// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitee

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/go-scm/scm"
)

type webhookService struct {
	client *wrapper
}

func (s *webhookService) Parse(req *http.Request, fn scm.SecretFunc) (scm.Webhook, error) {
	data, err := ioutil.ReadAll(
		io.LimitReader(req.Body, 10000000),
	)
	if err != nil {
		return nil, err
	}

	var hook scm.Webhook
	switch req.Header.Get("X-Gitee-Event") {
	case "Push Hook", "Tag Push Hook":
		hook, err = parsePushHook(data)
	case "Issue Hook":
		return nil, scm.ErrUnknownEvent
	case "Merge Request Hook":
		hook, err = parsePullRequestHook(data)
	default:
		return nil, scm.ErrUnknownEvent
	}
	if err != nil {
		return nil, err
	}

	// get the gitlab shared token to verify the payload
	// authenticity. If no key is provided, no validation
	// is performed.
	token, err := fn(hook)
	if err != nil {
		return hook, err
	} else if token == "" {
		return hook, nil
	}

	if token != req.Header.Get("X-Gitee-Token") {
		return hook, scm.ErrSignatureInvalid
	}

	return hook, nil
}

func parsePushHook(data []byte) (scm.Webhook, error) {
	src := new(pushHook)
	err := json.Unmarshal(data, src)
	if err != nil {
		return nil, err
	}
	switch {
	case src.HookName == "push_hooks" && src.Before == "0000000000000000000000000000000000000000":
		// TODO we previously considered returning a
		// branch creation hook, however, the push hook
		// returns more metadata (commit details).
		return convertPushHook(src), nil
	//case src.HookName == "push_hooks" && src.After == "0000000000000000000000000000000000000000":
	//	return converBranchHook(src), nil
	case src.HookName == "tag_push_hooks" && src.Before == "0000000000000000000000000000000000000000":
		// TODO we previously considered returning a
		// tag creation hook, however, the push hook
		// returns more metadata (commit details).
		return convertPushHook(src), nil
	case src.HookName == "tag_push_hooks" && src.After == "0000000000000000000000000000000000000000":
		return convertTagHook(src), nil
	default:
		return convertPushHook(src), nil
	}
}

func parsePullRequestHook(data []byte) (scm.Webhook, error) {
	src := new(pullRequestHook)
	err := json.Unmarshal(data, src)
	if err != nil {
		return nil, err
	}
	switch src.Action {
	case "open", "approved", "tested", "merge", "test", "assign":
		// no-op
	default:
		return nil, scm.ErrUnknownEvent
	}
	switch {
	default:
		return convertPullRequestHook(src), nil
	}
}

func convertPushHook(src *pushHook) *scm.PushHook {
	var commits []scm.Commit
	for _, c := range src.Commits {
		commits = append(commits,
			scm.Commit{
				Sha:     c.ID,
				Message: c.Message,
				Link:    c.URL,
				Author: scm.Signature{
					Name:  c.Author.Name,
					Email: c.Author.Email,
				},
				Committer: scm.Signature{
					Name:  c.Author.Name,
					Email: c.Author.Email,
				},
			})
	}
	//namespace, name := scm.Split(src.Project.PathWithNamespace)
	dst := &scm.PushHook{
		Ref:    src.Ref,
		Before: src.Before,
		After:  src.After,
		Repo: scm.Repository{
			ID:        src.Repository.ID,
			Namespace: src.Repository.NameSpace,
			Name:      src.Repository.Name,
			Clone:     src.Repository.GitUrl,
			CloneSSH:  src.Repository.SshUrl,
			Link:      src.Repository.HtmlUrl,
			Branch:    src.Repository.DefaultBranch,
			Private:   src.Repository.Private,
		},
		Commit: scm.Commit{
			Sha:     src.HeadCommit.ID,
			Message: src.HeadCommit.Message, // NOTE this is set below
			Author: scm.Signature{
				Login:  src.HeadCommit.Author.UserName,
				Name:   src.HeadCommit.Author.Name,
				Email:  src.HeadCommit.Author.Email,
				Avatar: "",
			},
			Committer: scm.Signature{
				Login:  src.HeadCommit.Committer.UserName,
				Name:   src.HeadCommit.Committer.Name,
				Email:  src.HeadCommit.Committer.Email,
				Avatar: "",
			},
			Link: src.HeadCommit.Url, // NOTE this is set below
		},
		Sender: scm.User{
			Login:  src.Sender.Login,
			Name:   src.Sender.Name,
			Email:  src.Sender.Email,
			Avatar: src.Sender.AvatarUrl,
		},
		Commits: commits,
	}
	return dst
}

func converBranchHook(src *pushHook) *scm.BranchHook {
	action := scm.ActionCreate
	commit := src.After
	if src.After == "0000000000000000000000000000000000000000" {
		action = scm.ActionDelete
		commit = src.Before
	}
	namespace, name := scm.Split(src.Project.PathWithNamespace)
	return &scm.BranchHook{
		Action: action,
		Ref: scm.Reference{
			Name: scm.TrimRef(src.Ref),
			Sha:  commit,
		},
		Repo: scm.Repository{
			ID:        strconv.Itoa(src.Project.ID),
			Namespace: namespace,
			Name:      name,
			Clone:     src.Project.GitHTTPURL,
			CloneSSH:  src.Project.GitSSHURL,
			Link:      src.Project.WebURL,
			Branch:    src.Project.DefaultBranch,
			Private:   false, // TODO how do we correctly set Private vs Public?
		},
		Sender: scm.User{
			Login:  src.Sender.Login,
			Name:   src.Sender.Name,
			Email:  src.Sender.Email,
			Avatar: src.Sender.AvatarUrl,
		},
	}
}

func convertTagHook(src *pushHook) *scm.TagHook {
	action := scm.ActionCreate
	commit := src.After
	if src.After == "0000000000000000000000000000000000000000" {
		action = scm.ActionDelete
		commit = src.Before
	}
	return &scm.TagHook{
		Action: action,
		Ref: scm.Reference{
			Name: scm.TrimRef(src.Ref),
			Sha:  commit,
		},
		Repo: scm.Repository{
			ID:        src.Repository.ID,
			Namespace: src.Repository.NameSpace,
			Name:      src.Repository.Name,
			Clone:     src.Repository.GitUrl,
			CloneSSH:  src.Repository.SshUrl,
			Link:      src.Repository.HtmlUrl,
			Branch:    src.Repository.DefaultBranch,
			Private:   src.Repository.Private,
		},
		Sender: scm.User{
			Login:  src.Sender.Login,
			Name:   src.Sender.Name,
			Email:  src.Sender.Email,
			Avatar: src.Sender.AvatarUrl,
		},
	}
}

func convertPullRequestHook(src *pullRequestHook) *scm.PullRequestHook {
	action := scm.ActionSync
	switch src.Action {
	case "open":
		action = scm.ActionOpen
	case "tested":
		action = scm.ActionTested
	case "approved":
		action = scm.ActionApproved
	case "merge":
		action = scm.ActionMerge
	case "test":
		action = scm.ActionTest
	}
	return &scm.PullRequestHook{
		Action: action,
		PullRequest: scm.PullRequest{
			Number:  src.PullRequest.ID,
			Title:   src.PullRequest.Title,
			Body:    src.PullRequest.Body,
			Sha:     src.PullRequest.MergeCommitSha,
			Ref:     src.PullRequest.MergeReferenceName,
			Source:  src.PullRequest.Head.Ref,
			Target:  src.PullRequest.Base.Ref,
			Fork:    src.PullRequest.PathWithNamespace,
			Link:    src.PullRequest.HtmlUrl,
			Closed:  src.PullRequest.ClosedAt != nil,
			Merged:  src.PullRequest.Merged,
			Created: src.PullRequest.CreatedAt,
			Updated: src.PullRequest.UpdatedAt,
			Author: scm.User{
				Login:  src.PullRequest.User.Login,
				Name:   src.PullRequest.User.Name,
				Email:  src.PullRequest.User.Email, // TODO how do we get the pull request author email?
				Avatar: src.PullRequest.User.AvatarUrl,
			},
		},
		Repo: scm.Repository{
			ID:        src.Repository.ID,
			Namespace: src.Repository.Namespace,
			Name:      src.Repository.Name,
			Clone:     src.Repository.CloneUrl,
			CloneSSH:  src.Repository.GitSshUrl,
			Link:      src.Repository.HtmlUrl,
			Branch:    src.Repository.DefaultBranch,
			Private:   src.Repository.Private, // TODO how do we correctly set Private vs Public?
		},
		Sender: scm.User{
			Login:  src.Sender.Login,
			Name:   src.Sender.Name,
			Email:  src.Sender.Email,
			Avatar: src.Sender.AvatarUrl,
		},
	}
}

type (
	pushHook struct {
		HookName   string `json:"hook_name"`
		Namespace  string `json:"namespace"`
		Before     string `json:"before"`
		After      string `json:"after"`
		Ref        string `json:"ref"`
		HeadCommit struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				UserName string `json:"user_name"`
				Username string `json:"username"`
				Url      string `json:"url"`
			} `json:"author"`
			Committer
			Url string `json:"url"`
		}
		Sender struct {
			Login     string `json:"login"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			AvatarUrl string `json:"avatar_url"`
		} `json:"sender"`
		Project struct {
			ID                int         `json:"id"`
			Name              string      `json:"name"`
			Description       string      `json:"description"`
			WebURL            string      `json:"web_url"`
			AvatarURL         interface{} `json:"avatar_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			GitHTTPURL        string      `json:"git_http_url"`
			Namespace         string      `json:"namespace"`
			VisibilityLevel   int         `json:"visibility_level"`
			PathWithNamespace string      `json:"path_with_namespace"`
			DefaultBranch     string      `json:"default_branch"`
			CiConfigPath      interface{} `json:"ci_config_path"`
			Homepage          string      `json:"homepage"`
			URL               string      `json:"url"`
			SSHURL            string      `json:"ssh_url"`
			HTTPURL           string      `json:"http_url"`
		} `json:"project"`
		Commits []struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			URL     string `json:"url"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
			Added    []string      `json:"added"`
			Modified []interface{} `json:"modified"`
			Removed  []interface{} `json:"removed"`
		} `json:"commits"`
		TotalCommitsCount int `json:"total_commits_count"`
		Repository        struct {
			ID            string `json:"id"`
			NameSpace     string `json:"namespace"`
			Name          string `json:"name"`
			GitUrl        string `json:"git_url"`
			SshUrl        string `json:"ssh_url"`
			HtmlUrl       string `json:"html_url"`
			DefaultBranch string `json:"default_branch"`
			Private       bool   `json:"private"`
		} `json:"repository"`
	}

	commentHook struct {
		ObjectKind string `json:"object_kind"`
		User       struct {
			Name      string `json:"name"`
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		ProjectID int `json:"project_id"`
		Project   struct {
			ID                int         `json:"id"`
			Name              string      `json:"name"`
			Description       string      `json:"description"`
			WebURL            string      `json:"web_url"`
			AvatarURL         interface{} `json:"avatar_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			GitHTTPURL        string      `json:"git_http_url"`
			Namespace         string      `json:"namespace"`
			VisibilityLevel   int         `json:"visibility_level"`
			PathWithNamespace string      `json:"path_with_namespace"`
			DefaultBranch     string      `json:"default_branch"`
			CiConfigPath      interface{} `json:"ci_config_path"`
			Homepage          string      `json:"homepage"`
			URL               string      `json:"url"`
			SSHURL            string      `json:"ssh_url"`
			HTTPURL           string      `json:"http_url"`
		} `json:"project"`
		ObjectAttributes struct {
			ID           int         `json:"id"`
			Note         string      `json:"note"`
			NoteableType string      `json:"noteable_type"`
			AuthorID     int         `json:"author_id"`
			CreatedAt    string      `json:"created_at"`
			UpdatedAt    string      `json:"updated_at"`
			ProjectID    int         `json:"project_id"`
			Attachment   interface{} `json:"attachment"`
			LineCode     string      `json:"line_code"`
			CommitID     string      `json:"commit_id"`
			NoteableID   int         `json:"noteable_id"`
			StDiff       interface{} `json:"st_diff"`
			System       bool        `json:"system"`
			UpdatedByID  interface{} `json:"updated_by_id"`
			Type         string      `json:"type"`
			Position     struct {
				BaseSha      string      `json:"base_sha"`
				StartSha     string      `json:"start_sha"`
				HeadSha      string      `json:"head_sha"`
				OldPath      string      `json:"old_path"`
				NewPath      string      `json:"new_path"`
				PositionType string      `json:"position_type"`
				OldLine      interface{} `json:"old_line"`
				NewLine      int         `json:"new_line"`
			} `json:"position"`
			OriginalPosition struct {
				BaseSha      string      `json:"base_sha"`
				StartSha     string      `json:"start_sha"`
				HeadSha      string      `json:"head_sha"`
				OldPath      string      `json:"old_path"`
				NewPath      string      `json:"new_path"`
				PositionType string      `json:"position_type"`
				OldLine      interface{} `json:"old_line"`
				NewLine      int         `json:"new_line"`
			} `json:"original_position"`
			ResolvedAt     interface{} `json:"resolved_at"`
			ResolvedByID   interface{} `json:"resolved_by_id"`
			DiscussionID   string      `json:"discussion_id"`
			ChangePosition struct {
				BaseSha      interface{} `json:"base_sha"`
				StartSha     interface{} `json:"start_sha"`
				HeadSha      interface{} `json:"head_sha"`
				OldPath      interface{} `json:"old_path"`
				NewPath      interface{} `json:"new_path"`
				PositionType string      `json:"position_type"`
				OldLine      interface{} `json:"old_line"`
				NewLine      interface{} `json:"new_line"`
			} `json:"change_position"`
			ResolvedByPush interface{} `json:"resolved_by_push"`
			URL            string      `json:"url"`
		} `json:"object_attributes"`
		Repository struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Homepage    string `json:"homepage"`
		} `json:"repository"`
		MergeRequest struct {
			AssigneeID                interface{} `json:"assignee_id"`
			AuthorID                  int         `json:"author_id"`
			CreatedAt                 string      `json:"created_at"`
			DeletedAt                 interface{} `json:"deleted_at"`
			Description               string      `json:"description"`
			HeadPipelineID            interface{} `json:"head_pipeline_id"`
			ID                        int         `json:"id"`
			Iid                       int         `json:"iid"`
			LastEditedAt              interface{} `json:"last_edited_at"`
			LastEditedByID            interface{} `json:"last_edited_by_id"`
			MergeCommitSha            interface{} `json:"merge_commit_sha"`
			MergeError                interface{} `json:"merge_error"`
			MergeParams               interface{} `json:"-"`
			MergeStatus               string      `json:"merge_status"`
			MergeUserID               interface{} `json:"merge_user_id"`
			MergeWhenPipelineSucceeds bool        `json:"merge_when_pipeline_succeeds"`
			MilestoneID               interface{} `json:"milestone_id"`
			SourceBranch              string      `json:"source_branch"`
			SourceProjectID           int         `json:"source_project_id"`
			State                     string      `json:"state"`
			TargetBranch              string      `json:"target_branch"`
			TargetProjectID           int         `json:"target_project_id"`
			TimeEstimate              int         `json:"time_estimate"`
			Title                     string      `json:"title"`
			UpdatedAt                 string      `json:"updated_at"`
			UpdatedByID               interface{} `json:"updated_by_id"`
			URL                       string      `json:"url"`
			Source                    struct {
				ID                int         `json:"id"`
				Name              string      `json:"name"`
				Description       string      `json:"description"`
				WebURL            string      `json:"web_url"`
				AvatarURL         interface{} `json:"avatar_url"`
				GitSSHURL         string      `json:"git_ssh_url"`
				GitHTTPURL        string      `json:"git_http_url"`
				Namespace         string      `json:"namespace"`
				VisibilityLevel   int         `json:"visibility_level"`
				PathWithNamespace string      `json:"path_with_namespace"`
				DefaultBranch     string      `json:"default_branch"`
				CiConfigPath      interface{} `json:"ci_config_path"`
				Homepage          string      `json:"homepage"`
				URL               string      `json:"url"`
				SSHURL            string      `json:"ssh_url"`
				HTTPURL           string      `json:"http_url"`
			} `json:"source"`
			Target struct {
				ID                int         `json:"id"`
				Name              string      `json:"name"`
				Description       string      `json:"description"`
				WebURL            string      `json:"web_url"`
				AvatarURL         interface{} `json:"avatar_url"`
				GitSSHURL         string      `json:"git_ssh_url"`
				GitHTTPURL        string      `json:"git_http_url"`
				Namespace         string      `json:"namespace"`
				VisibilityLevel   int         `json:"visibility_level"`
				PathWithNamespace string      `json:"path_with_namespace"`
				DefaultBranch     string      `json:"default_branch"`
				CiConfigPath      interface{} `json:"ci_config_path"`
				Homepage          string      `json:"homepage"`
				URL               string      `json:"url"`
				SSHURL            string      `json:"ssh_url"`
				HTTPURL           string      `json:"http_url"`
			} `json:"target"`
			LastCommit struct {
				ID        string `json:"id"`
				Message   string `json:"message"`
				Timestamp string `json:"timestamp"`
				URL       string `json:"url"`
				Author    struct {
					Name  string `json:"name"`
					Email string `json:"email"`
				} `json:"author"`
			} `json:"last_commit"`
			WorkInProgress      bool        `json:"work_in_progress"`
			TotalTimeSpent      int         `json:"total_time_spent"`
			HumanTotalTimeSpent interface{} `json:"human_total_time_spent"`
			HumanTimeEstimate   interface{} `json:"human_time_estimate"`
		} `json:"merge_request"`
	}

	tagHook struct {
		ObjectKind   string      `json:"object_kind"`
		EventName    string      `json:"event_name"`
		Before       string      `json:"before"`
		After        string      `json:"after"`
		Ref          string      `json:"ref"`
		CheckoutSha  string      `json:"checkout_sha"`
		Message      interface{} `json:"message"`
		UserID       int         `json:"user_id"`
		UserName     string      `json:"user_name"`
		UserUsername string      `json:"user_username"`
		UserEmail    string      `json:"user_email"`
		UserAvatar   string      `json:"user_avatar"`
		ProjectID    int         `json:"project_id"`
		Project      struct {
			ID                int         `json:"id"`
			Name              string      `json:"name"`
			Description       string      `json:"description"`
			WebURL            string      `json:"web_url"`
			AvatarURL         interface{} `json:"avatar_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			GitHTTPURL        string      `json:"git_http_url"`
			Namespace         string      `json:"namespace"`
			VisibilityLevel   int         `json:"visibility_level"`
			PathWithNamespace string      `json:"path_with_namespace"`
			DefaultBranch     string      `json:"default_branch"`
			CiConfigPath      interface{} `json:"ci_config_path"`
			Homepage          string      `json:"homepage"`
			URL               string      `json:"url"`
			SSHURL            string      `json:"ssh_url"`
			HTTPURL           string      `json:"http_url"`
		} `json:"project"`
		Commits []struct {
			ID        string `json:"id"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
			URL       string `json:"url"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
			Added    []string      `json:"added"`
			Modified []interface{} `json:"modified"`
			Removed  []interface{} `json:"removed"`
		} `json:"commits"`
		TotalCommitsCount int `json:"total_commits_count"`
		Repository        struct {
			Name            string `json:"name"`
			URL             string `json:"url"`
			Description     string `json:"description"`
			Homepage        string `json:"homepage"`
			GitHTTPURL      string `json:"git_http_url"`
			GitSSHURL       string `json:"git_ssh_url"`
			VisibilityLevel int    `json:"visibility_level"`
		} `json:"repository"`
	}

	issueHook struct {
		ObjectKind string `json:"object_kind"`
		User       struct {
			Name      string `json:"name"`
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		Project struct {
			ID                int         `json:"id"`
			Name              string      `json:"name"`
			Description       string      `json:"description"`
			WebURL            string      `json:"web_url"`
			AvatarURL         interface{} `json:"avatar_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			GitHTTPURL        string      `json:"git_http_url"`
			Namespace         string      `json:"namespace"`
			VisibilityLevel   int         `json:"visibility_level"`
			PathWithNamespace string      `json:"path_with_namespace"`
			DefaultBranch     string      `json:"default_branch"`
			CiConfigPath      interface{} `json:"ci_config_path"`
			Homepage          string      `json:"homepage"`
			URL               string      `json:"url"`
			SSHURL            string      `json:"ssh_url"`
			HTTPURL           string      `json:"http_url"`
		} `json:"project"`
		ObjectAttributes struct {
			AssigneeID          interface{}   `json:"assignee_id"`
			AuthorID            int           `json:"author_id"`
			BranchName          interface{}   `json:"branch_name"`
			ClosedAt            interface{}   `json:"closed_at"`
			Confidential        bool          `json:"confidential"`
			CreatedAt           string        `json:"created_at"`
			DeletedAt           interface{}   `json:"deleted_at"`
			Description         string        `json:"description"`
			DueDate             interface{}   `json:"due_date"`
			ID                  int           `json:"id"`
			Iid                 int           `json:"iid"`
			LastEditedAt        string        `json:"last_edited_at"`
			LastEditedByID      int           `json:"last_edited_by_id"`
			MilestoneID         interface{}   `json:"milestone_id"`
			MovedToID           interface{}   `json:"moved_to_id"`
			ProjectID           int           `json:"project_id"`
			RelativePosition    int           `json:"relative_position"`
			State               string        `json:"state"`
			TimeEstimate        int           `json:"time_estimate"`
			Title               string        `json:"title"`
			UpdatedAt           string        `json:"updated_at"`
			UpdatedByID         int           `json:"updated_by_id"`
			URL                 string        `json:"url"`
			TotalTimeSpent      int           `json:"total_time_spent"`
			HumanTotalTimeSpent interface{}   `json:"human_total_time_spent"`
			HumanTimeEstimate   interface{}   `json:"human_time_estimate"`
			AssigneeIds         []interface{} `json:"assignee_ids"`
			Action              string        `json:"action"`
		} `json:"object_attributes"`
		Labels []struct {
			ID          int         `json:"id"`
			Title       string      `json:"title"`
			Color       string      `json:"color"`
			ProjectID   int         `json:"project_id"`
			CreatedAt   string      `json:"created_at"`
			UpdatedAt   string      `json:"updated_at"`
			Template    bool        `json:"template"`
			Description string      `json:"description"`
			Type        string      `json:"type"`
			GroupID     interface{} `json:"group_id"`
		} `json:"labels"`
		Changes struct {
			Labels struct {
				Previous []interface{} `json:"previous"`
				Current  []struct {
					ID          int         `json:"id"`
					Title       string      `json:"title"`
					Color       string      `json:"color"`
					ProjectID   int         `json:"project_id"`
					CreatedAt   string      `json:"created_at"`
					UpdatedAt   string      `json:"updated_at"`
					Template    bool        `json:"template"`
					Description string      `json:"description"`
					Type        string      `json:"type"`
					GroupID     interface{} `json:"group_id"`
				} `json:"current"`
			} `json:"labels"`
		} `json:"changes"`
		Repository struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Homepage    string `json:"homepage"`
		} `json:"repository"`
	}

	pullRequestHook struct {
		Action string `json:"action"`
		Author struct {
			Name      string `json:"name"`
			Username  string `json:"login"`
			AvatarURL string `json:"avatar_url"`
		} `json:"author"`
		PullRequest struct {
			ID                 int        `json:"id"`
			Title              string     `json:"title"`
			Body               string     `json:"body"`
			Sha                string     `json:"sha"`
			Ref                string     `json:""`
			PathWithNamespace  string     `json:"path_with_namespace"`
			HtmlUrl            string     `json:"html_url"`
			Merged             bool       `json:"merged"`
			MergeCommitSha     string     `json:"merge_commit_sha"`
			MergeReferenceName string     `json:"merge_reference_name"`
			ClosedAt           *time.Time `json:"closed_at"`
			CreatedAt          time.Time  `json:"created_at"`
			UpdatedAt          time.Time  `json:"updated_at"`
			Base               struct {
				Ref string `json:"ref"`
			} `json:"base"`
			Head struct {
				Ref string `json:"ref"`
			} `json:"head"`
			User struct {
				Login     string `json:"login"`
				Name      string `json:"name"`
				Email     string `json:"email"`
				AvatarUrl string `json:"avatar_url"`
			}
		} `json:"pull_request"`
		Repository struct {
			ID            string `json:"id"`
			Namespace     string `json:"namespace"`
			Name          string `json:"name"`
			CloneUrl      string `json:"clone_url"`
			GitSshUrl     string `json:"git_ssh_url"`
			HtmlUrl       string `json:"html_url"`
			DefaultBranch string `json:"default_branch"`
			Private       bool   `json:"private"`
		} `json:"repository"`
		Sender struct {
			Login     string `json:"login"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			AvatarUrl string `json:"avatar_url"`
		}
	}
)
