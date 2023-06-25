package initializers

import (
	"fmt"
	"os"
	"github.com/google/go-github/v33/github"
)

type Env struct {
	RepoOwner    string `env:"INPUT_REPO_OWNER"`
	RepoName     string `env:"INPUT_REPO_NAME"`
	BaseBranch   string `env:"INPUT_BASE_BRANCH"`
	TargetBranch string `env:"INPUT_TARGET_BRANCH"`
	Token        string `env:"INPUT_GITHUB_TOKEN"`
	GithubEvent *github.PullRequestEvent
}

func LoadEnv() (env Env, err error) {
	env = Env{
		RepoOwner:    os.Getenv("INPUT_REPO_OWNER"),
		RepoName:     os.Getenv("INPUT_REPO_NAME"),
		BaseBranch:   os.Getenv("INPUT_BASE_BRANCH"),
		TargetBranch: os.Getenv("INPUT_TARGET_BRANCH"),
		Token:        os.Getenv("INPUT_GITHUB_TOKEN"),
		GithubEvent:  nil,
	}

	// validate config struct
	if env.RepoOwner == "" {
		return env, fmt.Errorf("missing repo owner")
	}
	if env.RepoName == "" {
		return env, fmt.Errorf("missing repo name")
	}
	if env.BaseBranch == "" {
		return env, fmt.Errorf("missing base branch")
	}
	if env.TargetBranch == "" {
		return env, fmt.Errorf("missing target branch")
	}
	if env.Token == "" {
		return env, fmt.Errorf("missing github token")
	}
	if env.GithubEvent == nil {
		return env, fmt.Errorf("missing github event")
	}

	return env, nil
}
