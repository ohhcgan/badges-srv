package github

import "github.com/shurcooL/githubv4"

type PullReqContributionsByRepo struct {
	Repository struct {
		Owner struct {
			Login githubv4.String
		}
	}
	Contributions struct {
		TotalCount githubv4.Int
	}
}

type MonthlyStatsQueryUserRepositories struct {
	Nodes []struct {
		CreatedAt githubv4.DateTime
	}
	PageInfo struct {
		HasNextPage githubv4.Boolean
		EndCursor   githubv4.String
	}
}

type MonthlyStatsQueryUserContributionsCollection struct {
	TotalCommitContributions             githubv4.Int
	PullRequestContributionsByRepository []PullReqContributionsByRepo `graphql:"pullRequestContributionsByRepository(maxRepositories: 100)"`
}

type MonthlyStatsQueryUser struct {
	ContributionsCollection MonthlyStatsQueryUserContributionsCollection `graphql:"contributionsCollection(from: $from, to: $to)"`
	Repositories            MonthlyStatsQueryUserRepositories            `graphql:"repositories(first: 100, affiliations: OWNER, orderBy: {field: CREATED_AT, direction: DESC})"`
}

type MonthlyStatsQuery struct {
	User MonthlyStatsQueryUser `graphql:"user(login: $login)"`
}
