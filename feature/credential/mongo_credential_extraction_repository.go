package credential

import (
	"context"
	"time"

	"CredChain_Golang/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const credentialExtractionsCollection = "credential_extractions"

type mongoCredentialExtractionRepository struct {
	coll *mongo.Collection
}

// NewMongoCredentialExtractionRepository is the exported FX factory.
func NewMongoCredentialExtractionRepository(db *mongo.Database) domain.CredentialExtractionRepository {
	return &mongoCredentialExtractionRepository{coll: db.Collection(credentialExtractionsCollection)}
}

func (r *mongoCredentialExtractionRepository) Store(ctx context.Context, e domain.CredentialExtraction) error {
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	_, err := r.coll.UpdateOne(ctx,
		bson.M{"credential_id": e.CredentialID},
		bson.M{
			"$set": bson.M{
				"file_hash":  e.FileHash,
				"text":       e.Text,
				"ids":        e.IDs,
				"embedding":  e.Embedding,
				"updated_at": e.UpdatedAt,
			},
			"$setOnInsert": bson.M{"credential_id": e.CredentialID, "created_at": e.CreatedAt},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (r *mongoCredentialExtractionRepository) FindByCredentialId(ctx context.Context, credentialID string) (*domain.CredentialExtraction, error) {
	var out domain.CredentialExtraction
	err := r.coll.FindOne(ctx, bson.M{"credential_id": credentialID}).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// FindRankedByIds runs a SINGLE aggregation: match docs sharing any id value,
// compute the intersection size, sort by it desc, limit. NO per-id queries.
func (r *mongoCredentialExtractionRepository) FindRankedByIds(ctx context.Context, values []string, limit int) ([]domain.CredentialExtraction, error) {
	if len(values) == 0 {
		return []domain.CredentialExtraction{}, nil
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"ids.value": bson.M{"$in": values}}}},
		{{Key: "$addFields", Value: bson.M{
			"match_count": bson.M{"$size": bson.M{"$setIntersection": bson.A{"$ids.value", values}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "match_count", Value: -1}}}},
		{{Key: "$limit", Value: int64(limit)}},
	}
	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []domain.CredentialExtraction
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
