// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitee

import (
	"context"
	"fmt"
	"time"

	"github.com/drone/go-scm/scm"
)

type pullService struct {
	client *wrapper
}

func (s *pullService) Find(ctx context.Context, repo string, number int) (*scm.PullRequest, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls/%d", encode(repo), number)
	out := new(pr)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertPullRequest(out), res, err
}

func (s *pullService) FindComment(ctx context.Context, repo string, index, id int) (*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls/comments/%d", encode(repo), id)
	out := new(issueComment)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertIssueComment(out), res, err
}

func (s *pullService) List(ctx context.Context, repo string, opts scm.PullRequestListOptions) ([]*scm.PullRequest, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls?%s", encode(repo), encodePullRequestListOptions(opts))
	out := []*pr{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertPullRequestList(out), res, err
}

func (s *pullService) ListChanges(ctx context.Context, repo string, number int, opts scm.ListOptions) ([]*scm.Change, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls/%d/files?%s", encode(repo), number, encodeListOptions(opts))
	out := new(changes)
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertChangeList(out.Changes), res, err
}

func (s *pullService) ListComments(ctx context.Context, repo string, index int, opts scm.ListOptions) ([]*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls/%d/comments", encode(repo), index, encodeListOptions(opts))
	out := []*issueComment{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertIssueCommentList(out), res, err
}

func (s *pullService) ListCommits(ctx context.Context, repo string, number int, opts scm.ListOptions) ([]*scm.Commit, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls/%d/commits", encode(repo), number, nil)
	out := []*commit{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertCommitList(out), res, err
}

func (s *pullService) Create(ctx context.Context, repo string, input *scm.PullRequestInput) (*scm.PullRequest, *scm.Response, error) {
	//in := url.Values{}
	//in.Set("title", input.Title)
	//in.Set("description", input.Body)
	//in.Set("source_branch", input.Source)
	//in.Set("target_branch", input.Target)
	in := prCreate{
		Title: input.Title,
		Head:  input.Source,
		Body:  input.Body,
		Base:  input.Target,
	}
	path := fmt.Sprintf("api/v5/repos/%s/pulls", encode(repo))
	out := new(pr)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertPullRequest(out), res, err
}

func (s *pullService) CreateComment(ctx context.Context, repo string, index int, input *scm.CommentInput) (*scm.Comment, *scm.Response, error) {
	//in := url.Values{}
	//in.Set("body", input.Body)
	in := issueCommentCreate{
		Body: input.Body,
	}
	path := fmt.Sprintf("api/v5/repos/%s/pulls/%d/comments", encode(repo), index)
	out := new(issueComment)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertIssueComment(out), res, err
}

func (s *pullService) DeleteComment(ctx context.Context, repo string, index, id int) (*scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls/%d/comments/%d", encode(repo), index, id)
	res, err := s.client.do(ctx, "DELETE", path, nil, nil)
	return res, err
}

func (s *pullService) Merge(ctx context.Context, repo string, number int) (*scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/pulls/%d/merge", encode(repo), number)
	res, err := s.client.do(ctx, "PUT", path, nil, nil)
	return res, err
}

func (s *pullService) Close(ctx context.Context, repo string, number int) (*scm.Response, error) {
	//path := fmt.Sprintf("api/v5/repos/%s/pulls/%d?state_event=closed", encode(repo), number)
	//res, err := s.client.do(ctx, "PUT", path, nil, nil)
	return nil, scm.ErrNotSupported
}

type pr struct {
	ID                int    `json:"id"`
	URL               string `json:"url"`
	HtmlUrl           string `json:"html_url"`
	DiffUrl           string `json:"diff_url"`
	PatchUrl          string `json:"patch_url"`
	IssueUrl          string `json:"issue_url"`
	CommitsUrl        string `json:"commits_url"`
	ReviewCommentsUrl string `json:"review_comments_url"`
	ReviewCommentUrl  string `json:"review_comment_url"`
	CommentsUrl       string `json:"comments_url"`
	Number            int    `json:"number"`
	State             string `json:"state"`
	Title             string `json:"title"`
	Body              string `json:"body"`
	User              struct {
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarUrl string `json:"avatar_url"`
	} `json:"user"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`

	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
	Closed  time.Time `json:"closed_at"`
	Labels  []struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"labels"`
}

type prCreate struct {
	Title             string `json:"title"`
	Head              string `json:"head"`
	Base              string `json:"base"`
	Body              string `json:"body"`
	MilestoneNumber   int    `json:"milestone_number"`
	Labels            string `json:"labels"`
	Issue             string `json:"issue"`
	Assignees         string `json:"assignees"`
	Testers           string `json:"testers"`
	AssigneesNumber   int    `json:"assignees_number"`
	TestersNumber     int    `json:"testers_number"`
	PruneSourceBranch bool   `json:"prune_source_branch"`
}

type changes struct {
	Changes []*change
}

type change struct {
	FileName  string `json:"filename"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Status    string `json:"status"`
}

func convertPullRequestList(from []*pr) []*scm.PullRequest {
	to := []*scm.PullRequest{}
	for _, v := range from {
		to = append(to, convertPullRequest(v))
	}
	return to
}

func convertPullRequest(from *pr) *scm.PullRequest {
	var labels []scm.Label
	for _, label := range from.Labels {
		labels = append(labels, scm.Label{
			Name: label.Name,
		})
	}
	return &scm.PullRequest{
		Number: from.Number,
		Title:  from.Title,
		Body:   from.Body,
		Sha:    nil,
		Ref:    fmt.Sprintf("refs/merge-requests/%d/head", from.Number),
		Source: from.Head.Ref,
		Target: from.Base.Ref,
		Link:   from.URL,
		Closed: from.State != "opened",
		Merged: from.State == "merged",
		Author: scm.User{
			Name:   from.User.Name,
			Login:  from.User.Login,
			Avatar: from.User.AvatarUrl,
		},
		Created: from.Created,
		Updated: from.Updated,
		Labels:  labels,
	}
}

func convertChangeList(from []*change) []*scm.Change {
	to := []*scm.Change{}
	for _, v := range from {
		to = append(to, convertChange(v))
	}
	return to
}

func convertChange(from *change) *scm.Change {
	to := &scm.Change{
		Path:    from.FileName,
		Added:   from.Additions == 1,
		Deleted: from.Deletions == 1,
		Renamed: from.Status == "modified",
	}
	if to.Path == "" {
		to.Path = from.FileName
	}
	return to
}
