package database

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestNullString(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		tests := []struct {
			name string
			in   NullString
			want interface{}
		}{
			{"valid returns string", NullString{Valid: true, String: "hello"}, "hello"},
			{"valid empty returns empty string", NullString{Valid: true, String: ""}, ""},
			{"invalid returns nil", NullString{Valid: false, String: "ignored"}, nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Value()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("expected %v, got %v", tt.want, got)
				}
			})
		}
	})

	t.Run("Scan", func(t *testing.T) {
		tests := []struct {
			name      string
			src       interface{}
			wantValid bool
			wantStr   string
		}{
			{"string", "world", true, "world"},
			{"bytes", []byte("bytes"), true, "bytes"},
			{"nil becomes invalid", nil, false, ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var ns NullString
				if err := ns.Scan(tt.src); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if ns.Valid != tt.wantValid || ns.String != tt.wantStr {
					t.Errorf("expected {Valid:%v String:%q}, got {Valid:%v String:%q}",
						tt.wantValid, tt.wantStr, ns.Valid, ns.String)
				}
			})
		}
	})

	t.Run("JSON roundtrip", func(t *testing.T) {
		tests := []struct {
			name     string
			in       NullString
			wantJSON string
		}{
			{"valid", NullString{Valid: true, String: "abc"}, `"abc"`},
			{"invalid marshals to null", NullString{Valid: false}, "null"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b, err := json.Marshal(tt.in)
				if err != nil {
					t.Fatalf("marshal error: %v", err)
				}
				if string(b) != tt.wantJSON {
					t.Fatalf("expected JSON %s, got %s", tt.wantJSON, b)
				}

				var out NullString
				if err := json.Unmarshal(b, &out); err != nil {
					t.Fatalf("unmarshal error: %v", err)
				}
				if out != tt.in {
					t.Errorf("roundtrip mismatch: expected %+v, got %+v", tt.in, out)
				}
			})
		}
	})

	t.Run("UnmarshalJSON invalid input", func(t *testing.T) {
		var ns NullString
		if err := ns.UnmarshalJSON([]byte(`{`)); err == nil {
			t.Error("expected error for malformed JSON")
		}
	})
}

func TestNullInt64(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		valid := NullInt64{Int64: 42, Valid: true}
		v, err := valid.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != int64(42) {
			t.Errorf("expected 42, got %v", v)
		}

		invalid := NullInt64{Int64: 42, Valid: false}
		v, err = invalid.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != nil {
			t.Errorf("expected nil for invalid, got %v", v)
		}
	})

	t.Run("Scan", func(t *testing.T) {
		tests := []struct {
			name      string
			src       interface{}
			wantValid bool
			wantInt   int64
		}{
			{"int64", int64(7), true, 7},
			{"nil becomes invalid", nil, false, 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var ni NullInt64
				if err := ni.Scan(tt.src); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if ni.Valid != tt.wantValid || ni.Int64 != tt.wantInt {
					t.Errorf("expected {Valid:%v Int64:%d}, got {Valid:%v Int64:%d}",
						tt.wantValid, tt.wantInt, ni.Valid, ni.Int64)
				}
			})
		}
	})

	t.Run("JSON roundtrip", func(t *testing.T) {
		tests := []struct {
			name     string
			in       NullInt64
			wantJSON string
		}{
			{"valid", NullInt64{Int64: 123, Valid: true}, "123"},
			{"negative", NullInt64{Int64: -9, Valid: true}, "-9"},
			{"invalid marshals to null", NullInt64{}, "null"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b, err := json.Marshal(tt.in)
				if err != nil {
					t.Fatalf("marshal error: %v", err)
				}
				if string(b) != tt.wantJSON {
					t.Fatalf("expected JSON %s, got %s", tt.wantJSON, b)
				}

				var out NullInt64
				if err := json.Unmarshal(b, &out); err != nil {
					t.Fatalf("unmarshal error: %v", err)
				}
				if out != tt.in {
					t.Errorf("roundtrip mismatch: expected %+v, got %+v", tt.in, out)
				}
			})
		}
	})

	t.Run("UnmarshalJSON invalid input", func(t *testing.T) {
		var ni NullInt64
		if err := ni.UnmarshalJSON([]byte(`"not-a-number"`)); err == nil {
			t.Error("expected error for non-numeric JSON")
		}
	})
}

func TestNullFloat64(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		valid := NullFloat64{Float64: 2.5, Valid: true}
		v, err := valid.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 2.5 {
			t.Errorf("expected 2.5, got %v", v)
		}

		invalid := NullFloat64{}
		v, err = invalid.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != nil {
			t.Errorf("expected nil for invalid, got %v", v)
		}
	})

	t.Run("Scan", func(t *testing.T) {
		var nf NullFloat64
		if err := nf.Scan(3.5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !nf.Valid || nf.Float64 != 3.5 {
			t.Errorf("expected {Valid:true Float64:3.5}, got %+v", nf)
		}

		var nilNf NullFloat64
		if err := nilNf.Scan(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nilNf.Valid {
			t.Error("expected invalid after scanning nil")
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		// Note: MarshalJSON uses %f, so output always has six decimal places.
		tests := []struct {
			name     string
			in       NullFloat64
			wantJSON string
		}{
			{"valid", NullFloat64{Float64: 3.14, Valid: true}, "3.140000"},
			{"invalid marshals to null", NullFloat64{}, "null"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b, err := json.Marshal(tt.in)
				if err != nil {
					t.Fatalf("marshal error: %v", err)
				}
				if string(b) != tt.wantJSON {
					t.Errorf("expected JSON %s, got %s", tt.wantJSON, b)
				}
			})
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		var nf NullFloat64
		if err := json.Unmarshal([]byte("1.25"), &nf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !nf.Valid || nf.Float64 != 1.25 {
			t.Errorf("expected {Valid:true Float64:1.25}, got %+v", nf)
		}

		var nullNf NullFloat64
		if err := json.Unmarshal([]byte("null"), &nullNf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nullNf.Valid {
			t.Error("expected invalid after unmarshaling null")
		}
	})
}

func TestNullBool(t *testing.T) {
	t.Run("Value", func(t *testing.T) {
		tests := []struct {
			name string
			in   NullBool
			want interface{}
		}{
			{"valid true", NullBool{Bool: true, Valid: true}, true},
			{"valid false", NullBool{Bool: false, Valid: true}, false},
			{"invalid returns nil", NullBool{Bool: true, Valid: false}, nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.in.Value()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("expected %v, got %v", tt.want, got)
				}
			})
		}
	})

	t.Run("Scan", func(t *testing.T) {
		tests := []struct {
			name      string
			src       interface{}
			wantValid bool
			wantBool  bool
		}{
			{"bool true", true, true, true},
			{"bool false", false, true, false},
			{"int64 one", int64(1), true, true},
			{"nil becomes invalid", nil, false, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var nb NullBool
				if err := nb.Scan(tt.src); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if nb.Valid != tt.wantValid || nb.Bool != tt.wantBool {
					t.Errorf("expected {Valid:%v Bool:%v}, got {Valid:%v Bool:%v}",
						tt.wantValid, tt.wantBool, nb.Valid, nb.Bool)
				}
			})
		}
	})

	t.Run("JSON roundtrip", func(t *testing.T) {
		tests := []struct {
			name     string
			in       NullBool
			wantJSON string
		}{
			{"true", NullBool{Bool: true, Valid: true}, "true"},
			{"false", NullBool{Bool: false, Valid: true}, "false"},
			{"invalid marshals to null", NullBool{}, "null"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				b, err := json.Marshal(tt.in)
				if err != nil {
					t.Fatalf("marshal error: %v", err)
				}
				if string(b) != tt.wantJSON {
					t.Fatalf("expected JSON %s, got %s", tt.wantJSON, b)
				}

				var out NullBool
				if err := json.Unmarshal(b, &out); err != nil {
					t.Fatalf("unmarshal error: %v", err)
				}
				if out != tt.in {
					t.Errorf("roundtrip mismatch: expected %+v, got %+v", tt.in, out)
				}
			})
		}
	})

	t.Run("UnmarshalJSON invalid input", func(t *testing.T) {
		var nb NullBool
		if err := nb.UnmarshalJSON([]byte(`"yes"`)); err == nil {
			t.Error("expected error for non-boolean JSON")
		}
	})
}

func TestNullTime(t *testing.T) {
	ref := time.Date(2024, 5, 1, 10, 30, 0, 0, time.UTC)

	t.Run("Value", func(t *testing.T) {
		valid := NullTime{Time: ref, Valid: true}
		v, err := valid.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := v.(time.Time)
		if !ok || !got.Equal(ref) {
			t.Errorf("expected %v, got %v", ref, v)
		}

		invalid := NullTime{}
		v, err = invalid.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != nil {
			t.Errorf("expected nil for invalid, got %v", v)
		}
	})

	t.Run("Scan", func(t *testing.T) {
		var nt NullTime
		if err := nt.Scan(ref); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !nt.Valid || !nt.Time.Equal(ref) {
			t.Errorf("expected valid time %v, got %+v", ref, nt)
		}

		// Non-time values are silently treated as NULL.
		var invalid NullTime
		if err := invalid.Scan("not a time"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if invalid.Valid {
			t.Error("expected invalid after scanning non-time value")
		}

		var nilTime NullTime
		if err := nilTime.Scan(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nilTime.Valid {
			t.Error("expected invalid after scanning nil")
		}
	})

	t.Run("JSON roundtrip", func(t *testing.T) {
		in := NullTime{Time: ref, Valid: true}
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		if string(b) != `"2024-05-01T10:30:00Z"` {
			t.Fatalf("unexpected JSON: %s", b)
		}

		var out NullTime
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if !out.Valid || !out.Time.Equal(ref) {
			t.Errorf("roundtrip mismatch: expected %v, got %+v", ref, out)
		}

		nullB, err := json.Marshal(NullTime{})
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		if string(nullB) != "null" {
			t.Errorf("expected null, got %s", nullB)
		}

		var nullOut NullTime
		if err := json.Unmarshal([]byte("null"), &nullOut); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if nullOut.Valid || !nullOut.Time.IsZero() {
			t.Errorf("expected zero invalid time, got %+v", nullOut)
		}
	})

	t.Run("String", func(t *testing.T) {
		if got := (NullTime{}).String(); got != "null" {
			t.Errorf("expected 'null', got %q", got)
		}
		valid := NullTime{Time: ref, Valid: true}
		if got := valid.String(); got != ref.String() {
			t.Errorf("expected %q, got %q", ref.String(), got)
		}
	})
}

func TestInt64Slice(t *testing.T) {
	t.Run("Value produces JSON array", func(t *testing.T) {
		v, err := Int64Slice{1, 2, 3}.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		b, ok := v.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", v)
		}
		if string(b) != "[1,2,3]" {
			t.Errorf("expected [1,2,3], got %s", b)
		}
	})

	t.Run("Scan", func(t *testing.T) {
		tests := []struct {
			name    string
			src     interface{}
			want    Int64Slice
			wantErr bool
		}{
			{"from bytes", []byte("[4,5,6]"), Int64Slice{4, 5, 6}, false},
			{"from string", "[7,8]", Int64Slice{7, 8}, false},
			{"empty array", "[]", Int64Slice{}, false},
			{"unsupported type", 42, nil, true},
			// NULL scans as empty input, which is not valid JSON.
			{"nil errors", nil, nil, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var s Int64Slice
				err := s.Scan(tt.src)
				if tt.wantErr {
					if err == nil {
						t.Fatal("expected error")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(s, tt.want) {
					t.Errorf("expected %v, got %v", tt.want, s)
				}
			})
		}
	})

	t.Run("Value/Scan roundtrip", func(t *testing.T) {
		in := Int64Slice{10, -20, 30}
		v, err := in.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var out Int64Slice
		if err := out.Scan(v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Errorf("roundtrip mismatch: expected %v, got %v", in, out)
		}
	})
}

func TestInt64Array(t *testing.T) {
	t.Run("Value produces PostgreSQL array format", func(t *testing.T) {
		v, err := Int64Array{1, 2, 3}.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		b, ok := v.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", v)
		}
		if string(b) != "{1,2,3}" {
			t.Errorf("expected {1,2,3}, got %s", b)
		}
	})

	t.Run("Value of empty array", func(t *testing.T) {
		v, err := Int64Array{}.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(v.([]byte)) != "{}" {
			t.Errorf("expected {}, got %s", v)
		}
	})

	t.Run("Scan", func(t *testing.T) {
		tests := []struct {
			name    string
			src     interface{}
			want    Int64Array
			wantErr bool
		}{
			{"from string", "{1,2,3}", Int64Array{1, 2, 3}, false},
			{"from bytes", []byte("{9}"), Int64Array{9}, false},
			{"empty", "{}", Int64Array{}, false},
			{"unsupported type", 3.14, nil, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var a Int64Array
				err := a.Scan(tt.src)
				if tt.wantErr {
					if err == nil {
						t.Fatal("expected error")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(a, tt.want) {
					t.Errorf("expected %v, got %v", tt.want, a)
				}
			})
		}
	})

	t.Run("Value/Scan roundtrip", func(t *testing.T) {
		in := Int64Array{5, 10, 15}
		v, err := in.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var out Int64Array
		if err := out.Scan(v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Errorf("roundtrip mismatch: expected %v, got %v", in, out)
		}
	})
}

func TestGenericJSONField(t *testing.T) {
	t.Run("Value/Scan roundtrip", func(t *testing.T) {
		in := GenericJSONField{
			"name":   "goframe",
			"count":  float64(3),
			"active": true,
			"nested": map[string]interface{}{"key": "value"},
		}
		v, err := in.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var out GenericJSONField
		if err := out.Scan(v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(map[string]interface{}(in), map[string]interface{}(out)) {
			t.Errorf("roundtrip mismatch: expected %v, got %v", in, out)
		}
	})

	t.Run("Scan from string", func(t *testing.T) {
		var g GenericJSONField
		if err := g.Scan(`{"a":1}`); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if g["a"] != float64(1) {
			t.Errorf("expected a=1, got %v", g["a"])
		}
	})

	t.Run("Scan nil leaves field untouched", func(t *testing.T) {
		var g GenericJSONField
		if err := g.Scan(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if g != nil {
			t.Errorf("expected nil map, got %v", g)
		}
	})

	t.Run("Scan unsupported type", func(t *testing.T) {
		var g GenericJSONField
		if err := g.Scan(42); err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}

func TestStringJSONArray(t *testing.T) {
	t.Run("Value/Scan roundtrip", func(t *testing.T) {
		in := StringJSONArray{"a", "b", "c"}
		v, err := in.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var out StringJSONArray
		if err := out.Scan(v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Errorf("roundtrip mismatch: expected %v, got %v", in, out)
		}
	})

	t.Run("Scan unsupported type", func(t *testing.T) {
		var out StringJSONArray
		if err := out.Scan(1); err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}

func TestStringMapJSONArray(t *testing.T) {
	t.Run("Value/Scan roundtrip", func(t *testing.T) {
		in := StringMapJSONArray{"tags": {"a", "b"}, "ids": {"1"}}
		v, err := in.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var out StringMapJSONArray
		if err := out.Scan(v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(in, out) {
			t.Errorf("roundtrip mismatch: expected %v, got %v", in, out)
		}
	})

	t.Run("Scan unsupported type", func(t *testing.T) {
		var out StringMapJSONArray
		if err := out.Scan(false); err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}
