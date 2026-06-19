package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Gerry3010/pipepush/internal/models"
)

// DB wraps a pgxpool and provides all data access methods.
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new DB connection pool and verifies connectivity.
func New(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the connection pool.
func (d *DB) Close() {
	d.pool.Close()
}

// --- Users ---

func (d *DB) CreateUser(ctx context.Context, email, passwordHash, publicKey, encPrivKey, kdfSalt string) (*models.User, error) {
	var u models.User
	err := d.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, public_key, encrypted_private_key, kdf_salt)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, public_key, encrypted_private_key, kdf_salt, created_at`,
		email, passwordHash, publicKey, encPrivKey, kdfSalt,
	).Scan(&u.ID, &u.Email, &u.PublicKey, &u.EncryptedPrivateKey, &u.KDFSalt, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return &u, nil
}

func (d *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, string, error) {
	var u models.User
	var passwordHash string
	err := d.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, public_key, encrypted_private_key, kdf_salt, created_at
		FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &passwordHash, &u.PublicKey, &u.EncryptedPrivateKey, &u.KDFSalt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("getting user: %w", err)
	}
	return &u, passwordHash, nil
}

func (d *DB) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := d.pool.QueryRow(ctx, `
		SELECT id, email, public_key, encrypted_private_key, kdf_salt, created_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PublicKey, &u.EncryptedPrivateKey, &u.KDFSalt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return &u, nil
}

// --- Projects ---

func (d *DB) CreateProject(ctx context.Context, userID, encName, encDesc string) (*models.Project, error) {
	var p models.Project
	err := d.pool.QueryRow(ctx, `
		INSERT INTO projects (user_id, encrypted_name, encrypted_description)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id, user_id, encrypted_name, COALESCE(encrypted_description, ''), created_at`,
		userID, encName, encDesc,
	).Scan(&p.ID, &p.UserID, &p.EncryptedName, &p.EncryptedDescription, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}
	return &p, nil
}

func (d *DB) ListProjects(ctx context.Context, userID string) ([]*models.Project, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, user_id, encrypted_name, COALESCE(encrypted_description, ''), created_at
		FROM projects WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	defer rows.Close()
	return scanProjects(rows)
}

func (d *DB) GetProject(ctx context.Context, id, userID string) (*models.Project, error) {
	var p models.Project
	err := d.pool.QueryRow(ctx, `
		SELECT id, user_id, encrypted_name, COALESCE(encrypted_description, ''), created_at
		FROM projects WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&p.ID, &p.UserID, &p.EncryptedName, &p.EncryptedDescription, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}
	return &p, nil
}

func (d *DB) DeleteProject(ctx context.Context, id, userID string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// --- Pipelines ---

func (d *DB) CreatePipeline(ctx context.Context, userID, projectID, encName, routingKey string) (*models.Pipeline, error) {
	var p models.Pipeline
	err := d.pool.QueryRow(ctx, `
		INSERT INTO pipelines (user_id, project_id, encrypted_name, routing_key)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id, user_id, project_id, encrypted_name, COALESCE(routing_key, ''), created_at`,
		userID, projectID, encName, routingKey,
	).Scan(&p.ID, &p.UserID, &p.ProjectID, &p.EncryptedName, &p.RoutingKey, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating pipeline: %w", err)
	}
	return &p, nil
}

func (d *DB) ListPipelines(ctx context.Context, projectID, userID string) ([]*models.Pipeline, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, user_id, project_id, encrypted_name, COALESCE(routing_key, ''), created_at
		FROM pipelines WHERE project_id = $1 AND user_id = $2
		ORDER BY created_at DESC`, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("listing pipelines: %w", err)
	}
	defer rows.Close()
	return scanPipelines(rows)
}

func (d *DB) GetPipeline(ctx context.Context, id, userID string) (*models.Pipeline, error) {
	var p models.Pipeline
	err := d.pool.QueryRow(ctx, `
		SELECT id, user_id, project_id, encrypted_name, COALESCE(routing_key, ''), created_at
		FROM pipelines WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&p.ID, &p.UserID, &p.ProjectID, &p.EncryptedName, &p.RoutingKey, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting pipeline: %w", err)
	}
	return &p, nil
}

// GetPipelineByRoutingKey finds a pipeline in a project by its routing key.
// Used to resolve a project-scoped token's target pipeline from the webhook's
// plaintext pipeline name. Returns (nil, nil) when no pipeline matches.
func (d *DB) GetPipelineByRoutingKey(ctx context.Context, projectID, routingKey string) (*models.Pipeline, error) {
	var p models.Pipeline
	err := d.pool.QueryRow(ctx, `
		SELECT id, user_id, project_id, encrypted_name, COALESCE(routing_key, ''), created_at
		FROM pipelines WHERE project_id = $1 AND routing_key = $2`, projectID, routingKey,
	).Scan(&p.ID, &p.UserID, &p.ProjectID, &p.EncryptedName, &p.RoutingKey, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting pipeline by routing key: %w", err)
	}
	return &p, nil
}

func (d *DB) DeletePipeline(ctx context.Context, id, userID string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM pipelines WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// --- Notification Tokens ---

func (d *DB) CreateNotificationToken(ctx context.Context, userID, projectID, pipelineID, encName, tokenHash string) (*models.NotificationToken, error) {
	var t models.NotificationToken
	var pipeIDPtr *string
	if pipelineID != "" {
		pipeIDPtr = &pipelineID
	}
	err := d.pool.QueryRow(ctx, `
		INSERT INTO notification_tokens (user_id, project_id, pipeline_id, encrypted_name, token_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, project_id, COALESCE(pipeline_id::text, ''), encrypted_name, token_hash, active, created_at, last_used_at`,
		userID, projectID, pipeIDPtr, encName, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.ProjectID, &t.PipelineID, &t.EncryptedName, &t.TokenHash, &t.Active, &t.CreatedAt, &t.LastUsedAt)
	if err != nil {
		return nil, fmt.Errorf("creating token: %w", err)
	}
	return &t, nil
}

func (d *DB) ListTokens(ctx context.Context, projectID, userID string) ([]*models.NotificationToken, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, user_id, project_id, COALESCE(pipeline_id::text, ''), encrypted_name, token_hash, active, created_at, last_used_at
		FROM notification_tokens WHERE project_id = $1 AND user_id = $2
		ORDER BY created_at DESC`, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("listing tokens: %w", err)
	}
	defer rows.Close()
	return scanTokens(rows)
}

// GetTokenByHash finds an active token by its SHA-256 hash.
// Returns the token along with the owning user's public key.
func (d *DB) GetTokenByHash(ctx context.Context, hash string) (*models.NotificationToken, string, error) {
	var t models.NotificationToken
	var publicKey string
	err := d.pool.QueryRow(ctx, `
		SELECT t.id, t.user_id, t.project_id, COALESCE(t.pipeline_id::text, ''), t.encrypted_name, t.token_hash, t.active, t.created_at, t.last_used_at, u.public_key
		FROM notification_tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = $1 AND t.active = true`, hash,
	).Scan(&t.ID, &t.UserID, &t.ProjectID, &t.PipelineID, &t.EncryptedName, &t.TokenHash, &t.Active, &t.CreatedAt, &t.LastUsedAt, &publicKey)
	if err == pgx.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("getting token by hash: %w", err)
	}
	return &t, publicKey, nil
}

func (d *DB) RevokeToken(ctx context.Context, id, userID string) error {
	_, err := d.pool.Exec(ctx, `UPDATE notification_tokens SET active = false WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// DeleteToken permanently removes a token. Runs reference it via ON DELETE SET
// NULL, so their history is preserved (token_id just becomes null).
func (d *DB) DeleteToken(ctx context.Context, id, userID string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM notification_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (d *DB) TouchToken(ctx context.Context, id string) error {
	_, err := d.pool.Exec(ctx, `UPDATE notification_tokens SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

// --- Runs ---

func (d *DB) CreateRun(ctx context.Context, userID, projectID, pipelineID, tokenID, status, encPayload string) (*models.Run, error) {
	var r models.Run
	err := d.pool.QueryRow(ctx, `
		INSERT INTO runs (user_id, project_id, pipeline_id, token_id, status, encrypted_payload)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, project_id, pipeline_id, token_id, status, encrypted_payload, received_at`,
		userID, projectID, pipelineID, tokenID, status, encPayload,
	).Scan(&r.ID, &r.UserID, &r.ProjectID, &r.PipelineID, &r.TokenID, &r.Status, &r.EncryptedPayload, &r.ReceivedAt)
	if err != nil {
		return nil, fmt.Errorf("creating run: %w", err)
	}
	return &r, nil
}

func (d *DB) ListRuns(ctx context.Context, pipelineID, userID string, limit int) ([]*models.Run, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := d.pool.Query(ctx, `
		SELECT id, user_id, project_id, pipeline_id, COALESCE(token_id::text, ''), status, encrypted_payload, received_at
		FROM runs WHERE pipeline_id = $1 AND user_id = $2
		ORDER BY received_at DESC LIMIT $3`, pipelineID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}
	defer rows.Close()
	return scanRuns(rows)
}

func (d *DB) GetRun(ctx context.Context, id, userID string) (*models.Run, error) {
	var r models.Run
	err := d.pool.QueryRow(ctx, `
		SELECT id, user_id, project_id, pipeline_id, COALESCE(token_id::text, ''), status, encrypted_payload, received_at
		FROM runs WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&r.ID, &r.UserID, &r.ProjectID, &r.PipelineID, &r.TokenID, &r.Status, &r.EncryptedPayload, &r.ReceivedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting run: %w", err)
	}
	return &r, nil
}

// --- VAPID Subscriptions ---

func (d *DB) UpsertVAPIDSubscription(ctx context.Context, userID, endpoint, p256dh, auth, deviceName string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO vapid_subscriptions (user_id, endpoint, p256dh_key, auth_key, device_name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, endpoint) DO UPDATE
		SET p256dh_key = EXCLUDED.p256dh_key, auth_key = EXCLUDED.auth_key, device_name = EXCLUDED.device_name`,
		userID, endpoint, p256dh, auth, deviceName)
	return err
}

func (d *DB) DeleteVAPIDSubscription(ctx context.Context, userID, endpoint string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM vapid_subscriptions WHERE user_id = $1 AND endpoint = $2`, userID, endpoint)
	return err
}

func (d *DB) ListVAPIDSubscriptions(ctx context.Context, userID string) ([]*models.VAPIDSubscription, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, user_id, endpoint, p256dh_key, auth_key, COALESCE(device_name, ''), created_at
		FROM vapid_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.VAPIDSubscription
	for rows.Next() {
		var s models.VAPIDSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256DHKey, &s.AuthKey, &s.DeviceName, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	return subs, rows.Err()
}

// --- Scanner helpers ---

func scanProjects(rows pgx.Rows) ([]*models.Project, error) {
	var projects []*models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.EncryptedName, &p.EncryptedDescription, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

func scanPipelines(rows pgx.Rows) ([]*models.Pipeline, error) {
	var pipelines []*models.Pipeline
	for rows.Next() {
		var p models.Pipeline
		if err := rows.Scan(&p.ID, &p.UserID, &p.ProjectID, &p.EncryptedName, &p.RoutingKey, &p.CreatedAt); err != nil {
			return nil, err
		}
		pipelines = append(pipelines, &p)
	}
	return pipelines, rows.Err()
}

func scanTokens(rows pgx.Rows) ([]*models.NotificationToken, error) {
	var tokens []*models.NotificationToken
	for rows.Next() {
		var t models.NotificationToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.ProjectID, &t.PipelineID, &t.EncryptedName, &t.TokenHash, &t.Active, &t.CreatedAt, &t.LastUsedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, &t)
	}
	return tokens, rows.Err()
}

func scanRuns(rows pgx.Rows) ([]*models.Run, error) {
	var runs []*models.Run
	for rows.Next() {
		var r models.Run
		if err := rows.Scan(&r.ID, &r.UserID, &r.ProjectID, &r.PipelineID, &r.TokenID, &r.Status, &r.EncryptedPayload, &r.ReceivedAt); err != nil {
			return nil, err
		}
		runs = append(runs, &r)
	}
	return runs, rows.Err()
}
