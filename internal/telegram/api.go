package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// APIClient is a helper to make raw requests for methods missing from tgbotapi
type APIClient struct {
	Token string
}

func NewAPIClient(token string) *APIClient {
	return &APIClient{Token: token}
}

func (c *APIClient) doRequest(method string, payload interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.Token, method)

	var buf bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return nil, err
		}
	}

	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Ok          bool            `json:"ok"`
		ErrorCode   int             `json:"error_code"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Ok {
		return nil, fmt.Errorf("telegram API error (%d): %s", result.ErrorCode, result.Description)
	}

	if len(result.Result) > 0 {
		if string(result.Result) == "true" || string(result.Result) == "false" {
			return nil, nil
		}
		var mapRes map[string]interface{}
		if err := json.Unmarshal(result.Result, &mapRes); err == nil {
			return mapRes, nil
		}
	}

	return nil, nil
}

// CreateForumTopic creates a new forum topic and returns the message thread ID
func (c *APIClient) CreateForumTopic(chatID interface{}, title string) (int, error) {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"name":    title,
	}

	res, err := c.doRequest("createForumTopic", payload)
	if err != nil {
		return 0, err
	}

	if threadID, ok := res["message_thread_id"].(float64); ok {
		return int(threadID), nil
	}
	return 0, fmt.Errorf("could not extract message_thread_id from response")
}

// CreateChatInviteLink creates a new invite link
func (c *APIClient) CreateChatInviteLink(chatID interface{}, expireDate int64, memberLimit int, createsJoinRequest bool) (string, error) {
	payload := map[string]interface{}{
		"chat_id": chatID,
	}
	if expireDate > 0 {
		payload["expire_date"] = expireDate
	}
	if memberLimit > 0 {
		payload["member_limit"] = memberLimit
	}
	if createsJoinRequest {
		payload["creates_join_request"] = true
	}

	res, err := c.doRequest("createChatInviteLink", payload)
	if err != nil {
		return "", err
	}

	if link, ok := res["invite_link"].(string); ok {
		return link, nil
	}
	return "", fmt.Errorf("could not extract invite_link from response")
}

// RevokeChatInviteLink revokes an existing invite link
func (c *APIClient) RevokeChatInviteLink(chatID interface{}, inviteLink string) error {
	payload := map[string]interface{}{
		"chat_id":     chatID,
		"invite_link": inviteLink,
	}
	_, err := c.doRequest("revokeChatInviteLink", payload)
	return err
}

// ApproveChatJoinRequest approves a user's join request
func (c *APIClient) ApproveChatJoinRequest(chatID interface{}, userID int64) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"user_id": userID,
	}
	_, err := c.doRequest("approveChatJoinRequest", payload)
	return err
}

// DeclineChatJoinRequest declines a user's join request
func (c *APIClient) DeclineChatJoinRequest(chatID interface{}, userID int64) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"user_id": userID,
	}
	_, err := c.doRequest("declineChatJoinRequest", payload)
	return err
}

// BanChatMember kicks (bans) a user
func (c *APIClient) BanChatMember(chatID interface{}, userID int64, untilDate int64, revokeMessages bool) error {
	payload := map[string]interface{}{
		"chat_id":         chatID,
		"user_id":         userID,
		"revoke_messages": revokeMessages,
	}
	if untilDate > 0 {
		payload["until_date"] = untilDate
	}
	_, err := c.doRequest("banChatMember", payload)
	return err
}

// UnbanChatMember unbans a user (allows them to rejoin)
func (c *APIClient) UnbanChatMember(chatID interface{}, userID int64, onlyIfBanned bool) error {
	payload := map[string]interface{}{
		"chat_id":        chatID,
		"user_id":        userID,
		"only_if_banned": onlyIfBanned,
	}
	_, err := c.doRequest("unbanChatMember", payload)
	return err
}

// GetChatMember gets member info
func (c *APIClient) GetChatMember(chatID interface{}, userID int64) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"user_id": userID,
	}
	return c.doRequest("getChatMember", payload)
}

// ParseChatID safely parses chat IDs
func ParseChatID(id string) interface{} {
	if intID, err := strconv.ParseInt(id, 10, 64); err == nil {
		return intID
	}
	return id
}

// DeleteMessage deletes a message in a chat
func (c *APIClient) DeleteMessage(chatID interface{}, messageID int) error {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	_, err := c.doRequest("deleteMessage", payload)
	return err
}
