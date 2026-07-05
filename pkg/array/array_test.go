package array

import (
	"reflect"
	"testing"
)

func TestStringInArray(t *testing.T) {
	tests := []struct {
		name string
		s    string
		arr  []string
		want bool
	}{
		{"found first", "a", []string{"a", "b", "c"}, true},
		{"found last", "c", []string{"a", "b", "c"}, true},
		{"not found", "x", []string{"a", "b", "c"}, false},
		{"empty array", "a", nil, false},
		{"empty string found", "", []string{"", "b"}, true},
		{"empty string not found", "", []string{"a", "b"}, false},
		{"case sensitive", "A", []string{"a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringInArray(tt.s, tt.arr...); got != tt.want {
				t.Errorf("StringInArray(%q, %v) = %v, want %v", tt.s, tt.arr, got, tt.want)
			}
		})
	}
}

func TestIntersectionInt64(t *testing.T) {
	tests := []struct {
		name string
		a    []int64
		b    []int64
		want bool
	}{
		{"common element", []int64{1, 2, 3}, []int64{3, 4, 5}, true},
		{"no common element", []int64{1, 2}, []int64{3, 4}, false},
		{"a longer than b", []int64{1, 2, 3, 4, 5}, []int64{5}, true},
		{"b longer than a", []int64{5}, []int64{1, 2, 3, 4, 5}, true},
		{"identical slices", []int64{7, 8}, []int64{7, 8}, true},
		{"empty a", nil, []int64{1, 2}, false},
		{"empty b", []int64{1, 2}, nil, false},
		{"both empty", nil, nil, false},
		{"negative values", []int64{-1, -2}, []int64{-2}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IntersectionInt64(tt.a, tt.b); got != tt.want {
				t.Errorf("IntersectionInt64(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIntersectionString(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{"partial overlap", []string{"a", "b", "c"}, []string{"b", "c", "d"}, []string{"b", "c"}},
		{"no overlap", []string{"a"}, []string{"b"}, nil},
		{"identical", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		{"empty a", nil, []string{"a"}, nil},
		{"empty b", []string{"a"}, nil, nil},
		{"both empty", nil, nil, nil},
		{"preserves b order", []string{"a", "b", "c"}, []string{"c", "a"}, []string{"c", "a"}},
		{"duplicates in b repeated", []string{"a"}, []string{"a", "a"}, []string{"a", "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntersectionString(tt.a, tt.b)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IntersectionString(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestUniqueString(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		want []string
	}{
		{"removes duplicates", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"all same", []string{"x", "x", "x"}, []string{"x"}},
		{"preserves first occurrence order", []string{"c", "a", "c", "b"}, []string{"c", "a", "b"}},
		{"empty input", nil, []string{}},
		{"empty strings deduped", []string{"", "", "a"}, []string{"", "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueString(tt.a)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqueString(%v) = %v, want %v", tt.a, got, tt.want)
			}
		})
	}
}

func TestCheckFullStringIntersection(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"a subset of b", []string{"a", "b"}, []string{"a", "b", "c"}, true},
		{"a equals b", []string{"a", "b"}, []string{"b", "a"}, true},
		{"a not subset", []string{"a", "x"}, []string{"a", "b"}, false},
		{"a superset of b", []string{"a", "b", "c"}, []string{"a", "b"}, false},
		{"duplicates in a ignored", []string{"a", "a", "b"}, []string{"a", "b"}, true},
		{"duplicates in b ignored", []string{"a"}, []string{"a", "a"}, true},
		{"empty a always contained", nil, []string{"a"}, true},
		{"both empty", nil, nil, true},
		{"empty b non-empty a", []string{"a"}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckFullStringIntersection(tt.a, tt.b); got != tt.want {
				t.Errorf("CheckFullStringIntersection(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
