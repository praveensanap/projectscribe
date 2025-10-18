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

	// Apply authentication middleware to all API routes
	//api.Use(middleware.SupabaseAuth(s.config.SupabaseJWTSecret))

	// Article routes
	// Initialize services
	geminiService := services.NewGeminiService(s.config.GeminiAPIKey)

	// Initialize storage service
	storageService, err := services.NewStorageService(
		s.config.StorageEndpoint,
		s.config.StoragePublicURL,
		s.config.StorageRegion,
		s.config.StorageAccessKey,
		s.config.StorageSecretKey,
		s.config.StorageBucketName,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize storage service: %v", err))
	}

	elevenLabsService := services.NewElevenLabsService(s.config.ElevenLabsAPIKey, storageService)
	jobProcessor := jobs.NewProcessor(s.db, geminiService, elevenLabsService, storageService)

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
