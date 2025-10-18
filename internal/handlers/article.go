package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"pocketscribe/internal/middleware"

	"github.com/gorilla/mux"
)

type Article struct {
	ID              int64   `json:"id"`
	UserID          string  `json:"user_id"`
	URL             string  `json:"url"`
	Title           *string `json:"title,omitempty"`
	Format          string  `json:"format"`
	Length          string  `json:"length"`
	Status          string  `json:"status"`
	ThumbnailPath   *string `json:"thumbnail_path,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	Language        *string `json:"language,omitempty"`
	Style           *string `json:"style,omitempty"`
	OriginalContent *string `json:"original_content,omitempty"`
	Summary         *string `json:"summary,omitempty"`
	TextBody        *string `json:"text_body,omitempty"`
	AudioFilePath   *string `json:"audio_file_path,omitempty"`
	VideoFilePath   *string `json:"video_file_path,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
	ErrorMessage    *string `json:"error_message,omitempty"`
}

type CreateArticleRequest struct {
	URL      string  `json:"url"`
	Format   string  `json:"format"`
	Length   string  `json:"length"`
	Language *string `json:"language,omitempty"`
	Style    *string `json:"style,omitempty"`
}

type ArticleHandler struct {
	db           *sql.DB
	jobProcessor JobProcessor
}

type JobProcessor interface {
	ProcessArticle(articleID int64)
}

func NewArticleHandler(db *sql.DB, jobProcessor JobProcessor) *ArticleHandler {
	return &ArticleHandler{
		db:           db,
		jobProcessor: jobProcessor,
	}
}

func (h *ArticleHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Validate format
	if req.Format != "text" && req.Format != "audio" && req.Format != "video" {
		http.Error(w, "Format must be 'text', 'audio', or 'video'", http.StatusBadRequest)
		return
	}

	// Validate length
	if req.Length != "s" && req.Length != "m" && req.Length != "l" {
		http.Error(w, "Length must be 's', 'm', or 'l'", http.StatusBadRequest)
		return
	}

	// Insert article with status 'queued' and user_id
	var article Article
	query := `INSERT INTO articles (user_id, url, format, length, language, style, status)
	          VALUES ($1, $2, $3, $4, $5, $6, 'queued')
	          RETURNING id, user_id, url, title, format, length, status, thumbnail_path,
	                    created_at, updated_at, language, style`

	err := h.db.QueryRow(query, userID, req.URL, req.Format, req.Length, req.Language, req.Style).Scan(
		&article.ID, &article.UserID, &article.URL, &article.Title, &article.Format, &article.Length,
		&article.Status, &article.ThumbnailPath, &article.CreatedAt, &article.UpdatedAt,
		&article.Language, &article.Style,
	)
	if err != nil {
		http.Error(w, "Failed to create article", http.StatusInternalServerError)
		return
	}

	// Trigger background processing
	go h.jobProcessor.ProcessArticle(article.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(article)
}

func (h *ArticleHandler) GetArticles(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.db.Query(`SELECT id, user_id, url, title, format, length, status, thumbnail_path,
	                         created_at, updated_at, language, style, summary, text_body,
	                         audio_file_path, video_file_path, duration_seconds, error_message
	                         FROM articles WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		http.Error(w, "Failed to fetch articles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	articles := []Article{}
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.UserID, &article.URL, &article.Title,
			&article.Format, &article.Length, &article.Status, &article.ThumbnailPath,
			&article.CreatedAt, &article.UpdatedAt, &article.Language, &article.Style,
			&article.Summary, &article.TextBody, &article.AudioFilePath, &article.VideoFilePath,
			&article.DurationSeconds, &article.ErrorMessage); err != nil {
			http.Error(w, "Failed to scan article", http.StatusInternalServerError)
			return
		}
		articles = append(articles, article)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

func (h *ArticleHandler) GetArticle(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	var article Article
	query := `SELECT id, user_id, url, title, format, length, status, thumbnail_path,
	          created_at, updated_at, language, style, original_content, summary, text_body,
	          audio_file_path, video_file_path, duration_seconds, error_message
	          FROM articles WHERE id = $1 AND user_id = $2`

	err = h.db.QueryRow(query, id, userID).Scan(
		&article.ID, &article.UserID, &article.URL, &article.Title, &article.Format, &article.Length,
		&article.Status, &article.ThumbnailPath, &article.CreatedAt, &article.UpdatedAt,
		&article.Language, &article.Style, &article.OriginalContent, &article.Summary, &article.TextBody,
		&article.AudioFilePath, &article.VideoFilePath, &article.DurationSeconds, &article.ErrorMessage,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch article", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(article)
}

func (h *ArticleHandler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec(`DELETE FROM articles WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		http.Error(w, "Failed to delete article", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
