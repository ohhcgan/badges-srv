package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github-badges-backend/internal/auth"
	"github-badges-backend/internal/stats"
	"github-badges-backend/internal/user"
)

type Controllers struct {
	logger       *zap.Logger
	userStore    *user.Store
	statsStore   *stats.Store
	monthlyJobFn func()
}

func NewControllers(
	userStore *user.Store,
	statsStore *stats.Store,
	logger *zap.Logger,
	monthlyJobFn func(),
) *Controllers {
	return &Controllers{
		logger:       logger,
		userStore:    userStore,
		statsStore:   statsStore,
		monthlyJobFn: monthlyJobFn,
	}
}

/**
 * Controllers
 */
func (h *Controllers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	/**
	 * TODO: move to gin, and remove writeJson function.
	 */
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Controllers) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	u, err := h.userStore.FindByID(r.Context(), userID)
	if errors.Is(err, user.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if err != nil {
		h.logger.Error("get user info query failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"github_id":  u.GithubID,
		"login":      u.GithubLogin,
		"name":       u.Name,
		"email":      u.Email,
		"avatar_url": u.AvatarURL,
		"created_at": u.CreatedAt,
	})
}

func (h *Controllers) GetStats(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")

	targetMonth, err := GetTargetMonth(year, month)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid year or month")
		return
	}

	st, err := h.statsStore.FindByUserAndMonth(r.Context(), userID, targetMonth)
	if errors.Is(err, stats.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no stats for this month yet")
		return
	}
	if err != nil {
		h.logger.Error("get stats query failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := statsResponse(st)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Controllers) GetStatsHistory(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())

	monthsNum := 12
	list, err := h.statsStore.ListForUser(r.Context(), userID, monthsNum)
	if err != nil {
		h.logger.Error("get stats history query failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	result := make([]map[string]any, 0, len(list))
	for _, st := range list {
		result = append(result, statsResponse(st))
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": result})
}

func (h *Controllers) TriggerMonthly(w http.ResponseWriter, r *http.Request) {
	if h.monthlyJobFn == nil {
		writeError(w, http.StatusServiceUnavailable, "job not available")
		return
	}

	go h.monthlyJobFn()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "job started"})
}

/**
 * Helpers
 */
func statsResponse(st *stats.MonthlyStats) map[string]any {
	resp := map[string]any{
		"id":                        st.ID,
		"user_id":                   st.UserID,
		"month":                     st.StatMonth.Format("2006-01"),
		"total_commits":             st.TotalCommits,
		"repos_created":             st.ReposCreated,
		"open_source_contributions": st.OpenSourceContributions,
		"created_at":                st.CreatedAt,
	}
	if st.CommitPctChange.Valid {
		resp["commit_pct_change"] = st.CommitPctChange.Float64
	} else {
		resp["commit_pct_change"] = nil
	}
	return resp
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Printf("error writingJson: %v\n", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func GetTargetMonth(year, month string) (time.Time, error) {
	var targetMonth time.Time

	if year != "" && month != "" {
		year, yearParseErr := strconv.Atoi(year)
		month, monthParseErr := strconv.Atoi(month)

		if month < 1 || month > 12 ||
			yearParseErr != nil || monthParseErr != nil ||
			(year >= int(time.Now().Year()) && month >= int(time.Now().Month())) {
			return time.Time{}, errors.New("invalid year or month")
		}
		targetMonth = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	} else {
		/* Fall back to prev month */

		now := time.Now().UTC()
		monthOffset := time.Month(1)
		yearOffset := 0

		if now.Month()-monthOffset <= 0 {
			yearOffset += 1
			monthOffset = -11
		}

		targetMonth = time.Date(now.Year()-yearOffset, now.Month()-monthOffset, 1, 0, 0, 0, 0, time.UTC)
	}
	return targetMonth, nil
}
