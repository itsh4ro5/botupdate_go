package main
import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
)
func main() {
	doc := bson.M{"123": "user1"}
	b, _ := bson.Marshal(doc)
	var m map[int64]string
	err := bson.Unmarshal(b, &m)
	fmt.Printf("Parsed: %v, Err: %v\n", m, err)
}
