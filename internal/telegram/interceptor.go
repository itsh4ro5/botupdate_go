package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/itsh4ro5/botupdate/internal/models"
)

// ReactionHandler defines the interface for handling parsed reaction updates
type ReactionHandler interface {
	HandleReaction(ctx context.Context, reaction *models.MessageReactionUpdated)
}

// UpdateInterceptor wraps an HTTP client to intercept Telegram getUpdates responses
// and parse message_reaction updates which are not natively supported by the older
// tgbotapi struct used in this project.
type UpdateInterceptor struct {
	Client  *http.Client
	Handler ReactionHandler
	Ctx     context.Context
}

func (i *UpdateInterceptor) Do(req *http.Request) (*http.Response, error) {
	resp, err := i.Client.Do(req)
	if err != nil {
		return resp, err
	}

	// Only intercept getUpdates requests
	if !strings.HasSuffix(req.URL.Path, "/getUpdates") {
		return resp, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	// Restore the response body so tgbotapi can read it as normal
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Parse the raw JSON payload to extract message_reaction
	var result struct {
		Ok     bool `json:"ok"`
		Result []struct {
			UpdateID        int                            `json:"update_id"`
			MessageReaction *models.MessageReactionUpdated `json:"message_reaction"`
		} `json:"result"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err == nil && result.Ok {
		for _, u := range result.Result {
			if u.MessageReaction != nil && i.Handler != nil && i.Ctx != nil {
				go i.Handler.HandleReaction(i.Ctx, u.MessageReaction)
			}
		}
	} else if err != nil {
		log.Printf("Failed to unmarshal raw updates for reaction sync: %v", err)
	}

	return resp, nil
}
