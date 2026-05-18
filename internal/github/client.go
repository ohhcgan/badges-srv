package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type ContributionStats struct {
	TotalCommits            int
	ReposCreated            int
	OpenSourceContributions int
}

type Client struct {
	v4    *githubv4.Client
	token string
}

func NewClient(accessToken, refreshToken string) *Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken, RefreshToken: refreshToken})
	httpClient := oauth2.NewClient(context.Background(), src)
	return &Client{
		v4:    githubv4.NewClient(httpClient),
		token: accessToken,
	}
}

func (c *Client) FetchMonthlyStats(ctx context.Context, login string, from, to time.Time) (*ContributionStats, error) {
	vars := map[string]any{
		"login": githubv4.String(login),
		"from":  githubv4.DateTime{Time: from},
		"to":    githubv4.DateTime{Time: to},
	}

	var q MonthlyStatsQuery

	if err := c.v4.Query(ctx, &q, vars); err != nil {
		return nil, fmt.Errorf("github graphql query: %w", err)
	}

	reposCreated := 0
	openSourceContributions := 0
	totalCommits := int(q.User.ContributionsCollection.TotalCommitContributions)

	/* TODO: use EndCursor & HasNextPage for fetching remaining pages */
	for _, node := range q.User.Repositories.Nodes {
		t := node.CreatedAt.Time.UTC()
		if t.Before(from) || t.After(to) {
			break
		}
		reposCreated++
	}

	for _, repo := range q.User.ContributionsCollection.PullRequestContributionsByRepository {
		if strings.EqualFold(string(repo.Repository.Owner.Login), login) {
			continue
		}
		openSourceContributions += int(repo.Contributions.TotalCount)
	}

	return &ContributionStats{
		TotalCommits:            totalCommits,
		ReposCreated:            reposCreated,
		OpenSourceContributions: openSourceContributions,
	}, nil
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func UserInfo(ctx context.Context, token string) (id int64, login, name, email, avatarURL string, err error) {
	helper := func(path string, dest any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+path, nil)
		if err != nil {
			return errors.New("unexpected error happen")
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("github API %s returned %d", path, resp.StatusCode)
		}
		return json.NewDecoder(resp.Body).Decode(dest)
	}

	var u githubUser
	if err = helper("/user", &u); err != nil {
		return 0, "", "", "", "", fmt.Errorf("fetching /user: %w", err)
	}

	id, login, name, avatarURL = u.ID, u.Login, u.Name, u.AvatarURL
	email = u.Email

	if email == "" {
		var emails []githubEmail
		if err = helper("/user/emails", &emails); err == nil {
			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					break
				}
			}
		}
	}
	return id, login, name, email, avatarURL, nil
}
