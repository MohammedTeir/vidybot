package database

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoClient wraps the MongoDB client
type MongoClient struct {
	client *mongo.Client
}

// NewMongoClient creates a new MongoDB client with improved connection handling
func NewMongoClient(ctx context.Context, uri string) (*MongoClient, error) {
	// Set client options with additional connection settings
	clientOptions := options.Client().ApplyURI(uri)
	
	// Add connection timeout and other settings for better reliability
	clientOptions.SetConnectTimeout(30 * time.Second)
	clientOptions.SetServerSelectionTimeout(30 * time.Second)
	clientOptions.SetSocketTimeout(30 * time.Second)
	clientOptions.SetMaxPoolSize(10)
	clientOptions.SetMinPoolSize(1)
	
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	
	// Check the connection with a longer timeout
	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	err = client.Ping(pingCtx, nil)
	if err != nil {
		client.Disconnect(ctx)
		return nil, err
	}
	
	return &MongoClient{
		client: client,
	}, nil
}

// GetCollection returns a MongoDB collection
func (m *MongoClient) GetCollection(database, collection string) *mongo.Collection {
	return m.client.Database(database).Collection(collection)
}

// Disconnect closes the MongoDB connection
func (m *MongoClient) Disconnect(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// GetClient returns the underlying MongoDB client
func (m *MongoClient) GetClient() *MongoClient {
	return m
}

// RedisClient wraps the Redis client
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(ctx context.Context, uri string) (*RedisClient, error) {
	// Create Redis client
	opt, err := redis.ParseURL(uri)
	if err != nil {
		return nil, err
	}
	
	client := redis.NewClient(opt)
	
	// Check connection
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	
	return &RedisClient{
		client: client,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Set sets a key-value pair in Redis
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value from Redis
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

