# PocketScribe Backend

A Go backend service for article summarization and text-to-speech conversion using Supabase PostgreSQL, Google Gemini AI, and ElevenLabs.

## Features

- RESTful API with proper routing
- PostgreSQL database integration with Supabase
- Article processing with AI-powered summarization (Google Gemini)
- Text-to-speech conversion (ElevenLabs)
- Background job processing for async operations
- CRUD operations for Users, Notes, and Articles
- Middleware for logging, CORS, and error recovery
- Clean project structure following Go best practices

## Project Structure

```
pocketscribe/
├── cmd/
│   └── server/
│       └── main.go                  # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go                # Configuration management
│   ├── database/
│   │   └── database.go              # Database connection and migrations
│   ├── handlers/
│   │   ├── health.go                # Health check endpoint
│   │   ├── user.go                  # User CRUD handlers
│   │   ├── note.go                  # Note CRUD handlers
│   │   └── article.go               # Article CRUD handlers
│   ├── services/
│   │   ├── gemini.go                # Google Gemini AI integration
│   │   └── elevenlabs.go            # ElevenLabs TTS integration
│   ├── jobs/
│   │   └── processor.go             # Background job processor
│   ├── middleware/
│   │   └── middleware.go            # HTTP middleware
│   └── server/
│       └── server.go                # HTTP server setup
├── storage/
│   └── audio/                       # Generated audio files
├── .env.example                     # Environment variables template
├── .gitignore
├── go.mod
└── README.md
```

## Prerequisites

- Go 1.21 or higher
- Supabase account with PostgreSQL database
- Google Gemini API key ([Get it here](https://makersuite.google.com/app/apikey))
- ElevenLabs API key ([Get it here](https://elevenlabs.io/))
- Git

## Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd pocketscribe
```

2. Copy the environment variables template:
```bash
cp .env.example .env
```

3. Edit `.env` and add your credentials:
```
DATABASE_URL=postgresql://postgres:[YOUR_PASSWORD]@db.fawgciilqoctwjwcjaqc.supabase.co:5432/postgres
PORT=8080
ENV=development

# API Keys
GEMINI_API_KEY=your_gemini_api_key_here
ELEVENLABS_API_KEY=your_elevenlabs_api_key_here

# Storage
AUDIO_STORAGE_PATH=./storage/audio
```

4. Install dependencies:
```bash
go mod tidy
```

5. Run the server:
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Health Check
- `GET /health` - Check if the server is running

### Users
- `POST /api/v1/users` - Create a new user
- `GET /api/v1/users` - Get all users
- `GET /api/v1/users/{id}` - Get a specific user
- `PUT /api/v1/users/{id}` - Update a user
- `DELETE /api/v1/users/{id}` - Delete a user

### Notes
- `POST /api/v1/notes` - Create a new note
- `GET /api/v1/notes` - Get all notes
- `GET /api/v1/notes/{id}` - Get a specific note
- `PUT /api/v1/notes/{id}` - Update a note
- `DELETE /api/v1/notes/{id}` - Delete a note
- `GET /api/v1/users/{userId}/notes` - Get all notes for a specific user

### Articles
- `POST /api/v1/articles` - Create and process a new article
- `GET /api/v1/articles` - Get all articles
- `GET /api/v1/articles/{id}` - Get a specific article
- `DELETE /api/v1/articles/{id}` - Delete an article

## Example Requests

### Create a User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","name":"John Doe"}'
```

### Get All Users
```bash
curl http://localhost:8080/api/v1/users
```

### Create a Note
```bash
curl -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"title":"My First Note","content":"This is the content of my note"}'
```

### Get User Notes
```bash
curl http://localhost:8080/api/v1/users/1/notes
```

### Create an Article (Text Only)
```bash
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/article",
    "format": "text",
    "length": "s"
  }'
```

### Create an Article (With Audio)
```bash
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/article",
    "format": "audio",
    "length": "m",
    "language": "English",
    "style": "professional"
  }'
```

### Get Article Status
```bash
curl http://localhost:8080/api/v1/articles/1
```

### Get All Articles
```bash
curl http://localhost:8080/api/v1/articles
```

## Database Schema

The application automatically creates the following tables on startup:

### Users Table
- `id` (SERIAL PRIMARY KEY)
- `email` (VARCHAR, UNIQUE)
- `name` (VARCHAR)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

### Notes Table
- `id` (SERIAL PRIMARY KEY)
- `user_id` (INTEGER, FOREIGN KEY)
- `title` (VARCHAR)
- `content` (TEXT)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

### Articles Table
- `id` (SERIAL PRIMARY KEY)
- `url` (TEXT) - URL of the article to process
- `format` (VARCHAR) - Output format: 'text' or 'audio'
- `length` (VARCHAR) - Summary length: 's' (1min), 'm' (5min), 'l' (full)
- `language` (VARCHAR) - Optional language preference
- `style` (VARCHAR) - Optional style preference
- `status` (VARCHAR) - Processing status: 'init', 'processing', 'available', 'failed'
- `original_content` (TEXT) - Extracted article content
- `summary` (TEXT) - AI-generated summary
- `audio_file_path` (TEXT) - Path to generated audio file
- `error_message` (TEXT) - Error message if processing failed
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

## Development

### Build the application
```bash
go build -o bin/server cmd/server/main.go
```

### Run the built binary
```bash
./bin/server
```

### Run tests
```bash
go test ./...
```

## Article Processing Workflow

When you create an article via the API:

1. **Article Created**: The article is immediately saved to the database with `status="init"` and an ID is returned
2. **Background Processing Starts**: A goroutine begins processing the article asynchronously
3. **Status Update**: Article status changes to `"processing"`
4. **Content Extraction**: Gemini AI extracts the article content from the URL, removing ads, navigation, and non-article elements
5. **Summarization**: Based on the `length` parameter:
   - `s` (short): ~1 minute read (150-200 words)
   - `m` (medium): ~5 minute read (750-1000 words)
   - `l` (long): Full article, cleaned and organized
6. **Text-to-Speech** (if format="audio"): ElevenLabs converts the summary to high-quality audio
7. **Completion**: Status changes to `"available"` and the article is ready

You can poll the article endpoint to check the status. Once `status="available"`, the summary and audio (if requested) are ready.

## Environment Variables

- `DATABASE_URL` - PostgreSQL connection string (required)
- `PORT` - Server port (default: 8080)
- `ENV` - Environment (development/production, default: development)
- `GEMINI_API_KEY` - Google Gemini API key (required)
- `ELEVENLABS_API_KEY` - ElevenLabs API key (required)
- `AUDIO_STORAGE_PATH` - Path to store audio files (default: ./storage/audio)

## License

MIT
