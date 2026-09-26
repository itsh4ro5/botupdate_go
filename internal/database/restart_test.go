package database

import (
	"context"
	"os"
	"testing"
)

func TestJSONStoreRestartRecovery(t *testing.T) {
	testFile := "test_state.json"
	defer os.Remove(testFile)

	store := NewJSONStore(testFile)
	ctx := context.Background()

	_, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// 2. Set mapping
	err = store.SetMessageMapping(ctx, "123:456", "789:101")
	if err != nil {
		t.Fatalf("Failed to set mapping: %v", err)
	}

	// 3. Set another mapping
	err = store.SetMessageMapping(ctx, "789:101", "123:456")
	if err != nil {
		t.Fatalf("Failed to set mapping: %v", err)
	}

	// 4. Simulate Restart (New Instance)
	store2 := NewJSONStore(testFile)
	state2, err := store2.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load on restart: %v", err)
	}

	// 5. Verify
	if val, ok := state2.MessageMap["123:456"]; !ok || val != "789:101" {
		t.Fatalf("Mapping 1 lost or corrupted! Expected 789:101, got %v", val)
	}
	if val, ok := state2.MessageMap["789:101"]; !ok || val != "123:456" {
		t.Fatalf("Mapping 2 lost or corrupted! Expected 123:456, got %v", val)
	}
}
