// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bitbucket

import (
	"context"
	"fmt"
	"time"

	"github.com/drone/go-scm/scm"
)

type repository struct {
	UUID       string    `json:"uuid"`
	SCM        string    `json:"scm"`
	FullName   string    `json:"full_name"`
	IsPrivate  bool      `json:"is_private"`
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`
	Mainbranch struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"mainbranch"`
}

type perms struct {
	Values []struct {
		Permissions string `json:"permission"`
	} `json:"values"`
}

type hook struct {
	ID     int      `json:"id"`
	Name   string   `json:"name"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
	Config struct {
		URL         string `json:"url"`
		Secret      string `json:"secret"`
		ContentType string `json:"content_type"`
	} `json:"config"`
}

type repositoryService struct {
	client *wrapper
}

// Find returns the repository by name.
func (s *repositoryService) Find(ctx context.Context, repo string) (*scm.Repository, *scm.Response, error) {
	path := fmt.Sprintf("2.0/repositories/%s", repo)
	out := new(repository)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertRepository(out), res, err
}

// FindHook returns a repository hook.
func (s *repositoryService) FindHook(ctx context.Context, repo string, id int) (*scm.Hook, *scm.Response, error) {
	// path := fmt.Sprintf("repos/%s/hooks/%d", repo, id)
	// out := new(hook)
	// res, err := s.client.do(ctx, "GET", path, nil, out)
	// return convertHook(out), res, err
	return nil, nil, scm.ErrNotSupported
}

// FindPerms returns the repository permissions.
func (s *repositoryService) FindPerms(ctx context.Context, repo string) (*scm.Perm, *scm.Response, error) {
	path := fmt.Sprintf("2.0/user/permissions/repositories?q=repository.full_name=%q", repo)
	out := new(perms)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertPerms(out), res, err
}

// List returns the user repository list.
func (s *repositoryService) List(ctx context.Context, opts scm.ListOptions) ([]*scm.Repository, *scm.Response, error) {
	// path := fmt.Sprintf("user/repos?%s", encodeListOptions(opts))
	// out := []*repository{}
	// res, err := s.client.do(ctx, "GET", path, nil, &out)
	// return convertRepositoryList(out), res, err
	return nil, nil, scm.ErrNotSupported
}

// ListHooks returns a list or repository hooks.
func (s *repositoryService) ListHooks(ctx context.Context, repo string, opts scm.ListOptions) ([]*scm.Hook, *scm.Response, error) {
	// path := fmt.Sprintf("repos/%s/hooks?%s", repo, encodeListOptions(opts))
	// out := []*hook{}
	// res, err := s.client.do(ctx, "GET", path, nil, &out)
	// return convertHookList(out), res, err
	return nil, nil, scm.ErrNotSupported
}

// ListStatus returns a list of commit statuses.
func (s *repositoryService) ListStatus(ctx context.Context, repo, ref string, opts scm.ListOptions) ([]*scm.Status, *scm.Response, error) {
	// path := fmt.Sprintf("repos/%s/statuses/%s?%s", repo, ref, encodeListOptions(opts))
	// out := []*status{}
	// res, err := s.client.do(ctx, "GET", path, nil, &out)
	// return convertStatusList(out), res, err
	return nil, nil, scm.ErrNotSupported
}

// CreateHook creates a new repository webhook.
func (s *repositoryService) CreateHook(ctx context.Context, repo string, input *scm.HookInput) (*scm.Hook, *scm.Response, error) {
	// path := fmt.Sprintf("repos/%s/hooks", repo)
	// in := new(hook)
	// in.Active = true
	// in.Name = "web"
	// in.Config.Secret = input.Secret
	// in.Config.ContentType = "json"
	// in.Config.URL = input.Target
	// in.Events = append(
	// 	input.NativeEvents,
	// 	convertHookEvents(input.Events)...,
	// )
	// out := new(hook)
	// res, err := s.client.do(ctx, "POST", path, in, out)
	// return convertHook(out), res, err
	return nil, nil, scm.ErrNotSupported
}

// CreateStatus creates a new commit status.
func (s *repositoryService) CreateStatus(ctx context.Context, repo, ref string, input *scm.StatusInput) (*scm.Status, *scm.Response, error) {
	// path := fmt.Sprintf("repos/%s/statuses/%s", repo, ref)
	// in := &status{
	// 	State:       convertFromState(input.State),
	// 	Context:     input.Label,
	// 	Description: input.Desc,
	// 	TargetURL:   input.Target,
	// }
	// out := new(status)
	// res, err := s.client.do(ctx, "POST", path, in, out)
	// return convertStatus(out), res, err
	return nil, nil, scm.ErrNotSupported
}

// DeleteHook deletes a repository webhook.
func (s *repositoryService) DeleteHook(ctx context.Context, repo string, id int) (*scm.Response, error) {
	// path := fmt.Sprintf("repos/%s/hooks/%d", repo, id)
	// return s.client.do(ctx, "DELETE", path, nil, nil)
	return nil, scm.ErrNotSupported
}

// helper function to convert from the gogs repository list to
// the common repository structure.
func convertRepositoryList(from []*repository) []*scm.Repository {
	to := []*scm.Repository{}
	for _, v := range from {
		to = append(to, convertRepository(v))
	}
	return to
}

// helper function to convert from the gogs repository structure
// to the common repository structure.
func convertRepository(from *repository) *scm.Repository {
	namespace, name := scm.Split(from.FullName)
	return &scm.Repository{
		ID:        from.UUID,
		Name:      name,
		Namespace: namespace,
		Link:      fmt.Sprintf("https://bitbucket.org/%s", from.FullName),
		Branch:    from.Mainbranch.Name,
		Private:   from.IsPrivate,
		Clone:     fmt.Sprintf("https://bitbucket.org/%s.git", from.FullName),
		CloneSSH:  fmt.Sprintf("git@bitbucket.org:%s.git", from.FullName),
		Created:   from.CreatedOn,
		Updated:   from.UpdatedOn,
	}
}

func convertPerms(from *perms) *scm.Perm {
	to := new(scm.Perm)
	if len(from.Values) != 1 {
		return to
	}
	switch from.Values[0].Permissions {
	case "admin":
		to.Pull = true
		to.Push = true
		to.Admin = true
	case "write":
		to.Pull = true
		to.Push = true
	default:
		to.Pull = true
	}
	return to
}

func convertHookList(from []*hook) []*scm.Hook {
	to := []*scm.Hook{}
	for _, v := range from {
		to = append(to, convertHook(v))
	}
	return to
}

func convertHook(from *hook) *scm.Hook {
	return &scm.Hook{
		ID:     from.ID,
		Active: from.Active,
		Target: from.Config.URL,
		Events: from.Events,
	}
}

func convertHookEvents(from scm.HookEvents) []string {
	var events []string
	if from.Push {
		events = append(events, "push")
	}
	if from.PullRequest {
		events = append(events, "pull_request")
	}
	if from.PullRequestComment {
		events = append(events, "pull_request_review_comment")
	}
	if from.Issue {
		events = append(events, "issues")
	}
	if from.IssueComment || from.PullRequestComment {
		events = append(events, "issue_comment")
	}
	if from.Branch || from.Tag {
		events = append(events, "create")
		events = append(events, "delete")
	}
	return events
}

type status struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	State       string    `json:"state"`
	TargetURL   string    `json:"target_url"`
	Description string    `json:"description"`
	Context     string    `json:"context"`
}

func convertStatusList(from []*status) []*scm.Status {
	to := []*scm.Status{}
	for _, v := range from {
		to = append(to, convertStatus(v))
	}
	return to
}

func convertStatus(from *status) *scm.Status {
	return &scm.Status{
		State:  convertState(from.State),
		Label:  from.Context,
		Desc:   from.Description,
		Target: from.TargetURL,
	}
}

func convertState(from string) scm.State {
	switch from {
	case "error":
		return scm.StateError
	case "failure":
		return scm.StateFailure
	case "pending":
		return scm.StatePending
	case "success":
		return scm.StateSuccess
	default:
		return scm.StateUnknown
	}
}

func convertFromState(from scm.State) string {
	switch from {
	case scm.StatePending, scm.StateRunning:
		return "pending"
	case scm.StateSuccess:
		return "success"
	case scm.StateFailure:
		return "failure"
	default:
		return "error"
	}
}
