package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"pocketscribe/internal/services"
)

type Processor struct {
	db                *sql.DB
	geminiService     *services.GeminiService
	elevenLabsService *services.ElevenLabsService
	storageService    *services.StorageService
	apnsService       *services.APNSService
	falService        *services.FalService
}

func NewProcessor(db *sql.DB, geminiService *services.GeminiService, elevenLabsService *services.ElevenLabsService, storageService *services.StorageService, apnsService *services.APNSService, falService *services.FalService) *Processor {
	return &Processor{
		db:                db,
		geminiService:     geminiService,
		elevenLabsService: elevenLabsService,
		storageService:    storageService,
		apnsService:       apnsService,
		falService:        falService,
	}
}

// ProcessArticle processes an article in the background
func (p *Processor) ProcessArticle(articleID int64) {
	log.Printf("Starting to process article %d", articleID)

	// Update status to processing
	if err := p.updateArticleStatus(articleID, "processing", ""); err != nil {
		log.Printf("Failed to update article %d status to processing: %v", articleID, err)
		return
	}

	// Get article details
	var url, format, length string
	var language, style sql.NullString
	query := `SELECT url, format, length, language, style FROM articles WHERE id = $1`
	err := p.db.QueryRow(query, articleID).Scan(&url, &format, &length, &language, &style)
	if err != nil {
		log.Printf("Failed to get article %d details: %v", articleID, err)
		p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to get article details: %v", err))
		return
	}

	// Step 1: Summarize the article using Gemini
	styleStr := "summarize" // default style
	if style.Valid && style.String != "" {
		styleStr = style.String
	}

	languageStr := "en"
	if language.Valid && language.String != "" {
		languageStr = language.String
	}

	log.Printf("Summarizing article %d with length %s and style %s", articleID, length, styleStr)
	originalContent, summary, err := p.geminiService.SummarizeArticle(url, length, languageStr, styleStr)
	if err != nil {
		log.Printf("Failed to summarize article %d: %v", articleID, err)
		p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to summarize: %v", err))
		p.sendFailureNotification(articleID, "Failed to summarize")
		return
	}

	// Generate title from the original content
	log.Printf("Generating title for article %d", articleID)
	title, err := p.geminiService.GenerateTitle(originalContent)
	if err != nil {
		log.Printf("Failed to generate title for article %d: %v", articleID, err)
		// Don't fail the entire process if title generation fails
		// Just use a default title
		title = "Untitled Article"
	}

	// Save the original content, title, and summary
	updateQuery := `UPDATE articles SET original_content = $1, title = $2, summary = $3, updated_at = CURRENT_TIMESTAMP
	                WHERE id = $4`
	if _, err := p.db.Exec(updateQuery, originalContent, title, summary, articleID); err != nil {
		log.Printf("Failed to save summary for article %d: %v", articleID, err)
		p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to save summary: %v", err))
		return
	}

	log.Printf("Successfully summarized article %d with title: %s", articleID, title)

	// Step 2: Generate thumbnail from summary
	log.Printf("Generating thumbnail for article %d", articleID)
	thumbnailData, err := p.geminiService.GenerateThumbnail(summary)
	if err != nil {
		log.Printf("Failed to generate thumbnail for article %d: %v", articleID, err)
		// Don't fail the entire process if thumbnail generation fails
		// Just log and continue
	} else {
		// Upload thumbnail to storage
		thumbnailKey := services.GenerateThumbnailKey(articleID)
		thumbnailURL, err := p.storageService.UploadFile(context.Background(), thumbnailKey, thumbnailData, "image/png")
		if err != nil {
			log.Printf("Failed to upload thumbnail for article %d: %v", articleID, err)
		} else {
			// Save thumbnail path
			updateQuery := `UPDATE articles SET thumbnail_path = $1, updated_at = CURRENT_TIMESTAMP
			                WHERE id = $2`
			if _, err := p.db.Exec(updateQuery, thumbnailURL, articleID); err != nil {
				log.Printf("Failed to save thumbnail path for article %d: %v", articleID, err)
			} else {
				log.Printf("Successfully generated and uploaded thumbnail for article %d", articleID)
			}
		}
	}

	// Step 3: If format is audio, convert to speech using ElevenLabs
	if format == "audio" {
		log.Printf("Converting article %d to speech", articleID)

		langStr := ""
		if language.Valid {
			langStr = language.String
		}

		styleStr := ""
		if style.Valid {
			styleStr = style.String
		}

		audioPath, err := p.elevenLabsService.ConvertTextToSpeech(summary, articleID, langStr, styleStr)
		if err != nil {
			log.Printf("Failed to convert article %d to speech: %v", articleID, err)
			p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to convert to speech: %v", err))
			p.sendFailureNotification(articleID, "Failed to convert to speech")
			return
		}

		// Save audio file path
		updateQuery := `UPDATE articles SET audio_file_path = $1, updated_at = CURRENT_TIMESTAMP
		                WHERE id = $2`
		if _, err := p.db.Exec(updateQuery, audioPath, articleID); err != nil {
			log.Printf("Failed to save audio path for article %d: %v", articleID, err)
			p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to save audio path: %v", err))
			return
		}

		log.Printf("Successfully converted article %d to speech", articleID)
	}

	// Step 4: If format is video, generate video using Fal API (Sora 2)
	if format == "video" && p.falService != nil {
		log.Printf("Generating video for article %d using Fal API (Sora 2)", articleID)

		// Determine video duration based on length
		var duration int
		switch length {
		case "s":
			duration = 10 // 10 seconds for short
		case "m":
			duration = 30 // 30 seconds for medium
		case "l":
			duration = 60 // 60 seconds for long
		default:
			duration = 30 // default to medium
		}

		// Generate video from summary
		videoURL, err := p.falService.GenerateVideo(summary, duration)
		if err != nil {
			log.Printf("Failed to generate video for article %d: %v", articleID, err)
			p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to generate video: %v", err))
			p.sendFailureNotification(articleID, "Failed to generate video")
			return
		}

		log.Printf("Video generated successfully for article %d, downloading from %s", articleID, videoURL)

		// Download and save the video
		videoPath, err := p.falService.DownloadVideo(videoURL, int(articleID))
		if err != nil {
			log.Printf("Failed to download video for article %d: %v", articleID, err)
			p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to download video: %v", err))
			p.sendFailureNotification(articleID, "Failed to download video")
			return
		}

		// Upload video to storage
		videoKey := services.GenerateVideoKey(articleID)
		videoStorageURL, err := p.storageService.UploadVideoFile(context.Background(), videoKey, videoPath)
		if err != nil {
			log.Printf("Failed to upload video to storage for article %d: %v", articleID, err)
			p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to upload video: %v", err))
			p.sendFailureNotification(articleID, "Failed to upload video")
			return
		}

		// Save video file path
		updateQuery := `UPDATE articles SET video_file_path = $1, updated_at = CURRENT_TIMESTAMP
		                WHERE id = $2`
		if _, err := p.db.Exec(updateQuery, videoStorageURL, articleID); err != nil {
			log.Printf("Failed to save video path for article %d: %v", articleID, err)
			p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to save video path: %v", err))
			return
		}

		log.Printf("Successfully generated and uploaded video for article %d", articleID)
	}

	// Update status to ready
	if err := p.updateArticleStatus(articleID, "ready", ""); err != nil {
		log.Printf("Failed to update article %d status to ready: %v", articleID, err)
		return
	}

	log.Printf("Successfully processed article %d", articleID)

	// Send push notification to Apple device
	if p.apnsService != nil {
		log.Printf("Sending push notification for article %d", articleID)
		if err := p.apnsService.SendArticleReadyNotification(articleID, title); err != nil {
			log.Printf("Failed to send push notification for article %d: %v", articleID, err)
			// Don't fail the entire process if notification fails
		} else {
			log.Printf("Successfully sent push notification for article %d to device %s", articleID, p.apnsService.GetDeviceToken())
		}
	}
}

func (p *Processor) updateArticleStatus(articleID int64, status, errorMessage string) error {
	query := `UPDATE articles SET status = $1, error_message = $2, updated_at = NOW()
	          WHERE id = $3`
	_, err := p.db.Exec(query, status, errorMessage, articleID)
	return err
}

func (p *Processor) sendFailureNotification(articleID int64, errorMsg string) {
	if p.apnsService != nil {
		log.Printf("Sending failure notification for article %d", articleID)
		if err := p.apnsService.SendArticleFailedNotification(articleID, errorMsg); err != nil {
			log.Printf("Failed to send failure notification for article %d: %v", articleID, err)
		} else {
			log.Printf("Successfully sent failure notification for article %d to device %s", articleID, p.apnsService.GetDeviceToken())
		}
	}
}
