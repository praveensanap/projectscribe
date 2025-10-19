package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

type APNSService struct {
	token       string
	bundleID    string
	production  bool
	deviceToken string
	client      *http.Client
}

type APNSPayload struct {
	APS APSData `json:"aps"`
}

type APSData struct {
	Alert APSAlert `json:"alert"`
	Badge int      `json:"badge,omitempty"`
	Sound string   `json:"sound,omitempty"`
}

type APSAlert struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Subtitle string `json:"subtitle,omitempty"`
}

// NewAPNSService creates a new APNS service with a static token
func NewAPNSService(token, deviceToken, bundleID string, production bool) *APNSService {
	production = false

	// Create HTTP/2 client
	transport := &http2.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &APNSService{
		token:       token,
		bundleID:    bundleID,
		production:  production,
		deviceToken: deviceToken,
		client:      client,
	}
}

// SendArticleReadyNotification sends a push notification when an article is ready
func (s *APNSService) SendArticleReadyNotification(articleID int64, title string) error {
	// Construct the payload
	payload := APNSPayload{
		APS: APSData{
			Alert: APSAlert{
				Title: "Article Ready!",
				Body:  fmt.Sprintf("Your article '%s' is ready to read", title),
			},
			Badge: 1,
			Sound: "default",
		},
	}

	return s.sendNotification(payload)
}

// SendArticleFailedNotification sends a push notification when an article fails
func (s *APNSService) SendArticleFailedNotification(articleID int64, errorMsg string) error {
	payload := APNSPayload{
		APS: APSData{
			Alert: APSAlert{
				Title: "Article Processing Failed",
				Body:  "There was an error processing your article",
			},
			Sound: "default",
		},
	}

	return s.sendNotification(payload)
}

// sendNotification sends the actual push notification to APNS
func (s *APNSService) sendNotification(payload APNSPayload) error {
	// Skip if no token configured (graceful degradation)
	if s.token == "" {
		log.Printf("APNS: No token configured, skipping push notification")
		return nil
	}

	// Determine APNS endpoint
	endpoint := "https://api.sandbox.push.apple.com"
	if s.production {
		endpoint = "https://api.push.apple.com"
	}

	url := fmt.Sprintf("%s/3/device/%s", endpoint, s.deviceToken)

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("apns-topic", s.bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-expiration", "0")
	req.Header.Set("authorization", fmt.Sprintf("bearer %s", s.token))

	// Send request using HTTP/2 client
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		log.Printf("APNS: Failed to send notification. Status: %d, Response: %+v", resp.StatusCode, errorResponse)
		return fmt.Errorf("APNS returned status %d: %v", resp.StatusCode, errorResponse)
	}

	log.Printf("APNS: Successfully sent notification to device %s", s.deviceToken)
	return nil
}

// GetDeviceToken returns the hardcoded device token
func (s *APNSService) GetDeviceToken() string {
	return s.deviceToken
}
