package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")

/**
 * Store handles all database operations for users.
 */
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Upsert(ctx context.Context, u *User) error {
	const UPSERT_QUERY = `
		INSERT INTO users (github_id, github_login, name, email, avatar_url, encrypted_token, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (github_id) DO UPDATE SET
			github_login    = EXCLUDED.github_login,
			name            = CASE WHEN EXCLUDED.name != '' THEN EXCLUDED.name ELSE users.name END,
			email           = CASE WHEN EXCLUDED.email != '' THEN EXCLUDED.email ELSE users.email END,
			avatar_url      = EXCLUDED.avatar_url,
			encrypted_token = EXCLUDED.encrypted_token,
			updated_at      = NOW()
		RETURNING id, created_at, updated_at
	`

	return s.pool.QueryRow(ctx, UPSERT_QUERY,
		u.GithubID, u.GithubLogin, u.Name, u.Email, u.AvatarURL, u.EncryptedToken,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (s *Store) FindByGithubID(ctx context.Context, githubId int64) (*User, error) {
	const FIND_USER_QUERY = `
		SELECT id, github_id, github_login, name, email, avatar_url, encrypted_token, created_at, updated_at
		FROM users WHERE github_id = $1
	`
	u := &User{}
	err := s.pool.QueryRow(ctx, FIND_USER_QUERY, githubId).Scan(
		&u.ID,
		&u.GithubID,
		&u.GithubLogin,
		&u.Name,
		&u.Email,
		&u.AvatarURL,
		&u.EncryptedToken,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user by github_id: %w", err)
	}
	return u, nil
}

func (s *Store) FindByID(ctx context.Context, id string) (*User, error) {
	const FIND_USER_QUERY = `
		SELECT id, github_id, github_login, name, email, avatar_url, encrypted_token, created_at, updated_at
		FROM users WHERE id = $1
	`
	u := &User{}
	err := s.pool.QueryRow(ctx, FIND_USER_QUERY, id).Scan(
		&u.ID,
		&u.GithubID,
		&u.GithubLogin,
		&u.Name,
		&u.Email,
		&u.AvatarURL,
		&u.EncryptedToken,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying user by id: %w", err)
	}
	return u, nil
}

func (s *Store) ListAll(ctx context.Context) ([]*User, error) {
	/**
	 * TODO: Add pagination here for more efficient batching.
	 */
	const LIST_USERS_QUERY = `
		SELECT id, github_id, github_login, name, email, avatar_url, encrypted_token, created_at, updated_at
		FROM users
		WHERE email != ''
		ORDER BY created_at
	`
	rows, err := s.pool.Query(ctx, LIST_USERS_QUERY)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}

		err := rows.Scan(
			&u.ID,
			&u.GithubID,
			&u.GithubLogin,
			&u.Name,
			&u.Email,
			&u.AvatarURL,
			&u.EncryptedToken,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
