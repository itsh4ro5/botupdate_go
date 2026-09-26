package models

import (
	"testing"
	"go.mongodb.org/mongo-driver/bson"
)

func TestSupportTopicUnmarshalBSONValue(t *testing.T) {
	tests := []struct {
		name          string
		bsonRaw       bson.M
		expectedTopic int
	}{
		{
			name: "Legacy int32",
			bsonRaw: bson.M{
				"USER_TOPICS": bson.M{
					"8197649993": int32(5),
				},
			},
			expectedTopic: 5,
		},
		{
			name: "Legacy int64",
			bsonRaw: bson.M{
				"USER_TOPICS": bson.M{
					"8197649993": int64(10),
				},
			},
			expectedTopic: 10,
		},
		{
			name: "Current SupportTopic struct",
			bsonRaw: bson.M{
				"USER_TOPICS": bson.M{
					"8197649993": bson.M{
						"user_id":           int64(8197649993),
						"topic_id":          int32(15),
						"message_thread_id": int32(15),
					},
				},
			},
			expectedTopic: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := bson.Marshal(tt.bsonRaw)
			if err != nil {
				t.Fatalf("Failed to marshal test data: %v", err)
			}

			var state BotState
			if err := bson.Unmarshal(b, &state); err != nil {
				t.Fatalf("Failed to unmarshal BotState: %v", err)
			}

			topic, ok := state.UserTopics[8197649993]
			if !ok {
				t.Fatalf("Expected user 8197649993 in UserTopics")
			}
			if topic.TopicID != tt.expectedTopic {
				t.Errorf("Expected TopicID %d, got %d", tt.expectedTopic, topic.TopicID)
			}
			if topic.MessageThread != tt.expectedTopic {
				t.Errorf("Expected MessageThread %d, got %d", tt.expectedTopic, topic.MessageThread)
			}
		})
	}
}

func TestSupportTopicEmptyMissing(t *testing.T) {
	// Missing
	raw := bson.M{}
	b, _ := bson.Marshal(raw)
	var state BotState
	if err := bson.Unmarshal(b, &state); err != nil {
		t.Fatalf("Failed to unmarshal empty BSON: %v", err)
	}

	// Empty UserTopics
	raw2 := bson.M{"USER_TOPICS": bson.M{}}
	b2, _ := bson.Marshal(raw2)
	var state2 BotState
	if err := bson.Unmarshal(b2, &state2); err != nil {
		t.Fatalf("Failed to unmarshal empty UserTopics: %v", err)
	}
}
