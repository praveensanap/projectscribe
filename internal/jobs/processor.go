package jobs

import (
	"database/sql"
	"fmt"
	"log"

	"pocketscribe/internal/services"
)

type Processor struct {
	db                *sql.DB
	geminiService     *services.GeminiService
	elevenLabsService *services.ElevenLabsService
}

func NewProcessor(db *sql.DB, geminiService *services.GeminiService, elevenLabsService *services.ElevenLabsService) *Processor {
	return &Processor{
		db:                db,
		geminiService:     geminiService,
		elevenLabsService: elevenLabsService,
	}
}

// ProcessArticle processes an article in the background
func (p *Processor) ProcessArticle(articleID int) {
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
	log.Printf("Summarizing article %d with length %s", articleID, length)
	originalContent, summary, err := p.geminiService.SummarizeArticle(url, length)
	if err != nil {
		log.Printf("Failed to summarize article %d: %v", articleID, err)
		p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to summarize: %v", err))
		return
	}

	// Save the original content and summary
	updateQuery := `UPDATE articles SET original_content = $1, summary = $2, updated_at = CURRENT_TIMESTAMP
	                WHERE id = $3`
	if _, err := p.db.Exec(updateQuery, originalContent, summary, articleID); err != nil {
		log.Printf("Failed to save summary for article %d: %v", articleID, err)
		p.updateArticleStatus(articleID, "failed", fmt.Sprintf("Failed to save summary: %v", err))
		return
	}

	log.Printf("Successfully summarized article %d", articleID)

	// Step 2: If format is audio, convert to speech using ElevenLabs
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

	// Update status to available
	if err := p.updateArticleStatus(articleID, "available", ""); err != nil {
		log.Printf("Failed to update article %d status to available: %v", articleID, err)
		return
	}

	log.Printf("Successfully processed article %d", articleID)
}

func (p *Processor) updateArticleStatus(articleID int, status, errorMessage string) error {
	query := `UPDATE articles SET status = $1, error_message = $2, updated_at = CURRENT_TIMESTAMP
	          WHERE id = $3`
	_, err := p.db.Exec(query, status, errorMessage, articleID)
	return err
}
