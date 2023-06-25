package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MateSousa/create-release-bot/initializers"
	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

const (
	prLabelPending string = "createrelease:pending"
	prLabelmerged  string = "createrelease:merged"
	releaseCommit  string = "Release is at:"
	releaseName    string = "Release "
)

func PREvent(client *github.Client, env initializers.Env, event *github.PullRequestEvent) error {
	switch *event.Action {
	case "closed":
		if *event.PullRequest.Merged {
			// Check if PR has label "createrelease:pending"
			if !HasPendingLabel(event.PullRequest) {
				return nil
			}
			// Remove label "createrelease:pending" and add "createrelease:merged" to PR
			err := AddMergedLabel(client, env, event.PullRequest)
			if err != nil {
				return err
			}
			// Create a new latest release tag and increment the minor version
			newReleaseTag, err := CreateNewLatestReleaseTag(client, env)
			if err != nil {
				return err
			}
			// Create a new release
			newRelease, err := CreateNewRelease(client, env, newReleaseTag)
			if err != nil {
				return err
			}
			// Create a new comment in PR with commit message
			commit := "Release is at: " + newRelease.GetHTMLURL()
			err = CreateNewComment(client, env, event.PullRequest, commit)
			if err != nil {
				return err
			}
		} else {
			if !HasPendingLabel(event.PullRequest) {
				return nil
			}
			// Remove label "createrelease:pending" from PR
			err := RemovePendingLabel(client, env, event.PullRequest)
			if err != nil {
				fmt.Printf("error removing label: %v", err)
				return err
			}
		}
	}

	return nil
}

func main() {
	env, err := initializers.LoadEnv()
	if err != nil {
		fmt.Printf("error loading env: %v", err)
		os.Exit(1)
	}

	client, err := CreateGithubClient(env)
	if err != nil {
		fmt.Printf("error creating github client: %v", err)
		os.Exit(1)
	}

	fmt.Printf("repo owner: %v\n", client.Reactions)

	fmt.Printf("event: %v\n", env.GithubEvent)

	// err = PREvent(client, env, env.GithubEvent)
	// if err != nil {
	// 	fmt.Printf("error handling pr event: %v", err)
	// 	os.Exit(1)
	// }
	os.Exit(0)
}

// Create a github client with a token
func CreateGithubClient(env initializers.Env) (*github.Client, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: env.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return client, nil
}

// Create a label "createrelease:pending" and add to PR
func AddPendingLabel(client *github.Client, env initializers.Env, pr *github.PullRequest) error {
	_, _, err := client.Issues.AddLabelsToIssue(context.Background(), env.RepoOwner, env.RepoName, *pr.Number, []string{prLabelPending})
	if err != nil {
		return err
	}

	return nil
}

func RemovePendingLabel(client *github.Client, env initializers.Env, pr *github.PullRequest) error {
	_, err := client.Issues.RemoveLabelForIssue(context.Background(), env.RepoOwner, env.RepoName, *pr.Number, prLabelPending)
	if err != nil {
		fmt.Printf("error removing label: %v", err)
		return err
	}

	return nil
}

// Remove label "createrelease:pending" and add "createrelease:merged" to PR
func AddMergedLabel(client *github.Client, env initializers.Env, pr *github.PullRequest) error {
	err := RemovePendingLabel(client, env, pr)
	if err != nil {
		fmt.Printf("error removing label: %v", err)
		return err
	}

	_, _, err = client.Issues.AddLabelsToIssue(context.Background(), env.RepoOwner, env.RepoName, *pr.Number, []string{prLabelmerged})
	if err != nil {
		return err
	}

	return nil
}

// Create a new latest release tag and increment the minor version
func CreateNewLatestReleaseTag(client *github.Client, env initializers.Env) (string, error) {
	var releaseTag string

	releaseList, _, err := client.Repositories.ListReleases(context.Background(), env.RepoOwner, env.RepoName, nil)
	if err != nil {
		return "", err
	}

	noReleaseTag := len(releaseList) == 0
	if noReleaseTag {
		releaseTag = "v0.0.1"
	} else {
		latestReleaseTag := releaseList[0].GetTagName()
		latestReleaseTagSplit := strings.Split(latestReleaseTag, ".")

		latestReleaseTagMinorVersion, err := strconv.Atoi(latestReleaseTagSplit[1])
		if err != nil {
			return "", err
		}

		newReleaseTagMinorVersion := latestReleaseTagMinorVersion + 1

		releaseTag = fmt.Sprintf("v0.%d.0", newReleaseTagMinorVersion)

	}

	// Create a new tag
	newReleaseTag, _, err := client.Git.CreateTag(context.Background(), env.RepoOwner, env.RepoName, &github.Tag{
		Tag: &releaseTag,
	})
	if err != nil {
		fmt.Printf("error creating tag: %v", err)
		return "", err
	}

	return *newReleaseTag.Tag, nil
}

// Create a new release
func CreateNewRelease(client *github.Client, env initializers.Env, newReleaseTag string) (*github.RepositoryRelease, error) {
	name := fmt.Sprintf("%s %s", releaseName, newReleaseTag)

	newRelease, _, err := client.Repositories.CreateRelease(context.Background(), env.RepoOwner, env.RepoName, &github.RepositoryRelease{
		TagName: &newReleaseTag,
		Name:    &name,
	})
	if err != nil {
		return nil, err
	}

	return newRelease, nil
}

// Create a new comment in PR with commit message
func CreateNewComment(client *github.Client, env initializers.Env, pr *github.PullRequest, commitMessage string) error {
	_, _, err := client.Issues.CreateComment(context.Background(), env.RepoOwner, env.RepoName, *pr.Number, &github.IssueComment{
		Body: &commitMessage,
	})
	if err != nil {
		return err
	}

	return nil
}

// Check if PR has label "createrelease:pending"
func HasPendingLabel(pr *github.PullRequest) bool {
	for _, label := range pr.Labels {
		if *label.Name == prLabelPending {
			return true
		}
	}
	return false
}
