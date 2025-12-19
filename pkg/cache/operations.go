package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

// Serializable represents an object that can be encoded/decoded for caching
type Serializable interface {
	Encode(w *bytes.Buffer) error
	Decode(r *bytes.Buffer) error
}

// Set stores a key-value pair with TTL in Redis
func (m *Manager) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return m.client.Set(ctx, key, value, ttl).Err()
}

// SetNX sets a key only if it doesn't exist (atomic)
func (m *Manager) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return m.client.SetNX(ctx, key, value, ttl).Result()
}

// Get retrieves a value by key from Redis
func (m *Manager) Get(ctx context.Context, key string) (string, error) {
	return m.client.Get(ctx, key).Result()
}

// GetDel atomically gets and deletes a key
func (m *Manager) GetDel(ctx context.Context, key string) (string, error) {
	return m.client.GetDel(ctx, key).Result()
}

// Del deletes one or more keys from Redis
func (m *Manager) Del(ctx context.Context, keys ...string) error {
	return m.client.Del(ctx, keys...).Err()
}

// Exists checks if one or more keys exist in Redis
func (m *Manager) Exists(ctx context.Context, keys ...string) (int64, error) {
	return m.client.Exists(ctx, keys...).Result()
}

// Expire sets a timeout on a key
func (m *Manager) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return m.client.Expire(ctx, key, ttl).Err()
}

// TTL returns the remaining time to live of a key
func (m *Manager) TTL(ctx context.Context, key string) (time.Duration, error) {
	return m.client.TTL(ctx, key).Result()
}

// Incr increments the integer value of a key by one
func (m *Manager) Incr(ctx context.Context, key string) (int64, error) {
	return m.client.Incr(ctx, key).Result()
}

// IncrBy increments the integer value of a key by the given amount
func (m *Manager) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return m.client.IncrBy(ctx, key, value).Result()
}

// Decr decrements the integer value of a key by one
func (m *Manager) Decr(ctx context.Context, key string) (int64, error) {
	return m.client.Decr(ctx, key).Result()
}

// DecrBy decrements the integer value of a key by the given amount
func (m *Manager) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return m.client.DecrBy(ctx, key, value).Result()
}

// SetJSON serializes and stores a JSON object with TTL
func (m *Manager) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.Set(ctx, key, string(data), ttl)
}

// GetJSON retrieves and deserializes a JSON object
func (m *Manager) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := m.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// SetObject serializes and stores a Serializable object with TTL
func (m *Manager) SetObject(ctx context.Context, key string, obj Serializable, ttl time.Duration) error {
	buf := &bytes.Buffer{}
	if err := obj.Encode(buf); err != nil {
		return err
	}
	return m.Set(ctx, key, buf.String(), ttl)
}

// GetObject retrieves and deserializes a Serializable object
func (m *Manager) GetObject(ctx context.Context, key string, obj Serializable) error {
	data, err := m.Get(ctx, key)
	if err != nil {
		return err
	}
	buf := bytes.NewBufferString(data)
	return obj.Decode(buf)
}

// MGet retrieves multiple keys at once
func (m *Manager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return m.client.MGet(ctx, keys...).Result()
}

// MSet sets multiple key-value pairs atomically
func (m *Manager) MSet(ctx context.Context, pairs ...interface{}) error {
	return m.client.MSet(ctx, pairs...).Err()
}

// HSet sets a field in a hash
func (m *Manager) HSet(ctx context.Context, key string, values ...interface{}) error {
	return m.client.HSet(ctx, key, values...).Err()
}

// HGet gets a field from a hash
func (m *Manager) HGet(ctx context.Context, key, field string) (string, error) {
	return m.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all fields from a hash
func (m *Manager) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return m.client.HGetAll(ctx, key).Result()
}

// HDel deletes one or more fields from a hash
func (m *Manager) HDel(ctx context.Context, key string, fields ...string) error {
	return m.client.HDel(ctx, key, fields...).Err()
}

// LPush prepends one or more values to a list
func (m *Manager) LPush(ctx context.Context, key string, values ...interface{}) error {
	return m.client.LPush(ctx, key, values...).Err()
}

// RPush appends one or more values to a list
func (m *Manager) RPush(ctx context.Context, key string, values ...interface{}) error {
	return m.client.RPush(ctx, key, values...).Err()
}

// LPop removes and returns the first element of a list
func (m *Manager) LPop(ctx context.Context, key string) (string, error) {
	return m.client.LPop(ctx, key).Result()
}

// RPop removes and returns the last element of a list
func (m *Manager) RPop(ctx context.Context, key string) (string, error) {
	return m.client.RPop(ctx, key).Result()
}

// LRange gets a range of elements from a list
func (m *Manager) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return m.client.LRange(ctx, key, start, stop).Result()
}

// SAdd adds one or more members to a set
func (m *Manager) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return m.client.SAdd(ctx, key, members...).Err()
}

// SRem removes one or more members from a set
func (m *Manager) SRem(ctx context.Context, key string, members ...interface{}) error {
	return m.client.SRem(ctx, key, members...).Err()
}

// SMembers gets all members of a set
func (m *Manager) SMembers(ctx context.Context, key string) ([]string, error) {
	return m.client.SMembers(ctx, key).Result()
}

// SIsMember checks if a value is a member of a set
func (m *Manager) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return m.client.SIsMember(ctx, key, member).Result()
}

// ZAdd adds one or more members to a sorted set
func (m *Manager) ZAdd(ctx context.Context, key string, members ...*Z) error {
	redisMembers := make([]*redis.Z, len(members))
	for i, m := range members {
		redisMembers[i] = &redis.Z{
			Score:  m.Score,
			Member: m.Member,
		}
	}
	return m.client.ZAdd(ctx, key, redisMembers...).Err()
}

// Z represents a sorted set member
type Z struct {
	Score  float64
	Member interface{}
}

// ZRange gets a range of members from a sorted set by index
func (m *Manager) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return m.client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeByScore gets members from a sorted set by score range
func (m *Manager) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	return m.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
}

// ZRem removes one or more members from a sorted set
func (m *Manager) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return m.client.ZRem(ctx, key, members...).Err()
}

// Publish publishes a message to a channel
func (m *Manager) Publish(ctx context.Context, channel string, message interface{}) error {
	return m.client.Publish(ctx, channel, message).Err()
}

// Keys finds all keys matching a pattern (use with caution in production)
func (m *Manager) Keys(ctx context.Context, pattern string) ([]string, error) {
	return m.client.Keys(ctx, pattern).Result()
}

// Scan iterates over keys matching a pattern
func (m *Manager) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return m.client.Scan(ctx, cursor, match, count).Result()
}

// FlushDB deletes all keys in the current database (use with extreme caution)
func (m *Manager) FlushDB(ctx context.Context) error {
	return m.client.FlushDB(ctx).Err()
}

// Ping tests the connection to Redis
func (m *Manager) Ping(ctx context.Context) error {
	return m.client.Ping(ctx).Err()
}
