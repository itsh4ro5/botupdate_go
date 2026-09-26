package models

// MessageReactionUpdated represents a change of a reaction on a message performed by a user.
type MessageReactionUpdated struct {
	Chat        Chat           `json:"chat"`
	MessageID   int            `json:"message_id"`
	User        *User          `json:"user,omitempty"`
	ActorChat   *Chat          `json:"actor_chat,omitempty"`
	Date        int            `json:"date"`
	OldReaction []ReactionType `json:"old_reaction"`
	NewReaction []ReactionType `json:"new_reaction"`
}

// Chat represents a telegram chat.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// ReactionType represents a reaction type (emoji or custom_emoji).
type ReactionType struct {
	Type          string `json:"type"` // "emoji" or "custom_emoji"
	Emoji         string `json:"emoji,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}
