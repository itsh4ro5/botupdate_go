package bot

import (
	"context"
	"os"
	"testing"

	"github.com/itsh4ro5/botupdate/internal/database"
)

func TestAuthService(t *testing.T) {
	// Setup test DB
	tmpFile := "test_data.json"
	defer os.Remove(tmpFile)

	store := database.NewJSONStore(tmpFile)
	auth := NewAuthService(store, 789)
	ctx := context.Background()

	// Initially, load state to inject mock data
	state, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Expected no error on load: %v", err)
	}

	state.AdminIDs[123] = struct{}{}
	state.BlockedUsers[456] = struct{}{}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	if !auth.IsAdmin(ctx, 123) {
		t.Error("Expected 123 to be admin")
	}
	if auth.IsAdmin(ctx, 999) {
		t.Error("Expected 999 to NOT be admin")
	}

	if !auth.IsBlocked(ctx, 456) {
		t.Error("Expected 456 to be blocked")
	}
	if auth.IsBlocked(ctx, 123) {
		t.Error("Expected 123 to NOT be blocked")
	}

	if !auth.IsOwner(ctx, 789) {
		t.Error("Expected 789 to be owner")
	}
}
