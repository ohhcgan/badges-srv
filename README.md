# GitHub Badges Backend

A Go backend for GitHub activity badges and monthly report posters. This service authenticates users with GitHub OAuth, collects GitHub activity statistics, stores monthly summaries, generates report images, and emails a visual activity report to users.

## Key Features

1. GitHub OAuth login flow
2. JWT session cookie authentication
3. PostgreSQL-backed user and monthly stats storage
4. GitHub contribution collection for the previous month
5. Monthly activity poster generation as PNG
6. Email delivery of report posters
7. Scheduled monthly processing at 02:00 UTC on the 1st of each month
8. Admin-triggered manual monthly job via API

## Architecture

1. `cmd/server/main.go` - application bootstrap and graceful shutdown
2. `internal/config` - environment configuration loader
3. `internal/auth` - OAuth login, JWT session handling, protected routes
4. `internal/database` - PostgreSQL connection and schema migration SQL
5. `internal/github` - GitHub API GraphQL integration and user data fetching
6. `internal/poster` - image generation for monthly report posters
7. `internal/mailer` - email composition and SMTP delivery
8. `internal/scheduler` - monthly batch job orchestration
9. `internal/server` - Gin HTTP server and API controllers
10. `internal/stats` - monthly statistics collection and storage
11. `internal/user` - user persistence and lookup

## Requirements

1. Go 1.26
2. PostgreSQL database
3. GitHub OAuth app credentials
4. SMTP credentials for email delivery

## Environment Variables

Create a `.env` file in the project root with these values:

```env
PORT=8080
FRONTEND_URL=http://localhost:3000
DATABASE_URL=postgres://user:pass@localhost:5432/github_badges?sslmode=disable

GITHUB_CLIENT_ID=your_github_client_id
GITHUB_REDIRECT_URL=http://localhost:8080/auth/callback
GITHUB_CLIENT_SECRET=your_github_client_secret

JWT_SECRET=your_jwt_secret
ENCRYPTION_KEY=your_32_byte_hex_key

SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=you@example.com
SMTP_PASS=your_smtp_password
EMAIL_FROM="GitHub Badges <noreply@example.com>"

ADMIN_KEY=your_admin_api_key
ENVIRONMENT=development
```

## Database Setup

The database schema is defined in `internal/database/migrations/init.sql`.

Run the SQL file manually or with your preferred migration tool:

```bash
psql "$DATABASE_URL" -f internal/database/migrations/init.sql
```

## Build & Run

```bash
go build ./cmd/server/main.go
./cmd/server/main
```

Or run directly:

```bash
go run ./cmd/server/main.go
```

## API Endpoints

### Public

- `GET /health`
  - Health check endpoint
- `GET /auth/login`
  - Redirects to GitHub OAuth consent page
- `GET /auth/callback`
  - GitHub OAuth callback
- `GET /auth/logout`
  - Clears the session cookie and redirects to the frontend

### Authenticated

All `/api` requests require a valid JWT session cookie set by GitHub login.

- `GET /api/me`
  - Returns authenticated user profile metadata
- `GET /api/stats`
  - Returns current or requested month stats
  - Query parameters: `year`, `month`
- `GET /api/stats/history`
  - Returns the last 12 months of stats history

### Admin

- `POST /api/admin/trigger-monthly`
  - Starts the monthly processing job on demand
  - Requires `X-Admin-Key: <ADMIN_KEY>` header
  - If `ADMIN_KEY` is empty, the admin endpoint is hidden and responds as `404`

## Monthly Job

The scheduler runs at `02:00 UTC` on the 1st day of every month, based on `cron` configuration.

The monthly job:

1. Loads all users from the database
2. Collects GitHub activity stats for the previous month
3. Generates a PNG poster report
4. Emails the report to the user

## Example

See the generated example report image:

![Example Report](./example/example.png)

## Notes

- User OAuth tokens are encrypted before storage.
- Sessions are managed via an HTTP-only JWT cookie named `session`.
- The service uses GitHub OAuth scopes `read:user` and `user:email`.
- The backend currently uses Gin for HTTP routing and is designed to be paired with a separate frontend.
