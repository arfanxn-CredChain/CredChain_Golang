package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

// IPFSClient represents an IPFS interaction client
type IPFSClient struct {
	logger *zap.Logger
}

// NewIPFSClient returns a stubbed IPFS client
func NewIPFSClient(logger *zap.Logger) *IPFSClient {
	return &IPFSClient{logger: logger}
}

// UploadJSON takes a metadata struct, converts to JSON, and "uploads" to IPFS
// For Phase 2, this is heavily stubbed to return a fake ipfs:// URI.
func (c *IPFSClient) UploadJSON(ctx context.Context, metadata any) (string, error) {
	bytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", err
	}

	// Pseudo-upload: Just log the payload that would be sent.
	c.logger.Info("stub ipfs upload payload", zap.Int("size", len(bytes)))

	// Create a fake CID (Content Identifier)
	cid := fmt.Sprintf("QmStub%s", ulid.Make().String())
	uri := fmt.Sprintf("ipfs://%s", cid)

	return uri, nil
}
