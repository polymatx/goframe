package binding

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// user is used for JSON/XML binding and validation tests.
type user struct {
	Name  string `json:"name" xml:"name" form:"name" validate:"required"`
	Age   int    `json:"age" xml:"age" form:"age" validate:"gte=0,lte=130"`
	Email string `json:"email" xml:"email" form:"email" validate:"omitempty,email"`
}

// formPayload exercises the supported form field kinds.
type formPayload struct {
	Name   string   `form:"name"`
	Age    int      `form:"age"`
	Active bool     `form:"active"`
	Score  float64  `form:"score"`
	Count  uint     `form:"count"`
	Tags   []string `form:"tags"`
	NoTag  string   // bound via lowercased field name "notag"
}

func newRequest(t *testing.T, method, target, contentType, body string) *http.Request {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}

func TestJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errContains string
		want        user
	}{
		{
			name: "valid payload",
			body: `{"name":"john","age":30,"email":"john@example.com"}`,
			want: user{Name: "john", Age: 30, Email: "john@example.com"},
		},
		{
			name:        "malformed JSON",
			body:        `{"name": "john"`,
			wantErr:     true,
			errContains: "invalid JSON",
		},
		{
			name:        "empty body",
			body:        "",
			wantErr:     true,
			errContains: "request body is empty",
		},
		{
			name:        "validation failure on missing required field",
			body:        `{"age":30}`,
			wantErr:     true,
			errContains: "Name",
		},
		{
			name:        "validation failure on out of range value",
			body:        `{"name":"john","age":200}`,
			wantErr:     true,
			errContains: "Age",
		},
		{
			name:        "validation failure on bad email",
			body:        `{"name":"john","age":30,"email":"not-an-email"}`,
			wantErr:     true,
			errContains: "Email",
		},
		{
			name:        "type mismatch",
			body:        `{"name":"john","age":"thirty"}`,
			wantErr:     true,
			errContains: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequest(t, http.MethodPost, "/users", "application/json", tt.body)

			var got user
			err := JSON(req, &got)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestXML(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errContains string
		want        user
	}{
		{
			name: "valid payload",
			body: `<user><name>jane</name><age>25</age><email>jane@example.com</email></user>`,
			want: user{Name: "jane", Age: 25, Email: "jane@example.com"},
		},
		{
			name:        "malformed XML",
			body:        `<user><name>jane</user>`,
			wantErr:     true,
			errContains: "invalid XML",
		},
		{
			name:        "validation failure on missing required field",
			body:        `<user><age>25</age></user>`,
			wantErr:     true,
			errContains: "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequest(t, http.MethodPost, "/users", "application/xml", tt.body)

			var got user
			err := XML(req, &got)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestForm(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
		want    formPayload
	}{
		{
			name: "all supported kinds",
			body: url.Values{
				"name":   {"john"},
				"age":    {"30"},
				"active": {"true"},
				"score":  {"9.5"},
				"count":  {"7"},
				"tags":   {"go", "web"},
				"notag":  {"fallback"},
			}.Encode(),
			want: formPayload{
				Name:   "john",
				Age:    30,
				Active: true,
				Score:  9.5,
				Count:  7,
				Tags:   []string{"go", "web"},
				NoTag:  "fallback",
			},
		},
		{
			name: "missing fields are left as zero values",
			body: url.Values{"name": {"solo"}}.Encode(),
			want: formPayload{Name: "solo"},
		},
		{
			name:    "invalid int value",
			body:    url.Values{"age": {"not-a-number"}}.Encode(),
			wantErr: true,
		},
		{
			name:    "invalid bool value",
			body:    url.Values{"active": {"maybe"}}.Encode(),
			wantErr: true,
		},
		{
			name:    "invalid float value",
			body:    url.Values{"score": {"high"}}.Encode(),
			wantErr: true,
		},
		{
			name:    "invalid uint value",
			body:    url.Values{"count": {"-1"}}.Encode(),
			wantErr: true,
		},
		{
			name:    "malformed percent encoding",
			body:    "name=%zz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequest(t, http.MethodPost, "/users", "application/x-www-form-urlencoded", tt.body)

			var got formPayload
			err := Form(req, &got)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestForm_UnsupportedFieldKind(t *testing.T) {
	type withMap struct {
		Meta map[string]string `form:"meta"`
	}

	req := newRequest(t, http.MethodPost, "/users", "application/x-www-form-urlencoded",
		url.Values{"meta": {"x"}}.Encode())

	var got withMap
	err := Form(req, &got)
	if err == nil {
		t.Fatal("expected error for unsupported field kind, got nil")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("expected 'unknown type' error, got %q", err.Error())
	}
}

func TestQuery(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
		want    formPayload
	}{
		{
			name:   "binds query parameters",
			target: "/search?name=jane&age=25&active=true&tags=a&tags=b",
			want:   formPayload{Name: "jane", Age: 25, Active: true, Tags: []string{"a", "b"}},
		},
		{
			name:   "no query parameters leaves zero values",
			target: "/search",
			want:   formPayload{},
		},
		{
			name:    "invalid numeric query value",
			target:  "/search?age=abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)

			var got formPayload
			err := Query(req, &got)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestBind_ContentTypeDispatch(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        user
	}{
		{
			name:        "application/json",
			contentType: "application/json",
			body:        `{"name":"john","age":30}`,
			want:        user{Name: "john", Age: 30},
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			body:        `{"name":"john","age":30}`,
			want:        user{Name: "john", Age: 30},
		},
		{
			name:        "application/xml",
			contentType: "application/xml",
			body:        `<user><name>jane</name><age>25</age></user>`,
			want:        user{Name: "jane", Age: 25},
		},
		{
			name:        "form urlencoded",
			contentType: "application/x-www-form-urlencoded",
			body:        url.Values{"name": {"form-user"}, "age": {"40"}}.Encode(),
			want:        user{Name: "form-user", Age: 40},
		},
		{
			name:        "unknown content type falls back to JSON",
			contentType: "text/weird",
			body:        `{"name":"fallback","age":1}`,
			want:        user{Name: "fallback", Age: 1},
		},
		{
			name:        "missing content type falls back to JSON",
			contentType: "",
			body:        `{"name":"fallback","age":1}`,
			want:        user{Name: "fallback", Age: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequest(t, http.MethodPost, "/users", tt.contentType, tt.body)

			var got user
			if err := Bind(req, &got); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestBind_InvalidPayload(t *testing.T) {
	req := newRequest(t, http.MethodPost, "/users", "application/json", `not json at all`)

	var got user
	if err := Bind(req, &got); err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		obj     user
		wantErr bool
	}{
		{
			name: "valid struct",
			obj:  user{Name: "john", Age: 30, Email: "john@example.com"},
		},
		{
			name: "valid struct without optional email",
			obj:  user{Name: "john", Age: 30},
		},
		{
			name:    "missing required field",
			obj:     user{Age: 30},
			wantErr: true,
		},
		{
			name:    "value out of range",
			obj:     user{Name: "john", Age: 131},
			wantErr: true,
		},
		{
			name:    "invalid email",
			obj:     user{Name: "john", Age: 30, Email: "nope"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.obj)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
