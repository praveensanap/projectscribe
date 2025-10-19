# Video Generation with Fal API (Sora 2)

This document explains how the video generation feature works in PocketScribe using Fal API's Sora 2 model.

## Overview

When creating an article with `format: "video"`, the system will automatically generate a video from the article summary using OpenAI's Sora 2 model via Fal API.

## Setup

### 1. Get Fal API Key

1. Sign up at [fal.ai](https://fal.ai)
2. Generate an API key from your dashboard
3. Add it to your `.env` file:

```bash
FAL_API_KEY=your_fal_api_key_here
```

### 2. Configure Storage

Ensure your Supabase storage is configured with a bucket that can handle video files. The system will automatically create a `videos/` folder in your storage bucket.

## Usage

### Create Article with Video Format

Make a POST request to `/api/v1/articles` with:

```json
{
  "url": "https://example.com/article",
  "format": "video",
  "length": "m",
  "language": "English",
  "style": "summarize"
}
```

### Video Duration by Length

The video duration is automatically determined by the `length` parameter:

- `"s"` (short): 10 seconds
- `"m"` (medium): 30 seconds
- `"l"` (long): 60 seconds

### Processing Flow

1. Article is created with status `queued`
2. Background processor starts:
   - Extracts article content using Gemini
   - Generates summary based on length and style
   - Generates title
   - Generates thumbnail
   - **Generates video using Fal API (Sora 2)**
   - Downloads video and uploads to Supabase storage
   - Updates article with `video_file_path`
3. Article status changes to `ready`
4. Push notification sent to device

### Response

The article response will include:

```json
{
  "id": 123,
  "title": "Article Title",
  "format": "video",
  "status": "ready",
  "video_file_path": "https://your-project.supabase.co/storage/v1/object/public/audio/videos/article_123.mp4",
  "thumbnail_path": "https://...",
  "summary": "Article summary...",
  ...
}
```

## Implementation Details

### Fal Service (`internal/services/fal.go`)

The `FalService` handles:
- Submitting video generation requests to Fal API
- Polling for completion (up to 5 minutes)
- Downloading the generated video
- Saving to local storage temporarily

### Video Generation Process

1. **Submit Request**: Send summary text to Fal API's Sora 2 endpoint
2. **Poll for Completion**: Check status every 5 seconds (max 60 attempts)
3. **Download Video**: Retrieve MP4 file from Fal's storage
4. **Upload to Storage**: Upload to Supabase storage bucket
5. **Clean Up**: Delete local temporary file

### Storage

Videos are stored in:
- **Local (temporary)**: `uploads/videos/article_{id}_{timestamp}.mp4`
- **Supabase**: `videos/article_{id}.mp4`

The local file is automatically deleted after successful upload to Supabase.

## Error Handling

If video generation fails:
- Article status is set to `failed`
- Error message is stored in `error_message` field
- Push notification sent with failure details

Common failure reasons:
- Invalid or missing FAL_API_KEY
- Fal API rate limits
- Video generation timeout (5 minutes)
- Network issues during download
- Storage upload failures

## Configuration

### Environment Variables

```bash
# Required
FAL_API_KEY=your_fal_api_key_here

# Storage (already configured for audio)
STORAGE_ENDPOINT=https://your-project-id.storage.supabase.co/storage/v1/s3
STORAGE_PUBLIC_URL=https://your-project-id.supabase.co
STORAGE_REGION=us-east-1
STORAGE_ACCESS_KEY=your_storage_access_key
STORAGE_SECRET_KEY=your_storage_secret_key
STORAGE_BUCKET_NAME=audio  # Videos will be in same bucket, different folder
```

## Limitations

- **Maximum Video Length**: 60 seconds (long format)
- **Generation Time**: Can take 2-5 minutes depending on video length
- **Video Format**: MP4 only
- **Aspect Ratio**: 16:9 (default)

## Future Enhancements

Potential improvements:
- Custom aspect ratio selection (16:9, 9:16, 1:1)
- Multiple video quality options
- Custom video styles and effects
- Video editing capabilities
- Combine audio narration with video
