package stats

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

/**
 * MonthlyStats holds aggregated GitHub activity for a single calendar month.
 */
type MonthlyStats struct {
	ID                      uuid.UUID
	UserID                  string
	StatMonth               time.Time /* always the 1st of the month, UTC */
	TotalCommits            int
	ReposCreated            int
	OpenSourceContributions int
	CommitPctChange         sql.NullFloat64 /* CommitPctChange is NULL for the user's very first tracked month */
	CreatedAt               time.Time
}
