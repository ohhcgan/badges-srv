package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("stats not found")

/**
 * Store handles all database operations for stats.
 */
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Upsert(ctx context.Context, st *MonthlyStats) error {
	const UPSERT_MONTHLY_USER_STATS_QUERY = `
		INSERT INTO monthly_stats
			(user_id, stat_month, total_commits, repos_created, open_source_contributions, commit_pct_change)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, stat_month) DO UPDATE SET
			total_commits               = EXCLUDED.total_commits,
			repos_created               = EXCLUDED.repos_created,
			open_source_contributions   = EXCLUDED.open_source_contributions,
			commit_pct_change           = EXCLUDED.commit_pct_change
		RETURNING id, created_at
	`
	return s.pool.QueryRow(ctx, UPSERT_MONTHLY_USER_STATS_QUERY,
		st.UserID,
		st.StatMonth,
		st.TotalCommits,
		st.ReposCreated,
		st.OpenSourceContributions,
		st.CommitPctChange,
	).Scan(&st.ID, &st.CreatedAt)
}

func (s *Store) FindByUserAndMonth(ctx context.Context, userID string, month time.Time) (*MonthlyStats, error) {
	const FIND_BY_USER_AND_MONTY_QUERY = `
		SELECT id, user_id, stat_month, total_commits, repos_created, open_source_contributions, commit_pct_change, created_at
		FROM monthly_stats WHERE user_id = $1 AND stat_month = $2
	`

	st := &MonthlyStats{}

	err := s.pool.QueryRow(ctx, FIND_BY_USER_AND_MONTY_QUERY, userID, month).Scan(
		&st.ID,
		&st.UserID,
		&st.StatMonth,
		&st.TotalCommits,
		&st.ReposCreated,
		&st.OpenSourceContributions,
		&st.CommitPctChange,
		&st.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying monthly stats: %w", err)
	}
	return st, nil
}

func (s *Store) FindLatestForUser(ctx context.Context, userID string) (*MonthlyStats, error) {
	const FIND_LATEST_STATS_QUERY = `
		SELECT id, user_id, stat_month, total_commits, repos_created, open_source_contributions, commit_pct_change, created_at
		FROM monthly_stats WHERE user_id = $1 ORDER BY stat_month DESC LIMIT 1
	`

	st := &MonthlyStats{}

	err := s.pool.QueryRow(ctx, FIND_LATEST_STATS_QUERY, userID).Scan(
		&st.ID,
		&st.UserID,
		&st.StatMonth,
		&st.TotalCommits,
		&st.ReposCreated,
		&st.OpenSourceContributions,
		&st.CommitPctChange,
		&st.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying latest stats: %w", err)
	}
	return st, nil
}

func (s *Store) ListForUser(ctx context.Context, userID string, limit int) ([]*MonthlyStats, error) {
	const LIST_USER_STATS_QUERY = `
		SELECT id, user_id, stat_month, total_commits, repos_created, open_source_contributions, commit_pct_change, created_at
		FROM monthly_stats WHERE user_id = $1 ORDER BY stat_month DESC LIMIT $2
	`

	rows, err := s.pool.Query(ctx, LIST_USER_STATS_QUERY, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing stats: %w", err)
	}
	defer rows.Close()

	var result []*MonthlyStats

	for rows.Next() {
		st := &MonthlyStats{}

		err := rows.Scan(
			&st.ID,
			&st.UserID,
			&st.StatMonth,
			&st.TotalCommits,
			&st.ReposCreated,
			&st.OpenSourceContributions,
			&st.CommitPctChange,
			&st.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scanning stats row: %w", err)
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

func (s *Store) LogEmail(ctx context.Context, userID string, month time.Time, status string, errMsg string) error {
	const INSERT_EMAIL_LOG = "INSERT INTO email_logs (user_id, stat_month, status, error_msg) VALUES ($1, $2, $3, NULLIF($4, ''))"
	_, err := s.pool.Exec(ctx, INSERT_EMAIL_LOG, userID, month, status, errMsg)
	return err
}
