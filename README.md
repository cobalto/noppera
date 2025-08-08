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

## Directory Structure

- cmd/api/ Main application entry point
- internal/handlers/ HTTP handlers for boards, posts, auth, flags, search, threads
- internal/models/ Data models and database operations
- internal/storage/ Image storage (local/S3)
- internal/middleware/ Authentication, rate-limiting, logging
- internal/jobs/ Background jobs (archiving)
- internal/config/ Configuration loading

