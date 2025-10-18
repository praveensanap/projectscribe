package server

import (
	"database/sql"
	"fmt"
	"net/http"

	"pocketscribe/internal/config"
	"pocketscribe/internal/handlers"
	"pocketscribe/internal/jobs"
	"pocketscribe/internal/middleware"
	"pocketscribe/internal/services"

	"github.com/gorilla/mux"
)

type Server struct {
	config *config.Config
	db     *sql.DB
	router *mux.Router
}

func New(cfg *config.Config, db *sql.DB) *Server {
	s := &Server{
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Apply global middleware
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.CORS)
	s.router.Use(middleware.Recovery)

	// Health check endpoint
	s.router.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// User routes
	userHandler := handlers.NewUserHandler(s.db)
	api.HandleFunc("/users", userHandler.CreateUser).Methods("POST")
	api.HandleFunc("/users", userHandler.GetUsers).Methods("GET")
	api.HandleFunc("/users/{id}", userHandler.GetUser).Methods("GET")
	api.HandleFunc("/users/{id}", userHandler.UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", userHandler.DeleteUser).Methods("DELETE")

	// Note routes
	noteHandler := handlers.NewNoteHandler(s.db)
	api.HandleFunc("/notes", noteHandler.CreateNote).Methods("POST")
	api.HandleFunc("/notes", noteHandler.GetNotes).Methods("GET")
	api.HandleFunc("/notes/{id}", noteHandler.GetNote).Methods("GET")
	api.HandleFunc("/notes/{id}", noteHandler.UpdateNote).Methods("PUT")
	api.HandleFunc("/notes/{id}", noteHandler.DeleteNote).Methods("DELETE")
	api.HandleFunc("/users/{userId}/notes", noteHandler.GetUserNotes).Methods("GET")

	// Article routes
	// Initialize services
	geminiService := services.NewGeminiService(s.config.GeminiAPIKey)
	elevenLabsService := services.NewElevenLabsService(s.config.ElevenLabsAPIKey, s.config.AudioStoragePath)
	jobProcessor := jobs.NewProcessor(s.db, geminiService, elevenLabsService)

	articleHandler := handlers.NewArticleHandler(s.db, jobProcessor)
	api.HandleFunc("/articles", articleHandler.CreateArticle).Methods("POST")
	api.HandleFunc("/articles", articleHandler.GetArticles).Methods("GET")
	api.HandleFunc("/articles/{id}", articleHandler.GetArticle).Methods("GET")
	api.HandleFunc("/articles/{id}", articleHandler.DeleteArticle).Methods("DELETE")
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.config.Port)
	return http.ListenAndServe(addr, s.router)
}
