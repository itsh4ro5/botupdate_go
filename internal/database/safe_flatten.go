package database

import (
	"github.com/itsh4ro5/botupdate/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

// safeFlatten converts BotState to a flattened map for $set
// Example: "data.admin_ids": state.AdminIDs
func safeFlatten(state *models.BotState) (bson.M, error) {
	b, err := bson.Marshal(state)
	if err != nil {
		return nil, err
	}
	var doc bson.M
	if err := bson.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	
	flattened := bson.M{}
	for k, v := range doc {
		flattened["data."+k] = v
	}
	return flattened, nil
}
