package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
			newReleaseTag, err := CreateNewLatestReleaseTag(client, env, *event.PullRequest.Head.SHA)
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

	event, err := ParsePullRequestEvent(env.GithubEvent)
	if err != nil {
		fmt.Printf("error parsing event: %v", err)
		os.Exit(1)
	}

	fmt.Printf("repo owner: %v\n", client.Reactions)

	fmt.Printf("event: %v\n", event)

	err = PREvent(client, env, event)
	if err != nil {
		fmt.Printf("error handling pr event: %v", err)
		os.Exit(1)
	}
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
func CreateNewLatestReleaseTag(client *github.Client, env initializers.Env, lastCommitSHA string) (string, error) {
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

		latestReleaseTagMajorVersion, err := strconv.Atoi(latestReleaseTagSplit[0])
		if err != nil {
			return "", err
		}
		latestReleaseTagMinorVersion, err := strconv.Atoi(latestReleaseTagSplit[1])
		if err != nil {
			return "", err
		}
		latestReleaseTagPatchVersion, err := strconv.Atoi(latestReleaseTagSplit[2])
		if err != nil {
			return "", err
		}

		if latestReleaseTagPatchVersion == 9 {
			latestReleaseTagMinorVersion = latestReleaseTagMinorVersion + 1
			latestReleaseTagPatchVersion = 0
		} else {
			latestReleaseTagPatchVersion = latestReleaseTagPatchVersion + 1
		}

		if latestReleaseTagMinorVersion == 9 {
			latestReleaseTagMajorVersion = latestReleaseTagMajorVersion + 1
			latestReleaseTagMinorVersion = 0
		} else {
			latestReleaseTagMinorVersion = latestReleaseTagMinorVersion + 1
		}

		if latestReleaseTagMinorVersion == 9 && latestReleaseTagPatchVersion == 9 {
			latestReleaseTagMajorVersion = latestReleaseTagMajorVersion + 1
			latestReleaseTagMinorVersion = 0
			latestReleaseTagPatchVersion = 0
		}

		releaseTag = fmt.Sprintf("v%d.%d.%d", latestReleaseTagMajorVersion, latestReleaseTagMinorVersion, latestReleaseTagPatchVersion)
	}

	// Create a new tag
	now := time.Now()
	newReleaseTag, _, err := client.Git.CreateTag(context.Background(), env.RepoOwner, env.RepoName, &github.Tag{
		Tag:     &releaseTag,
		Message: &releaseTag,
		Object: &github.GitObject{
			Type: github.String("commit"),
			SHA:  github.String(lastCommitSHA),
		},
		Tagger: &github.CommitAuthor{
			Name:  github.String("Create Release Action"),
			Email: github.String("githubaction@github.com"),
			Date:  &now,
		},
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

func ParsePullRequestEvent(pullRequestEvent string) (*github.PullRequestEvent, error) {
	// Read the event payload from the env vars
	payloadEnv := pullRequestEvent
	if payloadEnv == "" {
		return nil, fmt.Errorf("no payload found for event %s", pullRequestEvent)
	}

	// Parse the event payload
	prEvent := &github.PullRequestEvent{}
	err := json.Unmarshal([]byte(payloadEnv), prEvent)
	if err != nil {
		return nil, err
	}

	return prEvent, nil
}
