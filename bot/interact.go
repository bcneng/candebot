package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/bcneng/candebot/slackx"

	"github.com/bcneng/candebot/cmd"
	"github.com/slack-go/slack"
)

var urlRegex = regexp.MustCompile(`(?mi)^(?:http(s)?:\/\/)?[\w.-]+(?:\.[\w\.-]+)+[\w\-\._~:/?#[\]@!\$&'\(\)\*\+,;=.]+$`)

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
		case slack.InteractionTypeDialogSubmission:
			switch message.CallbackID {
			case "job_submission":
				validationErrors := make(map[string]string)
				if !urlRegex.MatchString(message.Submission["job_link"]) {
					validationErrors["job_link"] = "The link to the job spec is invalid"
				}

				if strings.Contains(strings.ToLower(message.Submission["max_salary"]), "stock") {
					validationErrors["max_salary"] = "The Salary Max field cannot contain extras like mentions to Stock Options."
				}

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

				msg := fmt.Sprintf(":computer: %s @ %s - :moneybag: %s - %s - :round_pushpin: %s - :link: <%s|Link> - :raised_hands: More info DM <@%s>",
					message.Submission["role"],
					message.Submission["company"],
					message.Submission["min_salary"],
					message.Submission["max_salary"],
					message.Submission["location"],
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
	companyInput.Hint = "Links or special characters are not allowed"
	companyInput.MaxLength = 20
	companyInput.MinLength = 2

	salaryMinInput := slack.NewTextInput("min_salary", "Salary min", "")
	salaryMinInput.Optional = true
	salaryMinInput.Placeholder = "60K"
	salaryMinInput.Hint = "Use thousands. Links or special characters are not allowed"
	salaryMinInput.MaxLength = 10
	salaryMinInput.MinLength = 2

	salaryMaxInput := slack.NewTextInput("max_salary", "Salary max", "")
	salaryMaxInput.Placeholder = "90K"
	salaryMaxInput.Hint = "Use thousands. Links or special characters are not allowed"
	salaryMaxInput.Subtype = slack.InputSubtypeNumber
	salaryMaxInput.MaxLength = 10
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
			Label: "Remote",
			Value: "Remote",
		},
	}
	locationInput := slack.NewStaticSelectDialogInput("location", "Location - Select", options)
	locationInput.Optional = false

	// Open a dialog
	elements := []slack.DialogElement{
		roleInput,
		companyInput,
		salaryMinInput,
		salaryMaxInput,
		locationInput,
		linkInput,
	}
	return slack.Dialog{
		CallbackID:  "job_submission",
		Title:       "New Job Post",
		SubmitLabel: "Submit",
		Elements:    elements,
	}
}
