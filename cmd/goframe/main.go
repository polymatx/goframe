package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const version = "1.0.0"

//go:embed embedded/pkg
var embeddedPkg embed.FS

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "new":
		handleNew()
	case "gen":
		handleGen()
	case "migrate":
		handleMigrate()
	case "serve":
		handleServe()
	case "build":
		handleBuild()
	case "version":
		fmt.Printf("GoFrame CLI v%s\n", version)
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`GoFrame CLI - A toolkit for GoFrame projects

Usage:
  goframe <command> [arguments]

Commands:
  new <name>           Create new project with embedded framework packages
  gen model <name>     Generate model
  gen handler <name>   Generate handler
  gen crud <name>      Generate full CRUD (model + handler)
  gen middleware <name> Generate middleware
  migrate              Run database migrations
  serve                Start development server with hot reload
  build [output]       Build production binary
  version              Show version
  help                 Show this help

Examples:
  goframe new myapp
  goframe gen model User
  goframe gen handler user
  goframe gen crud Product
  goframe serve
  goframe build`)
}

func handleNew() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: goframe new <project-name>")
		os.Exit(1)
	}

	name := os.Args[2]
	if err := createProject(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Project '%s' created successfully!\n", name)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  go mod tidy\n")
	fmt.Printf("  go run cmd/server/main.go\n")
}

func createProject(name string) error {
	// Create directory structure
	dirs := []string{
		name,
		filepath.Join(name, "cmd", "server"),
		filepath.Join(name, "internal", "handlers"),
		filepath.Join(name, "internal", "models"),
		filepath.Join(name, "internal", "services"),
		filepath.Join(name, "pkg"),
		filepath.Join(name, "config"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Copy embedded pkg files
	if err := copyEmbeddedPkg(name); err != nil {
		return fmt.Errorf("failed to copy embedded packages: %w", err)
	}

	// Create main.go using local packages
	mainGo := `package main

import (
	"{{.Module}}/internal/handlers"
	"{{.Module}}/pkg/app"
	"{{.Module}}/pkg/middleware"
)

func main() {
	a := app.New(&app.Config{
		Name: "{{.Name}}",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	// Register root routes
	handlers.RegisterRootRoutes(a)

	// Register API routes
	api := a.Group("/api/v1")
	handlers.RegisterRoutes(api)

	a.StartWithGracefulShutdown()
}
`

	if err := writeTemplate(filepath.Join(name, "cmd", "server", "main.go"), mainGo, map[string]string{
		"Name":   name,
		"Module": name,
	}); err != nil {
		return err
	}

	// Create routes.go using local packages
	routesGo := `package handlers

import (
	"net/http"

	"{{.Module}}/pkg/app"
)

func RegisterRoutes(router *app.RouteGroup) {
	router.GET("/health", HealthHandler)
}

// RegisterRootRoutes registers routes at the root level
func RegisterRootRoutes(a *app.App) {
	a.Router().HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := app.NewContext(w, r)
		ctx.JSON(200, map[string]interface{}{
			"name":    "{{.Name}}",
			"version": "1.0.0",
			"status":  "running",
			"endpoints": map[string]string{
				"health": "/api/v1/health",
			},
		})
	}).Methods("GET")
}
`
	if err := writeTemplate(filepath.Join(name, "internal", "handlers", "routes.go"), routesGo, map[string]string{
		"Module": name,
		"Name":   name,
	}); err != nil {
		return err
	}

	// Create health.go using local packages
	healthGo := `package handlers

import (
	"net/http"
	"{{.Module}}/pkg/app"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	ctx.JSON(200, map[string]string{"status": "ok"})
}
`
	if err := writeTemplate(filepath.Join(name, "internal", "handlers", "health.go"), healthGo, map[string]string{
		"Module": name,
	}); err != nil {
		return err
	}

	// Create go.mod with required dependencies (no goframe dependency needed)
	goMod := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	github.com/rs/cors v1.10.1
	github.com/sirupsen/logrus v1.9.3
	github.com/go-playground/validator/v10 v10.16.0
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/prometheus/client_golang v1.17.0
	golang.org/x/time v0.5.0
	gorm.io/gorm v1.25.5
	gorm.io/driver/postgres v1.5.4
	gorm.io/driver/mysql v1.5.2
	gorm.io/driver/sqlite v1.5.4
)
`, name)
	if err := os.WriteFile(filepath.Join(name, "go.mod"), []byte(goMod), 0644); err != nil {
		return err
	}

	// Create .gitignore
	gitignore := `*.exe
*.out
*.log
.env
.env.*
tmp/
bin/
coverage.out
coverage.html
`
	if err := os.WriteFile(filepath.Join(name, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return err
	}

	// Add .gitkeep to empty directories
	emptyDirs := []string{
		filepath.Join(name, "internal", "models"),
		filepath.Join(name, "internal", "services"),
		filepath.Join(name, "config"),
	}
	for _, dir := range emptyDirs {
		if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), []byte(""), 0644); err != nil {
			return err
		}
	}

	return nil
}

// copyEmbeddedPkg copies embedded pkg files to the new project
func copyEmbeddedPkg(projectName string) error {
	return fs.WalkDir(embeddedPkg, "embedded/pkg", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath := strings.TrimPrefix(path, "embedded/")
		destPath := filepath.Join(projectName, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Read embedded file
		content, err := embeddedPkg.ReadFile(path)
		if err != nil {
			return err
		}

		// Write to destination
		return os.WriteFile(destPath, content, 0644)
	})
}

func handleGen() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: goframe gen <model|handler|crud|middleware> <name>")
		os.Exit(1)
	}

	genType := os.Args[2]
	name := os.Args[3]

	// Detect module name from go.mod
	moduleName := detectModuleName()

	switch genType {
	case "model":
		if err := generateModel(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Model '%s' generated: internal/models/%s.go\n", name, strings.ToLower(name))
	case "handler":
		if err := generateHandler(name, moduleName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Handler '%s' generated: internal/handlers/%s.go\n", name, strings.ToLower(name))
	case "crud":
		if err := generateModel(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := generateHandler(name, moduleName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ CRUD '%s' generated\n", name)
	case "middleware":
		if err := generateMiddleware(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Middleware '%s' generated: internal/middleware/%s.go\n", name, strings.ToLower(name))
	default:
		fmt.Printf("Unknown type: %s\n", genType)
		os.Exit(1)
	}
}

func detectModuleName() string {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return "myapp" // default fallback
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "myapp"
}

func generateModel(name string) error {
	tmpl := `package models

import (
	"time"
	"gorm.io/gorm"
)

type {{.Name}} struct {
	ID        uint           ` + "`" + `json:"id" gorm:"primarykey"` + "`" + `
	CreatedAt time.Time      ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt time.Time      ` + "`" + `json:"updated_at"` + "`" + `
	DeletedAt gorm.DeletedAt ` + "`" + `json:"-" gorm:"index"` + "`" + `
}

type {{.Name}}Service struct {
	db *gorm.DB
}

func New{{.Name}}Service(db *gorm.DB) *{{.Name}}Service {
	return &{{.Name}}Service{db: db}
}

func (s *{{.Name}}Service) Create(item *{{.Name}}) error {
	return s.db.Create(item).Error
}

func (s *{{.Name}}Service) GetByID(id uint) (*{{.Name}}, error) {
	var item {{.Name}}
	err := s.db.First(&item, id).Error
	return &item, err
}

func (s *{{.Name}}Service) GetAll() ([]{{.Name}}, error) {
	var items []{{.Name}}
	err := s.db.Find(&items).Error
	return items, err
}

func (s *{{.Name}}Service) Update(item *{{.Name}}) error {
	return s.db.Save(item).Error
}

func (s *{{.Name}}Service) Delete(id uint) error {
	return s.db.Delete(&{{.Name}}{}, id).Error
}
`
	if err := os.MkdirAll("internal/models", 0755); err != nil {
		return err
	}
	return writeTemplate(filepath.Join("internal", "models", strings.ToLower(name)+".go"), tmpl, map[string]string{
		"Name": name,
	})
}

func generateHandler(name, moduleName string) error {
	tmpl := `package handlers

import (
	"net/http"

	"{{.Module}}/pkg/app"
	"{{.Module}}/pkg/binding"
)

type {{.Name}}Handler struct{}

func New{{.Name}}Handler() *{{.Name}}Handler {
	return &{{.Name}}Handler{}
}

func (h *{{.Name}}Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	var req map[string]interface{}
	if err := binding.JSON(r, &req); err != nil {
		ctx.JSONError(400, err)
		return
	}
	ctx.JSON(201, map[string]string{"message": "created"})
}

func (h *{{.Name}}Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")
	ctx.JSON(200, map[string]string{"id": id})
}

func (h *{{.Name}}Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	ctx.JSON(200, []interface{}{})
}

func (h *{{.Name}}Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")
	var req map[string]interface{}
	if err := binding.JSON(r, &req); err != nil {
		ctx.JSONError(400, err)
		return
	}
	ctx.JSON(200, map[string]string{"id": id})
}

func (h *{{.Name}}Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	id := ctx.Param("id")
	ctx.JSON(200, map[string]string{"id": id})
}

func (h *{{.Name}}Handler) RegisterRoutes(router *app.RouteGroup) {
	router.POST("/{{.NameLower}}", h.Create)
	router.GET("/{{.NameLower}}", h.List)
	router.GET("/{{.NameLower}}/{id}", h.Get)
	router.PUT("/{{.NameLower}}/{id}", h.Update)
	router.DELETE("/{{.NameLower}}/{id}", h.Delete)
}
`
	if err := os.MkdirAll("internal/handlers", 0755); err != nil {
		return err
	}
	return writeTemplate(filepath.Join("internal", "handlers", strings.ToLower(name)+".go"), tmpl, map[string]string{
		"Name":      name,
		"NameLower": strings.ToLower(name),
		"Module":    moduleName,
	})
}

func generateMiddleware(name string) error {
	tmpl := `package middleware

import "net/http"

func {{.Name}}() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
`
	if err := os.MkdirAll("internal/middleware", 0755); err != nil {
		return err
	}
	return writeTemplate(filepath.Join("internal", "middleware", strings.ToLower(name)+".go"), tmpl, map[string]string{
		"Name": name,
	})
}

func handleMigrate() {
	// Check if migrations directory exists
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		fmt.Println("No migrations directory found. Create 'migrations/' directory with SQL files.")
		os.Exit(1)
	}

	fmt.Println("Running migrations...")
	fmt.Println("⚠ Note: Auto-migration requires database connection. Use GORM AutoMigrate in your app.")
	fmt.Println("✓ Migration check completed")
}

func handleServe() {
	fmt.Println("Starting dev server...")
	if _, err := exec.LookPath("air"); err != nil {
		fmt.Println("Installing air...")
		if err := exec.Command("go", "install", "github.com/cosmtrek/air@latest").Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install air: %v\n", err)
			os.Exit(1)
		}
	}
	cmd := exec.Command("air")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running air: %v\n", err)
		os.Exit(1)
	}
}

func handleBuild() {
	output := "bin/app"
	if len(os.Args) > 2 {
		output = os.Args[2]
	}
	fmt.Println("Building...")
	if err := os.MkdirAll("bin", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating bin directory: %v\n", err)
		os.Exit(1)
	}
	cmd := exec.Command("go", "build", "-o", output, "./cmd/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Built: %s\n", output)
}

func writeTemplate(path, tmplStr string, data map[string]string) error {
	tmpl, err := template.New("t").Parse(tmplStr)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}
