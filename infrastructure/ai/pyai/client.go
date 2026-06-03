package pyai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"CredChain_Golang/config"
)

// ExtractFile is a single file to send to the Python /extract or /verify endpoint.
type ExtractFile struct {
	Filename string
	MIMEType string
	Data     []byte
}

// PythonExtractResult is the per-file result from Python /extract.
// nil Embeddings means the file failed OCR on the Python side.
type PythonExtractResult struct {
	RawText    string
	Embeddings []float64
}

// VerifyResult is the result from Python /verify for a single file.
type VerifyResult struct {
	Verdict           string
	SimilarityScore   float64
	SimilarityPercent string
	Description       VerifyDescription
}

// VerifyDescription is the bilingual description block from Python /verify.
type VerifyDescription struct {
	ID string `json:"id"`
	EN string `json:"en"`
}

// PythonAIClient is the interface for calling the CredChain Python AI service.
// Calls are serialized via an internal mutex because the Python service runs a
// single uvicorn worker and LaBSE is not thread-safe.
type PythonAIClient interface {
	Extract(ctx context.Context, files ...ExtractFile) ([]PythonExtractResult, error)
	Verify(ctx context.Context, file ExtractFile, storedEmbeddings []float64) (*VerifyResult, error)
}

type pythonAIClient struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.Mutex
}

type PythonAIClientParams struct {
	Config *config.Config
}

func NewPythonAIClient(cfg *config.Config) PythonAIClient {
	timeout := 120 * time.Second
	if cfg.PythonAITimeoutSeconds != nil {
		timeout = time.Duration(*cfg.PythonAITimeoutSeconds) * time.Second
	}
	baseURL := "http://localhost:8081"
	if cfg.PythonAIBaseURL != nil {
		baseURL = *cfg.PythonAIBaseURL
	}
	return &pythonAIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// pythonResponse is the generic envelope returned by the Python service.
type pythonResponse[T any] struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    []T                 `json:"data"`
	Errors  map[string][]string `json:"errors"`
}

type extractData struct {
	RawText    string    `json:"raw_text"`
	Embeddings []float64 `json:"embeddings"`
}

type verifyData struct {
	SimilarityScore   float64           `json:"similarity_score"`
	SimilarityPercent string            `json:"similarity_percent"`
	Verdict           string            `json:"verdict"`
	Description       VerifyDescription `json:"description"`
}

func (c *pythonAIClient) Extract(ctx context.Context, files ...ExtractFile) ([]PythonExtractResult, error) {
	if len(files) == 0 {
		return nil, nil
	}

	body, contentType, err := buildMultipartFiles(files)
	if err != nil {
		return nil, fmt.Errorf("python extract: build multipart: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/extract", body)
	if err != nil {
		return nil, fmt.Errorf("python extract: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python extract: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python extract: read body: %w", err)
	}

	var parsed pythonResponse[*extractData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python extract: decode: %w", err)
	}

	if parsed.Code == 500140 {
		return nil, fmt.Errorf("python extract: all files failed OCR (code %d)", parsed.Code)
	}

	results := make([]PythonExtractResult, len(parsed.Data))
	for i, d := range parsed.Data {
		if d != nil {
			results[i] = PythonExtractResult{RawText: d.RawText, Embeddings: d.Embeddings}
		}
	}
	return results, nil
}

func (c *pythonAIClient) Verify(ctx context.Context, file ExtractFile, storedEmbeddings []float64) (*VerifyResult, error) {
	embJSON, err := json.Marshal([]map[string]any{{"stored_embeddings": storedEmbeddings}})
	if err != nil {
		return nil, fmt.Errorf("python verify: marshal embeddings: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("files", file.Filename)
	if err != nil {
		return nil, fmt.Errorf("python verify: create form file: %w", err)
	}
	if _, err := part.Write(file.Data); err != nil {
		return nil, fmt.Errorf("python verify: write file: %w", err)
	}
	if err := writer.WriteField("metadata", string(embJSON)); err != nil {
		return nil, fmt.Errorf("python verify: write metadata: %w", err)
	}
	writer.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/verify", body)
	if err != nil {
		return nil, fmt.Errorf("python verify: build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python verify: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("python verify: read body: %w", err)
	}

	var parsed pythonResponse[*verifyData]
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("python verify: decode: %w", err)
	}

	if len(parsed.Data) == 0 || parsed.Data[0] == nil {
		return nil, fmt.Errorf("python verify: empty result (code %d)", parsed.Code)
	}

	d := parsed.Data[0]
	return &VerifyResult{
		Verdict:           d.Verdict,
		SimilarityScore:   d.SimilarityScore,
		SimilarityPercent: d.SimilarityPercent,
		Description:       d.Description,
	}, nil
}

func buildMultipartFiles(files []ExtractFile) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, f := range files {
		part, err := writer.CreateFormFile("files", f.Filename)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(f.Data); err != nil {
			return nil, "", err
		}
	}
	writer.Close()
	return body, writer.FormDataContentType(), nil
}

// FileExtToMIME returns the MIME type for common credential file extensions.
func FileExtToMIME(filename string) string {
	switch filepath.Ext(filename) {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".tiff", ".tif":
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}
