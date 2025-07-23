package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChatID           int64              `bson:"chat_id" json:"chat_id"`
	InterfaceLanguage string            `bson:"interface_language" json:"interface_language"`
	CaptionLanguage  string             `bson:"caption_language" json:"caption_language"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
	LastActivity     time.Time          `bson:"last_activity" json:"last_activity"`
	RequestCount     int                `bson:"request_count" json:"request_count"`
	RateLimitReset   time.Time          `bson:"rate_limit_reset" json:"rate_limit_reset"`
}

// NewUser creates a new user with default values
func NewUser(chatID int64) *User {
	return &User{
		ChatID:           chatID,
		InterfaceLanguage: "en", // Default to English
		CaptionLanguage:  "en", // Default to English
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		LastActivity:     time.Now(),
		RequestCount:     0,
		RateLimitReset:   time.Now(),
	}
}

// DownloadRequest represents a video download request
type DownloadRequest struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChatID      int64              `bson:"chat_id" json:"chat_id"`
	URL         string             `bson:"url" json:"url"`
	Status      string             `bson:"status" json:"status"` // pending, processing, completed, failed
	RetryCount  int                `bson:"retry_count" json:"retry_count"`
	ErrorReason string             `bson:"error_reason,omitempty" json:"error_reason,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	CompletedAt time.Time          `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}

// NewDownloadRequest creates a new download request
func NewDownloadRequest(chatID int64, url string) *DownloadRequest {
	return &DownloadRequest{
		ChatID:     chatID,
		URL:        url,
		Status:     "pending",
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// DownloadResult represents the result of a video download
type DownloadResult struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RequestID       primitive.ObjectID `bson:"request_id" json:"request_id"`
	ChatID          int64              `bson:"chat_id" json:"chat_id"`
	VideoPath       string             `bson:"video_path" json:"video_path"`
	VideoWithSubPath string            `bson:"video_with_sub_path" json:"video_with_sub_path"`
	AudioPath       string             `bson:"audio_path" json:"audio_path"`
	SubtitlePath    string             `bson:"subtitle_path" json:"subtitle_path"`
	HasSubtitle     bool               `bson:"has_subtitle" json:"has_subtitle"`
	FileSize        int64              `bson:"file_size" json:"file_size"`
	Duration        int                `bson:"duration" json:"duration"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
}

// SupportedLanguage represents a supported language for the bot interface and captions
type SupportedLanguage struct {
	Code        string `bson:"code" json:"code"`
	Name        string `bson:"name" json:"name"`
	NativeName  string `bson:"native_name" json:"native_name"`
	IsAvailable bool   `bson:"is_available" json:"is_available"`
}

// GetSupportedLanguages returns the list of supported languages
func GetSupportedLanguages() []SupportedLanguage {
	return []SupportedLanguage{
		{
			Code:        "ar",
			Name:        "Arabic",
			NativeName:  "العربية",
			IsAvailable: true,
		},
		{
			Code:        "en",
			Name:        "English",
			NativeName:  "English",
			IsAvailable: true,
		},
		{
			Code:        "de",
			Name:        "German",
			NativeName:  "Deutsch",
			IsAvailable: true,
		},
		{
			Code:        "fr",
			Name:        "French",
			NativeName:  "Français",
			IsAvailable: true,
		},
	}
}

// RateLimitEntry represents a rate limit entry for a user
type RateLimitEntry struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChatID    int64              `bson:"chat_id" json:"chat_id"`
	Count     int                `bson:"count" json:"count"`
	ResetTime time.Time          `bson:"reset_time" json:"reset_time"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// NewRateLimitEntry creates a new rate limit entry
func NewRateLimitEntry(chatID int64, resetTime time.Time) *RateLimitEntry {
	return &RateLimitEntry{
		ChatID:    chatID,
		Count:     1,
		ResetTime: resetTime,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ErrorLog represents an error log entry
type ErrorLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChatID    int64              `bson:"chat_id,omitempty" json:"chat_id,omitempty"`
	RequestID primitive.ObjectID `bson:"request_id,omitempty" json:"request_id,omitempty"`
	Level     string             `bson:"level" json:"level"`
	Message   string             `bson:"message" json:"message"`
	Error     string             `bson:"error" json:"error"`
	Stack     string             `bson:"stack,omitempty" json:"stack,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// NewErrorLog creates a new error log entry
func NewErrorLog(level, message, errorStr, stack string) *ErrorLog {
	return &ErrorLog{
		Level:     level,
		Message:   message,
		Error:     errorStr,
		Stack:     stack,
		CreatedAt: time.Now(),
	}
}

// WithChatID adds a chat ID to the error log
func (e *ErrorLog) WithChatID(chatID int64) *ErrorLog {
	e.ChatID = chatID
	return e
}

// WithRequestID adds a request ID to the error log
func (e *ErrorLog) WithRequestID(requestID primitive.ObjectID) *ErrorLog {
	e.RequestID = requestID
	return e
}
