package database

import (
	"context"
	"time"

	"github.com/mohammedteir/telegram-video-downloader-bot/internal/models"
	"github.com/mohammedteir/telegram-video-downloader-bot/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepository handles user data operations
type UserRepository struct {
	client   *MongoClient
	database string
	logger   *utils.EnhancedLogger
}

// NewUserRepository creates a new user repository
func NewUserRepository(client *MongoClient, database string, logger *utils.EnhancedLogger) *UserRepository {
	return &UserRepository{
		client:   client,
		database: database,
		logger:   logger,
	}
}

// GetUserCollection returns the users collection
func (r *UserRepository) GetUserCollection() *mongo.Collection {
	return r.client.GetCollection(r.database, "users")
}

// GetClient returns the underlying MongoDB client
func (r *UserRepository) GetClient() *MongoClient {
    return r.client
}


// FindUserByChatID finds a user by chat ID
func (r *UserRepository) FindUserByChatID(ctx context.Context, chatID int64) (*models.User, error) {
	collection := r.GetUserCollection()
	
	var user models.User
	filter := bson.M{"chat_id": chatID}
	
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("Error finding user by chat ID %d: %v", chatID, err)
		return nil, err
	}
	
	return &user, nil
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	collection := r.GetUserCollection()
	
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.LastActivity = time.Now()
	
	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		r.logger.Error("Error creating user: %v", err)
		return nil, err
	}
	
	user.ID = result.InsertedID.(primitive.ObjectID)
	r.logger.Info("Created new user with chat ID %d", user.ChatID)
	return user, nil
}

// UpdateUser updates an existing user
func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	collection := r.GetUserCollection()
	
	user.UpdatedAt = time.Now()
	
	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": user}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error updating user %s: %v", user.ID.Hex(), err)
	}
	return err
}

// UpdateUserLanguage updates a user's interface and caption language
func (r *UserRepository) UpdateUserLanguage(ctx context.Context, chatID int64, interfaceLanguage, captionLanguage string) error {
	collection := r.GetUserCollection()
	
	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"interface_language": interfaceLanguage,
			"caption_language":   captionLanguage,
			"updated_at":         time.Now(),
			"last_activity":      time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		r.logger.Error("Error updating user language for chat ID %d: %v", chatID, err)
	} else {
		r.logger.Info("Updated language settings for chat ID %d: interface=%s, caption=%s", 
			chatID, interfaceLanguage, captionLanguage)
	}
	return err
}

// UpdateUserInterfaceLanguage updates a user's interface language
func (r *UserRepository) UpdateUserInterfaceLanguage(ctx context.Context, chatID int64, language string) error {
	collection := r.GetUserCollection()
	
	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"interface_language": language,
			"updated_at":         time.Now(),
			"last_activity":      time.Now(),
		},
	}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error updating interface language for chat ID %d: %v", chatID, err)
	} else {
		r.logger.Info("Updated interface language for chat ID %d: %s", chatID, language)
	}
	return err
}

// UpdateUserCaptionLanguage updates a user's caption language
func (r *UserRepository) UpdateUserCaptionLanguage(ctx context.Context, chatID int64, language string) error {
	collection := r.GetUserCollection()
	
	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"caption_language": language,
			"updated_at":       time.Now(),
			"last_activity":    time.Now(),
		},
	}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error updating caption language for chat ID %d: %v", chatID, err)
	} else {
		r.logger.Info("Updated caption language for chat ID %d: %s", chatID, language)
	}
	return err
}

// UpdateUserActivity updates a user's last activity timestamp and increments request count
func (r *UserRepository) UpdateUserActivity(ctx context.Context, chatID int64) error {
	collection := r.GetUserCollection()
	
	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"last_activity": time.Now(),
			"updated_at":    time.Now(),
		},
		"$inc": bson.M{
			"request_count": 1,
		},
	}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error updating user activity for chat ID %d: %v", chatID, err)
	}
	return err
}

// ResetUserRateLimit resets a user's rate limit
func (r *UserRepository) ResetUserRateLimit(ctx context.Context, chatID int64, resetTime time.Time) error {
	collection := r.GetUserCollection()
	
	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"rate_limit_reset": resetTime,
			"request_count":    0,
			"updated_at":       time.Now(),
		},
	}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error resetting rate limit for chat ID %d: %v", chatID, err)
	} else {
		r.logger.Info("Reset rate limit for chat ID %d, next reset at %v", chatID, resetTime)
	}
	return err
}

// DownloadRepository handles download request and result operations
type DownloadRepository struct {
	client   *MongoClient
	database string
	logger   *utils.EnhancedLogger
}

// NewDownloadRepository creates a new download repository
func NewDownloadRepository(client *MongoClient, database string, logger *utils.EnhancedLogger) *DownloadRepository {
	return &DownloadRepository{
		client:   client,
		database: database,
		logger:   logger,
	}
}

// GetRequestCollection returns the download requests collection
func (r *DownloadRepository) GetRequestCollection() *mongo.Collection {
	return r.client.GetCollection(r.database, "download_requests")
}

// GetResultCollection returns the download results collection
func (r *DownloadRepository) GetResultCollection() *mongo.Collection {
	return r.client.GetCollection(r.database, "download_results")
}

// CreateDownloadRequest creates a new download request
func (r *DownloadRepository) CreateDownloadRequest(ctx context.Context, request *models.DownloadRequest) (*models.DownloadRequest, error) {
	collection := r.GetRequestCollection()
	
	result, err := collection.InsertOne(ctx, request)
	if err != nil {
		r.logger.Error("Error creating download request: %v", err)
		return nil, err
	}
	
	request.ID = result.InsertedID.(primitive.ObjectID)
	r.logger.Info("Created download request %s for chat ID %d: %s", 
		request.ID.Hex(), request.ChatID, request.URL)
	return request, nil
}

// UpdateDownloadRequestStatus updates a download request status
func (r *DownloadRepository) UpdateDownloadRequestStatus(ctx context.Context, requestID primitive.ObjectID, status string) error {
	collection := r.GetRequestCollection()
	
	filter := bson.M{"_id": requestID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	
	// If status is completed, set completed_at
	if status == "completed" {
		update["$set"].(bson.M)["completed_at"] = time.Now()
	}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error updating download request status %s to %s: %v", 
			requestID.Hex(), status, err)
	} else {
		r.logger.Info("Updated download request %s status to %s", requestID.Hex(), status)
	}
	return err
}

// UpdateDownloadRequestRetry updates a download request retry count and error reason
func (r *DownloadRepository) UpdateDownloadRequestRetry(ctx context.Context, requestID primitive.ObjectID, errorReason string) error {
	collection := r.GetRequestCollection()
	
	filter := bson.M{"_id": requestID}
	update := bson.M{
		"$inc": bson.M{
			"retry_count": 1,
		},
		"$set": bson.M{
			"error_reason": errorReason,
			"updated_at":   time.Now(),
		},
	}
	
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error updating download request retry %s: %v", requestID.Hex(), err)
		return err
	}
	
	r.logger.Info("Updated download request %s retry count, matched: %d, modified: %d", 
		requestID.Hex(), result.MatchedCount, result.ModifiedCount)
	return nil
}

// CreateDownloadResult creates a new download result
func (r *DownloadRepository) CreateDownloadResult(ctx context.Context, result *models.DownloadResult) (*models.DownloadResult, error) {
	collection := r.GetResultCollection()
	
	insertResult, err := collection.InsertOne(ctx, result)
	if err != nil {
		r.logger.Error("Error creating download result: %v", err)
		return nil, err
	}
	
	result.ID = insertResult.InsertedID.(primitive.ObjectID)
	r.logger.Info("Created download result %s for request %s", 
		result.ID.Hex(), result.RequestID.Hex())
	return result, nil
}

// GetDownloadResultByRequestID gets a download result by request ID
func (r *DownloadRepository) GetDownloadResultByRequestID(ctx context.Context, requestID primitive.ObjectID) (*models.DownloadResult, error) {
	collection := r.GetResultCollection()
	
	var result models.DownloadResult
	filter := bson.M{"request_id": requestID}
	
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("Error finding download result by request ID %s: %v", 
			requestID.Hex(), err)
		return nil, err
	}
	
	return &result, nil
}

// ErrorLogRepository handles error logging operations
type ErrorLogRepository struct {
	client   *MongoClient
	database string
	logger   *utils.EnhancedLogger
}

// NewErrorLogRepository creates a new error log repository
func NewErrorLogRepository(client *MongoClient, database string, logger *utils.EnhancedLogger) *ErrorLogRepository {
	return &ErrorLogRepository{
		client:   client,
		database: database,
		logger:   logger,
	}
}

// GetErrorLogCollection returns the error logs collection
func (r *ErrorLogRepository) GetErrorLogCollection() *mongo.Collection {
	return r.client.GetCollection(r.database, "error_logs")
}

// LogError logs an error to the database
func (r *ErrorLogRepository) LogError(ctx context.Context, errorLog *models.ErrorLog) error {
	collection := r.GetErrorLogCollection()
	
	_, err := collection.InsertOne(ctx, errorLog)
	if err != nil {
		r.logger.Error("Error inserting error log: %v", err)
		return err
	}
	
	return nil
}

// GetErrorLogs gets error logs with optional filtering
func (r *ErrorLogRepository) GetErrorLogs(ctx context.Context, filter bson.M, limit int64) ([]*models.ErrorLog, error) {
	collection := r.GetErrorLogCollection()
	
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	if limit > 0 {
		findOptions.SetLimit(limit)
	}
	
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		r.logger.Error("Error finding error logs: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var logs []*models.ErrorLog
	if err := cursor.All(ctx, &logs); err != nil {
		r.logger.Error("Error decoding error logs: %v", err)
		return nil, err
	}
	
	return logs, nil
}

// RateLimitRepository handles rate limiting operations
type RateLimitRepository struct {
	client   *MongoClient
	database string
	logger   *utils.EnhancedLogger
}

// NewRateLimitRepository creates a new rate limit repository
func NewRateLimitRepository(client *MongoClient, database string, logger *utils.EnhancedLogger) *RateLimitRepository {
	return &RateLimitRepository{
		client:   client,
		database: database,
		logger:   logger,
	}
}

// GetRateLimitCollection returns the rate limits collection
func (r *RateLimitRepository) GetRateLimitCollection() *mongo.Collection {
	return r.client.GetCollection(r.database, "rate_limits")
}

// GetRateLimit gets a rate limit entry for a chat ID
func (r *RateLimitRepository) GetRateLimit(ctx context.Context, chatID int64) (*models.RateLimitEntry, error) {
	collection := r.GetRateLimitCollection()
	
	var entry models.RateLimitEntry
	filter := bson.M{
		"chat_id": chatID,
		"reset_time": bson.M{
			"$gt": time.Now(),
		},
	}
	
	err := collection.FindOne(ctx, filter).Decode(&entry)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("Error finding rate limit for chat ID %d: %v", chatID, err)
		return nil, err
	}
	
	return &entry, nil
}

// CreateOrUpdateRateLimit creates or updates a rate limit entry
func (r *RateLimitRepository) CreateOrUpdateRateLimit(ctx context.Context, chatID int64, resetTime time.Time) error {
	collection := r.GetRateLimitCollection()
	
	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"reset_time": resetTime,
			"updated_at": time.Now(),
		},
		"$inc": bson.M{
			"count": 1,
		},
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		r.logger.Error("Error updating rate limit for chat ID %d: %v", chatID, err)
		return err
	}
	
	if result.UpsertedCount > 0 {
		r.logger.Info("Created new rate limit for chat ID %d, reset at %v", chatID, resetTime)
	} else {
		r.logger.Debug("Updated rate limit for chat ID %d, reset at %v", chatID, resetTime)
	}
	
	return nil
}

// CleanupExpiredRateLimits removes expired rate limit entries
func (r *RateLimitRepository) CleanupExpiredRateLimits(ctx context.Context) (int64, error) {
	collection := r.GetRateLimitCollection()
	
	filter := bson.M{
		"reset_time": bson.M{
			"$lt": time.Now(),
		},
	}
	
	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		r.logger.Error("Error cleaning up expired rate limits: %v", err)
		return 0, err
	}
	
	if result.DeletedCount > 0 {
		r.logger.Info("Cleaned up %d expired rate limit entries", result.DeletedCount)
	}
	
	return result.DeletedCount, nil
}
