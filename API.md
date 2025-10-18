# PocketScribe API Documentation

## Overview

PocketScribe is an article processing service that extracts, summarizes, and optionally converts web articles to audio using AI.

## Base URL

```
http://localhost:8080
```

## Article Processing

### Create Article

Creates a new article and starts background processing.

**Endpoint:** `POST /api/v1/articles`

**Request Body:**
```json
{
  "url": "string (required)",
  "format": "text|audio (required)",
  "length": "s|m|l (required)",
  "language": "string (optional)",
  "style": "string (optional)"
}
```

**Parameters:**
- `url`: The URL of the article to process
- `format`: Output format
  - `text`: Text summary only
  - `audio`: Text summary + audio file
- `length`: Summary length
  - `s`: Short (~1 minute read, 150-200 words)
  - `m`: Medium (~5 minute read, 750-1000 words)
  - `l`: Long (full article, cleaned)
- `language`: Optional language preference (e.g., "English", "Spanish")
- `style`: Optional style preference (e.g., "professional", "casual")

**Response:** `201 Created`
```json
{
  "id": 1,
  "url": "https://example.com/article",
  "format": "audio",
  "length": "m",
  "language": "English",
  "style": "professional",
  "status": "init",
  "created_at": "2025-10-18T12:00:00Z",
  "updated_at": "2025-10-18T12:00:00Z"
}
```

**Status Codes:**
- `201`: Article created successfully
- `400`: Invalid request (missing fields or invalid values)
- `500`: Server error

**Example:**
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

---

### Get Article

Retrieves a specific article with its processing status and results.

**Endpoint:** `GET /api/v1/articles/{id}`

**Response:** `200 OK`
```json
{
  "id": 1,
  "url": "https://example.com/article",
  "format": "audio",
  "length": "m",
  "language": "English",
  "style": "professional",
  "status": "available",
  "original_content": "Full extracted article text...",
  "summary": "Summarized article text...",
  "audio_file_path": "./storage/audio/article_1.mp3",
  "error_message": null,
  "created_at": "2025-10-18T12:00:00Z",
  "updated_at": "2025-10-18T12:05:00Z"
}
```

**Status Values:**
- `init`: Article created, processing not started
- `processing`: Currently being processed
- `available`: Processing complete, summary and audio (if requested) are ready
- `failed`: Processing failed (check error_message)

**Status Codes:**
- `200`: Success
- `404`: Article not found
- `500`: Server error

**Example:**
```bash
curl http://localhost:8080/api/v1/articles/1
```

---

### List Articles

Retrieves all articles ordered by creation date (newest first).

**Endpoint:** `GET /api/v1/articles`

**Response:** `200 OK`
```json
[
  {
    "id": 2,
    "url": "https://example.com/article-2",
    "format": "text",
    "length": "s",
    "status": "available",
    "summary": "Brief summary...",
    "created_at": "2025-10-18T12:30:00Z",
    "updated_at": "2025-10-18T12:31:00Z"
  },
  {
    "id": 1,
    "url": "https://example.com/article-1",
    "format": "audio",
    "length": "m",
    "status": "processing",
    "created_at": "2025-10-18T12:00:00Z",
    "updated_at": "2025-10-18T12:05:00Z"
  }
]
```

**Example:**
```bash
curl http://localhost:8080/api/v1/articles
```

---

### Delete Article

Deletes an article and its associated audio file (if exists).

**Endpoint:** `DELETE /api/v1/articles/{id}`

**Response:** `204 No Content`

**Status Codes:**
- `204`: Successfully deleted
- `404`: Article not found
- `500`: Server error

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/v1/articles/1
```

---

## Processing Workflow

1. **Client submits article**: POST request with URL and preferences
2. **Immediate response**: Article ID and status="init" returned
3. **Background processing begins**:
   - Status updates to "processing"
   - Gemini AI extracts clean article content from URL
   - Gemini AI generates summary based on length preference
   - (If format="audio") ElevenLabs converts summary to MP3
4. **Completion**: Status updates to "available"
5. **Client polls**: GET requests to check status and retrieve results

---

## Error Handling

All errors follow this format:

```json
{
  "error": "Error message description"
}
```

Common errors:
- `400 Bad Request`: Invalid input (wrong format, missing required fields)
- `404 Not Found`: Resource doesn't exist
- `500 Internal Server Error`: Server-side error (check logs)

When an article processing fails, the status will be "failed" and the error_message field will contain details.

---

## Rate Limiting

Currently no rate limiting is implemented. Consider implementing rate limiting for production use.

---

## Health Check

**Endpoint:** `GET /health`

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```

Use this endpoint to verify the service is running.

---

## Notes

- Processing time varies based on article length and API response times (typically 30-60 seconds)
- Audio files are stored locally at the path specified in AUDIO_STORAGE_PATH
- The Gemini API automatically filters out ads, navigation, and other non-article content
- ElevenLabs uses the "Rachel" voice by default (configurable in elevenlabs.go:44)
