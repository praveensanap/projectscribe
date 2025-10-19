package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type FalService struct {
	apiKey     string
	httpClient *http.Client
}

func NewFalService() *FalService {
	return &FalService{
		apiKey: os.Getenv("FAL_API_KEY"),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Video generation can take time
		},
	}
}

type SoraGenerateRequest struct {
	Prompt string `json:"prompt"`
	// Add additional parameters as needed
	AspectRatio string `json:"aspect_ratio,omitempty"` // e.g., "16:9", "9:16"
	Duration    int    `json:"duration,omitempty"`     // Duration in seconds
}

type SoraGenerateResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	VideoURL  string `json:"video_url,omitempty"`
}

type SoraStatusResponse struct {
	RequestID   string                 `json:"request_id"`
	Status      string                 `json:"status"` // "pending", "processing", "completed", "failed"
	ResponseURL string                 `json:"response_url"`
	Error       string                 `json:"error,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
}

// GenerateVideo generates a video from text using Fal's Sora 2 model
func (f *FalService) GenerateVideo(prompt string, duration int) (string, error) {
	if f.apiKey == "" {
		return "", fmt.Errorf("FAL_API_KEY not set")
	}

	// Submit the video generation request
	requestID, err := f.submitRequest(prompt, duration)
	if err != nil {
		return "", fmt.Errorf("failed to submit request: %w", err)
	}

	// Poll for completion
	videoURL, err := f.pollForCompletion(requestID)
	if err != nil {
		return "", fmt.Errorf("failed to get video: %w", err)
	}

	return videoURL, nil
}

func (f *FalService) submitRequest(prompt string, duration int) (string, error) {
	// Fal API endpoint for Sora 2
	url := "https://queue.fal.run/fal-ai/sora-2"

	reqBody := SoraGenerateRequest{
		Prompt:      prompt,
		AspectRatio: "16:9",
		Duration:    duration,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result SoraGenerateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.RequestID, nil
}

func (f *FalService) fetchVideoURL(responseURL string) (string, error) {
	req, err := http.NewRequest("GET", responseURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create result request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch result: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read result response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("result request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response as a generic map to handle various response structures
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse result response: %w", err)
	}

	fmt.Printf("Fal API result response: %+v\n", result)

	// Try to extract video URL from various possible locations in the Fal API response
	// The structure might be: {"video": {"url": "..."}} or {"data": {"video": {"url": "..."}}}
	if video, ok := result["video"].(map[string]interface{}); ok {
		if url, ok := video["url"].(string); ok {
			fmt.Printf("Found video URL in result.video.url: %s\n", url)
			return url, nil
		}
	}
	if data, ok := result["data"].(map[string]interface{}); ok {
		if video, ok := data["video"].(map[string]interface{}); ok {
			if url, ok := video["url"].(string); ok {
				fmt.Printf("Found video URL in result.data.video.url: %s\n", url)
				return url, nil
			}
		}
		if url, ok := data["url"].(string); ok {
			fmt.Printf("Found video URL in result.data.url: %s\n", url)
			return url, nil
		}
	}
	// Check if URL is at the top level
	if url, ok := result["url"].(string); ok {
		fmt.Printf("Found video URL in result.url: %s\n", url)
		return url, nil
	}

	return "", fmt.Errorf("video URL not found in result response: %s", string(body))
}

func (f *FalService) pollForCompletion(requestID string) (string, error) {
	url := fmt.Sprintf("https://queue.fal.run/fal-ai/sora-2/requests/%s/status", requestID)

	// Poll for up to 5 minutes (60 attempts with 5 second intervals)
	maxAttempts := 60
	pollInterval := 5 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(pollInterval)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create status request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))

		resp, err := f.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to check status: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed to read status response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("status request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var status SoraStatusResponse
		if err := json.Unmarshal(body, &status); err != nil {
			return "", fmt.Errorf("failed to parse status response: %w", err)
		}

		switch status.Status {
		case "COMPLETED":
			fmt.Printf("Video generation completed. Fetching result from: %s\n", status.ResponseURL)
			// When completed, we need to fetch the actual result from the response_url
			if status.ResponseURL != "" {
				videoURL, err := f.fetchVideoURL(status.ResponseURL)
				if err != nil {
					return "", fmt.Errorf("failed to fetch video URL: %w", err)
				}
				return videoURL, nil
			}
			// Fallback: check if output is already present
			if status.Output != nil {
				if videoURL, ok := status.Output["video"].(string); ok {
					return videoURL, nil
				}
				if videoURL, ok := status.Output["url"].(string); ok {
					return videoURL, nil
				}
			}
			return "", fmt.Errorf("video completed but no URL found in response")
		case "FAILED":
			return "", fmt.Errorf("video generation failed: %s", status.Error)
		case "PENDING", "PROCESSING":
			// Continue polling
			continue
		default:
			return "", fmt.Errorf("unknown status: %s", status.Status)
		}
	}

	return "", fmt.Errorf("video generation timed out after %d attempts", maxAttempts)
}

// DownloadVideo downloads the video from a URL and returns the file path
func (f *FalService) DownloadVideo(videoURL string, articleID int) (string, error) {

	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create status request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download video: status %d", resp.StatusCode)
	}

	// Create videos directory if it doesn't exist
	videosDir := "uploads/videos"
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create videos directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("%s/article_%d_%d.mp4", videosDir, articleID, time.Now().Unix())

	// Create the file
	out, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create video file: %w", err)
	}
	defer out.Close()

	// Write the video content to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save video: %w", err)
	}

	return filename, nil
}
