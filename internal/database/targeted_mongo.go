package database

import (
	"context"
	"fmt"

	"github.com/itsh4ro5/botupdate/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Helper for targeted updates
func (m *MongoStore) updateField(ctx context.Context, field string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}
	update := bson.M{"$set": bson.M{field: value}}

	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) deleteField(ctx context.Context, field string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}
	update := bson.M{"$unset": bson.M{field: ""}}

	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) SetUser(ctx context.Context, userID int64, user *models.User) error {
	return m.updateField(ctx, fmt.Sprintf("data.users.%d", userID), user)
}

func (m *MongoStore) SetBlockedUser(ctx context.Context, userID int64, blocked bool) error {
	field := fmt.Sprintf("data.blocked_users.%d", userID)
	if blocked {
		return m.updateField(ctx, field, struct{}{})
	}
	return m.deleteField(ctx, field)
}

func (m *MongoStore) SetAdmin(ctx context.Context, userID int64, isAdmin bool) error {
	field := fmt.Sprintf("data.admin_ids.%d", userID)
	if isAdmin {
		return m.updateField(ctx, field, struct{}{})
	}
	return m.deleteField(ctx, field)
}

func (m *MongoStore) SetSupportTopic(ctx context.Context, userID int64, topic *models.SupportTopic) error {
	return m.updateField(ctx, fmt.Sprintf("data.user_topics.%d", userID), topic)
}

func (m *MongoStore) SetPendingRequest(ctx context.Context, requestID string, req *models.PendingRequest) error {
	if req == nil {
		return m.deleteField(ctx, fmt.Sprintf("data.pending_requests.%s", requestID))
	}
	return m.updateField(ctx, fmt.Sprintf("data.pending_requests.%s", requestID), req)
}

func (m *MongoStore) SetInviteLink(ctx context.Context, hash string, link *models.InviteMapping) error {
	safeHash := encodeLinkMapKey(hash)
	if link == nil {
		return m.deleteField(ctx, fmt.Sprintf("data.link_map.%s", safeHash))
	}
	return m.updateField(ctx, fmt.Sprintf("data.link_map.%s", safeHash), link)
}

func (m *MongoStore) SetBatchCategory(ctx context.Context, batchID int64, category string) error {
	return m.updateField(ctx, fmt.Sprintf("data.batch_categories.%d", batchID), category)
}

func (m *MongoStore) SetCategories(ctx context.Context, categories []string) error {
	return m.updateField(ctx, "data.categories", categories)
}

func (m *MongoStore) SetLockState(ctx context.Context, lockType string, locked bool) error {
	switch lockType {
	case "free":
		return m.updateField(ctx, "data.free_locked", locked)
	case "paid":
		return m.updateField(ctx, "data.paid_locked", locked)
	case "test":
		return m.updateField(ctx, "data.test_bot_locked", locked)
	}
	return nil
}

func (m *MongoStore) SetMaintenanceMode(ctx context.Context, enabled bool) error {
	return m.updateField(ctx, "data.maintenance_mode", enabled)
}

func (m *MongoStore) AddScheduledDelete(ctx context.Context, sd *models.ScheduledDelete) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}
	update := bson.M{"$push": bson.M{"data.scheduled_deletes": sd}}

	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) RemoveScheduledDelete(ctx context.Context, chatID int64, msgID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update()
	filter := bson.M{"_id": "main_settings"}
	update := bson.M{"$pull": bson.M{"data.scheduled_deletes": bson.M{"chat_id": chatID, "message_id": msgID}}}

	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) SetCustomWelcome(ctx context.Context, batchID int64, text string) error {
	if text == "" {
		return m.deleteField(ctx, fmt.Sprintf("data.custom_welcomes.%d", batchID))
	}
	return m.updateField(ctx, fmt.Sprintf("data.custom_welcomes.%d", batchID), text)
}

func (m *MongoStore) SetUserbotSession(ctx context.Context, session string) error {
	return m.updateField(ctx, "data.userbot_session", session)
}

func (m *MongoStore) SetMessageMapping(ctx context.Context, key string, val string) error {
	return m.updateField(ctx, fmt.Sprintf("data.message_map.%s", key), val)
}

func (m *MongoStore) RemoveMessageMapping(ctx context.Context, key string) error {
	return m.deleteField(ctx, fmt.Sprintf("data.message_map.%s", key))
}

func (m *MongoStore) SetBatch(ctx context.Context, id int64, batch *models.Batch, batchType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}

	// Create multiple sets atomically
	setOps := bson.M{
		fmt.Sprintf("data.all_chats.%d", id):        batch.Name,
		fmt.Sprintf("data.batch_categories.%d", id): batch.Category,
	}

	switch batchType {
	case "free":
		setOps[fmt.Sprintf("data.free_batches.%d", id)] = batch
	case "paid":
		setOps[fmt.Sprintf("data.paid_batches.%d", id)] = batch
	case "special":
		setOps[fmt.Sprintf("data.special_batches.%d", id)] = batch
	}

	update := bson.M{"$set": setOps}
	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) SetBatchCoin(ctx context.Context, id int64, coins int64) error {
	return m.updateField(ctx, fmt.Sprintf("data.batch_coins.%d", id), coins)
}

func (m *MongoStore) RemoveBatch(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}

	unsetOps := bson.M{
		fmt.Sprintf("data.free_batches.%d", id):     "",
		fmt.Sprintf("data.paid_batches.%d", id):     "",
		fmt.Sprintf("data.special_batches.%d", id):  "",
		fmt.Sprintf("data.all_chats.%d", id):        "",
		fmt.Sprintf("data.batch_categories.%d", id): "",
		fmt.Sprintf("data.custom_welcomes.%d", id):  "",
		fmt.Sprintf("data.batch_coins.%d", id):      "",
	}

	update := bson.M{"$unset": unsetOps}
	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}
