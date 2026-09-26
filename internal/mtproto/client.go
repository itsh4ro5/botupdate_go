package mtproto

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/itsh4ro5/botupdate/config"
)

// Client wraps the gotd telegram client for userbot operations
type Client struct {
	apiClient *telegram.Client
	cfg       *config.Config
}

// NewClient initializes the MTProto client
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.APIID == 0 || cfg.APIHash == "" || cfg.SessionString == "" {
		return nil, fmt.Errorf("MTProto credentials missing: requires API_ID, API_HASH, and SESSION_STRING")
	}

	// Create client with session storage
	// We use the basic initialization for now
	client := telegram.NewClient(int(cfg.APIID), cfg.APIHash, telegram.Options{})

	return &Client{
		apiClient: client,
		cfg:       cfg,
	}, nil
}

// Run executes a callback within an active MTProto session safely
func (c *Client) Run(ctx context.Context, callback func(ctx context.Context, api *tg.Client) error) error {
	return c.apiClient.Run(ctx, func(ctx context.Context) error {
		// Attempt to auth using session string (Not fully mapped, mocked for interface completeness)
		status, err := c.apiClient.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to check auth status: %w", err)
		}
		if !status.Authorized {
			return fmt.Errorf("userbot session is not authorized")
		}

		api := c.apiClient.API()
		return callback(ctx, api)
	})
}
