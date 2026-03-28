package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"

	"github.com/yourusername/project/internal/cases"
	"github.com/yourusername/project/internal/cases/fts"
)

// Pool implements cases.CaseRepository using pgxpool.
// Caller must ensure pgxvec.RegisterTypes is called on connections (e.g. AfterConnect).
type Pool struct {
	pool *pgxpool.Pool
}

// NewPool returns a repository using the given pool.
func NewPool(pool *pgxpool.Pool) *Pool {
	return &Pool{pool: pool}
}

func (p *Pool) conn(ctx context.Context) (*pgxpool.Conn, error) {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	if err := pgxvec.RegisterTypes(ctx, conn.Conn()); err != nil {
		conn.Release()
		return nil, err
	}
	return conn, nil
}

func (p *Pool) Create(ctx context.Context, c *cases.Case, searchTSVInput string) error {
	conn, err := p.conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	qJSON, _ := json.Marshal(c.Questions)
	kJSON, _ := json.Marshal(c.Keywords)
	query := `
		INSERT INTO cases (category, title, questions, keywords, response_template, status, created_by, search_tsv)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6, $7, to_tsvector('simple', $8))
		RETURNING id, created_at, updated_at`
	return conn.QueryRow(ctx, query,
		c.Category, c.Title, qJSON, kJSON, c.ResponseTemplate, c.Status, ptrToStr(c.CreatedBy), searchTSVInput,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (p *Pool) GetByID(ctx context.Context, id string) (*cases.Case, error) {
	conn, err := p.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		SELECT id, category, title, questions, keywords, response_template, status,
		       created_by, created_at, updated_by, updated_at, approved_by, approved_at, notes
		FROM cases WHERE id = $1`
	var c cases.Case
	var qJSON, kJSON []byte
	var notes *string
	err = conn.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Category, &c.Title, &qJSON, &kJSON, &c.ResponseTemplate, &c.Status,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt, &c.ApprovedBy, &c.ApprovedAt, &notes,
	)
	if err != nil {
		if isPgNoRows(err) {
			return nil, cases.ErrNotFound
		}
		return nil, err
	}
	_ = json.Unmarshal(qJSON, &c.Questions)
	_ = json.Unmarshal(kJSON, &c.Keywords)
	c.Notes = notes
	return &c, nil
}

func (p *Pool) List(ctx context.Context, filters cases.ListFilters, principalID string) ([]cases.Case, error) {
	conn, err := p.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		SELECT id, category, title, questions, keywords, response_template, status,
		       created_by, created_at, updated_by, updated_at, approved_by, approved_at, notes
		FROM cases
		WHERE (status != $1 OR created_by = $2)
		  AND ($3 = '' OR status = $3)
		  AND ($4 = '' OR category = $4)
		  AND (NOT $5::boolean OR created_by = $2)
		ORDER BY updated_at DESC`
	rows, err := conn.Query(ctx, query,
		cases.StatusDraft, principalID,
		filters.Status, filters.Category, filters.Mine,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []cases.Case
	for rows.Next() {
		var c cases.Case
		var qJSON, kJSON []byte
		var notes *string
		if err := rows.Scan(
			&c.ID, &c.Category, &c.Title, &qJSON, &kJSON, &c.ResponseTemplate, &c.Status,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt, &c.ApprovedBy, &c.ApprovedAt, &notes,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(qJSON, &c.Questions)
		_ = json.Unmarshal(kJSON, &c.Keywords)
		c.Notes = notes
		list = append(list, c)
	}
	return list, rows.Err()
}

// ListCategories returns distinct non-empty category names from cases, sorted.
func (p *Pool) ListCategories(ctx context.Context) ([]string, error) {
	rows, err := p.pool.Query(ctx, `SELECT DISTINCT category FROM cases WHERE category != '' ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (p *Pool) Update(ctx context.Context, id string, u *cases.CaseUpdate, searchTSVInput string, updatedBy string) (*cases.Case, error) {
	conn, err := p.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Build dynamic update: only set non-nil fields; always set search_tsv, updated_by, updated_at
	// We need current row to merge; then build SET clause.
	cur, err := p.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cat := cur.Category
	title := cur.Title
	questions := cur.Questions
	keywords := cur.Keywords
	tpl := cur.ResponseTemplate
	notes := cur.Notes
	if u.Category != nil {
		cat = *u.Category
	}
	if u.Title != nil {
		title = *u.Title
	}
	if len(u.Questions) != 0 {
		questions = u.Questions
	}
	if len(u.Keywords) != 0 {
		keywords = u.Keywords
	}
	if u.ResponseTemplate != nil {
		tpl = *u.ResponseTemplate
	}
	if u.Notes != nil {
		notes = u.Notes
	}
	qJSON, _ := json.Marshal(questions)
	kJSON, _ := json.Marshal(keywords)
	query := `
		UPDATE cases
		SET category = $1, title = $2, questions = $3::jsonb, keywords = $4::jsonb, response_template = $5,
		    notes = $6, search_tsv = to_tsvector('simple', $7), updated_by = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING id, category, title, questions, keywords, response_template, status,
		          created_by, created_at, updated_by, updated_at, approved_by, approved_at, notes`
	var c cases.Case
	var qJ, kJ []byte
	var n *string
	err = conn.QueryRow(ctx, query,
		cat, title, qJSON, kJSON, tpl, notes, searchTSVInput, updatedBy, id,
	).Scan(
		&c.ID, &c.Category, &c.Title, &qJ, &kJ, &c.ResponseTemplate, &c.Status,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt, &c.ApprovedBy, &c.ApprovedAt, &n,
	)
	if err != nil {
		if isPgNoRows(err) {
			return nil, cases.ErrNotFound
		}
		return nil, err
	}
	_ = json.Unmarshal(qJ, &c.Questions)
	_ = json.Unmarshal(kJ, &c.Keywords)
	c.Notes = n
	return &c, nil
}

func (p *Pool) SetStatus(ctx context.Context, id string, status string, comment string, principalID string) (*cases.Case, error) {
	conn, err := p.conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		UPDATE cases
		SET status = $1::varchar, updated_by = $2, updated_at = NOW(),
		    approved_by = CASE WHEN $1::varchar = 'approved' THEN $2 ELSE approved_by END,
		    approved_at = CASE WHEN $1::varchar = 'approved' THEN NOW() ELSE approved_at END,
		    notes = CASE WHEN $3::text != '' THEN COALESCE(notes || E'\n' || $3, $3) ELSE notes END
		WHERE id = $4
		RETURNING id, category, title, questions, keywords, response_template, status,
		          created_by, created_at, updated_by, updated_at, approved_by, approved_at, notes`
	var c cases.Case
	var qJ, kJ []byte
	var n *string
	err = conn.QueryRow(ctx, query, status, principalID, comment, id).Scan(
		&c.ID, &c.Category, &c.Title, &qJ, &kJ, &c.ResponseTemplate, &c.Status,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedBy, &c.UpdatedAt, &c.ApprovedBy, &c.ApprovedAt, &n,
	)
	if err != nil {
		if isPgNoRows(err) {
			return nil, cases.ErrNotFound
		}
		return nil, err
	}
	_ = json.Unmarshal(qJ, &c.Questions)
	_ = json.Unmarshal(kJ, &c.Keywords)
	c.Notes = n
	return &c, nil
}

func (p *Pool) SoftDelete(ctx context.Context, id string) error {
	conn, err := p.conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	// 1) Remove embedding if any
	_, _ = conn.Exec(ctx, `DELETE FROM case_embeddings WHERE case_id = $1`, id)
	// 2) Set status to archived
	res, err := conn.Exec(ctx, `UPDATE cases SET status = $1, updated_at = NOW() WHERE id = $2`, cases.StatusArchived, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return cases.ErrNotFound
	}
	return nil
}

func (p *Pool) UpsertEmbedding(ctx context.Context, caseID string, embedding []float32) error {
	conn, err := p.conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	vec := pgvector.NewVector(embedding)
	query := `
		INSERT INTO case_embeddings (case_id, embedding, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (case_id) DO UPDATE SET embedding = EXCLUDED.embedding, updated_at = NOW()`
	_, err = conn.Exec(ctx, query, caseID, vec)
	return err
}

func (p *Pool) DeleteEmbeddingByCaseID(ctx context.Context, caseID string) error {
	conn, err := p.conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	_, err = conn.Exec(ctx, `DELETE FROM case_embeddings WHERE case_id = $1`, caseID)
	return err
}

func (p *Pool) DeleteAll(ctx context.Context) error {
	conn, err := p.conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	_, err = conn.Exec(ctx, `DELETE FROM cases`)
	return err
}

// ReplaceAll deletes all cases (CASCADE clears case_embeddings) and inserts items as new drafts.
func (p *Pool) ReplaceAll(ctx context.Context, items []cases.CaseImportItem, createdBy string) (imported int, errs []string) {
	conn, err := p.conn(ctx)
	if err != nil {
		return 0, []string{err.Error()}
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, []string{err.Error()}
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM cases`)
	if err != nil {
		return 0, []string{err.Error()}
	}

	for _, item := range items {
		qJSON, _ := json.Marshal(item.Questions)
		kJSON, _ := json.Marshal(item.Keywords)
		tsvInput := fts.BuildSearchTSVInput(item.Title, item.Keywords, item.Questions)
		_, err := tx.Exec(ctx,
			`INSERT INTO cases (id, category, title, questions, keywords, response_template, status, created_by, search_tsv)
			 VALUES (gen_random_uuid(), $1, $2, $3::jsonb, $4::jsonb, $5, 'draft', $6, to_tsvector('simple', $7))`,
			item.Category, item.Title, qJSON, kJSON, item.ResponseTemplate, createdBy, tsvInput,
		)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		imported++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, append(errs, err.Error())
	}
	return imported, errs
}

// ImportMerge updates existing by id or creates new; returns imported count, updated count, errors.
func (p *Pool) ImportMerge(ctx context.Context, items []cases.CaseImportItem, createdBy string) (imported, updated int, errs []string) {
	conn, err := p.conn(ctx)
	if err != nil {
		return 0, 0, []string{err.Error()}
	}
	defer conn.Release()

	for _, item := range items {
		qJSON, _ := json.Marshal(item.Questions)
		kJSON, _ := json.Marshal(item.Keywords)
		tsvInput := fts.BuildSearchTSVInput(item.Title, item.Keywords, item.Questions)
		if item.ID != "" {
			_, err := p.GetByID(ctx, item.ID)
			if err == nil {
				// Update existing
				_, err = conn.Exec(ctx, `
					UPDATE cases SET category = $1, title = $2, questions = $3::jsonb, keywords = $4::jsonb,
						response_template = $5, search_tsv = to_tsvector('simple', $6), updated_by = $7, updated_at = NOW()
					WHERE id = $8`,
					item.Category, item.Title, qJSON, kJSON, item.ResponseTemplate, tsvInput, createdBy, item.ID,
				)
				if err != nil {
					errs = append(errs, err.Error())
				} else {
					updated++
				}
				continue
			}
		}
		// Create new
		_, err := conn.Exec(ctx,
			`INSERT INTO cases (id, category, title, questions, keywords, response_template, status, created_by, search_tsv)
			 VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, 'draft', $7, to_tsvector('simple', $8))`,
			uuid.New().String(), item.Category, item.Title, qJSON, kJSON, item.ResponseTemplate, createdBy, tsvInput,
		)
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			imported++
		}
	}
	return imported, updated, errs
}

// ImportDraft creates all items as draft.
func (p *Pool) ImportDraft(ctx context.Context, items []cases.CaseImportItem, createdBy string) (imported int, errs []string) {
	conn, err := p.conn(ctx)
	if err != nil {
		return 0, []string{err.Error()}
	}
	defer conn.Release()

	for _, item := range items {
		qJSON, _ := json.Marshal(item.Questions)
		kJSON, _ := json.Marshal(item.Keywords)
		tsvInput := fts.BuildSearchTSVInput(item.Title, item.Keywords, item.Questions)
		_, err := conn.Exec(ctx,
			`INSERT INTO cases (id, category, title, questions, keywords, response_template, status, created_by, search_tsv)
			 VALUES (gen_random_uuid(), $1, $2, $3::jsonb, $4::jsonb, $5, 'draft', $6, to_tsvector('simple', $7))`,
			item.Category, item.Title, qJSON, kJSON, item.ResponseTemplate, createdBy, tsvInput,
		)
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			imported++
		}
	}
	return imported, errs
}

func ptrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func strPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func isPgNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

