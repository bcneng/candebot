package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bcneng/candebot/slackx"

	"github.com/bcneng/candebot/cmd"
	"github.com/slack-go/slack"
)

func interactAPIHandler(botContext cmd.BotContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Fail to read request body: %v\n", err)
			return
		}
		defer r.Body.Close()

		if err := botContext.VerifyRequest(r, body); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("Fail to verify SigningSecret: %v", err)
		}

		str, _ := url.QueryUnescape(string(body))
		str = strings.Replace(str, "payload=", "", 1)
		var message slack.InteractionCallback
		if err := json.Unmarshal([]byte(str), &message); err != nil {
			log.Printf("Fail to unmarshal json: %v", err)
			return
		}

		switch message.Type {
		case slack.InteractionTypeMessageAction:
			switch message.CallbackID {
			case "report_message":
				dialog := generateReportMessageDialog()
				dialog.State = slackx.LinkToMessage(message.Channel.ID, message.MessageTs) // persist the message link across submission
				if err := botContext.Client.OpenDialog(message.TriggerID, dialog); err != nil {
					log.Println(err.Error())
				}
			}
		case slack.InteractionTypeDialogSubmission:
			switch message.CallbackID {
			case "report_message":
				msg := fmt.Sprintf("<@%s> sent a message report:\n- *Reason*: %s\n- *Feeling Scale*: %s of 5\n%s",
					message.User.Name,
					message.Submission["reason"],
					message.Submission["scale"],
					sanitizeReportState(message.State),
				)
				_ = slackx.Send(botContext.Client, "", channelStaff, msg, false)
			case "job_submission":

				messageJobLink := message.Submission["job_link"]
				messageMaxSalary := message.Submission["max_salary"]
				messageMinSalary := message.Submission["min_salary"]

				maxSalary, minSalary, validationErrors := validateSubmission(messageJobLink, messageMaxSalary, messageMinSalary)

				if len(validationErrors) > 0 {
					var errs []slack.DialogInputValidationError
					for f, err := range validationErrors {
						errs = append(errs, slack.DialogInputValidationError{
							Name:  f,
							Error: err,
						})
					}

					_ = json.NewEncoder(w).Encode(slack.DialogInputValidationErrors{
						Errors: errs,
					})

					return
				}

				minSalaryStr := fmt.Sprintf("%dK", minSalary)
				if minSalary == -1 {
					minSalaryStr = ""
				}

				msg := fmt.Sprintf(":computer: %s @ %s - :moneybag: %s - %dK %s - :round_pushpin: %s - :lower_left_fountain_pen: %s - :link: <%s|Link> - :raised_hands: More info DM <@%s>",
					message.Submission["role"],
					message.Submission["company"],
					minSalaryStr,
					maxSalary,
					message.Submission["currency"],
					message.Submission["location"],
					message.Submission["publisher"],
					message.Submission["job_link"],
					message.User.Name,
				)
				_ = slackx.Send(botContext.Client, "", channelHiringJobBoard, msg, false, slack.MsgOptionDisableLinkUnfurl())
			}
		case slack.InteractionTypeShortcut:
			switch message.CallbackID {
			case "submit_job":
				if err := botContext.Client.OpenDialog(message.TriggerID, generateSubmitJobFormDialog()); err != nil {
					log.Println(err.Error())
				}
			}
		}
	}
}

// validateSubmission runs validations over the submitted salary range and job offer link. Produces a list of errors if any.
//
// Arguments are the strings as read from the submissions (no previous transform/parsing/filter)
// Returns:
//  - the parsed max salary as int
//  - the parsed min salary as int, or -1 if the field was empty (it's an optional field)
//  - a map of field name to error message
func validateSubmission(messageJobLink, messageMaxSalary, messageMinSalary string) (int, int, map[string]string) {

	validationErrors := make(map[string]string)
	if _, err := url.ParseRequestURI(messageJobLink); err != nil {
		validationErrors["job_link"] = "The link to the job spec is invalid."
	}

	maxSalary, err := strconv.Atoi(strings.TrimSpace(messageMaxSalary))
	if err != nil || maxSalary == 0 {
		validationErrors["max_salary"] = "The Salary Max field should be a non-zero numeric value."
	} else if maxSalary < 0 {
		validationErrors["max_salary"] = "The Salary Max field should be positive numeric value."
	}

	minSalary := -1
	if minSalaryStr := strings.TrimSpace(messageMinSalary); minSalaryStr != "" {
		minSalary, err = strconv.Atoi(minSalaryStr)
		if err != nil || maxSalary == 0 {
			validationErrors["min_salary"] = "The Salary Min field, if specified, should be a non-zero numeric value."
		}

		if minSalary < 0 {
			validationErrors["min_salary"] = "The Salary Min field should be positive numeric value."
		}

		if minSalary > maxSalary {
			validationErrors["min_salary"] = "The Salary Min field should contain a lower value than the specified in Salary Max field."
		}

		if 2.5*float32(minSalary) < float32(maxSalary) {
			validationErrors["max_salary"] = "The min-max salary range is too wide. Salary is a relevant field; keep it meaningful to increase offer impact."
		}
	}
	return maxSalary, minSalary, validationErrors
}

func generateSubmitJobFormDialog() slack.Dialog {
	// Make new dialog components and open a dialog.
	// Component-Text
	roleInput := slack.NewTextInput("role", "Role", "")
	roleInput.Placeholder = "Software Engineer"
	roleInput.Hint = "Links or special characters are not allowed."
	roleInput.MaxLength = 50
	roleInput.MinLength = 2

	companyInput := slack.NewTextInput("company", "Company", "")
	companyInput.Placeholder = "BcnEng"
	companyInput.Hint = "It MUST be the final company name, no agencies/intermediaries allowed. Links or special characters are not allowed"
	companyInput.MaxLength = 20
	companyInput.MinLength = 2

	salaryCurrencyOptions := []slack.DialogSelectOption{{"EUR", "EUR"}, {"USD", "USD"}, {"GBP", "GBP"}, {"CHF", "CHF"}}
	salaryCurrencyInput := slack.NewStaticSelectDialogInput("currency", "Currency", salaryCurrencyOptions)
	salaryCurrencyInput.Optional = false
	salaryCurrencyInput.Hint = "Choose the salary currency from the dropdown"

	salaryMinInput := slack.NewTextInput("min_salary", "Salary min (per year)", "")
	salaryMinInput.Optional = true
	salaryMinInput.Placeholder = "60"
	salaryMinInput.Hint = "Use thousand abbreviation representation. Example: write 60 for 60,000 EUR. Only numbers allowed"
	salaryMinInput.Subtype = slack.InputSubtypeNumber
	salaryMinInput.MaxLength = 3
	salaryMinInput.MinLength = 2

	salaryMaxInput := slack.NewTextInput("max_salary", "Salary max (per year)", "")
	salaryMaxInput.Placeholder = "90"
	salaryMaxInput.Hint = "Use thousand abbreviation representation. Example: write 90 for 90,000 EUR. Only numbers allowed"
	salaryMaxInput.Subtype = slack.InputSubtypeNumber
	salaryMaxInput.MaxLength = 3
	salaryMaxInput.MinLength = 2

	linkInput := slack.NewTextInput("job_link", "Link to the job spec", "")
	linkInput.Hint = "Only valid links allowed"
	linkInput.MinLength = 5
	linkInput.Subtype = slack.InputSubtypeEmail

	// Component-Select menu
	options := []slack.DialogSelectOption{
		{
			Label: "Barcelona",
			Value: "Barcelona",
		},
		{
			Label: "Barcelona/Remote",
			Value: "Barcelona/Remote",
		},
		{
			Label: "Remote (temporary because of COVID)",
			Value: "Remote (COVID)",
		},
		{
			Label: "Remote",
			Value: "Remote",
		},
	}
	locationInput := slack.NewStaticSelectDialogInput("location", "Location - Select", options)
	locationInput.Optional = false

	publisherInput := buildPublisherInput()

	// Open a dialog
	elements := []slack.DialogElement{
		roleInput,
		companyInput,
		salaryMinInput,
		salaryMaxInput,
		salaryCurrencyInput,
		locationInput,
		linkInput,
		publisherInput,
	}
	return slack.Dialog{
		CallbackID:  "job_submission",
		Title:       "New Job Post",
		SubmitLabel: "Submit",
		Elements:    elements,
	}
}

func generateReportMessageDialog() slack.Dialog {
	reasonInput := slack.NewTextInput("reason", "Reason", "")
	reasonInput.Placeholder = "Violates BcnEng's COC by using a violent language"
	reasonInput.Hint = "Explain the reason of this report."
	reasonInput.MinLength = 5
	reasonInput.Optional = false

	feelingScale := slack.NewStaticSelectDialogInput("scale", "How hurtful their words felt to you?", []slack.DialogSelectOption{
		{Label: "1", Value: "1"},
		{Label: "2", Value: "2"},
		{Label: "3", Value: "3"},
		{Label: "4", Value: "4"},
		{Label: "5", Value: "5"},
	})
	feelingScale.Hint = "5 point scale ranging starting from 1 (minimum) to 5 (extremely), where a greater score corresponds to a more hurtful feeling"

	elements := []slack.DialogElement{
		reasonInput,
		feelingScale,
	}
	return slack.Dialog{
		CallbackID:  "report_message",
		Title:       "Report message",
		SubmitLabel: "Report",
		Elements:    elements,
	}
}

func buildPublisherInput() *slack.DialogInputSelect {
	publisherOptions := []slack.DialogSelectOption{
		{
			Label: "Employer",
			Value: "Employer",
		},
		{
			Label: "Agency",
			Value: "Agency",
		},
		{
			Label: "Referral",
			Value: "Referral",
		},
	}
	publisherInput := slack.NewStaticSelectDialogInput("publisher", "Published by", publisherOptions)
	publisherInput.Optional = false

	return publisherInput
}

func sanitizeReportState(state string) string {
	return strings.ReplaceAll(state, "\\/", "/")
}
