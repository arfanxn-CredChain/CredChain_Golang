package gorm

import (
	"CredChain_Golang/domain"
	"context"
)

type contextKey string

const (
	transactionMetadataKey contextKey = "transaction_metadata"
)

// WithTransactionMetadata adds transaction metadata to context
func WithTransactionMetadata(ctx context.Context, metadata domain.TransactionMetadata) context.Context {
	return context.WithValue(ctx, transactionMetadataKey, metadata)
}

// GetTransactionMetadata retrieves transaction metadata from context
func GetTransactionMetadata(ctx context.Context) *domain.TransactionMetadata {
	metadata, ok := ctx.Value(transactionMetadataKey).(domain.TransactionMetadata)
	if !ok {
		return nil
	}
	return &metadata
}
