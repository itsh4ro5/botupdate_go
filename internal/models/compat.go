package models

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// parseIntSetFromJSON handles legacy array format and newer map format.
func parseIntSetFromJSON(raw json.RawMessage) (map[int64]struct{}, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return make(map[int64]struct{}), nil
	}

	res := make(map[int64]struct{})

	// Try parsing as array first (legacy Python list).
	var arr []int64
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, v := range arr {
			res[v] = struct{}{}
		}
		return res, nil
	}

	// Try parsing as map[string]interface{}.
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err == nil {
		for k := range m {
			if id, err := strconv.ParseInt(k, 10, 64); err == nil {
				res[id] = struct{}{}
			}
		}
		return res, nil
	}

	return nil, fmt.Errorf("failed to parse IntSet from JSON")
}

// parseIntSetFromBSON handles legacy array format and newer map format.
func parseIntSetFromBSON(raw bson.RawValue) (map[int64]struct{}, error) {
	if len(raw.Value) == 0 || raw.Type == bsontype.Null {
		return make(map[int64]struct{}), nil
	}

	res := make(map[int64]struct{})

	// Legacy Array.
	if raw.Type == bsontype.Array {
		var arr bson.A
		if err := raw.Unmarshal(&arr); err == nil {
			for _, v := range arr {
				switch val := v.(type) {
				case int32:
					res[int64(val)] = struct{}{}
				case int64:
					res[val] = struct{}{}
				case float64:
					res[int64(val)] = struct{}{}
				}
			}
			return res, nil
		}
	}

	// Map.
	if raw.Type == bsontype.EmbeddedDocument {
		var m bson.M
		if err := raw.Unmarshal(&m); err == nil {
			for k := range m {
				if id, err := strconv.ParseInt(k, 10, 64); err == nil {
					res[id] = struct{}{}
				}
			}
			return res, nil
		}
	}

	return nil, fmt.Errorf("failed to parse IntSet from BSON: unknown format")
}



// UnmarshalJSON parses BotState, safely handling legacy lists.
func (b *BotState) UnmarshalJSON(data []byte) error {
	type Alias BotState

	aux := &struct {
		AdminIDs     json.RawMessage `json:"admin_ids"`
		BlockedUsers json.RawMessage `json:"blocked_users"`
		Alias        `json:",inline"`
	}{
		Alias: (Alias)(*b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*b = (BotState)(aux.Alias)

	// Admin IDs.
	if aux.AdminIDs != nil {
		parsed, err := parseIntSetFromJSON(aux.AdminIDs)
		if err != nil {
			return fmt.Errorf("admin_ids decoding failed: %v", err)
		}
		b.AdminIDs = parsed
	} else {
		b.AdminIDs = make(map[int64]struct{})
	}

	// Blocked users.
	if aux.BlockedUsers != nil {
		parsed, err := parseIntSetFromJSON(aux.BlockedUsers)
		if err != nil {
			return fmt.Errorf("blocked_users decoding failed: %v", err)
		}
		b.BlockedUsers = parsed
	} else {
		b.BlockedUsers = make(map[int64]struct{})
	}

	return nil
}

// UnmarshalBSON parses BotState and safely handles legacy BSON formats.
func (b *BotState) UnmarshalBSON(data []byte) error {
	type Alias BotState

	aux := &struct {
		AdminIDs          bson.RawValue `bson:"ADMIN_IDS"`
		BlockedUsers      bson.RawValue `bson:"BLOCKED_USERS"`
		FreeBatchesRaw    bson.RawValue `bson:"FREE_CHANNELS"`
		PaidBatchesRaw    bson.RawValue `bson:"PAID_CHANNELS"`
		SpecialBatchesRaw bson.RawValue `bson:"SPECIAL_CHANNELS"`
		Alias             `bson:",inline"`
	}{
		Alias: (Alias)(*b),
	}

	if err := bson.Unmarshal(data, &aux); err != nil {
		return err
	}

	*b = (BotState)(aux.Alias)

	// ---------------------------------------------------------
	// Admin IDs
	// ---------------------------------------------------------
	parsedAdmins, err := parseIntSetFromBSON(aux.AdminIDs)
	if err != nil {
		return fmt.Errorf("admin_ids decoding failed: %v", err)
	}
	b.AdminIDs = parsedAdmins

	// ---------------------------------------------------------
	// Blocked users
	// ---------------------------------------------------------
	parsedBlocked, err := parseIntSetFromBSON(aux.BlockedUsers)
	if err != nil {
		return fmt.Errorf("blocked_users decoding failed: %v", err)
	}
	b.BlockedUsers = parsedBlocked

	// ---------------------------------------------------------
	// Legacy Batches
	// ---------------------------------------------------------
	b.FreeBatches = parseLegacyBatches(aux.FreeBatchesRaw, b.BatchCategories, b.CustomWelcomes, b.BatchCoins, "free")
	b.PaidBatches = parseLegacyBatches(aux.PaidBatchesRaw, b.BatchCategories, b.CustomWelcomes, b.BatchCoins, "paid")
	b.SpecialBatches = parseLegacyBatches(aux.SpecialBatchesRaw, b.BatchCategories, b.CustomWelcomes, b.BatchCoins, "special")

	return nil
}

func parseLegacyBatches(raw bson.RawValue, categories map[int64]string, welcomes map[int64]string, coins map[int64]int64, batchType string) map[int64]*Batch {
	if raw.Type != bsontype.EmbeddedDocument {
		return make(map[int64]*Batch)
	}

	m := make(map[string]bson.RawValue)
	if err := raw.Unmarshal(&m); err != nil {
		return make(map[int64]*Batch)
	}

	batches := make(map[int64]*Batch)
	for k, v := range m {
		id, _ := strconv.ParseInt(k, 10, 64)

		if v.Type == bsontype.String {
			// Legacy Python string format
			batches[id] = &Batch{
				ID:          id,
				Name:        v.StringValue(),
				Type:        batchType,
				Category:    categories[id],
				WelcomeText: welcomes[id],
			}
		} else if v.Type == bsontype.EmbeddedDocument {
			// New Go struct format
			var batch Batch
			_ = v.Unmarshal(&batch)
			batches[id] = &batch
		}
	}
	return batches
}

// MarshalJSON serializes BotState while keeping AdminIDs and
// BlockedUsers compatible with the legacy JSON representation.
func (b *BotState) MarshalJSON() ([]byte, error) {
	type Alias BotState

	adminIDsArray := make([]int64, 0, len(b.AdminIDs))
	for id := range b.AdminIDs {
		adminIDsArray = append(adminIDsArray, id)
	}

	blockedUsersArray := make([]int64, 0, len(b.BlockedUsers))
	for id := range b.BlockedUsers {
		blockedUsersArray = append(blockedUsersArray, id)
	}

	return json.Marshal(&struct {
		AdminIDs     []int64 `json:"admin_ids"`
		BlockedUsers []int64 `json:"blocked_users"`
		Alias        `json:",inline"`
	}{
		AdminIDs:     adminIDsArray,
		BlockedUsers: blockedUsersArray,
		Alias:        (Alias)(*b),
	})
}

// MarshalBSON serializes BotState while keeping AdminIDs and
// BlockedUsers in the normalized BSON representation.
func (b *BotState) MarshalBSON() ([]byte, error) {
	type Alias BotState

	adminIDsArray := make([]int64, 0, len(b.AdminIDs))
	for id := range b.AdminIDs {
		adminIDsArray = append(adminIDsArray, id)
	}

	blockedUsersArray := make([]int64, 0, len(b.BlockedUsers))
	for id := range b.BlockedUsers {
		blockedUsersArray = append(blockedUsersArray, id)
	}

	return bson.Marshal(&struct {
		AdminIDs     []int64 `bson:"ADMIN_IDS"`
		BlockedUsers []int64 `bson:"BLOCKED_USERS"`
		Alias        `bson:",inline"`
	}{
		AdminIDs:     adminIDsArray,
		BlockedUsers: blockedUsersArray,
		Alias:        (Alias)(*b),
	})
}

// UnmarshalBSON safely handles floats for JoinedAt since Python uses time.time()
// and converts ints to strings for UnlockedBatches
func (u *User) UnmarshalBSON(data []byte) error {
	type Alias User
	aux := &struct {
		JoinedAt        bson.RawValue `bson:"joined_at"`
		UnlockedBatches bson.RawValue `bson:"unlocked_batches"`
		DemoHistory     bson.RawValue `bson:"demo_history"`
		Alias           `bson:",inline"`
	}{
		Alias: (Alias)(*u),
	}

	if err := bson.Unmarshal(data, &aux); err != nil {
		return err
	}

	*u = (User)(aux.Alias)

	if aux.JoinedAt.Type == bsontype.Double {
		var f float64
		_ = aux.JoinedAt.Unmarshal(&f)
		u.JoinedAt = int64(f)
	} else if aux.JoinedAt.Type == bsontype.Int64 {
		_ = aux.JoinedAt.Unmarshal(&u.JoinedAt)
	} else if aux.JoinedAt.Type == bsontype.Int32 {
		var i int32
		_ = aux.JoinedAt.Unmarshal(&i)
		u.JoinedAt = int64(i)
	}

	if aux.UnlockedBatches.Type == bsontype.Array {
		var arr bson.A
		if err := aux.UnlockedBatches.Unmarshal(&arr); err == nil {
			var parsed []string
			for _, v := range arr {
				switch val := v.(type) {
				case string:
					parsed = append(parsed, val)
				case int32:
					parsed = append(parsed, fmt.Sprintf("%d", val))
				case int64:
					parsed = append(parsed, fmt.Sprintf("%d", val))
				case float64:
					parsed = append(parsed, fmt.Sprintf("%.0f", val))
				}
			}
			u.UnlockedBatches = parsed
		}
	}

	if aux.DemoHistory.Type == bsontype.Array {
		var arr bson.A
		if err := aux.DemoHistory.Unmarshal(&arr); err == nil {
			var parsed []string
			for _, v := range arr {
				switch val := v.(type) {
				case string:
					parsed = append(parsed, val)
				case int32:
					parsed = append(parsed, fmt.Sprintf("%d", val))
				case int64:
					parsed = append(parsed, fmt.Sprintf("%d", val))
				case float64:
					parsed = append(parsed, fmt.Sprintf("%.0f", val))
				}
			}
			u.DemoHistory = parsed
		}
	}
	return nil
}
