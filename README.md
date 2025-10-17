# Noppera Image Board API

A 4chan-inspired image board API built with Go, Chi, PostgreSQL, and JSONB. Supports boards, posts, threads, user authentication, post flagging, search, and image storage (local or S3).

## Features
- **Boards**: Create/list boards (admin-only creation).
- **Posts**: Create threads/replies, upload images (local or S3).
- **Auth**: User/admin registration, login with JWT.
- **Flags**: Flag posts for moderation, admin review.
- **Search**: Full-text search on post content and tags.
- **Threads**: View threads with replies.
- **Archiving**: Auto-archive threads after 7 days, delete after 30 days.
- **Rate-Limiting**: Prevent spam on public endpoints.
- **Logging**: Structured request logging with zerolog.
- **Health Checks**: Kubernetes-ready health, readiness, and liveness probes.
- **CORS Support**: Configurable Cross-Origin Resource Sharing.
- **API Documentation**: Interactive Swagger/OpenAPI documentation.

## Setup
1. **Prerequisites**:
   - Go 1.23
   - Docker, Docker Compose
   - AWS credentials (if using S3)

2. **Clone Repository**:
   ```bash
   git clone https://github.com/cobalto/noppera.git
   cd noppera
   ```

3. **Configure Environment**:
   Copy `.env.example` to `.env` and update values:
   ```bash
   cp .env.example .env
   ```

4. **Run with Docker**:
   ```bash
   docker-compose up -d
   ```

5. **Test Endpoints**:
   ```bash
   # Health check
   curl http://localhost:8080/health
   
   # Register user
   curl -X POST http://localhost:8080/auth/register -d '{"username":"test","password":"pass123"}'
   
   # Login
   curl -X POST http://localhost:8080/auth/login -d '{"username":"test","password":"pass123"}'
   
   # Create thread
   curl -X POST http://localhost:8080/boards/g/threads -d '{"title":"Test Thread","content":"Hello","image":"base64image"}'
   
   # Search posts
   curl "http://localhost:8080/posts/search?query=hello"
   
   # View thread
   curl http://localhost:8080/threads/1
   
   # View API documentation
   open http://localhost:8080/swagger/index.html
   ```

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string.
- `API_HOST`, `API_PORT`: API server host/port.
- `JWT_SECRET`: Secret for JWT signing.
- `STORAGE_TYPE`: `local` or `s3`.
- `UPLOAD_DIR`, `UPLOAD_URL_PREFIX`: Local storage directory and URL base (if STORAGE_TYPE=local).
- `S3_REGION`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `S3_BUCKET`: S3 settings.
- `MAX_POST_LENGTH`, `RATE_LIMIT_REQUESTS`, `RATE_LIMIT_BURST`, `ARCHIVE_DELETE_DAYS`: API settings.
- `LOG_LEVEL`, `LOG_FILE`: Logging settings.
- `CORS_ALLOWED_ORIGINS`, `CORS_ALLOWED_METHODS`, `CORS_ALLOWED_HEADERS`, `CORS_ALLOW_CREDENTIALS`: CORS settings.

## Directory Structure

- cmd/api/ Main application entry point
- internal/handlers/ HTTP handlers for boards, posts, auth, flags, search, threads, health
- internal/models/ Data models and database operations
- internal/storage/ Image storage (local/S3)
- internal/middleware/ Authentication, rate-limiting, logging, CORS
- internal/jobs/ Background jobs (archiving)
- internal/config/ Configuration loading
- docs/ Generated Swagger/OpenAPI documentation

## API Endpoints

### Health & Documentation
- `GET /health` - Health check with service status
- `GET /health/ready` - Readiness probe for Kubernetes
- `GET /health/live` - Liveness probe for Kubernetes
- `GET /swagger/*` - Interactive API documentation

### Authentication
- `POST /auth/register` - Register new user
- `POST /auth/login` - Login and get JWT token
- `POST /auth/register/admin` - Register admin user (admin only)

### Boards
- `GET /boards` - List all boards
- `POST /boards` - Create new board (admin only)

### Posts & Threads
- `POST /boards/{boardSlug}/threads` - Create new thread
- `POST /threads/{threadID}/replies` - Reply to thread
- `GET /threads/{threadID}` - Get thread with replies
- `DELETE /posts/{postID}/user` - Delete own post (authenticated)
- `DELETE /posts/{postID}/admin` - Delete any post (admin only)

### Search & Moderation
- `GET /posts/search` - Search posts by content, tags, or board
- `POST /posts/{postID}/flag` - Flag post for moderation
- `GET /flags` - List all flags (admin only)

## Development

### Generate Swagger Documentation
```bash
swag init -g cmd/api/main.go -o ./docs
```

### Build and Run
```bash
go build -o noppera ./cmd/api
./noppera
```

