package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/bcneng/candebot/slackx"
	"github.com/newrelic/newrelic-telemetry-sdk-go/telemetry"
	"github.com/slack-go/slack"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func interactAPIHandler(botContext Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch message.Type {
		case slack.InteractionTypeMessageAction:
			switch message.CallbackID {
			case "report_message":
				dialog := generateReportMessageDialog()
				dialog.State = slackx.LinkToMessage(message.Channel.ID, message.MessageTs) // persist the message link across submission
				if err := botContext.Client.OpenDialog(message.TriggerID, dialog); err != nil {
					log.Println(err)
				}
			case "delete_job_post":
				modal := generateDeleteJobPostModal()
				modal.PrivateMetadata = fmt.Sprintf("%s|%s|%s", message.Channel.ID, message.Message.Text, message.MessageTs) // persist the message channel, text, and ts across submission
				if resp, err := botContext.Client.OpenView(message.TriggerID, modal); err != nil {
					logModalError(err, resp)
				}
			case "delete_thread":
				if !botContext.IsStaff(message.User.ID) {
					if resp, err := botContext.Client.OpenView(message.TriggerID, userNotAllowedModal()); err != nil {
						logModalError(err, resp)
					}
					log.Printf("The user @%s (%s) is trying to execute the message action `delete_thread` and it doesn't have permissions", message.User.Name, message.User.ID)
					break
				}

				modal := generateDeleteThreadModal()
				modal.PrivateMetadata = fmt.Sprintf("%s|%s", message.Channel.ID, message.MessageTs) // persist the message channel and ts across submission
				if resp, err := botContext.Client.OpenView(message.TriggerID, modal); err != nil {
					logModalError(err, resp)
				}

			}
		case slack.InteractionTypeViewSubmission:
			switch message.View.CallbackID {
			case "delete_job_post":
				// We early set the Content-Type header for any response. This is important.
				w.Header().Set("Content-Type", "application/json")

				messageData := strings.Split(strings.Trim(message.View.PrivateMetadata, `"`), "|") // For some reason, slack adds an extra double quote
				channelID := messageData[0]
				messageText := messageData[1]
				messageTS := messageData[2]

				if channelID != botContext.Config.Channels.Jobs {
					_ = json.NewEncoder(w).Encode(
						slack.NewErrorsViewSubmissionResponse(map[string]string{"input_block": "The message is not a valid #hiring-job-board job post"}),
					)

					return
				}

				if !strings.Contains(messageText, fmt.Sprintf(":raised_hands: More info DM <@%s>", message.User.ID)) {
					_ = json.NewEncoder(w).Encode(
						slack.NewErrorsViewSubmissionResponse(map[string]string{"input_block": "You are not the author of this job post"}),
					)

					return
				}

				if _, _, err := botContext.AdminClient.DeleteMessage(channelID, messageTS); err != nil {
					log.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				_ = json.NewEncoder(w).Encode(slack.NewClearViewSubmissionResponse())

				log.Println("Job post message deleted successfully", message.View.PrivateMetadata)

				// Sending metrics
				botContext.Harvester.RecordMetric(telemetry.Count{
					Name:      fmt.Sprintf("%s_%s", strings.ToLower(botContext.Config.Bot.Name), "job_post.deleted"),
					Value:     1,
					Timestamp: time.Now(),
				})
			case "delete_thread":
				// We early set the Content-Type header for any response. This is important.
				w.Header().Set("Content-Type", "application/json")

				if !botContext.IsStaff(message.User.ID) {
					_ = json.NewEncoder(w).Encode(
						slack.NewErrorsViewSubmissionResponse(map[string]string{"input_block": "You are not allowed to delete threads. Please contact any Staff member."}),
					)
					return
				}

				messageData := strings.Split(strings.Trim(message.View.PrivateMetadata, `"`), "|") // For some reason, slack adds an extra double quote
				channelID := messageData[0]
				messageTS := messageData[1]

				repliesParams := &slack.GetConversationRepliesParameters{
					ChannelID: channelID,
					Timestamp: messageTS,
				}

				var threadMessages []slack.Message
				var cursor = ""
				var more = true

				for more {
					repliesParams.Cursor = cursor
					var replies []slack.Message
					replies, more, cursor, err = botContext.Client.GetConversationReplies(repliesParams)
					if err != nil {
						_ = json.NewEncoder(w).Encode(slack.NewErrorsViewSubmissionResponse(map[string]string{"input_block": err.Error()}))
						return
					}

					threadMessages = append(threadMessages, replies...)
				}

				go func() {
					deleted, ok := deleteThreadMessages(botContext, threadMessages, channelID)
					if !ok {
						log.Println("Thread deletion finished with errors (see logs)", message.View.PrivateMetadata)
					} else {
						log.Printf("Thread deletion finished successfully, %d messages where removed, including parent message: %s", deleted, message.View.PrivateMetadata)
					}

					//Sending metrics
					botContext.Harvester.RecordMetric(telemetry.Count{
						Name: fmt.Sprintf("%s.%s", strings.ToLower(botContext.Config.Bot.Name), "thread.deleted"),
						Attributes: map[string]interface{}{
							"errored": !ok,
							"deleted": deleted,
						},
						Value:     1,
						Timestamp: time.Now(),
					})
				}()

				_ = json.NewEncoder(w).Encode(slack.NewClearViewSubmissionResponse())
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
				_ = slackx.Send(botContext.Client, "", botContext.Config.Channels.Staff, msg, false)

				// Sending metrics
				botContext.Harvester.RecordMetric(telemetry.Count{
					Name: fmt.Sprintf("%s.%s", strings.ToLower(botContext.Config.Bot.Name), "report_message.received"),
					Attributes: map[string]interface{}{
						"scale": message.Submission["scale"],
					},
					Value:     1,
					Timestamp: time.Now(),
				})
			case "job_submission":
				link, maxSalary, minSalary, validationErrors := validateSubmission(message.Submission["job_link"], message.Submission["max_salary"], message.Submission["min_salary"])
				if link != nil && link.Query().Get("utm_source") == "" {
					// Add utm_source to the job link only if doesn't have one already
					query := link.Query()
					query.Add("utm_source", "bcneng")
					link.RawQuery = query.Encode()
					message.Submission["job_link"] = link.String()
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
				_ = slackx.Send(botContext.Client, "", botContext.Config.Channels.Jobs, msg, false, slack.MsgOptionDisableLinkUnfurl())

				// Sending metrics
				botContext.Harvester.RecordMetric(telemetry.Count{
					Name: fmt.Sprintf("%s.%s", strings.ToLower(botContext.Config.Bot.Name), "job_post.published"),
					Attributes: map[string]interface{}{
						"role":      cases.Title(language.English).String(strings.ToLower(message.Submission["role"])),
						"company":   cases.Title(language.English).String(strings.ToLower(message.Submission["company"])),
						"minSalary": minSalary,
						"maxSalary": maxSalary,
						"currency":  message.Submission["currency"],
						"location":  message.Submission["location"],
						"publisher": message.Submission["publisher"],
						"job_link":  message.Submission["job_link"],
						"user":      message.User.Name,
					},
					Value:     1,
					Timestamp: time.Now(),
				})
			}
		case slack.InteractionTypeShortcut:
			switch message.CallbackID {
			case "submit_job":
				if err := botContext.Client.OpenDialog(message.TriggerID, generateSubmitJobFormDialog()); err != nil {
					log.Println(err)
				}
			case "suggest_channel":
				if resp, err := botContext.Client.OpenView(message.TriggerID, suggestChannelModal()); err != nil {
					logModalError(err, resp)
				}
			}
		}
	}
}

func logModalError(err error, resp *slack.ViewResponse) {
	log.Println(err)
	log.Println(strings.Join(resp.ResponseMetadata.Messages, "\n"))
	log.Println(strings.Join(resp.ResponseMetadata.Warnings, "\n"))
}

func deleteThreadMessages(botContext Context, threadMessages []slack.Message, channelID string) (int, bool) {
	var errored bool
	var deleted int
	for _, threadMessage := range threadMessages {
		deleteMessageErr := retry.Do(
			func() error {
				_, _, err := botContext.AdminClient.DeleteMessage(channelID, threadMessage.Timestamp)
				if err != nil {
					switch actualErr := err.(type) {
					case *slack.RateLimitedError:
						log.Printf("Rate limit reached (Slack Web API Rate Limit: Tier 3. 50+ per minute). Waiting %s for making next request...", actualErr.RetryAfter)
						time.Sleep(actualErr.RetryAfter)

						return err
					}

					return retry.Unrecoverable(err) // Only retry if rate limited
				}

				deleted++
				return nil
			},
			retry.Attempts(0), // unlimited unless the error is not rate limit reached
			retry.OnRetry(func(_ uint, err error) {
				log.Printf("Retrying thread message %s deletion because of err: %s", threadMessage.Timestamp, err)
			}),
		)

		if deleteMessageErr != nil {
			errored = true
			log.Printf("Thread message %s deletion errored: %s", threadMessage.Timestamp, deleteMessageErr)
		}

	}
	return deleted, !errored
}

// validateSubmission runs validations over the submitted salary range and job offer link. Produces a list of errors if any.
//
// Arguments are the strings as read from the submissions (no previous transform/parsing/filter)
// Returns:
//   - the parsed URL. Nil if invalid
//   - the parsed max salary as int
//   - the parsed min salary as int, or -1 if the field was empty (it's an optional field)
//   - a map of field name to error message
func validateSubmission(messageJobLink, messageMaxSalary, messageMinSalary string) (*url.URL, int, int, map[string]string) {
	validationErrors := make(map[string]string)
	link, linkErr := url.ParseRequestURI(messageJobLink)
	if linkErr != nil {
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
	return link, maxSalary, minSalary, validationErrors
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
	companyInput.Hint = "It MUST be the final company name, no name of agencies/intermediaries allowed. Links or special characters are not allowed"
	companyInput.MaxLength = 20
	companyInput.MinLength = 2

	salaryCurrencyOptions := []slack.DialogSelectOption{{"EUR", "EUR"}, {"USD", "USD"}, {"GBP", "GBP"}, {"CHF", "CHF"}} // nolint: govet
	salaryCurrencyInput := slack.NewStaticSelectDialogInput("currency", "Currency", salaryCurrencyOptions)
	salaryCurrencyInput.Optional = false
	salaryCurrencyInput.Hint = "Choose the salary currency from the dropdown"

	salaryMinInput := slack.NewTextInput("min_salary", "Salary min (yearly fix income;no variable/bonus)", "")
	salaryMinInput.Optional = true
	salaryMinInput.Placeholder = "60"
	salaryMinInput.Hint = "Use thousand abbreviation representation. Example: write 60 for 60,000 EUR. Only numbers allowed"
	salaryMinInput.Subtype = slack.InputSubtypeNumber
	salaryMinInput.MaxLength = 3
	salaryMinInput.MinLength = 2

	salaryMaxInput := slack.NewTextInput("max_salary", "Salary max (yearly fix income;no variable/bonus)", "")
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

func generateDeleteJobPostModal() slack.ModalViewRequest {
	checkBoxOptionText := slack.NewTextBlockObject("plain_text", "I am sure I want to delete the selected job post", false, false)
	checkBoxDescriptionText := slack.NewTextBlockObject("plain_text", "By selecting this, you confirm to be the author of the selected job post, and to understand that the content of it is going to be deleted permanently", false, false)
	checkbox := slack.NewCheckboxGroupsBlockElement("some_action", slack.NewOptionBlockObject("confirmed", checkBoxOptionText, checkBoxDescriptionText))
	block := slack.NewInputBlock("input_block", slack.NewTextBlockObject(slack.PlainTextType, " ", false, false), nil, checkbox)

	return slack.ModalViewRequest{
		Type:  slack.VTModal,
		Title: slack.NewTextBlockObject(slack.PlainTextType, "Confirm deletion", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			block,
		}},
		Submit:     slack.NewTextBlockObject(slack.PlainTextType, "Confirm", false, false),
		CallbackID: "delete_job_post",
	}
}

func generateDeleteThreadModal() slack.ModalViewRequest {
	checkBoxOptionText := slack.NewTextBlockObject("plain_text", "I am sure I want to delete the thread", false, false)
	checkBoxDescriptionText := slack.NewTextBlockObject("plain_text", "By selecting this, you confirm you understand that whole thread is going to be deleted permanently", false, false)
	checkbox := slack.NewCheckboxGroupsBlockElement("some_action", slack.NewOptionBlockObject("confirmed", checkBoxOptionText, checkBoxDescriptionText))
	block := slack.NewInputBlock("input_block", slack.NewTextBlockObject(slack.PlainTextType, " ", false, false), nil, checkbox)
	noticeBlock := slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "Note that this could take some time due to <https://api.slack.com/docs/rate-limits|Slack Web API Rate Limit limitations: Tier 3. (50+ per minute)>. The execution will take place in a background process.", false, false), nil, nil)

	return slack.ModalViewRequest{
		Type:  slack.VTModal,
		Title: slack.NewTextBlockObject(slack.PlainTextType, "Confirm deletion", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			block,
			noticeBlock,
		}},
		Submit:     slack.NewTextBlockObject(slack.PlainTextType, "Confirm", false, false),
		CallbackID: "delete_thread",
	}
}

func userNotAllowedModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:  slack.VTModal,
		Title: slack.NewTextBlockObject(slack.PlainTextType, "Not allowed", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject("plain_text", "You are not allowed to perform this action.", false, false),
				nil,
				nil,
			),
		}},
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

func suggestChannelModal() slack.ModalViewRequest {
	text := "To suggest a new channel, edit the channels file and submit a Pull Request:\n\n" +
		"<https://github.com/bcneng/website/edit/main/data/channels.json|:pencil: Edit channels.json on GitHub>"

	return slack.ModalViewRequest{
		Type:  slack.VTModal,
		Title: slack.NewTextBlockObject(slack.PlainTextType, "Suggest new channel", false, false),
		Blocks: slack.Blocks{BlockSet: []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, text, false, false),
				nil,
				nil,
			),
		}},
	}
}

func sanitizeReportState(state string) string {
	return strings.ReplaceAll(state, "\\/", "/")
}
