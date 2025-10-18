package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type ElevenLabsService struct {
	apiKey      string
	storagePath string
	client      *http.Client
}

func NewElevenLabsService(apiKey, storagePath string) *ElevenLabsService {
	return &ElevenLabsService{
		apiKey:      apiKey,
		storagePath: storagePath,
		client:      &http.Client{},
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

// ConvertTextToSpeech converts text to speech and saves it to a file
// Returns the file path where audio is saved
func (e *ElevenLabsService) ConvertTextToSpeech(text string, articleID int, language, style string) (string, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(e.storagePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

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

	// Generate filename
	filename := fmt.Sprintf("article_%d.mp3", articleID)
	filePath := filepath.Join(e.storagePath, filename)

	// Save audio file
	if err := os.WriteFile(filePath, audioData, 0644); err != nil {
		return "", fmt.Errorf("failed to save audio file: %w", err)
	}

	return filePath, nil
}
