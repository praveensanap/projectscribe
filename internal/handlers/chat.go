package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"pocketscribe/internal/middleware"
	"pocketscribe/internal/services"

	"github.com/gorilla/mux"
)

type ChatHandler struct {
	db            *sql.DB
	geminiService *services.GeminiService
}

func NewChatHandler(db *sql.DB, geminiService *services.GeminiService) *ChatHandler {
	return &ChatHandler{
		db:            db,
		geminiService: geminiService,
	}
}

type ChatRequest struct {
	ArticleID   int64                     `json:"article_id"`
	Message     string                    `json:"message"`
	ChatHistory []services.ChatMessage    `json:"chat_history"`
}

type ChatResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatWithArticle handles chat requests for a specific article
func (h *ChatHandler) ChatWithArticle(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse article ID from URL
	vars := mux.Vars(r)
	articleID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Fetch the article to ensure it exists and belongs to the user
	var article Article
	query := `SELECT id, user_id, original_content, summary, status
	          FROM articles WHERE id = $1 AND user_id = $2`

	err = h.db.QueryRow(query, articleID, userID).Scan(
		&article.ID, &article.UserID, &article.OriginalContent, &article.Summary, &article.Status,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch article", http.StatusInternalServerError)
		return
	}

	// Check if article is ready for chatting
	if article.Status != "ready" {
		http.Error(w, "Article is not ready for chat. Current status: "+article.Status, http.StatusBadRequest)
		return
	}

	// Check if article has content
	if article.OriginalContent == nil || *article.OriginalContent == "" {
		http.Error(w, "Article content is not available", http.StatusBadRequest)
		return
	}

	// Generate response using Gemini
	response, err := h.geminiService.ChatWithArticle(
		*article.OriginalContent,
		req.ChatHistory,
		req.Message,
	)
	if err != nil {
		http.Error(w, "Failed to generate response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Role:    "assistant",
		Content: response,
	})
}
