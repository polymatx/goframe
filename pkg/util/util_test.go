package util

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// ---------- crypto.go ----------

func TestHashPassword(t *testing.T) {
	t.Run("hash and verify", func(t *testing.T) {
		hash, err := HashPassword("secret123")
		if err != nil {
			t.Fatalf("HashPassword returned error: %v", err)
		}
		if hash == "" {
			t.Fatal("expected non-empty hash")
		}
		if hash == "secret123" {
			t.Fatal("hash must not equal plaintext password")
		}
		if !CheckPassword("secret123", hash) {
			t.Error("CheckPassword should accept correct password")
		}
		if CheckPassword("wrong-password", hash) {
			t.Error("CheckPassword should reject wrong password")
		}
	})

	t.Run("hashes are salted", func(t *testing.T) {
		h1, err1 := HashPassword("same-password")
		h2, err2 := HashPassword("same-password")
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected errors: %v, %v", err1, err2)
		}
		if h1 == h2 {
			t.Error("two hashes of same password should differ (bcrypt salt)")
		}
	})

	t.Run("empty password", func(t *testing.T) {
		hash, err := HashPassword("")
		if err != nil {
			t.Fatalf("HashPassword(\"\") returned error: %v", err)
		}
		if !CheckPassword("", hash) {
			t.Error("CheckPassword should accept empty password with its hash")
		}
		if CheckPassword("x", hash) {
			t.Error("CheckPassword should reject non-empty password against empty-password hash")
		}
	})

	t.Run("password longer than bcrypt 72-byte limit", func(t *testing.T) {
		_, err := HashPassword(strings.Repeat("a", 100))
		if err == nil {
			t.Error("expected error for password longer than 72 bytes")
		}
	})
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("password", "not-a-bcrypt-hash") {
		t.Error("CheckPassword should return false for malformed hash")
	}
	if CheckPassword("password", "") {
		t.Error("CheckPassword should return false for empty hash")
	}
}

func TestMD5(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", "d41d8cd98f00b204e9800998ecf8427e"},
		{"known vector abc", "abc", "900150983cd24fb0d6963f7d28e17f72"},
		{"hello", "hello", "5d41402abc4b2a76b9719d911017c592"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MD5(tt.input); got != tt.want {
				t.Errorf("MD5(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSHA1(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"known vector abc", "abc", "a9993e364706816aba3e25717850c26c9cd0d89d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SHA1(tt.input); got != tt.want {
				t.Errorf("SHA1(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSHA256(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"known vector abc", "abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SHA256(tt.input); got != tt.want {
				t.Errorf("SHA256(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRandomBytes(t *testing.T) {
	t.Run("lengths", func(t *testing.T) {
		for _, n := range []int{0, 1, 16, 32, 64} {
			b, err := RandomBytes(n)
			if err != nil {
				t.Fatalf("RandomBytes(%d) returned error: %v", n, err)
			}
			if len(b) != n {
				t.Errorf("RandomBytes(%d) length = %d, want %d", n, len(b), n)
			}
		}
	})

	t.Run("randomness", func(t *testing.T) {
		a, err1 := RandomBytes(32)
		b, err2 := RandomBytes(32)
		if err1 != nil || err2 != nil {
			t.Fatalf("unexpected errors: %v, %v", err1, err2)
		}
		if reflect.DeepEqual(a, b) {
			t.Error("two 32-byte random reads should not be equal")
		}
	})
}

func TestRandomString(t *testing.T) {
	base64URLCharset := regexp.MustCompile(`^[A-Za-z0-9_=-]*$`)

	t.Run("lengths and charset", func(t *testing.T) {
		for _, n := range []int{0, 1, 8, 16, 32, 64} {
			s, err := RandomString(n)
			if err != nil {
				t.Fatalf("RandomString(%d) returned error: %v", n, err)
			}
			if len(s) != n {
				t.Errorf("RandomString(%d) length = %d, want %d", n, len(s), n)
			}
			if !base64URLCharset.MatchString(s) {
				t.Errorf("RandomString(%d) = %q contains chars outside base64 URL alphabet", n, s)
			}
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			s, err := RandomString(32)
			if err != nil {
				t.Fatalf("RandomString returned error: %v", err)
			}
			if seen[s] {
				t.Fatalf("duplicate random string generated: %q", s)
			}
			seen[s] = true
		}
	})
}

func TestRandomToken(t *testing.T) {
	hexCharset := regexp.MustCompile(`^[0-9a-f]{64}$`)

	t.Run("format", func(t *testing.T) {
		token, err := RandomToken()
		if err != nil {
			t.Fatalf("RandomToken returned error: %v", err)
		}
		if !hexCharset.MatchString(token) {
			t.Errorf("RandomToken() = %q, want 64 lowercase hex chars", token)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			token, err := RandomToken()
			if err != nil {
				t.Fatalf("RandomToken returned error: %v", err)
			}
			if seen[token] {
				t.Fatalf("duplicate token generated: %q", token)
			}
			seen[token] = true
		}
	})
}

func TestBase64EncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		encoded string
	}{
		{"simple text", []byte("hello"), "aGVsbG8="},
		{"empty", []byte{}, ""},
		{"binary data", []byte{0x00, 0xff, 0x10}, "AP8Q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Base64Encode(tt.data)
			if got != tt.encoded {
				t.Errorf("Base64Encode(%v) = %q, want %q", tt.data, got, tt.encoded)
			}

			decoded, err := Base64Decode(got)
			if err != nil {
				t.Fatalf("Base64Decode(%q) returned error: %v", got, err)
			}
			if !reflect.DeepEqual(decoded, tt.data) && (len(decoded) != 0 || len(tt.data) != 0) {
				t.Errorf("round trip: Base64Decode(%q) = %v, want %v", got, decoded, tt.data)
			}
		})
	}

	t.Run("invalid base64", func(t *testing.T) {
		if _, err := Base64Decode("!!!not-base64!!!"); err == nil {
			t.Error("expected error decoding invalid base64")
		}
	})
}

func TestUUIDv4(t *testing.T) {
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	t.Run("format version and variant", func(t *testing.T) {
		id, err := UUIDv4()
		if err != nil {
			t.Fatalf("UUIDv4 returned error: %v", err)
		}
		if !uuidPattern.MatchString(id) {
			t.Errorf("UUIDv4() = %q, does not match RFC 4122 v4 format", id)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id, err := UUIDv4()
			if err != nil {
				t.Fatalf("UUIDv4 returned error: %v", err)
			}
			if seen[id] {
				t.Fatalf("duplicate UUID generated: %q", id)
			}
			seen[id] = true
		}
	})
}

// ---------- slice.go ----------

func TestContains(t *testing.T) {
	t.Run("ints", func(t *testing.T) {
		tests := []struct {
			name    string
			slice   []int
			element int
			want    bool
		}{
			{"found", []int{1, 2, 3}, 2, true},
			{"not found", []int{1, 2, 3}, 4, false},
			{"empty slice", nil, 1, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := Contains(tt.slice, tt.element); got != tt.want {
					t.Errorf("Contains(%v, %v) = %v, want %v", tt.slice, tt.element, got, tt.want)
				}
			})
		}
	})

	t.Run("strings", func(t *testing.T) {
		if !Contains([]string{"a", "b"}, "b") {
			t.Error("expected Contains to find string element")
		}
		if Contains([]string{"a", "b"}, "A") {
			t.Error("Contains should be case sensitive")
		}
	})
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		predicate func(int) bool
		want      []int
	}{
		{"keep evens", []int{1, 2, 3, 4, 5}, func(n int) bool { return n%2 == 0 }, []int{2, 4}},
		{"keep all", []int{1, 2}, func(int) bool { return true }, []int{1, 2}},
		{"keep none", []int{1, 2}, func(int) bool { return false }, []int{}},
		{"empty input", nil, func(int) bool { return true }, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(tt.slice, tt.predicate)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter(%v) = %v, want %v", tt.slice, got, tt.want)
			}
		})
	}
}

func TestMap(t *testing.T) {
	t.Run("int to int", func(t *testing.T) {
		got := Map([]int{1, 2, 3}, func(n int) int { return n * n })
		want := []int{1, 4, 9}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map squares = %v, want %v", got, want)
		}
	})

	t.Run("int to string", func(t *testing.T) {
		got := Map([]int{1, 2}, func(n int) string { return strings.Repeat("x", n) })
		want := []string{"x", "xx"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map to strings = %v, want %v", got, want)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := Map(nil, func(n int) int { return n })
		if len(got) != 0 {
			t.Errorf("Map(nil) = %v, want empty", got)
		}
	})
}

func TestReduce(t *testing.T) {
	t.Run("sum", func(t *testing.T) {
		got := Reduce([]int{1, 2, 3, 4}, 0, func(acc, n int) int { return acc + n })
		if got != 10 {
			t.Errorf("Reduce sum = %d, want 10", got)
		}
	})

	t.Run("concat", func(t *testing.T) {
		got := Reduce([]string{"a", "b", "c"}, "", func(acc, s string) string { return acc + s })
		if got != "abc" {
			t.Errorf("Reduce concat = %q, want %q", got, "abc")
		}
	})

	t.Run("empty returns initial", func(t *testing.T) {
		got := Reduce(nil, 42, func(acc, n int) int { return acc + n })
		if got != 42 {
			t.Errorf("Reduce(nil) = %d, want initial 42", got)
		}
	})
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name  string
		slice []int
		want  []int
	}{
		{"removes duplicates", []int{1, 2, 1, 3, 2}, []int{1, 2, 3}},
		{"no duplicates", []int{1, 2, 3}, []int{1, 2, 3}},
		{"all same", []int{5, 5, 5}, []int{5}},
		{"empty", nil, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unique(tt.slice)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unique(%v) = %v, want %v", tt.slice, got, tt.want)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name  string
		slice []int
		want  []int
	}{
		{"even length", []int{1, 2, 3, 4}, []int{4, 3, 2, 1}},
		{"odd length", []int{1, 2, 3}, []int{3, 2, 1}},
		{"single element", []int{1}, []int{1}},
		{"empty", nil, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reverse(tt.slice)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reverse(%v) = %v, want %v", tt.slice, got, tt.want)
			}
		})
	}

	t.Run("does not mutate input", func(t *testing.T) {
		input := []int{1, 2, 3}
		Reverse(input)
		if !reflect.DeepEqual(input, []int{1, 2, 3}) {
			t.Errorf("Reverse mutated its input: %v", input)
		}
	})
}

func TestChunk(t *testing.T) {
	tests := []struct {
		name  string
		slice []int
		size  int
		want  [][]int
	}{
		{"even split", []int{1, 2, 3, 4}, 2, [][]int{{1, 2}, {3, 4}}},
		{"remainder chunk", []int{1, 2, 3, 4, 5}, 2, [][]int{{1, 2}, {3, 4}, {5}}},
		{"size larger than slice", []int{1, 2}, 10, [][]int{{1, 2}}},
		{"size one", []int{1, 2}, 1, [][]int{{1}, {2}}},
		{"empty slice", nil, 3, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Chunk(tt.slice, tt.size)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Chunk(%v, %d) = %v, want %v", tt.slice, tt.size, got, tt.want)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name   string
		slices [][]int
		want   []int
	}{
		{"multiple slices", [][]int{{1, 2}, {3}, {4, 5}}, []int{1, 2, 3, 4, 5}},
		{"with empty inner", [][]int{{1}, {}, {2}}, []int{1, 2}},
		{"empty outer", nil, []int{}},
		{"single slice", [][]int{{1, 2}}, []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Flatten(tt.slices)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Flatten(%v) = %v, want %v", tt.slices, got, tt.want)
			}
		})
	}
}

func TestFirst(t *testing.T) {
	if got := First([]int{7, 8, 9}, -1); got != 7 {
		t.Errorf("First = %d, want 7", got)
	}
	if got := First(nil, -1); got != -1 {
		t.Errorf("First(nil) = %d, want default -1", got)
	}
	if got := First([]string{}, "fallback"); got != "fallback" {
		t.Errorf("First(empty) = %q, want %q", got, "fallback")
	}
}

func TestLast(t *testing.T) {
	if got := Last([]int{7, 8, 9}, -1); got != 9 {
		t.Errorf("Last = %d, want 9", got)
	}
	if got := Last(nil, -1); got != -1 {
		t.Errorf("Last(nil) = %d, want default -1", got)
	}
	if got := Last([]string{"only"}, "fallback"); got != "only" {
		t.Errorf("Last(single) = %q, want %q", got, "only")
	}
}

// ---------- strings.go ----------

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"two words", "CamelCase", "camel_case"},
		{"leading lowercase", "camelCase", "camel_case"},
		{"single word lowercase", "camel", "camel"},
		{"single word capitalized", "Camel", "camel"},
		{"already snake", "already_snake", "already_snake"},
		{"digit before upper", "user2Name", "user2_name"},
		{"consecutive capitals each get underscore", "HTTPServer", "h_t_t_p_server"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CamelToSnake(tt.input); got != tt.want {
				t.Errorf("CamelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"two words", "snake_case", "SnakeCase"},
		{"three words", "a_b_c", "ABC"},
		{"single word", "single", "Single"},
		{"double underscore collapsed", "double__underscore", "DoubleUnderscore"},
		{"empty string", "", ""},
		{"trailing underscore", "word_", "Word"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SnakeToCamel(tt.input); got != tt.want {
				t.Errorf("SnakeToCamel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"shorter than max", "hi", 5, "hi"},
		{"exactly max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello..."},
		{"zero max", "abc", 0, "..."},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.input, tt.maxLen); got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase word", "hello", "Hello"},
		{"already capitalized", "Hello", "Hello"},
		{"single char", "h", "H"},
		{"digit first", "1abc", "1abc"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Capitalize(tt.input); got != tt.want {
				t.Errorf("Capitalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRemoveSpaces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"spaces between words", "a b c", "abc"},
		{"leading and trailing", " abc ", "abc"},
		{"no spaces", "abc", "abc"},
		{"only spaces", "   ", ""},
		{"tabs preserved", "a\tb", "a\tb"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveSpaces(tt.input); got != tt.want {
				t.Errorf("RemoveSpaces(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
