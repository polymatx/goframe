package cache

// fakeRedis is a minimal in-process Redis server speaking RESP2, implementing
// just enough of the protocol for these tests. It exists because the tests
// must not rely on a live external Redis and the module must not gain new
// dependencies. go-redis connects with HELLO 3 first; replying with a Redis
// error makes the client fall back to RESP2 transparently.

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fakeEntry struct {
	kind string // "string", "hash", "list", "set", "zset"
	str  string
	hash map[string]string
	list []string
	set  map[string]struct{}
	zset map[string]float64
}

type fakeRedis struct {
	ln net.Listener

	mu     sync.Mutex
	data   map[string]*fakeEntry
	expiry map[string]time.Time
}

func startFakeRedis() (*fakeRedis, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &fakeRedis{
		ln:     ln,
		data:   make(map[string]*fakeEntry),
		expiry: make(map[string]time.Time),
	}
	go s.acceptLoop()
	return s, nil
}

func (s *fakeRedis) Addr() string { return s.ln.Addr().String() }
func (s *fakeRedis) Close()       { _ = s.ln.Close() }

func (s *fakeRedis) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *fakeRedis) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}
		if len(args) == 0 {
			continue
		}
		if _, err := w.WriteString(s.exec(args)); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
	}
}

// --- RESP protocol helpers ---

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("expected array header, got %q", line)
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, err := readLine(r)
		if err != nil {
			return nil, err
		}
		if len(hdr) == 0 || hdr[0] != '$' {
			return nil, fmt.Errorf("expected bulk string header, got %q", hdr)
		}
		size, err := strconv.Atoi(hdr[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2) // payload + trailing \r\n
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}

const (
	respNullBulk  = "$-1\r\n"
	respWrongType = "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
)

func respSimple(s string) string { return "+" + s + "\r\n" }
func respError(m string) string  { return "-" + m + "\r\n" }
func respInt(n int64) string     { return ":" + strconv.FormatInt(n, 10) + "\r\n" }
func respBulk(s string) string   { return "$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n" }

func respArray(items []string) string {
	var b strings.Builder
	b.WriteString("*" + strconv.Itoa(len(items)) + "\r\n")
	for _, item := range items {
		b.WriteString(respBulk(item))
	}
	return b.String()
}

// --- command dispatch (s.mu held) ---

func (s *fakeRedis) exec(args []string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := strings.ToUpper(args[0])
	switch cmd {
	case "PING":
		return respSimple("PONG")
	case "HELLO":
		// Deny RESP3 so go-redis falls back to RESP2.
		return respError("ERR unknown command 'HELLO'")
	case "CLIENT", "SELECT":
		return respSimple("OK")
	case "FLUSHDB":
		s.data = make(map[string]*fakeEntry)
		s.expiry = make(map[string]time.Time)
		return respSimple("OK")
	case "SET":
		return s.cmdSet(args[1:])
	case "GET":
		return s.cmdGet(args[1])
	case "GETDEL":
		reply := s.cmdGet(args[1])
		delete(s.data, args[1])
		delete(s.expiry, args[1])
		return reply
	case "DEL":
		return s.cmdDel(args[1:])
	case "EXISTS":
		n := int64(0)
		for _, key := range args[1:] {
			if s.live(key) != nil {
				n++
			}
		}
		return respInt(n)
	case "EXPIRE":
		return s.cmdExpire(args[1], args[2])
	case "TTL":
		return s.cmdTTL(args[1])
	case "INCR":
		return s.cmdIncrBy(args[1], 1)
	case "DECR":
		return s.cmdIncrBy(args[1], -1)
	case "INCRBY", "DECRBY":
		delta, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return respError("ERR value is not an integer or out of range")
		}
		if cmd == "DECRBY" {
			delta = -delta
		}
		return s.cmdIncrBy(args[1], delta)
	case "MSET":
		for i := 1; i+1 < len(args); i += 2 {
			s.data[args[i]] = &fakeEntry{kind: "string", str: args[i+1]}
			delete(s.expiry, args[i])
		}
		return respSimple("OK")
	case "MGET":
		var b strings.Builder
		b.WriteString("*" + strconv.Itoa(len(args)-1) + "\r\n")
		for _, key := range args[1:] {
			if e := s.live(key); e != nil && e.kind == "string" {
				b.WriteString(respBulk(e.str))
			} else {
				b.WriteString(respNullBulk)
			}
		}
		return b.String()
	case "HSET":
		return s.cmdHSet(args[1], args[2:])
	case "HGET":
		return s.cmdHGet(args[1], args[2])
	case "HGETALL":
		return s.cmdHGetAll(args[1])
	case "HDEL":
		return s.cmdHDel(args[1], args[2:])
	case "LPUSH", "RPUSH":
		return s.cmdPush(args[1], args[2:], cmd == "LPUSH")
	case "LPOP", "RPOP":
		return s.cmdPop(args[1], cmd == "LPOP")
	case "LRANGE":
		return s.cmdLRange(args[1], args[2], args[3])
	case "SADD":
		return s.cmdSAdd(args[1], args[2:])
	case "SREM":
		return s.cmdSRem(args[1], args[2:])
	case "SISMEMBER":
		e := s.live(args[1])
		if e == nil {
			return respInt(0)
		}
		if e.kind != "set" {
			return respWrongType
		}
		if _, ok := e.set[args[2]]; ok {
			return respInt(1)
		}
		return respInt(0)
	case "SMEMBERS":
		e := s.live(args[1])
		if e == nil {
			return respArray(nil)
		}
		if e.kind != "set" {
			return respWrongType
		}
		members := make([]string, 0, len(e.set))
		for m := range e.set {
			members = append(members, m)
		}
		sort.Strings(members)
		return respArray(members)
	case "ZADD":
		return s.cmdZAdd(args[1], args[2:])
	case "ZRANGE":
		return s.cmdZRange(args[1], args[2], args[3])
	case "ZRANGEBYSCORE":
		return s.cmdZRangeByScore(args[1], args[2], args[3])
	case "ZREM":
		return s.cmdZRem(args[1], args[2:])
	default:
		return respError("ERR unknown command '" + args[0] + "'")
	}
}

// live returns the entry for key, lazily removing it if expired.
func (s *fakeRedis) live(key string) *fakeEntry {
	if deadline, ok := s.expiry[key]; ok && time.Now().After(deadline) {
		delete(s.data, key)
		delete(s.expiry, key)
	}
	return s.data[key]
}

func (s *fakeRedis) cmdSet(args []string) string {
	key, value := args[0], args[1]
	var ttl time.Duration
	nx := false
	for i := 2; i < len(args); {
		switch strings.ToUpper(args[i]) {
		case "EX", "PX":
			if i+1 >= len(args) {
				return respError("ERR syntax error")
			}
			n, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return respError("ERR value is not an integer or out of range")
			}
			if strings.ToUpper(args[i]) == "EX" {
				ttl = time.Duration(n) * time.Second
			} else {
				ttl = time.Duration(n) * time.Millisecond
			}
			i += 2
		case "NX":
			nx = true
			i++
		case "XX", "KEEPTTL":
			i++
		default:
			return respError("ERR syntax error")
		}
	}
	if nx && s.live(key) != nil {
		return respNullBulk
	}
	s.data[key] = &fakeEntry{kind: "string", str: value}
	if ttl > 0 {
		s.expiry[key] = time.Now().Add(ttl)
	} else {
		delete(s.expiry, key)
	}
	return respSimple("OK")
}

func (s *fakeRedis) cmdGet(key string) string {
	e := s.live(key)
	if e == nil {
		return respNullBulk
	}
	if e.kind != "string" {
		return respWrongType
	}
	return respBulk(e.str)
}

func (s *fakeRedis) cmdDel(keys []string) string {
	n := int64(0)
	for _, key := range keys {
		if s.live(key) != nil {
			delete(s.data, key)
			delete(s.expiry, key)
			n++
		}
	}
	return respInt(n)
}

func (s *fakeRedis) cmdExpire(key, secStr string) string {
	sec, err := strconv.ParseInt(secStr, 10, 64)
	if err != nil {
		return respError("ERR value is not an integer or out of range")
	}
	if s.live(key) == nil {
		return respInt(0)
	}
	s.expiry[key] = time.Now().Add(time.Duration(sec) * time.Second)
	return respInt(1)
}

func (s *fakeRedis) cmdTTL(key string) string {
	if s.live(key) == nil {
		return respInt(-2)
	}
	deadline, ok := s.expiry[key]
	if !ok {
		return respInt(-1)
	}
	return respInt(int64(math.Ceil(time.Until(deadline).Seconds())))
}

func (s *fakeRedis) cmdIncrBy(key string, delta int64) string {
	current := int64(0)
	if e := s.live(key); e != nil {
		if e.kind != "string" {
			return respWrongType
		}
		n, err := strconv.ParseInt(e.str, 10, 64)
		if err != nil {
			return respError("ERR value is not an integer or out of range")
		}
		current = n
	}
	current += delta
	s.data[key] = &fakeEntry{kind: "string", str: strconv.FormatInt(current, 10)}
	return respInt(current)
}

func (s *fakeRedis) cmdHSet(key string, fieldValues []string) string {
	e := s.live(key)
	if e == nil {
		e = &fakeEntry{kind: "hash", hash: make(map[string]string)}
		s.data[key] = e
	}
	if e.kind != "hash" {
		return respWrongType
	}
	added := int64(0)
	for i := 0; i+1 < len(fieldValues); i += 2 {
		if _, ok := e.hash[fieldValues[i]]; !ok {
			added++
		}
		e.hash[fieldValues[i]] = fieldValues[i+1]
	}
	return respInt(added)
}

func (s *fakeRedis) cmdHGet(key, field string) string {
	e := s.live(key)
	if e == nil {
		return respNullBulk
	}
	if e.kind != "hash" {
		return respWrongType
	}
	value, ok := e.hash[field]
	if !ok {
		return respNullBulk
	}
	return respBulk(value)
}

func (s *fakeRedis) cmdHGetAll(key string) string {
	e := s.live(key)
	if e == nil {
		return respArray(nil)
	}
	if e.kind != "hash" {
		return respWrongType
	}
	fields := make([]string, 0, len(e.hash))
	for f := range e.hash {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	flat := make([]string, 0, len(fields)*2)
	for _, f := range fields {
		flat = append(flat, f, e.hash[f])
	}
	return respArray(flat)
}

func (s *fakeRedis) cmdHDel(key string, fields []string) string {
	e := s.live(key)
	if e == nil {
		return respInt(0)
	}
	if e.kind != "hash" {
		return respWrongType
	}
	n := int64(0)
	for _, f := range fields {
		if _, ok := e.hash[f]; ok {
			delete(e.hash, f)
			n++
		}
	}
	return respInt(n)
}

func (s *fakeRedis) cmdPush(key string, values []string, left bool) string {
	e := s.live(key)
	if e == nil {
		e = &fakeEntry{kind: "list"}
		s.data[key] = e
	}
	if e.kind != "list" {
		return respWrongType
	}
	for _, v := range values {
		if left {
			e.list = append([]string{v}, e.list...)
		} else {
			e.list = append(e.list, v)
		}
	}
	return respInt(int64(len(e.list)))
}

func (s *fakeRedis) cmdPop(key string, left bool) string {
	e := s.live(key)
	if e == nil || len(e.list) == 0 {
		return respNullBulk
	}
	if e.kind != "list" {
		return respWrongType
	}
	var v string
	if left {
		v, e.list = e.list[0], e.list[1:]
	} else {
		v, e.list = e.list[len(e.list)-1], e.list[:len(e.list)-1]
	}
	if len(e.list) == 0 {
		delete(s.data, key)
		delete(s.expiry, key)
	}
	return respBulk(v)
}

// normalizeRange converts Redis-style start/stop indices (which may be
// negative) into slice bounds. ok is false when the range is empty.
func normalizeRange(start, stop, length int) (int, int, bool) {
	if start < 0 {
		start = length + start
		if start < 0 {
			start = 0
		}
	}
	if stop < 0 {
		stop = length + stop
	}
	if stop >= length {
		stop = length - 1
	}
	if length == 0 || start >= length || start > stop {
		return 0, 0, false
	}
	return start, stop, true
}

func (s *fakeRedis) cmdLRange(key, startStr, stopStr string) string {
	start, err1 := strconv.Atoi(startStr)
	stop, err2 := strconv.Atoi(stopStr)
	if err1 != nil || err2 != nil {
		return respError("ERR value is not an integer or out of range")
	}
	e := s.live(key)
	if e == nil {
		return respArray(nil)
	}
	if e.kind != "list" {
		return respWrongType
	}
	from, to, ok := normalizeRange(start, stop, len(e.list))
	if !ok {
		return respArray(nil)
	}
	return respArray(e.list[from : to+1])
}

func (s *fakeRedis) cmdSAdd(key string, members []string) string {
	e := s.live(key)
	if e == nil {
		e = &fakeEntry{kind: "set", set: make(map[string]struct{})}
		s.data[key] = e
	}
	if e.kind != "set" {
		return respWrongType
	}
	added := int64(0)
	for _, m := range members {
		if _, ok := e.set[m]; !ok {
			e.set[m] = struct{}{}
			added++
		}
	}
	return respInt(added)
}

func (s *fakeRedis) cmdSRem(key string, members []string) string {
	e := s.live(key)
	if e == nil {
		return respInt(0)
	}
	if e.kind != "set" {
		return respWrongType
	}
	n := int64(0)
	for _, m := range members {
		if _, ok := e.set[m]; ok {
			delete(e.set, m)
			n++
		}
	}
	return respInt(n)
}

func (s *fakeRedis) cmdZAdd(key string, scoreMembers []string) string {
	e := s.live(key)
	if e == nil {
		e = &fakeEntry{kind: "zset", zset: make(map[string]float64)}
		s.data[key] = e
	}
	if e.kind != "zset" {
		return respWrongType
	}
	added := int64(0)
	for i := 0; i+1 < len(scoreMembers); i += 2 {
		score, err := strconv.ParseFloat(scoreMembers[i], 64)
		if err != nil {
			return respError("ERR value is not a valid float")
		}
		if _, ok := e.zset[scoreMembers[i+1]]; !ok {
			added++
		}
		e.zset[scoreMembers[i+1]] = score
	}
	return respInt(added)
}

// sortedZSetMembers returns members ordered by (score, member).
func sortedZSetMembers(zset map[string]float64) []string {
	members := make([]string, 0, len(zset))
	for m := range zset {
		members = append(members, m)
	}
	sort.Slice(members, func(i, j int) bool {
		si, sj := zset[members[i]], zset[members[j]]
		if si != sj {
			return si < sj
		}
		return members[i] < members[j]
	})
	return members
}

func (s *fakeRedis) cmdZRange(key, startStr, stopStr string) string {
	start, err1 := strconv.Atoi(startStr)
	stop, err2 := strconv.Atoi(stopStr)
	if err1 != nil || err2 != nil {
		return respError("ERR value is not an integer or out of range")
	}
	e := s.live(key)
	if e == nil {
		return respArray(nil)
	}
	if e.kind != "zset" {
		return respWrongType
	}
	members := sortedZSetMembers(e.zset)
	from, to, ok := normalizeRange(start, stop, len(members))
	if !ok {
		return respArray(nil)
	}
	return respArray(members[from : to+1])
}

func parseScoreBound(s string) (float64, error) {
	switch strings.ToLower(s) {
	case "-inf":
		return math.Inf(-1), nil
	case "+inf", "inf":
		return math.Inf(1), nil
	default:
		return strconv.ParseFloat(s, 64)
	}
}

func (s *fakeRedis) cmdZRangeByScore(key, minStr, maxStr string) string {
	minScore, err1 := parseScoreBound(minStr)
	maxScore, err2 := parseScoreBound(maxStr)
	if err1 != nil || err2 != nil {
		return respError("ERR min or max is not a float")
	}
	e := s.live(key)
	if e == nil {
		return respArray(nil)
	}
	if e.kind != "zset" {
		return respWrongType
	}
	var result []string
	for _, m := range sortedZSetMembers(e.zset) {
		if score := e.zset[m]; score >= minScore && score <= maxScore {
			result = append(result, m)
		}
	}
	return respArray(result)
}

func (s *fakeRedis) cmdZRem(key string, members []string) string {
	e := s.live(key)
	if e == nil {
		return respInt(0)
	}
	if e.kind != "zset" {
		return respWrongType
	}
	n := int64(0)
	for _, m := range members {
		if _, ok := e.zset[m]; ok {
			delete(e.zset, m)
			n++
		}
	}
	return respInt(n)
}
