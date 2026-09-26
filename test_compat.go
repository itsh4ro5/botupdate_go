package main
import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/itsh4ro5/botupdate/internal/models"
)
func main() {
	doc := bson.M{
		"admin_ids": bson.A{int64(999)},
		"users": bson.M{"123": bson.M{"id": int64(123), "username": "foo"}},
	}
	b, _ := bson.Marshal(doc)

	var state models.BotState
	type Alias models.BotState
	aux := &struct {
		AdminIDs     bson.RawValue `bson:"admin_ids"`
		Alias        `bson:",inline"`
	}{
		Alias: (Alias)(state),
	}
	err := bson.Unmarshal(b, &aux)
	state = (models.BotState)(aux.Alias)
	fmt.Printf("Users: %d, AdminIDs: %v, Err: %v\n", len(state.Users), state.AdminIDs, err)
}
