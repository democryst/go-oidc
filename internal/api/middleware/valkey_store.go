package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
)

// ValkeyStore implements RateLimitStore using the native Valkey client.
type ValkeyStore struct {
	client valkey.Client
	script *valkey.Lua
}

// NewValkeyStore creates a new ValkeyStore instance using the native client.
func NewValkeyStore(client valkey.Client) *ValkeyStore {
	return &ValkeyStore{
		client: client,
		script: valkey.NewLuaScript(rateLimitScript),
	}
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

// Check implements the rate limiting logic using the native Valkey client and LUA script.
func (s *ValkeyStore) Check(key string, limit int, window time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Execute the script using valkey-go's Exec
	// It automatically handles EVALSHA and EVAL.
	res := s.script.Exec(ctx, s.client, []string{key}, []string{
		strconv.Itoa(limit),
		strconv.Itoa(int(window.Seconds())),
	})
	
	if err := res.Error(); err != nil {
		// On valkey error, we fail-closed for security.
		return false
	}

	count, err := res.AsInt64()
	if err != nil {
		return false
	}

	return count <= int64(limit)
}
