package middleware

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements RateLimitStore using Redis.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new RedisStore instance.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// script is the LUA script for atomic increment with expiration.
const rateLimitScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current_count = redis.call('INCR', key)
if current_count == 1 then
    redis.call('EXPIRE', key, window)
end
return current_count
`

// Check implements the rate limiting logic using Redis LUA script.
func (s *RedisStore) Check(key string, limit int, window time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Execute the script
	val, err := s.client.Eval(ctx, rateLimitScript, []string{key}, limit, int(window.Seconds())).Result()
	if err != nil {
		// On redis error, we might want to default to allow (fail-open) 
		// or block (fail-closed) depending on business requirements.
		// For high-security OIDC, we fail-closed if the database is down.
		return false
	}

	count, ok := val.(int64)
	if !ok {
		return false
	}

	return count <= int64(limit)
}
