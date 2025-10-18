package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL          string
	Port                 string
	Environment          string
	GeminiAPIKey         string
	ElevenLabsAPIKey     string
	AudioStoragePath     string
	StorageEndpoint      string
	StoragePublicURL     string
	StorageRegion        string
	StorageAccessKey     string
	StorageSecretKey     string
	StorageBucketName    string
	SupabaseURL          string
	SupabaseJWTSecret    string
}

func Load() (*Config, error) {
	// Try to load .env file, but don't fail if it doesn't exist
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		Port:              getEnv("PORT", "8080"),
		Environment:       getEnv("ENV", "development"),
		GeminiAPIKey:      getEnv("GEMINI_API_KEY", ""),
		ElevenLabsAPIKey:  getEnv("ELEVENLABS_API_KEY", ""),
		AudioStoragePath:  getEnv("AUDIO_STORAGE_PATH", "./storage/audio"),
		StorageEndpoint:   getEnv("STORAGE_ENDPOINT", ""),
		StoragePublicURL:  getEnv("STORAGE_PUBLIC_URL", ""),
		StorageRegion:     getEnv("STORAGE_REGION", "us-east-1"),
		StorageAccessKey:  getEnv("STORAGE_ACCESS_KEY", ""),
		StorageSecretKey:  getEnv("STORAGE_SECRET_KEY", ""),
		StorageBucketName: getEnv("STORAGE_BUCKET_NAME", "audio"),
		SupabaseURL:       getEnv("SUPABASE_URL", ""),
		SupabaseJWTSecret: getEnv("SUPABASE_JWT_SECRET", ""),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	if cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	if cfg.ElevenLabsAPIKey == "" {
		return nil, fmt.Errorf("ELEVENLABS_API_KEY environment variable is required")
	}

	if cfg.SupabaseJWTSecret == "" {
		return nil, fmt.Errorf("SUPABASE_JWT_SECRET environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
