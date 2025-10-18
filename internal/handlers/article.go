package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Article struct {
	ID              int     `json:"id"`
	URL             string  `json:"url"`
	Format          string  `json:"format"`
	Length          string  `json:"length"`
	Language        *string `json:"language,omitempty"`
	Style           *string `json:"style,omitempty"`
	Status          string  `json:"status"`
	OriginalContent *string `json:"original_content,omitempty"`
	Summary         *string `json:"summary,omitempty"`
	AudioFilePath   *string `json:"audio_file_path,omitempty"`
	ErrorMessage    *string `json:"error_message,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type CreateArticleRequest struct {
	URL      string  `json:"url"`
	Format   string  `json:"format"`
	Length   string  `json:"length"`
	Language *string `json:"language,omitempty"`
	Style    *string `json:"style,omitempty"`
}

type ArticleHandler struct {
	db            *sql.DB
	jobProcessor  JobProcessor
}

type JobProcessor interface {
	ProcessArticle(articleID int)
}

func NewArticleHandler(db *sql.DB, jobProcessor JobProcessor) *ArticleHandler {
	return &ArticleHandler{
		db:           db,
		jobProcessor: jobProcessor,
	}
}

func (h *ArticleHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
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
	if req.Format != "text" && req.Format != "audio" {
		http.Error(w, "Format must be 'text' or 'audio'", http.StatusBadRequest)
		return
	}

	// Validate length
	if req.Length != "s" && req.Length != "m" && req.Length != "l" {
		http.Error(w, "Length must be 's', 'm', or 'l'", http.StatusBadRequest)
		return
	}

	// Insert article with status 'init'
	var article Article
	query := `INSERT INTO articles (url, format, length, language, style, status)
	          VALUES ($1, $2, $3, $4, $5, 'init')
	          RETURNING id, url, format, length, language, style, status, created_at, updated_at`

	err := h.db.QueryRow(query, req.URL, req.Format, req.Length, req.Language, req.Style).Scan(
		&article.ID, &article.URL, &article.Format, &article.Length,
		&article.Language, &article.Style, &article.Status, &article.CreatedAt, &article.UpdatedAt,
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
	rows, err := h.db.Query(`SELECT id, url, format, length, language, style, status,
	                         summary, audio_file_path, error_message, created_at, updated_at
	                         FROM articles ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "Failed to fetch articles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	articles := []Article{}
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.URL, &article.Format, &article.Length,
			&article.Language, &article.Style, &article.Status, &article.Summary,
			&article.AudioFilePath, &article.ErrorMessage, &article.CreatedAt, &article.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan article", http.StatusInternalServerError)
			return
		}
		articles = append(articles, article)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

func (h *ArticleHandler) GetArticle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	var article Article
	query := `SELECT id, url, format, length, language, style, status, original_content,
	          summary, audio_file_path, error_message, created_at, updated_at
	          FROM articles WHERE id = $1`

	err = h.db.QueryRow(query, id).Scan(
		&article.ID, &article.URL, &article.Format, &article.Length, &article.Language,
		&article.Style, &article.Status, &article.OriginalContent, &article.Summary,
		&article.AudioFilePath, &article.ErrorMessage, &article.CreatedAt, &article.UpdatedAt,
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
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec(`DELETE FROM articles WHERE id = $1`, id)
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
