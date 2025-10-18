package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"google.golang.org/genai"
)

type GeminiService struct {
	apiKey     string
	client     *http.Client
	genaiClient *genai.Client
}

func NewGeminiService(apiKey string) *GeminiService {
	// Set the API key as environment variable for the genai client
	os.Setenv("GEMINI_API_KEY", apiKey)

	ctx := context.Background()
	genaiClient, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Printf("Warning: Failed to create genai client: %v", err)
		genaiClient = nil
	}

	return &GeminiService{
		apiKey:     apiKey,
		client:     &http.Client{},
		genaiClient: genaiClient,
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

const PROMPT = `Extract the main article content from this URL: %s

Please:
1. Remove all navigation menus, headers, footers, ads, and other non-article content
2. Keep only the article title and main body text
3. Preserve paragraph structure
4. Remove any JavaScript, CSS, or HTML tags
5. Return clean, readable text

Return only the extracted article content.`

func (g *GeminiService) extractArticleContent(url string) (string, error) {
	prompt := fmt.Sprintf(PROMPT, url)

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

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent?key=%s", g.apiKey)
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

IMPORTANT: This summary will be converted to speech, so:
- Use only spoken language and natural phrasing
- Avoid special characters, symbols, URLs, hashtags, and markdown formatting
- Avoid parentheses, brackets, asterisks, underscores, and other punctuation marks that aren't naturally spoken
- Use periods for natural pauses between sentences
- Use commas for shorter pauses within sentences
- Spell out numbers, percentages, and abbreviations (e.g., "ten percent" not "10%%", "doctor" not "Dr.")
- Write out acronyms on first use, then use the full term
- Use complete sentences with clear, natural flow
- Organize with paragraph breaks (blank lines) to indicate longer pauses between topics
- Be conversational and engaging, as if explaining to a listener
- Return ONLY the summary text, nothing else

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

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent?key=%s", g.apiKey)
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

// GenerateTitle generates a concise title from the article content
func (g *GeminiService) GenerateTitle(content string) (string, error) {
	// Create a snippet of the content (first 1000 characters to avoid token limits)
	contentSnippet := content
	if len(content) > 1000 {
		contentSnippet = content[:1000]
	}

	prompt := fmt.Sprintf(`Generate a concise, engaging title (maximum 10 words) for the following article content. The title should be clear, informative, and capture the main topic. Return ONLY the title, nothing else.

Article content:
%s

Title:`, contentSnippet)

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

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent?key=%s", g.apiKey)
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

	title := geminiResp.Candidates[0].Content.Parts[0].Text
	// Clean up the title - remove quotes and extra whitespace
	title = strings.Trim(strings.TrimSpace(title), "\"'")
	return title, nil
}

// GenerateThumbnail generates a thumbnail image from text using Imagen via Gemini SDK
func (g *GeminiService) GenerateThumbnail(summary string) ([]byte, error) {
	if g.genaiClient == nil {
		return nil, fmt.Errorf("genai client not initialized")
	}

	// Create a concise image prompt from the summary (limit to 500 chars)
	summarySnippet := summary
	if len(summary) > 500 {
		summarySnippet = summary[:500]
	}

	prompt := fmt.Sprintf(`Create a professional, visually appealing thumbnail image for an article. The image should be abstract and artistic, representing the following content: %s. Style: modern, clean, professional, eye-catching.`, summarySnippet)

	ctx := context.Background()

	// Use the Gemini 2.5 Flash Image model for image generation
	result, err := g.genaiClient.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash-image",
		genai.Text(prompt),
		nil, // config parameter
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	// Extract the image data from the response
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil {
			imageBytes := part.InlineData.Data
			return imageBytes, nil
		}
	}

	return nil, fmt.Errorf("no image data in response")
}
