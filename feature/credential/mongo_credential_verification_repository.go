package credential

import (
	"context"
	"errors"
	"time"

	"CredChain_Golang/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const credentialVerificationsCollection = "credential_verifications"

type mongoCredentialVerificationRepository struct {
	coll *mongo.Collection
}

// NewMongoCredentialVerificationRepository is the exported FX factory.
func NewMongoCredentialVerificationRepository(db *mongo.Database) domain.CredentialVerificationRepository {
	return &mongoCredentialVerificationRepository{coll: db.Collection(credentialVerificationsCollection)}
}

func (r *mongoCredentialVerificationRepository) FindByUploadedFileHash(ctx context.Context, hash string) (*domain.CredentialVerification, error) {
	var out domain.CredentialVerification
	err := r.coll.FindOne(ctx, bson.M{"uploaded_file_hash": hash}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *mongoCredentialVerificationRepository) Store(ctx context.Context, v domain.CredentialVerification) error {
	// created_at is in $set (not $setOnInsert) intentionally — re-verifying the
	// same file resets the TTL window (sliding expiry). This bounds storage while
	// keeping recently-active files cached. See also migrate-mongo's TTL index.
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now()
	}
	_, err := r.coll.UpdateOne(ctx,
		bson.M{"uploaded_file_hash": v.UploadedFileHash},
		bson.M{"$set": bson.M{
			"verdict_code":          v.VerdictCode,
			"matched_credential_id": v.MatchedCredentialID,
			"similarity_score":      v.SimilarityScore,
			"similarity_percent":    v.SimilarityPercent,
			"created_at":            v.CreatedAt,
		}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}
