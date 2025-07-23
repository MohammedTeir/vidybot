package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	enabled     bool
	requestsMax int
	timeWindow  time.Duration
	userLimit   bool
	redisClient *redis.Client
	logger      *EnhancedLogger
	mu          sync.Mutex
	counters    map[string]counter
}

type counter struct {
	count     int
	timestamp time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(enabled bool, requestsMax int, timeWindow int, userLimit bool, redisClient *redis.Client, logger *EnhancedLogger) *RateLimiter {
	return &RateLimiter{
		enabled:     enabled,
		requestsMax: requestsMax,
		timeWindow:  time.Duration(timeWindow) * time.Second,
		userLimit:   userLimit,
		redisClient: redisClient,
		logger:      logger,
		counters:    make(map[string]counter),
	}
}

// Allow checks if a request is allowed based on rate limits
func (rl *RateLimiter) Allow(ctx context.Context, identifier string) (bool, error) {
	if !rl.enabled {
		return true, nil
	}

	// If user limiting is disabled, use a global identifier
	if !rl.userLimit {
		identifier = "global"
	}

	// If Redis is available, use it for distributed rate limiting
	if rl.redisClient != nil {
		return rl.allowRedis(ctx, identifier)
	}

	// Otherwise, use in-memory rate limiting
	return rl.allowMemory(identifier), nil
}

// allowRedis implements rate limiting using Redis
func (rl *RateLimiter) allowRedis(ctx context.Context, identifier string) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s", identifier)
	now := time.Now().Unix()
	windowStart := now - int64(rl.timeWindow.Seconds())

	// Remove counts older than the time window
	if err := rl.redisClient.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart)).Err(); err != nil {
		rl.logger.Error("Failed to remove old rate limit counts: %v", err)
		return true, err // Allow on error
	}

	// Count requests in the current time window
	count, err := rl.redisClient.ZCard(ctx, key).Result()
	if err != nil {
		rl.logger.Error("Failed to count rate limit requests: %v", err)
		return true, err // Allow on error
	}

	// Check if limit is exceeded
	if count >= int64(rl.requestsMax) {
		rl.logger.Warn("Rate limit exceeded for %s: %d requests in %v", identifier, count, rl.timeWindow)
		return false, nil
	}

	// Add current request to the sorted set with score as current timestamp
	if err := rl.redisClient.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)}).Err(); err != nil {
		rl.logger.Error("Failed to add rate limit request: %v", err)
		return true, err // Allow on error
	}

	// Set expiration on the key to clean up automatically
	if err := rl.redisClient.Expire(ctx, key, rl.timeWindow).Err(); err != nil {
		rl.logger.Error("Failed to set rate limit expiration: %v", err)
	}

	return true, nil
}

// allowMemory implements rate limiting using in-memory counters
func (rl *RateLimiter) allowMemory(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	c, exists := rl.counters[identifier]

	// If counter doesn't exist or time window has passed, reset it
	if !exists || now.Sub(c.timestamp) > rl.timeWindow {
		rl.counters[identifier] = counter{
			count:     1,
			timestamp: now,
		}
		return true
	}

	// Check if limit is exceeded
	if c.count >= rl.requestsMax {
		rl.logger.Warn("Rate limit exceeded for %s: %d requests in %v", identifier, c.count, rl.timeWindow)
		return false
	}

	// Increment counter
	c.count++
	rl.counters[identifier] = c
	return true
}

// CleanupExpiredCounters removes expired counters from memory
func (rl *RateLimiter) CleanupExpiredCounters() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for id, c := range rl.counters {
		if now.Sub(c.timestamp) > rl.timeWindow {
			delete(rl.counters, id)
		}
	}
}

// StartCleanupScheduler starts a scheduler to clean up expired counters
func (rl *RateLimiter) StartCleanupScheduler(ctx context.Context) {
	if !rl.enabled || rl.redisClient != nil {
		return // No need to clean up if disabled or using Redis
	}

	ticker := time.NewTicker(rl.timeWindow)
	go func() {
		for {
			select {
			case <-ticker.C:
				rl.CleanupExpiredCounters()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
