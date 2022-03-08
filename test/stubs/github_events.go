package stubs

import (
	"time"

	"github.com/google/go-github/v42/github"
)

var RepoURL = "https://github.com/username/repository"

const (
	RepoFullName         = "username/repository"
	HeadCommitID         = "commit-id"
	HeadCommitMsg        = "commit message"
	HeadCommitAuthorName = "Author's Name"
	BeforeCommitID       = "before-commit-id"
	GitRef               = "refs/heads/main"
)

func GitHubPushEvent() *github.PushEvent {
	return &github.PushEvent{
		Repo: &github.PushEventRepository{
			HTMLURL:  github.String(RepoURL),
			FullName: github.String(RepoFullName),
		},
		HeadCommit: &github.HeadCommit{
			ID:      github.String(HeadCommitID),
			Message: github.String(HeadCommitMsg),
			Timestamp: &github.Timestamp{
				Time: time.Now(),
			},
			Author: &github.CommitAuthor{
				Name: github.String(HeadCommitAuthorName),
			},
		},
		Before: github.String(BeforeCommitID),
		Ref:    github.String(GitRef),
	}
}
