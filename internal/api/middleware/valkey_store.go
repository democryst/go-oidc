package middleware

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// ValkeyStore implements RateLimitStore using Valkey (wire-compatible with Redis).
type ValkeyStore struct {
	client *redis.Client
}

// NewValkeyStore creates a new ValkeyStore instance.
func NewValkeyStore(client *redis.Client) *ValkeyStore {
	return &ValkeyStore{client: client}
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

// Check implements the rate limiting logic using Valkey LUA script.
func (s *ValkeyStore) Check(key string, limit int, window time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Execute the script
	val, err := s.client.Eval(ctx, rateLimitScript, []string{key}, limit, int(window.Seconds())).Result()
	if err != nil {
		// On valkey error, we might want to default to allow (fail-open) 
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
