package models

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestLegacyBatchesUnmarshalBSON(t *testing.T) {
	raw := bson.M{
		"FREE_CHANNELS": bson.M{
			"123": "Python Batch",
			"456": bson.M{"id": int64(456), "name": "Go Batch", "type": "free", "category": "GoCat", "coins": int64(100)},
		},
		"BATCH_CATEGORIES": bson.M{"123": "PyCat"},
		"BATCH_COINS":      bson.M{"123": int64(50)},
		"CUSTOM_WELCOMES":  bson.M{"123": "Welcome to python"},
	}

	b, err := bson.Marshal(raw)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var state BotState
	if err := bson.Unmarshal(b, &state); err != nil {
		t.Fatalf("Failed to unmarshal BotState: %v", err)
	}

	if len(state.FreeBatches) != 2 {
		t.Fatalf("Expected 2 free batches, got %d", len(state.FreeBatches))
	}

	b1 := state.FreeBatches[123]
	if b1 == nil || b1.Name != "Python Batch" || b1.Category != "PyCat" || b1.WelcomeText != "Welcome to python" || b1.Type != "free" {
		t.Errorf("Legacy Python batch decoded incorrectly: %+v", b1)
	}

	b2 := state.FreeBatches[456]
	if b2 == nil || b2.Name != "Go Batch" || b2.Category != "GoCat" {
		t.Errorf("Go batch decoded incorrectly: %+v", b2)
	}
}
