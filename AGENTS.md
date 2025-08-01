# Agent Guidelines for Salon Catalog

## Build/Test Commands
- **Build**: `go build -o bin/salon_catalog .` or `go run .`
- **Test**: `go test ./...` (no tests currently exist)
- **Single test**: `go test -run TestName ./package/path`
- **CSS build**: `cd web/style && bun run dev` (Tailwind CSS watch mode)
- **Templ generate**: `templ generate` (for .templ files)

## Code Style & Conventions
- **Package comments**: Start with "Package name provides..." format
- **Imports**: Group stdlib, third-party, then local packages with blank lines
- **Error handling**: Use predefined errors (e.g., `ErrNoConnStr`), log with `log.Printf`
- **Naming**: Use camelCase for variables, PascalCase for exported functions/types
- **HTTP handlers**: Follow pattern `func HandlerName(w http.ResponseWriter, r *http.Request)`
- **Database**: Use pgx/v5 with connection pooling, context.Background() for queries
- **Templates**: Use templ library, render with `.Render(context.Background(), w)`
- **Routes**: Use new Go 1.22 HTTP routing with method prefixes (e.g., "GET /path")
- **Auth**: Custom JWT middleware with auth.PopulateAuth wrapper
- **Static files**: Serve from `web/static/` directory
- **Environment**: Use godotenv for .env files, os.Getenv for config

## Project Structure
- `internal/`: Core application logic (db, routes, templates, auth, forms)
- `cmd/`: Command-line utilities
- `web/static/`: Static assets (CSS, JS, images)
- `sql/migrations/`: Database migration files
- `vendor/`: Go module dependencies