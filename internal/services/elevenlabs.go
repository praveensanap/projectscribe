package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ElevenLabsService struct {
	apiKey         string
	storageService *StorageService
	client         *http.Client
}

func NewElevenLabsService(apiKey string, storageService *StorageService) *ElevenLabsService {
	return &ElevenLabsService{
		apiKey:         apiKey,
		storageService: storageService,
		client:         &http.Client{},
	}
}

type ttsRequest struct {
	Text          string        `json:"text"`
	ModelID       string        `json:"model_id"`
	VoiceSettings voiceSettings `json:"voice_settings"`
}

type voiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
}

// ConvertTextToSpeech converts text to speech and uploads it to Supabase storage
// Returns the public URL where audio is stored
func (e *ElevenLabsService) ConvertTextToSpeech(text string, articleID int64, language, style string) (string, error) {
	// Use default voice ID (Rachel - a versatile voice)
	// You can change this to other voice IDs from ElevenLabs
	voiceID := "21m00Tcm4TlvDq8ikWAM"

	reqBody := ttsRequest{
		Text:    text,
		ModelID: "eleven_monolingual_v1",
		VoiceSettings: voiceSettings{
			Stability:       0.5,
			SimilarityBoost: 0.75,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "audio/mpeg")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("elevenlabs API error: %s - %s", resp.Status, string(body))
	}

	// Read audio data
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read audio data: %w", err)
	}

	// Generate storage key
	key := GenerateAudioKey(articleID)

	// Upload to Supabase storage
	publicURL, err := e.storageService.UploadFile(context.Background(), key, audioData, "audio/mpeg")
	if err != nil {
		return "", fmt.Errorf("failed to upload audio to storage: %w", err)
	}

	return publicURL, nil
}
