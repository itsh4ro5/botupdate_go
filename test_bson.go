package main

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/itsh4ro5/botupdate/internal/models"
)

func main() {
	// Try array to map
	doc := bson.M{
		"admin_ids":     bson.A{int64(123), int64(456)},
		"blocked_users": bson.A{int64(789)},
	}
	b, err := bson.Marshal(doc)
	if err != nil {
		panic(err)
	}
	
	var state models.BotState
	err = bson.Unmarshal(b, &state)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("State AdminIDs: %+v\n", state.AdminIDs)
		fmt.Printf("State BlockedUsers: %+v\n", state.BlockedUsers)
	}
}
