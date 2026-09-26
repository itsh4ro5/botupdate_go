package database

import (
	"github.com/itsh4ro5/botupdate/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

func TestPythonGoMigrationSafety(t *testing.T) {
	// Mock the initial Python document
	legacyID := "main_settings"
	pythonLegacyBSON := bson.D{
		{Key: "_id", Value: legacyID},
		{Key: "bot_token", Value: "123:ABC"},
		{Key: "unknown_python_field", Value: "should_survive_go"},
		{Key: "legacy_nested_map", Value: bson.D{{Key: "foo", Value: "bar"}}},
		{Key: "data", Value: bson.D{
			{Key: "admin_ids", Value: bson.A{int64(999), int64(888)}}, // Python stored arrays
			{Key: "blocked_users", Value: bson.A{int64(777)}},
			{Key: "legacy_flashcards", Value: bson.A{"fc1", "fc2"}}, // Unknown to Go
		}},
	}

	bytes, err := bson.Marshal(pythonLegacyBSON)
	if err != nil {
		t.Fatalf("Failed to marshal python BSON: %v", err)
	}

	// 1. Go Load (via Compat layer unmarshal)
	type MongoDoc struct {
		ID   string          `bson:"_id"`
		Data models.BotState `bson:"data"`
	}
	var doc MongoDoc
	err = bson.Unmarshal(bytes, &doc)
	if err != nil {
		t.Fatalf("Failed to unmarshal python BSON into MongoDoc: %v", err)
	}
	state := doc.Data

	// Verify Compat mapped Arrays -> Maps inside Go correctly
	if _, ok := state.AdminIDs[999]; !ok {
		t.Errorf("Compat failed to load admin_ids into map")
	}
	if _, ok := state.BlockedUsers[777]; !ok {
		t.Errorf("Compat failed to load blocked_users into map")
	}

	// 2. Go Mutation
	state.AdminIDs[555] = struct{}{}

	// 3. Go Save (safeFlatten phase)
	updateDoc, err := safeFlatten(&state)
	if err != nil {
		t.Fatalf("safeFlatten failed: %v", err)
	}

	// The safeFlatten produces a map[string]interface{} representing $set instructions
	// Ensure it targets "data.admin_ids" directly
	if _, ok := updateDoc["data.admin_ids"]; !ok {
		t.Errorf("safeFlatten did not map AdminIDs correctly, missed data.admin_ids")
	}

	// Ensure that it does NOT overwrite "data" as a whole object
	if _, ok := updateDoc["data"]; ok {
		t.Errorf("CRITICAL FAILURE: safeFlatten returned 'data' as a root key. This will DROP all legacy Python fields!")
	}

	// Ensure it uses the compat serializer for AdminIDs (arrays)
	adminIDsInterface := updateDoc["data.admin_ids"]
	adminIDsSlice, ok := adminIDsInterface.(primitive.A)
	if !ok {
		t.Errorf("CRITICAL FAILURE: safeFlatten did not serialize AdminIDs to primitive.A. It returned %T.", adminIDsInterface)
	}

	found555 := false
	found999 := false
	for _, idInterface := range adminIDsSlice {
		id, _ := idInterface.(int64)
		if id == 555 {
			found555 = true
		}
		if id == 999 {
			found999 = true
		}
	}
	if !found555 || !found999 {
		t.Errorf("Mutated admin_id 555 or existing 999 not found in serialized array")
	}
}
