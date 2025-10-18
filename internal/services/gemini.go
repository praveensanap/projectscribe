package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GeminiService struct {
	apiKey string
	client *http.Client
}

func NewGeminiService(apiKey string) *GeminiService {
	return &GeminiService{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// SummarizeArticle fetches and summarizes an article from a URL
// length: "s" (1min), "m" (5min), "l" (full article)
func (g *GeminiService) SummarizeArticle(url string, length string) (string, string, error) {
	// First, extract the article content from the webpage
	content, err := g.extractArticleContent(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to extract article: %w", err)
	}

	// Then summarize based on length
	summary, err := g.summarize(content, length)
	if err != nil {
		return "", "", fmt.Errorf("failed to summarize: %w", err)
	}

	return content, summary, nil
}

func (g *GeminiService) extractArticleContent(url string) (string, error) {
	prompt := fmt.Sprintf(`Extract the main article content from this URL: %s

Please:
1. Remove all navigation menus, headers, footers, ads, and other non-article content
2. Keep only the article title and main body text
3. Preserve paragraph structure
4. Remove any JavaScript, CSS, or HTML tags
5. Return clean, readable text

Return only the extracted article content.`, url)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=%s", g.apiKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error: %s - %s", resp.Status, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiService) summarize(content string, length string) (string, error) {
	var targetLength string
	switch length {
	case "s":
		targetLength = "approximately 1 minute of reading time (about 150-200 words)"
	case "m":
		targetLength = "approximately 5 minutes of reading time (about 750-1000 words)"
	case "l":
		targetLength = "keep the full article content, but clean it up and organize it well"
	default:
		targetLength = "approximately 5 minutes of reading time"
	}

	prompt := fmt.Sprintf(`Please summarize the following article to %s.

Keep the summary:
- Clear and well-structured
- In plain text format
- Easy to understand
- Covering the main points

Article content:
%s

Summary:`, targetLength, content)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=%s", g.apiKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error: %s - %s", resp.Status, string(body))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	summary := geminiResp.Candidates[0].Content.Parts[0].Text
	return strings.TrimSpace(summary), nil
}
