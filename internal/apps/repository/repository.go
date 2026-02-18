package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourusername/project/internal/apps"
)

// Repository implements apps persistence (CRUD + settings).
type Repository interface {
	Create(ctx context.Context, a *apps.App) error
	GetByID(ctx context.Context, id string) (*apps.App, error)
	List(ctx context.Context) ([]apps.App, error)
	Update(ctx context.Context, id string, u *apps.AppUpdate) (*apps.App, error)
	GetSettings(ctx context.Context, appID string) (map[string]interface{}, error)
	UpdateSettings(ctx context.Context, appID string, settings map[string]interface{}) error
	GetByName(ctx context.Context, name string) (*apps.App, error)
}

// Pool implements Repository using pgxpool.
type Pool struct {
	pool *pgxpool.Pool
}

// NewPool returns a repository using the given pool.
func NewPool(pool *pgxpool.Pool) *Pool {
	return &Pool{pool: pool}
}

func isPgNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (p *Pool) Create(ctx context.Context, a *apps.App) error {
	settingsJSON, _ := json.Marshal(a.Settings)
	if settingsJSON == nil {
		settingsJSON = []byte("{}")
	}
	query := `
		INSERT INTO apps (id, name, settings)
		VALUES (gen_random_uuid(), $1, $2::jsonb)
		RETURNING id, created_at, updated_at`
	err := p.pool.QueryRow(ctx, query, a.Name, settingsJSON).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if isPgUniqueViolation(err) {
			return apps.ErrConflict
		}
		return err
	}
	return nil
}

func (p *Pool) GetByID(ctx context.Context, id string) (*apps.App, error) {
	query := `SELECT id, name, settings, created_at, updated_at FROM apps WHERE id = $1`
	var a apps.App
	var settingsJSON []byte
	err := p.pool.QueryRow(ctx, query, id).Scan(&a.ID, &a.Name, &settingsJSON, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if isPgNoRows(err) {
			return nil, apps.ErrNotFound
		}
		return nil, err
	}
	if len(settingsJSON) > 0 {
		_ = json.Unmarshal(settingsJSON, &a.Settings)
	}
	if a.Settings == nil {
		a.Settings = make(map[string]interface{})
	}
	return &a, nil
}

func (p *Pool) List(ctx context.Context) ([]apps.App, error) {
	query := `SELECT id, name, settings, created_at, updated_at FROM apps ORDER BY name`
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []apps.App
	for rows.Next() {
		var a apps.App
		var settingsJSON []byte
		if err := rows.Scan(&a.ID, &a.Name, &settingsJSON, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		if len(settingsJSON) > 0 {
			_ = json.Unmarshal(settingsJSON, &a.Settings)
		}
		if a.Settings == nil {
			a.Settings = make(map[string]interface{})
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (p *Pool) Update(ctx context.Context, id string, u *apps.AppUpdate) (*apps.App, error) {
	cur, err := p.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	name := cur.Name
	settings := cur.Settings
	if u.Name != nil {
		name = *u.Name
	}
	if u.Settings != nil {
		settings = u.Settings
	}
	settingsJSON, _ := json.Marshal(settings)
	if settingsJSON == nil {
		settingsJSON = []byte("{}")
	}
	query := `UPDATE apps SET name = $1, settings = $2::jsonb, updated_at = NOW() WHERE id = $3 RETURNING id, name, settings, created_at, updated_at`
	var a apps.App
	var settingsRaw []byte
	err = p.pool.QueryRow(ctx, query, name, settingsJSON, id).Scan(&a.ID, &a.Name, &settingsRaw, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if isPgNoRows(err) {
			return nil, apps.ErrNotFound
		}
		if isPgUniqueViolation(err) {
			return nil, apps.ErrConflict
		}
		return nil, err
	}
	if len(settingsRaw) > 0 {
		_ = json.Unmarshal(settingsRaw, &a.Settings)
	}
	if a.Settings == nil {
		a.Settings = make(map[string]interface{})
	}
	return &a, nil
}

func (p *Pool) GetSettings(ctx context.Context, appID string) (map[string]interface{}, error) {
	query := `SELECT settings FROM apps WHERE id = $1`
	var settingsJSON []byte
	err := p.pool.QueryRow(ctx, query, appID).Scan(&settingsJSON)
	if err != nil {
		if isPgNoRows(err) {
			return nil, apps.ErrNotFound
		}
		return nil, err
	}
	var settings map[string]interface{}
	if len(settingsJSON) > 0 {
		_ = json.Unmarshal(settingsJSON, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}
	return settings, nil
}

func (p *Pool) UpdateSettings(ctx context.Context, appID string, settings map[string]interface{}) error {
	settingsJSON, _ := json.Marshal(settings)
	if settingsJSON == nil {
		settingsJSON = []byte("{}")
	}
	cmd, err := p.pool.Exec(ctx, `UPDATE apps SET settings = $1::jsonb, updated_at = NOW() WHERE id = $2`, settingsJSON, appID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return apps.ErrNotFound
	}
	return nil
}

func (p *Pool) GetByName(ctx context.Context, name string) (*apps.App, error) {
	query := `SELECT id, name, settings, created_at, updated_at FROM apps WHERE name = $1`
	var a apps.App
	var settingsJSON []byte
	err := p.pool.QueryRow(ctx, query, name).Scan(&a.ID, &a.Name, &settingsJSON, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if isPgNoRows(err) {
			return nil, apps.ErrNotFound
		}
		return nil, err
	}
	if len(settingsJSON) > 0 {
		_ = json.Unmarshal(settingsJSON, &a.Settings)
	}
	if a.Settings == nil {
		a.Settings = make(map[string]interface{})
	}
	return &a, nil
}
