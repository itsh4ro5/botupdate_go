package models

import (
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"strconv"
)

// parseIntSetFromRaw handles legacy array format and newer map format
func parseIntSetFromJSON(raw json.RawMessage) (map[int64]struct{}, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return make(map[int64]struct{}), nil
	}
	
	res := make(map[int64]struct{})
	
	// Try parsing as array first (legacy Python list)
	var arr []int64
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, v := range arr {
			res[v] = struct{}{}
		}
		return res, nil
	}
	
	// Try parsing as map[string]struct{} or map[int64]struct{}
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

func parseIntSetFromBSON(raw bson.RawValue) (map[int64]struct{}, error) {
	if len(raw.Value) == 0 || raw.Type == bsontype.Null {
		return make(map[int64]struct{}), nil
	}

	res := make(map[int64]struct{})

	// Legacy Array
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

	// Map
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

// UnmarshalJSON parses BotState, safely handling legacy lists
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

	if aux.AdminIDs != nil {
		parsed, err := parseIntSetFromJSON(aux.AdminIDs)
		if err != nil {
			return fmt.Errorf("admin_ids decoding failed: %v", err)
		}
		b.AdminIDs = parsed
	} else {
		b.AdminIDs = make(map[int64]struct{})
	}

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

// UnmarshalBSON parses BotState, safely handling legacy arrays
func (b *BotState) UnmarshalBSON(data []byte) error {
	type Alias BotState
	aux := &struct {
		AdminIDs     bson.RawValue `bson:"admin_ids"`
		BlockedUsers bson.RawValue `bson:"blocked_users"`
		Alias        `bson:",inline"`
	}{
		Alias: (Alias)(*b),
	}

	if err := bson.Unmarshal(data, &aux); err != nil {
		return err
	}
	*b = (BotState)(aux.Alias)

	parsedAdmins, err := parseIntSetFromBSON(aux.AdminIDs)
	if err != nil {
		return fmt.Errorf("admin_ids decoding failed: %v", err)
	}
	b.AdminIDs = parsedAdmins

	parsedBlocked, err := parseIntSetFromBSON(aux.BlockedUsers)
	if err != nil {
		return fmt.Errorf("blocked_users decoding failed: %v", err)
	}
	b.BlockedUsers = parsedBlocked

	return nil
}

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
		AdminIDs     []int64 `bson:"admin_ids"`
		BlockedUsers []int64 `bson:"blocked_users"`
		Alias        `bson:",inline"`
	}{
		AdminIDs:     adminIDsArray,
		BlockedUsers: blockedUsersArray,
		Alias:        (Alias)(*b),
	})
}
