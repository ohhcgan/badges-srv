package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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
func (h *Controllers) HealthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "ok",
		"success": true,
		"data":    nil,
		"error":   nil,
	})
}

func (h *Controllers) GetMe(ctx *gin.Context) {
	userID, _ := auth.UserIDFromContext(ctx.Request.Context())

	u, err := h.userStore.FindByID(ctx.Request.Context(), userID)
	if errors.Is(err, user.ErrNotFound) {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"data":    nil,
			"message": "user not found",
			"error": gin.H{
				"message": "user not found",
				"code":    "NOT_FOUND",
			},
		})
		return
	}

	if err != nil {
		h.logger.Error("get user info query failed", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"data":    nil,
			"message": "something unexpected happened",
			"error": gin.H{
				"message": "something unexpected happened",
				"code":    "INTERNAL_SERVER_ERROR",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":         u.ID,
			"github_id":  u.GithubID,
			"login":      u.GithubLogin,
			"name":       u.Name,
			"email":      u.Email,
			"avatar_url": u.AvatarURL,
			"created_at": u.CreatedAt,
		},
		"message": "user fetched",
		"error":   nil,
	})
}

func (h *Controllers) GetStats(ctx *gin.Context) {
	userID, _ := auth.UserIDFromContext(ctx.Request.Context())

	year := ctx.Request.URL.Query().Get("year")
	month := ctx.Request.URL.Query().Get("month")

	targetMonth, err := GetTargetMonth(year, month)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid year or month",
			"data":    nil,
			"error": gin.H{
				"message": "invalid year or month",
				"code":    "BAD_REQUEST",
			},
		})
		return
	}

	st, err := h.statsStore.FindByUserAndMonth(ctx.Request.Context(), userID, targetMonth)
	if errors.Is(err, stats.ErrNotFound) {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": true,
			"message": "no stats for this month yet",
			"data":    nil,
			"error":   nil,
		})
		return
	}

	if err != nil {
		h.logger.Error("get stats query failed", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "something unexpected happened",
			"data":    nil,
			"error": gin.H{
				"message": "something unexpected happened",
				"code":    "INTERNAL_SERVER_ERROR",
			},
		})
		return
	}

	resp := statsResponse(st)
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "fetched",
		"data":    resp,
		"error":   nil,
	})
}

func (h *Controllers) GetStatsHistory(ctx *gin.Context) {
	userID, _ := auth.UserIDFromContext(ctx.Request.Context())

	monthsNum := 12
	list, err := h.statsStore.ListForUser(ctx.Request.Context(), userID, monthsNum)
	if err != nil {
		h.logger.Error("get stats history query failed", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "something unexpected happened",
			"data":    nil,
			"error": gin.H{
				"message": "something unexpected happened",
				"code":    "INTERNAL_SERVER_ERROR",
			},
		})
		return
	}

	result := make([]map[string]any, 0, len(list))
	for _, st := range list {
		result = append(result, statsResponse(st))
	}
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "history fetched",
		"data":    gin.H{"history": result},
		"error":   nil,
	})
}

func (h *Controllers) TriggerMonthly(ctx *gin.Context) {
	if h.monthlyJobFn == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"message": "job not available",
			"success": false,
			"data":    nil,
			"error": gin.H{
				"message": "job not available",
				"code":    "SERVICE_TEMP_UNAVAILABLE",
			},
		})
		return
	}

	go h.monthlyJobFn()

	ctx.JSON(http.StatusAccepted, gin.H{
		"message": "job started",
		"success": true,
		"data":    nil,
		"error":   nil,
	})
}

/**
 * Helpers
 */
func statsResponse(st *stats.MonthlyStats) map[string]any {
	resp := gin.H{
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

func (h *Controllers) NoRoute(ctx *gin.Context) {
	ctx.JSON(http.StatusNotFound, gin.H{
		"message": "404 not found",
		"success": false,
		"data":    nil,
		"error": gin.H{
			"message": "404 not found",
			"code":    "NOT_FOUND",
		},
	})
}
