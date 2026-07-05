package render

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type person struct {
	Name string `json:"name" xml:"name"`
	Age  int    `json:"age" xml:"age"`
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file %s: %v", path, err)
	}
	return path
}

func TestJSON(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		obj      interface{}
		wantBody string
	}{
		{"struct", http.StatusOK, person{Name: "john", Age: 30}, `{"name":"john","age":30}` + "\n"},
		{"map", http.StatusCreated, map[string]int{"a": 1}, `{"a":1}` + "\n"},
		{"nil object", http.StatusOK, nil, "null\n"},
		{"empty slice", http.StatusOK, []int{}, "[]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			if err := JSON(rec, tt.code, tt.obj); err != nil {
				t.Fatalf("JSON returned error: %v", err)
			}
			if rec.Code != tt.code {
				t.Errorf("status = %d, want %d", rec.Code, tt.code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}

	t.Run("unencodable object returns error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := JSON(rec, http.StatusOK, make(chan int)); err == nil {
			t.Error("expected error encoding channel to JSON")
		}
	})
}

func TestJSONIndent(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := JSONIndent(rec, http.StatusOK, person{Name: "john", Age: 30}); err != nil {
		t.Fatalf("JSONIndent returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}

	want := "{\n  \"name\": \"john\",\n  \"age\": 30\n}\n"
	if rec.Body.String() != want {
		t.Errorf("body = %q, want %q", rec.Body.String(), want)
	}

	// Indented output must still be valid JSON.
	var decoded person
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("indented output is not valid JSON: %v", err)
	}
	if decoded != (person{Name: "john", Age: 30}) {
		t.Errorf("decoded = %+v, want original struct", decoded)
	}
}

func TestXML(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := XML(rec, http.StatusOK, person{Name: "john", Age: 30}); err != nil {
			t.Fatalf("XML returned error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/xml; charset=utf-8" {
			t.Errorf("Content-Type = %q, want application/xml; charset=utf-8", ct)
		}
		want := "<person><name>john</name><age>30</age></person>"
		if rec.Body.String() != want {
			t.Errorf("body = %q, want %q", rec.Body.String(), want)
		}
	})

	t.Run("unencodable object returns error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := XML(rec, http.StatusOK, map[string]int{"a": 1}); err == nil {
			t.Error("expected error encoding map to XML")
		}
	})
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		format   string
		values   []interface{}
		wantBody string
	}{
		{"plain text", http.StatusOK, "hello", nil, "hello"},
		{"with format args", http.StatusAccepted, "user %s has %d points", []interface{}{"john", 42}, "user john has 42 points"},
		{"empty format", http.StatusOK, "", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			if err := String(rec, tt.code, tt.format, tt.values...); err != nil {
				t.Fatalf("String returned error: %v", err)
			}
			if rec.Code != tt.code {
				t.Errorf("status = %d, want %d", rec.Code, tt.code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", ct)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHTML(t *testing.T) {
	t.Run("renders template file", func(t *testing.T) {
		path := writeTempFile(t, t.TempDir(), "page.html", "<h1>{{.Title}}</h1>")

		rec := httptest.NewRecorder()
		err := HTML(rec, http.StatusOK, path, map[string]string{"Title": "Welcome"})
		if err != nil {
			t.Fatalf("HTML returned error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
		}
		if rec.Body.String() != "<h1>Welcome</h1>" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "<h1>Welcome</h1>")
		}
	})

	t.Run("escapes data", func(t *testing.T) {
		path := writeTempFile(t, t.TempDir(), "page.html", "{{.Title}}")

		rec := httptest.NewRecorder()
		if err := HTML(rec, http.StatusOK, path, map[string]string{"Title": "<script>"}); err != nil {
			t.Fatalf("HTML returned error: %v", err)
		}
		if strings.Contains(rec.Body.String(), "<script>") {
			t.Errorf("body %q should have HTML-escaped the data", rec.Body.String())
		}
	})

	t.Run("missing template file returns error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := HTML(rec, http.StatusOK, filepath.Join(t.TempDir(), "missing.html"), nil)
		if err == nil {
			t.Error("expected error for nonexistent template file")
		}
	})
}

func TestHTMLString(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := HTMLString(rec, http.StatusOK, "<p>hi</p>"); err != nil {
		t.Fatalf("HTMLString returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
	if rec.Body.String() != "<p>hi</p>" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "<p>hi</p>")
	}

	t.Run("empty html", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := HTMLString(rec, http.StatusNotFound, ""); err != nil {
			t.Fatalf("HTMLString returned error: %v", err)
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", rec.Body.String())
		}
	})
}

func TestFile(t *testing.T) {
	t.Run("sends existing file", func(t *testing.T) {
		content := "file-content-123"
		path := writeTempFile(t, t.TempDir(), "download.txt", content)

		rec := httptest.NewRecorder()
		if err := File(rec, path); err != nil {
			t.Fatalf("File returned error: %v", err)
		}
		if rec.Body.String() != content {
			t.Errorf("body = %q, want %q", rec.Body.String(), content)
		}
		if cd := rec.Header().Get("Content-Disposition"); cd != "attachment; filename=download.txt" {
			t.Errorf("Content-Disposition = %q, want attachment; filename=download.txt", cd)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
			t.Errorf("Content-Type = %q, want application/octet-stream", ct)
		}
		if cl := rec.Header().Get("Content-Length"); cl != fmt.Sprintf("%d", len(content)) {
			t.Errorf("Content-Length = %q, want %d", cl, len(content))
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := writeTempFile(t, t.TempDir(), "empty.bin", "")

		rec := httptest.NewRecorder()
		if err := File(rec, path); err != nil {
			t.Fatalf("File returned error: %v", err)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", rec.Body.String())
		}
		if cl := rec.Header().Get("Content-Length"); cl != "0" {
			t.Errorf("Content-Length = %q, want 0", cl)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := File(rec, filepath.Join(t.TempDir(), "nope.txt")); err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestFileAttachment(t *testing.T) {
	t.Run("uses custom filename", func(t *testing.T) {
		content := "attachment-content"
		path := writeTempFile(t, t.TempDir(), "internal-name.dat", content)

		rec := httptest.NewRecorder()
		if err := FileAttachment(rec, path, "report.pdf"); err != nil {
			t.Fatalf("FileAttachment returned error: %v", err)
		}
		if cd := rec.Header().Get("Content-Disposition"); cd != "attachment; filename=report.pdf" {
			t.Errorf("Content-Disposition = %q, want attachment; filename=report.pdf", cd)
		}
		if rec.Body.String() != content {
			t.Errorf("body = %q, want %q", rec.Body.String(), content)
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := FileAttachment(rec, filepath.Join(t.TempDir(), "nope.txt"), "x.txt"); err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestData(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		contentType string
		data        []byte
	}{
		{"binary payload", http.StatusOK, "application/pdf", []byte{0x25, 0x50, 0x44, 0x46}},
		{"text payload", http.StatusCreated, "text/csv", []byte("a,b\n1,2")},
		{"empty payload", http.StatusOK, "application/octet-stream", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			if err := Data(rec, tt.code, tt.contentType, tt.data); err != nil {
				t.Fatalf("Data returned error: %v", err)
			}
			if rec.Code != tt.code {
				t.Errorf("status = %d, want %d", rec.Code, tt.code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.contentType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.contentType)
			}
			if rec.Body.String() != string(tt.data) {
				t.Errorf("body = %q, want %q", rec.Body.String(), string(tt.data))
			}
		})
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	NoContent(rec)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestRedirect(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		location string
	}{
		{"found", http.StatusFound, "/login"},
		{"moved permanently", http.StatusMovedPermanently, "https://example.com/new"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/old", nil)
			Redirect(rec, req, tt.code, tt.location)
			if rec.Code != tt.code {
				t.Errorf("status = %d, want %d", rec.Code, tt.code)
			}
			if loc := rec.Header().Get("Location"); loc != tt.location {
				t.Errorf("Location = %q, want %q", loc, tt.location)
			}
		})
	}
}

func TestNewTemplateRenderer(t *testing.T) {
	t.Run("parses glob", func(t *testing.T) {
		dir := t.TempDir()
		writeTempFile(t, dir, "a.html", "<p>A: {{.}}</p>")
		writeTempFile(t, dir, "b.html", "<p>B: {{.}}</p>")

		tr, err := NewTemplateRenderer(filepath.Join(dir, "*.html"))
		if err != nil {
			t.Fatalf("NewTemplateRenderer returned error: %v", err)
		}
		if tr == nil {
			t.Fatal("expected non-nil renderer")
		}
	})

	t.Run("pattern matching no files returns error", func(t *testing.T) {
		tr, err := NewTemplateRenderer(filepath.Join(t.TempDir(), "*.html"))
		if err == nil {
			t.Error("expected error for glob matching no files")
		}
		if tr != nil {
			t.Error("expected nil renderer on error")
		}
	})
}

func TestTemplateRenderer_Render(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "greet.html", "<p>Hello {{.Name}}</p>")

	tr, err := NewTemplateRenderer(filepath.Join(dir, "*.html"))
	if err != nil {
		t.Fatalf("NewTemplateRenderer returned error: %v", err)
	}

	t.Run("renders known template", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := tr.Render(rec, http.StatusOK, "greet.html", map[string]string{"Name": "john"})
		if err != nil {
			t.Fatalf("Render returned error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
		}
		if rec.Body.String() != "<p>Hello john</p>" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "<p>Hello john</p>")
		}
	})

	t.Run("unknown template name returns error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := tr.Render(rec, http.StatusOK, "missing.html", nil); err == nil {
			t.Error("expected error rendering unknown template name")
		}
	})
}

func TestTemplateRenderer_AddTemplate(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "base.html", "base")

	tr, err := NewTemplateRenderer(filepath.Join(dir, "*.html"))
	if err != nil {
		t.Fatalf("NewTemplateRenderer returned error: %v", err)
	}

	t.Run("adds new template file", func(t *testing.T) {
		extra := writeTempFile(t, t.TempDir(), "extra.html", "extra: {{.}}")
		if err := tr.AddTemplate(extra); err != nil {
			t.Fatalf("AddTemplate returned error: %v", err)
		}

		rec := httptest.NewRecorder()
		if err := tr.Render(rec, http.StatusOK, "extra.html", "value"); err != nil {
			t.Fatalf("Render of added template returned error: %v", err)
		}
		if rec.Body.String() != "extra: value" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "extra: value")
		}
	})

	t.Run("original template still renders", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := tr.Render(rec, http.StatusOK, "base.html", nil); err != nil {
			t.Fatalf("Render returned error: %v", err)
		}
		if rec.Body.String() != "base" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "base")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		if err := tr.AddTemplate(filepath.Join(t.TempDir(), "missing.html")); err == nil {
			t.Error("expected error adding nonexistent template file")
		}
	})
}

func TestTemplateRenderer_AddTemplateGlob(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "base.html", "base")

	tr, err := NewTemplateRenderer(filepath.Join(dir, "*.html"))
	if err != nil {
		t.Fatalf("NewTemplateRenderer returned error: %v", err)
	}

	t.Run("adds templates from glob", func(t *testing.T) {
		other := t.TempDir()
		writeTempFile(t, other, "one.html", "one")
		writeTempFile(t, other, "two.html", "two")

		if err := tr.AddTemplateGlob(filepath.Join(other, "*.html")); err != nil {
			t.Fatalf("AddTemplateGlob returned error: %v", err)
		}

		for _, name := range []string{"one.html", "two.html", "base.html"} {
			rec := httptest.NewRecorder()
			if err := tr.Render(rec, http.StatusOK, name, nil); err != nil {
				t.Errorf("Render(%q) returned error: %v", name, err)
			}
		}
	})

	t.Run("glob matching no files returns error", func(t *testing.T) {
		if err := tr.AddTemplateGlob(filepath.Join(t.TempDir(), "*.html")); err == nil {
			t.Error("expected error for glob matching no files")
		}
	})
}
