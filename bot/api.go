package bot

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/bcneng/candebot/slackx"
	"github.com/slack-go/slack"
)

type createChannelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PRLink      string `json:"pr_link"`
}

type createChannelResponse struct {
	ChannelID string `json:"channel_id"`
}

func apiCreateChannelHandler(botCtx Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !found || subtle.ConstantTimeCompare([]byte(token), []byte(botCtx.Config.APIKey)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req createChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Description == "" {
			http.Error(w, "name and description are required", http.StatusBadRequest)
			return
		}

		channel, err := botCtx.AdminClient.CreateConversation(slack.CreateConversationParams{
			ChannelName: req.Name,
			IsPrivate:   false,
		})
		if err != nil {
			log.Printf("[ERROR] Failed to create channel %q: %s", req.Name, err)
			http.Error(w, fmt.Sprintf("failed to create channel: %s", err), http.StatusInternalServerError)
			return
		}

		if _, err := botCtx.AdminClient.SetPurposeOfConversation(channel.ID, req.Description); err != nil {
			log.Printf("[WARN] Failed to set purpose for channel %q: %s", req.Name, err)
		}

		announcement := fmt.Sprintf("A new channel <#%s> has been created! :tada:\nDescription: %s", channel.ID, req.Description)
		if req.PRLink != "" {
			announcement += fmt.Sprintf("\nPull Request: <%s>", req.PRLink)
		}
		if err := slackx.Send(botCtx.AdminClient, "", botCtx.Config.Channels.General, announcement, false); err != nil {
			log.Printf("[WARN] Failed to announce new channel %q in #general: %s", req.Name, err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(createChannelResponse{ChannelID: channel.ID})
	}
}
