package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"

	"github.com/yourusername/project/internal/search"
)

// Pool runs search queries (pgvector + FTS). Implements search.SearchRepository.
type Pool struct {
	pool *pgxpool.Pool
}

// NewPool returns a repository that uses the given pool. Caller must ensure
// pgxvec.RegisterTypes is called on connections (e.g. via AfterConnect).
func NewPool(pool *pgxpool.Pool) *Pool {
	return &Pool{pool: pool}
}

// SearchApproved runs vector (cosine) and FTS retrieval for approved cases only,
// merges by case_id, and returns candidates with both cosine_similarity and fts_rank where available.
func (p *Pool) SearchApproved(ctx context.Context, params search.SearchParams) ([]search.Candidate, error) {
	vec := pgvector.NewVector(params.QueryVector)
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	categoryFilter := params.Category != ""

	// Register pgvector types on the connection used for this query
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	if err := pgxvec.RegisterTypes(ctx, conn.Conn()); err != nil {
		return nil, err
	}

	byID := make(map[string]*search.Candidate)

	// 1) Vector search: cosine similarity (1 - distance), only approved
	vectorQuery := `
		SELECT c.id, c.title, c.response_template, c.category,
		       (1 - (e.embedding <=> $1::vector)) AS cosine_similarity
		FROM cases c
		JOIN case_embeddings e ON e.case_id = c.id
		WHERE c.status = 'approved'
		  AND (NOT $2::boolean OR c.category = $3)
		ORDER BY e.embedding <=> $1::vector
		LIMIT $4`
	vectorRows, err := conn.Query(ctx, vectorQuery, vec, categoryFilter, params.Category, limit)
	if err != nil {
		return nil, err
	}
	for vectorRows.Next() {
		var c search.Candidate
		err := vectorRows.Scan(&c.CaseID, &c.Title, &c.ResponseTemplate, &c.Category, &c.CosineSimilarity)
		if err != nil {
			vectorRows.Close()
			return nil, err
		}
		c.FTSRank = 0
		byID[c.CaseID] = &c
	}
	vectorRows.Close()
	if err := vectorRows.Err(); err != nil {
		return nil, err
	}

	// 2) FTS: search_tsv, only approved
	ftsQuery := `
		SELECT c.id, c.title, c.response_template, c.category,
		       ts_rank_cd(c.search_tsv, plainto_tsquery('english', $1)) AS fts_rank
		FROM cases c
		WHERE c.status = 'approved'
		  AND c.search_tsv IS NOT NULL
		  AND c.search_tsv @@ plainto_tsquery('english', $1)
		  AND (NOT $2::boolean OR c.category = $3)
		ORDER BY fts_rank DESC
		LIMIT $4`
	ftsRows, err := conn.Query(ctx, ftsQuery, params.QueryFTS, categoryFilter, params.Category, limit)
	if err != nil {
		return nil, err
	}
	for ftsRows.Next() {
		var caseID, title, responseTemplate, category string
		var ftsRank float64
		if err := ftsRows.Scan(&caseID, &title, &responseTemplate, &category, &ftsRank); err != nil {
			ftsRows.Close()
			return nil, err
		}
		if existing, ok := byID[caseID]; ok {
			existing.FTSRank = ftsRank
		} else {
			byID[caseID] = &search.Candidate{
				CaseID:           caseID,
				Title:            title,
				ResponseTemplate: responseTemplate,
				Category:         category,
				CosineSimilarity: 0,
				FTSRank:          ftsRank,
			}
		}
	}
	ftsRows.Close()
	if err := ftsRows.Err(); err != nil {
		return nil, err
	}

	out := make([]search.Candidate, 0, len(byID))
	for _, c := range byID {
		out = append(out, *c)
	}
	return out, nil
}
