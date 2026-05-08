package ai

import (
	"CredChain_Golang/config"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// ExtractResult represents the clean JSON returned by Gemini
type ExtractResult struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DateIssued  string `json:"date_issued"`
	Institution string `json:"institution"`
	Description string `json:"description"`
}

// Client wraps the Gemini AI logic
type Client struct {
	client *genai.Client
}

// NewClient initializes a Google GenAI client
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.GeminiAPIKey == nil {
		return nil, fmt.Errorf("gemini api key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(*cfg.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gemini client: %w", err)
	}

	return &Client{client: client}, nil
}

// Close cleans up
func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// ExtractMetadata takes raw document bytes, sends to Gemini, and enforces JSON output
func (c *Client) ExtractMetadata(ctx context.Context, mimeType string, fileBytes []byte) (*ExtractResult, string, error) {
	model := c.client.GenerativeModel("gemini-2.5-flash")
	model.ResponseMIMEType = "application/json"

	prompt := `Analyze the attached credential/certificate document. 
Extract the following information into a strict JSON object:
- "name": The name of the credential or degree (e.g. "Bachelor of Science")
- "type": Classify type (e.g. "academic", "professional", "identity", "other")
- "date_issued": The date of issuance or completion
- "institution": The issuing organization
- "description": A brief summary of what this document proves

Return ONLY valid JSON without markdown wrapping.`

	partData := genai.Blob{MIMEType: mimeType, Data: fileBytes}

	resp, err := model.GenerateContent(ctx, partData, genai.Text(prompt))
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, "", fmt.Errorf("gemini returned empty response")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, "", fmt.Errorf("gemini returned non-text response")
	}

	rawLog := string(text) // This will be sent to MongoDB

	var result ExtractResult
	if err := json.Unmarshal([]byte(rawLog), &result); err != nil {
		return nil, rawLog, fmt.Errorf("failed to parse gemini response as JSON: %w", err)
	}

	return &result, rawLog, nil
}
