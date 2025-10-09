package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient   *mongo.Client
	mongoDatabase *mongo.Database
	repoColl      *mongo.Collection
)

func InitMongo(ctx context.Context) error {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return fmt.Errorf("MONGODB_URI is not set")
	}

	clientOpts := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("mongo connect error: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongo ping error: %w", err)
	}

	mongoClient = client

	dbName := uri[strings.LastIndex(uri, "/")+1:]
	mongoDatabase = client.Database(dbName)
	repoColl = mongoDatabase.Collection("repos")

	fmt.Printf("Connected to MongoDB (%s)\n", dbName)
	return nil
}

func SaveAllToMongo(ctx context.Context) error {
	if repoColl == nil {
		return fmt.Errorf("mongo not initialized")
	}

	fmt.Printf("Saving %d repos to MongoDB...\n", len(RepoStore))
	start := time.Now()

	var ops []mongo.WriteModel
	StoreMu.Lock()
	for _, r := range RepoStore {
		doc := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"id": r.ID}).
			SetUpdate(bson.M{"$set": r}).
			SetUpsert(true)
		ops = append(ops, doc)
	}
	StoreMu.Unlock()

	if len(ops) == 0 {
		fmt.Println("No repos to save.")
		return nil
	}

	_, err := repoColl.BulkWrite(ctx, ops)
	if err != nil {
		return fmt.Errorf("bulk insert error: %w", err)
	}

	fmt.Printf("Saved %d repos in %s\n", len(ops), time.Since(start))
	return nil
}

func CloseMongo(ctx context.Context) {
	if mongoClient != nil {
		_ = mongoClient.Disconnect(ctx)
		fmt.Println("🔌 MongoDB disconnected.")
	}
}
