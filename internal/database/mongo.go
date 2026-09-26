package database

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/itsh4ro5/botupdate/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	mu                      sync.RWMutex
	client                  *mongo.Client
	collection              *mongo.Collection
	batchContentsCollection *mongo.Collection
}

func NewMongoStore(ctx context.Context, uri string) (*MongoStore, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI).
		SetConnectTimeout(5 * time.Second).
		SetSocketTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	collection := client.Database("telegram_bot_db").Collection("bot_settings")
	batchContentsCollection := client.Database("telegram_bot_db").Collection("batch_contents")

	return &MongoStore{
		client:                  client,
		collection:              collection,
		batchContentsCollection: batchContentsCollection,
	}, nil
}

func encodeLinkMapKey(hash string) string {
	return base64.URLEncoding.EncodeToString([]byte(hash))
}

func decodeLinkMapKey(key string) string {
	b, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		return key
	}
	return string(b)
}

func (m *MongoStore) Load(ctx context.Context) (*models.BotState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var doc struct {
		ID   string          `bson:"_id"`
		Data models.BotState `bson:"data"`
	}

	err := m.collection.FindOne(ctx, bson.M{"_id": "main_settings"}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return m.emptyState(), nil
		}
		return nil, fmt.Errorf("failed to load state from mongo: %w", err)
	}

	state := &doc.Data

	// Initialize maps if they are nil
	if state.AdminIDs == nil {
		state.AdminIDs = make(map[int64]struct{})
	}
	if state.Users == nil {
		state.Users = make(map[int64]*models.User)
	}
	if state.FreeBatches == nil {
		state.FreeBatches = make(map[int64]*models.Batch)
	}
	if state.PaidBatches == nil {
		state.PaidBatches = make(map[int64]*models.Batch)
	}
	if state.SpecialBatches == nil {
		state.SpecialBatches = make(map[int64]*models.Batch)
	}
	if state.AllChats == nil {
		state.AllChats = make(map[int64]string)
	}
	if state.BlockedUsers == nil {
		state.BlockedUsers = make(map[int64]struct{})
	}
	if state.UserTopics == nil {
		state.UserTopics = make(map[int64]*models.SupportTopic)
	}
	if state.PendingRequests == nil {
		state.PendingRequests = make(map[string]*models.PendingRequest)
	}
	if state.LinkMap == nil {
		state.LinkMap = make(map[string]*models.InviteMapping)
	}
	if state.CustomWelcomes == nil {
		state.CustomWelcomes = make(map[int64]string)
	}
	if state.BatchCategories == nil {
		state.BatchCategories = make(map[int64]string)
	}
	if state.Categories == nil {
		state.Categories = make([]string, 0)
	}
	if state.BatchCoins == nil {
		state.BatchCoins = make(map[int64]int64)
	}
	if state.WebAdmins == nil {
		state.WebAdmins = make(map[string]*models.WebAdmin)
	}
	if state.WebSessions == nil {
		state.WebSessions = make(map[string]*models.WebSession)
	}

	if state.LinkMap != nil {
		decodedMap := make(map[string]*models.InviteMapping)
		for k, v := range state.LinkMap {
			if v == nil || v.UserID <= 0 || v.BatchID == 0 {
				continue
			}
			decodedK := decodeLinkMapKey(k)
			decodedMap[decodedK] = v
		}
		state.LinkMap = decodedMap
	}

	return state, nil
}

func (m *MongoStore) Save(ctx context.Context, state *models.BotState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": "main_settings"}

	stateCopy := *state
	if state.LinkMap != nil {
		encodedLinkMap := make(map[string]*models.InviteMapping)
		for k, v := range state.LinkMap {
			encodedLinkMap[encodeLinkMapKey(k)] = v
		}
		stateCopy.LinkMap = encodedLinkMap
	}

	flattened, err := safeFlatten(&stateCopy)
	if err != nil {
		return fmt.Errorf("failed to serialize state for mongo update: %w", err)
	}

	update := bson.M{"$set": flattened}

	_, err = m.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save state to mongo: %w", err)
	}
	return nil
}

func (m *MongoStore) emptyState() *models.BotState {
	return &models.BotState{
		AdminIDs:        make(map[int64]struct{}),
		FreeBatches:     make(map[int64]*models.Batch),
		PaidBatches:     make(map[int64]*models.Batch),
		SpecialBatches:  make(map[int64]*models.Batch),
		AllChats:        make(map[int64]string),
		Users:           make(map[int64]*models.User),
		BlockedUsers:    make(map[int64]struct{}),
		UserTopics:      make(map[int64]*models.SupportTopic),
		PendingRequests: make(map[string]*models.PendingRequest),
		LinkMap:         make(map[string]*models.InviteMapping),
		CustomWelcomes:  make(map[int64]string),
		BatchCategories: make(map[int64]string),
		Categories:      make([]string, 0),
		BatchCoins:      make(map[int64]int64),
		WebAdmins:       make(map[string]*models.WebAdmin),
		WebSessions:     make(map[string]*models.WebSession),
	}
}

func (m *MongoStore) SetWebAdmin(ctx context.Context, username string, admin *models.WebAdmin) error {
	state, err := m.Load(ctx)
	if err != nil {
		return err
	}
	state.WebAdmins[username] = admin
	return m.Save(ctx, state)
}

func (m *MongoStore) SetWebSession(ctx context.Context, sessionID string, session *models.WebSession) error {
	state, err := m.Load(ctx)
	if err != nil {
		return err
	}
	state.WebSessions[sessionID] = session
	return m.Save(ctx, state)
}

func (m *MongoStore) DeleteWebSession(ctx context.Context, sessionID string) error {
	state, err := m.Load(ctx)
	if err != nil {
		return err
	}
	delete(state.WebSessions, sessionID)
	return m.Save(ctx, state)
}

func (m *MongoStore) GetDashboardOverview(ctx context.Context) (*models.DashboardOverview, error) {
	overview := &models.DashboardOverview{
		GeneratedAt: time.Now().UTC(),
	}

	pipeline := []bson.M{
		{"$match": bson.M{"_id": "main_settings"}},
		{"$project": bson.M{
			"total_users":           bson.M{"$size": bson.M{"$objectToArray": bson.M{"$ifNull": []interface{}{"$data.users", bson.M{}}}}},
			"total_free_batches":    bson.M{"$size": bson.M{"$objectToArray": bson.M{"$ifNull": []interface{}{"$data.free_batches", bson.M{}}}}},
			"total_paid_batches":    bson.M{"$size": bson.M{"$objectToArray": bson.M{"$ifNull": []interface{}{"$data.paid_batches", bson.M{}}}}},
			"total_special_batches": bson.M{"$size": bson.M{"$objectToArray": bson.M{"$ifNull": []interface{}{"$data.special_batches", bson.M{}}}}},
			"pending_requests":      bson.M{"$size": bson.M{"$objectToArray": bson.M{"$ifNull": []interface{}{"$data.pending_requests", bson.M{}}}}},
		}},
	}

	cursor, err := m.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %w", err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result struct {
			TotalUsers          int `bson:"total_users"`
			TotalFreeBatches    int `bson:"total_free_batches"`
			TotalPaidBatches    int `bson:"total_paid_batches"`
			TotalSpecialBatches int `bson:"total_special_batches"`
			PendingRequests     int `bson:"pending_requests"`
		}
		if err := cursor.Decode(&result); err == nil {
			overview.Users.Total = result.TotalUsers
			// Activity timestamp is not uniformly stored in models.User, so Active is 0 or -1 (unavailable)
			overview.Users.Active = -1

			overview.Batches.Total = result.TotalFreeBatches + result.TotalPaidBatches + result.TotalSpecialBatches
			overview.Requests.Pending = result.PendingRequests
		}
	} else {
		// Document missing, return zero values
		overview.Users.Active = -1
	}

	overview.System.Database = "online"
	// other system statuses are handled by the controller
	return overview, nil
}

func (m *MongoStore) SaveBatchContents(ctx context.Context, chatID string, data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": chatID}
	update := bson.M{"$set": data}

	_, err := m.batchContentsCollection.UpdateOne(ctx, filter, update, opts)
	return err
}
