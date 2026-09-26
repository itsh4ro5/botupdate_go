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
	return m.updateField(ctx, fmt.Sprintf("data.USER_DATA.%d", userID), user)
}

func (m *MongoStore) SetBlockedUser(ctx context.Context, userID int64, blocked bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}
	var update bson.M
	if blocked {
		update = bson.M{"$addToSet": bson.M{"data.BLOCKED_USERS": userID}}
	} else {
		update = bson.M{"$pull": bson.M{"data.BLOCKED_USERS": userID}}
	}
	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) SetAdmin(ctx context.Context, userID int64, isAdmin bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}
	var update bson.M
	if isAdmin {
		update = bson.M{"$addToSet": bson.M{"data.ADMIN_IDS": userID}}
	} else {
		update = bson.M{"$pull": bson.M{"data.ADMIN_IDS": userID}}
	}
	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) SetSupportTopic(ctx context.Context, userID int64, topic *models.SupportTopic) error {
	return m.updateField(ctx, fmt.Sprintf("data.USER_TOPICS.%d", userID), topic)
}

func (m *MongoStore) SetPendingRequest(ctx context.Context, requestID string, req *models.PendingRequest) error {
	if req == nil {
		return m.deleteField(ctx, fmt.Sprintf("data.PENDING_REQUESTS.%s", requestID))
	}
	return m.updateField(ctx, fmt.Sprintf("data.PENDING_REQUESTS.%s", requestID), req)
}

func (m *MongoStore) SetInviteLink(ctx context.Context, hash string, link *models.InviteMapping) error {
	safeHash := encodeLinkMapKey(hash)
	if link == nil {
		return m.deleteField(ctx, fmt.Sprintf("data.LINK_MAP.%s", safeHash))
	}
	return m.updateField(ctx, fmt.Sprintf("data.LINK_MAP.%s", safeHash), link)
}

func (m *MongoStore) SetBatchCategory(ctx context.Context, batchID int64, category string) error {
	return m.updateField(ctx, fmt.Sprintf("data.BATCH_CATEGORIES.%d", batchID), category)
}

func (m *MongoStore) SetCategories(ctx context.Context, categories []string) error {
	return m.updateField(ctx, "data.CATEGORIES", categories)
}

func (m *MongoStore) SetLockState(ctx context.Context, lockType string, locked bool) error {
	switch lockType {
	case "free":
		return m.updateField(ctx, "data.FREE_LOCKED", locked)
	case "paid":
		return m.updateField(ctx, "data.PAID_LOCKED", locked)
	case "test":
		return m.updateField(ctx, "data.TEST_BOT_LOCKED", locked)
	}
	return nil
}

func (m *MongoStore) SetMaintenanceMode(ctx context.Context, enabled bool) error {
	return m.updateField(ctx, "data.MAINTENANCE_MODE", enabled)
}

func (m *MongoStore) AddScheduledDelete(ctx context.Context, sd *models.ScheduledDelete) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}
	update := bson.M{"$push": bson.M{"data.SCHEDULED_DELETES": sd}}

	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) RemoveScheduledDelete(ctx context.Context, chatID int64, msgID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update()
	filter := bson.M{"_id": "main_settings"}
	update := bson.M{"$pull": bson.M{"data.SCHEDULED_DELETES": bson.M{"chat_id": chatID, "message_id": msgID}}}

	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) SetCustomWelcome(ctx context.Context, batchID int64, text string) error {
	if text == "" {
		return m.deleteField(ctx, fmt.Sprintf("data.CUSTOM_WELCOMES.%d", batchID))
	}
	return m.updateField(ctx, fmt.Sprintf("data.CUSTOM_WELCOMES.%d", batchID), text)
}

func (m *MongoStore) SetUserbotSession(ctx context.Context, session string) error {
	return m.updateField(ctx, "data.USERBOT_SESSION", session)
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
		fmt.Sprintf("data.ALL_CHATS.%d", id):        batch.Name,
		fmt.Sprintf("data.BATCH_CATEGORIES.%d", id): batch.Category,
	}

	switch batchType {
	case "free":
		setOps[fmt.Sprintf("data.FREE_CHANNELS.%d", id)] = batch
	case "paid":
		setOps[fmt.Sprintf("data.PAID_CHANNELS.%d", id)] = batch
	case "special":
		setOps[fmt.Sprintf("data.SPECIAL_CHANNELS.%d", id)] = batch
	}

	update := bson.M{"$set": setOps}
	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (m *MongoStore) SetBatchCoin(ctx context.Context, id int64, coins int64) error {
	return m.updateField(ctx, fmt.Sprintf("data.BATCH_COINS.%d", id), coins)
}

func (m *MongoStore) RemoveBatch(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}

	unsetOps := bson.M{
		fmt.Sprintf("data.FREE_CHANNELS.%d", id):     "",
		fmt.Sprintf("data.PAID_CHANNELS.%d", id):     "",
		fmt.Sprintf("data.SPECIAL_CHANNELS.%d", id):  "",
		fmt.Sprintf("data.ALL_CHATS.%d", id):        "",
		fmt.Sprintf("data.BATCH_CATEGORIES.%d", id): "",
		fmt.Sprintf("data.CUSTOM_WELCOMES.%d", id):  "",
		fmt.Sprintf("data.BATCH_COINS.%d", id):      "",
	}

	update := bson.M{"$unset": unsetOps}
	_, err := m.collection.UpdateOne(ctx, filter, update, opts)
	return err
}
