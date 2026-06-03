package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// ── Repository struct & factory ──────────────────────────────────────────

type gormCredentialRepository struct {
	db *gorm.DB
}

// NewGormCredentialRepository is the exported factory for FX injection.
func NewGormCredentialRepository(db *gorm.DB) domain.CredentialRepository {
	return &gormCredentialRepository{db: db}
}

// ── Column allowlists (dialect-agnostic; Postgres + SQLite) ─────────────

// allowedFilterColumns whitelists credential columns clients may filter on.
// holder_user_id is intentionally included so the user-detail UI can scope
// credentials to a specific holder.
var allowedFilterColumns = map[string]bool{
	"name":           true,
	"issued_at":      true,
	"revoked_at":     true,
	"holder_user_id": true,
}

// allowedSortColumns whitelists credential columns plus virtual joined-user
// columns (prefixed "holder_") that clients may sort on. Joined columns
// trigger an additional LEFT JOIN users AS holder at query build time.
// Sorts on non-allowlisted columns are silently ignored.
var allowedSortColumns = map[string]bool{
	"name":          true,
	"issued_at":     true,
	"revoked_at":    true,
	"holder_name":   true,
	"holder_email":  true,
	"holder_number": true,
	"holder_phone":  true,
}

// ── Preload helper ────────────────────────────────────────────────────────

// preloadByIncludes applies GORM Preload for each include key present in the
// query. Supported keys: "holder", "issuer", "revoker". A single batch
// IN-clause query runs per Preload regardless of result size (no N+1).
func preloadByIncludes(db *gorm.DB, query *domainQuery.Query) *gorm.DB {
	for _, inc := range query.Includes {
		switch inc {
		case "holder":
			db = db.Preload("HolderUser")
		case "issuer":
			db = db.Preload("IssuerUser")
		case "revoker":
			db = db.Preload("RevokerUser")
		}
	}
	return db
}

// ── Join helper ───────────────────────────────────────────────────────────

// needsHolderJoin reports whether we must LEFT JOIN users AS holder for the
// given query. Search always needs the holder join (name/email/number/phone
// search predicates); sorts on holder_* columns also require it.
func needsHolderJoin(query *domainQuery.Query) bool {
	return query.HasSearch() ||
		lo.ContainsBy(query.Sorts, func(s domainQuery.Sort) bool { return strings.HasPrefix(s.Column, "holder_") })
}

// mapSortColumn translates a user-facing sort column into a DB-qualified
// column expression (e.g. "holder_name" → "holder.name").
func mapSortColumn(col string) string {
	switch col {
	case "name", "issued_at", "revoked_at":
		return "credentials." + col
	case "holder_name":
		return "holder.name"
	case "holder_email":
		return "holder.email"
	case "holder_number":
		return "holder.number"
	case "holder_phone":
		return "holder.phone_number"
	default:
		return col
	}
}

// ── Pagination ────────────────────────────────────────────────────────────

// Get retrieves credentials with pagination, search, filters, sorts, and
// optional includes. Search spans credentials.name, credentials.meta, and
// the holder user's name/email/number/phone_number (via holder_user_id JOIN).
//
// When query.Includes contains "holder", "issuer", or "revoker", the
// corresponding GORM Preload runs — a single batch IN-clause query per
// Preload regardless of result size.
func (r *gormCredentialRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Credential{})

	if needsHolderJoin(query) {
		db = db.Joins("LEFT JOIN users AS holder ON holder.id = credentials.holder_user_id")
	}

	if query.HasSearch() {
		needle := "%" + query.Search + "%"
		db = db.Where(
			"LOWER(credentials.name) LIKE LOWER(?) OR "+
				"LOWER(CAST(credentials.meta AS TEXT)) LIKE LOWER(?) OR "+
				"LOWER(holder.name) LIKE LOWER(?) OR "+
				"LOWER(holder.email) LIKE LOWER(?) OR "+
				"LOWER(holder.number) LIKE LOWER(?) OR "+
				"LOWER(holder.phone_number) LIKE LOWER(?)",
			needle, needle, needle, needle, needle, needle,
		)
	}

	if query.HasFilters() {
		db = applyCredentialFilters(db, query.Filters)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.HasSorts() {
		for _, s := range query.Sorts {
			if !allowedSortColumns[s.Column] {
				continue
			}
			col := mapSortColumn(s.Column)
			db = db.Order(fmt.Sprintf("%s %s", col, s.Order))
		}
	} else {
		db = db.Order("credentials.issued_at DESC")
	}

	db = preloadByIncludes(db, query)

	if query.HasPagination() {
		db = db.Limit(query.Limit).Offset(query.Offset())
	}

	var credentials []model.Credential
	if err := db.Find(&credentials).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.Credential, len(credentials))
	for i, c := range credentials {
		out[i] = c.ToDomain()
	}
	return out, int(total), nil
}

// applyCredentialFilters maps domainQuery.Filter operators to GORM Where
// clauses, gated by the allowedFilterColumns allowlist for SQL injection
// safety. Columns are always scoped with the "credentials." prefix.
func applyCredentialFilters(db *gorm.DB, filters []domainQuery.Filter) *gorm.DB {
	for _, f := range filters {
		if !allowedFilterColumns[f.Column] {
			continue
		}
		col := "credentials." + f.Column
		switch f.Operator {
		case domainQuery.OperatorEqual:
			db = db.Where(col+" = ?", f.GetValue())
		case domainQuery.OperatorNotEqual:
			db = db.Where(col+" != ?", f.GetValue())
		case domainQuery.OperatorGreaterThan:
			db = db.Where(col+" > ?", f.GetValue())
		case domainQuery.OperatorLessThan:
			db = db.Where(col+" < ?", f.GetValue())
		case domainQuery.OperatorGreaterThanOrEqual:
			db = db.Where(col+" >= ?", f.GetValue())
		case domainQuery.OperatorLessThanOrEqual:
			db = db.Where(col+" <= ?", f.GetValue())
		case domainQuery.OperatorLike, domainQuery.OperatorILike:
			db = db.Where("LOWER("+col+") LIKE LOWER(?)", "%"+f.GetValue()+"%")
		case domainQuery.OperatorNotLike, domainQuery.OperatorNotILike:
			db = db.Where("LOWER("+col+") NOT LIKE LOWER(?)", "%"+f.GetValue()+"%")
		case domainQuery.OperatorIn:
			if len(f.Values) > 0 {
				db = db.Where(col+" IN ?", f.Values)
			}
		case domainQuery.OperatorNotIn:
			if len(f.Values) > 0 {
				db = db.Where(col+" NOT IN ?", f.Values)
			}
		case domainQuery.OperatorBetween:
			if len(f.Values) == 2 {
				db = db.Where(col+" BETWEEN ? AND ?", f.Values[0], f.Values[1])
			}
		case domainQuery.OperatorNotBetween:
			if len(f.Values) == 2 {
				db = db.Where(col+" NOT BETWEEN ? AND ?", f.Values[0], f.Values[1])
			}
		case domainQuery.OperatorNull:
			db = db.Where(col + " IS NULL")
		case domainQuery.OperatorNotNull:
			db = db.Where(col + " IS NOT NULL")
		}
	}
	return db
}

// ── Single-row lookups ────────────────────────────────────────────────────

// Find retrieves a single credential by ID, applying optional Preloads
// from query.Includes.
func (r *gormCredentialRepository) Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	db := r.db.WithContext(ctx).Model(&model.Credential{})
	db = preloadByIncludes(db, query)
	var c model.Credential
	if err := db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	d := c.ToDomain()
	return &d, nil
}

// ── Batch lookups ─────────────────────────────────────────────────────────

// FindByIds retrieves credentials by ID list (batch lookup).
func (r *gormCredentialRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	if len(ids) == 0 {
		return []domain.Credential{}, nil
	}
	var rows []model.Credential
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}

// FindByHolderId retrieves all credentials owned by a single holder.
func (r *gormCredentialRepository) FindByHolderId(ctx context.Context, holderID string) ([]domain.Credential, error) {
	var rows []model.Credential
	if err := r.db.WithContext(ctx).Where("holder_user_id = ?", holderID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}

// FindByFileHashes retrieves credentials whose file_hash matches any of the
// supplied hashes. Used during issue to detect duplicate uploads.
func (r *gormCredentialRepository) FindByFileHashes(ctx context.Context, hashes ...string) ([]domain.Credential, error) {
	if len(hashes) == 0 {
		return []domain.Credential{}, nil
	}
	var rows []model.Credential
	if err := r.db.WithContext(ctx).Where("file_hash IN ?", hashes).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, c := range rows {
		out[i] = c.ToDomain()
	}
	return out, nil
}

// ── Mutations ─────────────────────────────────────────────────────────────

// Store batch-inserts new credentials. Generates ULIDs for any missing IDs
// and defaults ExtractStatus to pending.
func (r *gormCredentialRepository) Store(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	if len(credentials) == 0 {
		return []domain.Credential{}, nil
	}
	for i := range credentials {
		if credentials[i].ID == "" {
			credentials[i].ID = ulid.Make().String()
		}
		if credentials[i].ExtractStatus == "" {
			credentials[i].ExtractStatus = domain.ExtractStatusPending
		}
	}
	rows := make([]model.Credential, len(credentials))
	for i, c := range credentials {
		rows[i] = model.FromDomainCredential(c)
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Credential, len(rows))
	for i, m := range rows {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Update partially updates one or more credentials using a single batched
// UPDATE with per-column CASE expressions. Only non-nil / non-zero fields
// are touched; unspecified columns fall through to ELSE column (preserving
// the existing value). This eliminates the N+1 per-row UPDATE pattern.
func (r *gormCredentialRepository) Update(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	if len(credentials) == 0 {
		return []domain.Credential{}, nil
	}
	if err := r.updateBatchCase(ctx, credentials); err != nil {
		return nil, err
	}
	ids := make([]string, len(credentials))
	for i, c := range credentials {
		ids[i] = c.ID
	}
	return r.FindByIds(ctx, ids...)
}

// updateBatchCase builds and executes a single UPDATE statement using CASE
// expressions for each column that at least one credential provides. Users
// sorted by ID for deterministic arg ordering.
func (r *gormCredentialRepository) updateBatchCase(ctx context.Context, items []domain.Credential) error {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	var setClauses []string
	var setArgs []interface{}

	addCaseClause := func(col string, getValue func(domain.Credential) (interface{}, bool)) {
		var caseArgs []interface{}
		for _, c := range items {
			if v, ok := getValue(c); ok {
				caseArgs = append(caseArgs, c.ID, v)
			}
		}
		if len(caseArgs) == 0 {
			return
		}
		caseSQL := "CASE id"
		for i := 0; i < len(caseArgs)/2; i++ {
			caseSQL += " WHEN ? THEN ?"
		}
		caseSQL += " ELSE " + col + " END"
		setClauses = append(setClauses, col+" = "+caseSQL)
		setArgs = append(setArgs, caseArgs...)
	}

	addCaseClause("name", func(c domain.Credential) (interface{}, bool) {
		if c.Name != "" {
			return c.Name, true
		}
		return nil, false
	})
	addCaseClause("meta", func(c domain.Credential) (interface{}, bool) {
		if c.Meta == nil {
			return nil, false
		}
		b, err := json.Marshal(c.Meta)
		if err != nil {
			return nil, false
		}
		return string(b), true
	})
	addCaseClause("token_id", func(c domain.Credential) (interface{}, bool) {
		if c.TokenID == nil {
			return nil, false
		}
		return *c.TokenID, true
	})
	addCaseClause("file_uri", func(c domain.Credential) (interface{}, bool) {
		if c.FileURI == nil {
			return nil, false
		}
		return *c.FileURI, true
	})
	addCaseClause("revoked_at", func(c domain.Credential) (interface{}, bool) {
		if c.RevokedAt == nil {
			return nil, false
		}
		return *c.RevokedAt, true
	})
	addCaseClause("revoker_user_id", func(c domain.Credential) (interface{}, bool) {
		if c.RevokerUserID == nil {
			return nil, false
		}
		return *c.RevokerUserID, true
	})
	addCaseClause("extract_status", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractStatus == "" {
			return nil, false
		}
		return string(c.ExtractStatus), true
	})
	addCaseClause("embeddings", func(c domain.Credential) (interface{}, bool) {
		if c.Embeddings == nil {
			return nil, false
		}
		b, err := json.Marshal(c.Embeddings)
		if err != nil {
			return nil, false
		}
		return string(b), true
	})
	addCaseClause("extract_error", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractError == nil {
			return nil, false
		}
		return *c.ExtractError, true
	})
	addCaseClause("extracted_at", func(c domain.Credential) (interface{}, bool) {
		if c.ExtractedAt == nil {
			return nil, false
		}
		return *c.ExtractedAt, true
	})

	if len(setClauses) == 0 {
		return nil
	}

	ids := make([]interface{}, len(items))
	for i, c := range items {
		ids[i] = c.ID
	}
	sql := "UPDATE credentials SET " + strings.Join(setClauses, ", ") + " WHERE id IN (?)"
	finalArgs := append(setArgs, ids)
	return r.db.WithContext(ctx).Exec(sql, finalArgs...).Error
}

// ── Compile-time interface check ──────────────────────────────────────────

var _ domain.CredentialRepository = (*gormCredentialRepository)(nil)
