package cache

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"
)

// flushCache clears the fake Redis so tests stay independent even when the
// test binary reruns them in the same process (e.g. -count=2).
func flushCache(t *testing.T) {
	t.Helper()
	if err := testCache.FlushDB(context.Background()); err != nil {
		t.Fatalf("flushdb failed: %v", err)
	}
}

func TestManager_SetGet(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	t.Run("set then get", func(t *testing.T) {
		if err := testCache.Set(ctx, "kv:greeting", "hello", 0); err != nil {
			t.Fatalf("set failed: %v", err)
		}
		got, err := testCache.Get(ctx, "kv:greeting")
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if got != "hello" {
			t.Errorf("expected 'hello', got %q", got)
		}
	})

	t.Run("get missing key returns ErrNotFound", func(t *testing.T) {
		_, err := testCache.Get(ctx, "kv:missing")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("set overwrites existing value", func(t *testing.T) {
		if err := testCache.Set(ctx, "kv:overwrite", "first", 0); err != nil {
			t.Fatalf("set failed: %v", err)
		}
		if err := testCache.Set(ctx, "kv:overwrite", "second", 0); err != nil {
			t.Fatalf("set failed: %v", err)
		}
		got, err := testCache.Get(ctx, "kv:overwrite")
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if got != "second" {
			t.Errorf("expected 'second', got %q", got)
		}
	})

	t.Run("set with TTL expires", func(t *testing.T) {
		if err := testCache.Set(ctx, "kv:short-lived", "gone soon", 30*time.Millisecond); err != nil {
			t.Fatalf("set failed: %v", err)
		}
		time.Sleep(80 * time.Millisecond)
		_, err := testCache.Get(ctx, "kv:short-lived")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound after TTL, got %v", err)
		}
	})
}

func TestManager_SetNX(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	ok, err := testCache.SetNX(ctx, "nx:lock", "owner-1", time.Minute)
	if err != nil {
		t.Fatalf("setnx failed: %v", err)
	}
	if !ok {
		t.Fatal("expected first SetNX to succeed")
	}

	ok, err = testCache.SetNX(ctx, "nx:lock", "owner-2", time.Minute)
	if err != nil {
		t.Fatalf("setnx failed: %v", err)
	}
	if ok {
		t.Fatal("expected second SetNX to fail")
	}

	got, err := testCache.Get(ctx, "nx:lock")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != "owner-1" {
		t.Errorf("expected original value 'owner-1', got %q", got)
	}
}

func TestManager_GetDel(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	if err := testCache.Set(ctx, "getdel:key", "one-shot", 0); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	got, err := testCache.GetDel(ctx, "getdel:key")
	if err != nil {
		t.Fatalf("getdel failed: %v", err)
	}
	if got != "one-shot" {
		t.Errorf("expected 'one-shot', got %q", got)
	}

	_, err = testCache.Get(ctx, "getdel:key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected key to be deleted, got %v", err)
	}

	_, err = testCache.GetDel(ctx, "getdel:missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing key, got %v", err)
	}
}

func TestManager_DelExists(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	if err := testCache.Set(ctx, "del:a", "1", 0); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := testCache.Set(ctx, "del:b", "2", 0); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	n, err := testCache.Exists(ctx, "del:a", "del:b", "del:missing")
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 existing keys, got %d", n)
	}

	if err := testCache.Del(ctx, "del:a"); err != nil {
		t.Fatalf("del failed: %v", err)
	}

	n, err = testCache.Exists(ctx, "del:a", "del:b")
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 existing key after delete, got %d", n)
	}
}

func TestManager_ExpireTTL(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	if err := testCache.Set(ctx, "ttl:key", "value", 0); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	t.Run("no expiry set", func(t *testing.T) {
		ttl, err := testCache.TTL(ctx, "ttl:key")
		if err != nil {
			t.Fatalf("ttl failed: %v", err)
		}
		if ttl >= 0 {
			t.Errorf("expected negative TTL for key without expiry, got %v", ttl)
		}
	})

	t.Run("expire sets TTL", func(t *testing.T) {
		if err := testCache.Expire(ctx, "ttl:key", 5*time.Minute); err != nil {
			t.Fatalf("expire failed: %v", err)
		}
		ttl, err := testCache.TTL(ctx, "ttl:key")
		if err != nil {
			t.Fatalf("ttl failed: %v", err)
		}
		if ttl <= 4*time.Minute || ttl > 5*time.Minute {
			t.Errorf("expected TTL close to 5m, got %v", ttl)
		}
	})

	t.Run("missing key TTL", func(t *testing.T) {
		ttl, err := testCache.TTL(ctx, "ttl:missing")
		if err != nil {
			t.Fatalf("ttl failed: %v", err)
		}
		if ttl >= 0 {
			t.Errorf("expected negative TTL for missing key, got %v", ttl)
		}
	})
}

func TestManager_IncrDecr(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	tests := []struct {
		name string
		op   func() (int64, error)
		want int64
	}{
		{"Incr on new key", func() (int64, error) { return testCache.Incr(ctx, "counter:n") }, 1},
		{"Incr again", func() (int64, error) { return testCache.Incr(ctx, "counter:n") }, 2},
		{"IncrBy 5", func() (int64, error) { return testCache.IncrBy(ctx, "counter:n", 5) }, 7},
		{"Decr", func() (int64, error) { return testCache.Decr(ctx, "counter:n") }, 6},
		{"DecrBy 4", func() (int64, error) { return testCache.DecrBy(ctx, "counter:n", 4) }, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.op()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestManager_HashOps(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	if err := testCache.HSet(ctx, "hash:user", "name", "alice", "role", "admin"); err != nil {
		t.Fatalf("hset failed: %v", err)
	}

	t.Run("HGet existing field", func(t *testing.T) {
		got, err := testCache.HGet(ctx, "hash:user", "name")
		if err != nil {
			t.Fatalf("hget failed: %v", err)
		}
		if got != "alice" {
			t.Errorf("expected 'alice', got %q", got)
		}
	})

	t.Run("HGet missing field returns ErrNotFound", func(t *testing.T) {
		_, err := testCache.HGet(ctx, "hash:user", "missing")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("HGetAll", func(t *testing.T) {
		got, err := testCache.HGetAll(ctx, "hash:user")
		if err != nil {
			t.Fatalf("hgetall failed: %v", err)
		}
		want := map[string]string{"name": "alice", "role": "admin"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("HDel removes field", func(t *testing.T) {
		if err := testCache.HDel(ctx, "hash:user", "role"); err != nil {
			t.Fatalf("hdel failed: %v", err)
		}
		_, err := testCache.HGet(ctx, "hash:user", "role")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected field to be deleted, got %v", err)
		}
	})
}

func TestManager_ListOps(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	if err := testCache.LPush(ctx, "list:queue", "a", "b"); err != nil {
		t.Fatalf("lpush failed: %v", err)
	}
	if err := testCache.RPush(ctx, "list:queue", "z"); err != nil {
		t.Fatalf("rpush failed: %v", err)
	}

	t.Run("LRange returns elements in order", func(t *testing.T) {
		got, err := testCache.LRange(ctx, "list:queue", 0, -1)
		if err != nil {
			t.Fatalf("lrange failed: %v", err)
		}
		// LPush prepends one value at a time: "a" then "b" -> [b a], RPush z -> [b a z]
		want := []string{"b", "a", "z"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("LPop returns head", func(t *testing.T) {
		got, err := testCache.LPop(ctx, "list:queue")
		if err != nil {
			t.Fatalf("lpop failed: %v", err)
		}
		if got != "b" {
			t.Errorf("expected 'b', got %q", got)
		}
	})

	t.Run("RPop returns tail", func(t *testing.T) {
		got, err := testCache.RPop(ctx, "list:queue")
		if err != nil {
			t.Fatalf("rpop failed: %v", err)
		}
		if got != "z" {
			t.Errorf("expected 'z', got %q", got)
		}
	})

	t.Run("LPop on empty list returns ErrNotFound", func(t *testing.T) {
		_, err := testCache.LPop(ctx, "list:empty")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestManager_SetOps(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	if err := testCache.SAdd(ctx, "set:tags", "go", "redis", "cache"); err != nil {
		t.Fatalf("sadd failed: %v", err)
	}

	t.Run("SIsMember", func(t *testing.T) {
		ok, err := testCache.SIsMember(ctx, "set:tags", "go")
		if err != nil {
			t.Fatalf("sismember failed: %v", err)
		}
		if !ok {
			t.Error("expected 'go' to be a member")
		}

		ok, err = testCache.SIsMember(ctx, "set:tags", "python")
		if err != nil {
			t.Fatalf("sismember failed: %v", err)
		}
		if ok {
			t.Error("expected 'python' to not be a member")
		}
	})

	t.Run("SMembers", func(t *testing.T) {
		got, err := testCache.SMembers(ctx, "set:tags")
		if err != nil {
			t.Fatalf("smembers failed: %v", err)
		}
		sort.Strings(got)
		want := []string{"cache", "go", "redis"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("SRem", func(t *testing.T) {
		if err := testCache.SRem(ctx, "set:tags", "cache"); err != nil {
			t.Fatalf("srem failed: %v", err)
		}
		ok, err := testCache.SIsMember(ctx, "set:tags", "cache")
		if err != nil {
			t.Fatalf("sismember failed: %v", err)
		}
		if ok {
			t.Error("expected 'cache' to be removed")
		}
	})
}

func TestManager_SortedSetOps(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	err := testCache.ZAdd(ctx, "zset:scores",
		&Z{Score: 3, Member: "carol"},
		&Z{Score: 1, Member: "alice"},
		&Z{Score: 2, Member: "bob"},
	)
	if err != nil {
		t.Fatalf("zadd failed: %v", err)
	}

	t.Run("ZRange returns members ordered by score", func(t *testing.T) {
		got, err := testCache.ZRange(ctx, "zset:scores", 0, -1)
		if err != nil {
			t.Fatalf("zrange failed: %v", err)
		}
		want := []string{"alice", "bob", "carol"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("ZRangeByScore filters by score", func(t *testing.T) {
		got, err := testCache.ZRangeByScore(ctx, "zset:scores", "2", "3")
		if err != nil {
			t.Fatalf("zrangebyscore failed: %v", err)
		}
		want := []string{"bob", "carol"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("ZRem removes member", func(t *testing.T) {
		if err := testCache.ZRem(ctx, "zset:scores", "bob"); err != nil {
			t.Fatalf("zrem failed: %v", err)
		}
		got, err := testCache.ZRange(ctx, "zset:scores", 0, -1)
		if err != nil {
			t.Fatalf("zrange failed: %v", err)
		}
		want := []string{"alice", "carol"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})
}

func TestManager_JSON(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	in := payload{Name: "goframe", Count: 7}
	if err := testCache.SetJSON(ctx, "json:payload", in, 0); err != nil {
		t.Fatalf("setjson failed: %v", err)
	}

	var out payload
	if err := testCache.GetJSON(ctx, "json:payload", &out); err != nil {
		t.Fatalf("getjson failed: %v", err)
	}
	if out != in {
		t.Errorf("roundtrip mismatch: expected %+v, got %+v", in, out)
	}

	var missing payload
	err := testCache.GetJSON(ctx, "json:missing", &missing)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// bufferSerializable is a minimal Serializable implementation for tests.
type bufferSerializable struct {
	payload string
}

func (b *bufferSerializable) Encode(w *bytes.Buffer) error {
	_, err := w.WriteString(b.payload)
	return err
}

func (b *bufferSerializable) Decode(r *bytes.Buffer) error {
	b.payload = r.String()
	return nil
}

func TestManager_SetGetObject(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	in := &bufferSerializable{payload: "serialized-state"}
	if err := testCache.SetObject(ctx, "obj:key", in, 0); err != nil {
		t.Fatalf("setobject failed: %v", err)
	}

	out := &bufferSerializable{}
	if err := testCache.GetObject(ctx, "obj:key", out); err != nil {
		t.Fatalf("getobject failed: %v", err)
	}
	if out.payload != in.payload {
		t.Errorf("roundtrip mismatch: expected %q, got %q", in.payload, out.payload)
	}

	err := testCache.GetObject(ctx, "obj:missing", &bufferSerializable{})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestManager_MSetMGet(t *testing.T) {
	ctx := context.Background()
	flushCache(t)

	if err := testCache.MSet(ctx, "m:a", "1", "m:b", "2"); err != nil {
		t.Fatalf("mset failed: %v", err)
	}

	got, err := testCache.MGet(ctx, "m:a", "m:b", "m:missing")
	if err != nil {
		t.Fatalf("mget failed: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}
	if got[0] != "1" || got[1] != "2" {
		t.Errorf("unexpected values: %v", got)
	}
	if got[2] != nil {
		t.Errorf("expected nil for missing key, got %v", got[2])
	}
}
