// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitee

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/internal/null"
)

type issueService struct {
	client *wrapper
}

func (s *issueService) Find(ctx context.Context, repo string, number int) (*scm.Issue, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/issues/%d", encode(repo), number)
	out := new(issue)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertIssue(out), res, err
}

func (s *issueService) FindComment(ctx context.Context, repo string, index, id int) (*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/issues/notes/%d", encode(repo), id)
	out := new(issueComment)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertIssueComment(out), res, err
}

func (s *issueService) List(ctx context.Context, repo string, opts scm.IssueListOptions) ([]*scm.Issue, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/issues?%s", encode(repo), encodeIssueListOptions(opts))
	out := []*issue{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertIssueList(out), res, err
}

func (s *issueService) ListComments(ctx context.Context, repo string, index int, opts scm.ListOptions) ([]*scm.Comment, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/issues/%d/notes?%s", encode(repo), index, encodeListOptions(opts))
	out := []*issueComment{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertIssueCommentList(out), res, err
}

func (s *issueService) Create(ctx context.Context, repo string, input *scm.IssueInput) (*scm.Issue, *scm.Response, error) {
	in := url.Values{}
	in.Set("title", input.Title)
	in.Set("description", input.Body)
	path := fmt.Sprintf("api/v5/repos/%s/issues?%s", encode(repo), in.Encode())
	out := new(issue)
	res, err := s.client.do(ctx, "POST", path, nil, out)
	return convertIssue(out), res, err
}

func (s *issueService) CreateComment(ctx context.Context, repo string, number int, input *scm.CommentInput) (*scm.Comment, *scm.Response, error) {
	in := issueCommentInput{
		Body: input.Body,
	}
	path := fmt.Sprintf("api/v5/repos/%s/issues/%d/comments", encode(repo), number)
	out := new(issueComment)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertIssueComment(out), res, err
}

func (s *issueService) DeleteComment(ctx context.Context, repo string, number, id int) (*scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/issues/comments/%d", encode(repo), id)
	return s.client.do(ctx, "DELETE", path, nil, nil)
}

func (s *issueService) Close(ctx context.Context, repo string, number int) (*scm.Response, error) {
	repos := strings.Split(repo, "/")
	path := fmt.Sprintf("api/v5/repos/%s/issues/%d", encode(repos[0]), number)
	in := issueCommentEdit{
		Repo:  repos[1],
		State: "closed",
	}
	res, err := s.client.do(ctx, "PATCH", path, in, nil)
	return res, err
}

func (s *issueService) Lock(ctx context.Context, repo string, number int) (*scm.Response, error) {
	//path := fmt.Sprintf("api/v5/repos/%s/issues/%d?discussion_locked=true", encode(repo), number)
	//res, err := s.client.do(ctx, "PUT", path, nil, nil)
	//gitee好像没有关闭评论的功能及接口
	return nil, nil
}

func (s *issueService) Unlock(ctx context.Context, repo string, number int) (*scm.Response, error) {
	//path := fmt.Sprintf("api/v5/repos/%s/issues/%d?discussion_locked=false", encode(repo), number)
	//res, err := s.client.do(ctx, "PUT", path, nil, nil)
	//gitee好像没有关闭评论的功能及接口
	return nil, nil
}

type issue struct {
	ID       int     `json:"id"`
	Number   int     `json:"number"`
	State    string  `json:"state"`
	Title    string  `json:"title"`
	Body     string  `json:"body"`
	Link     string  `json:"web_url"`
	Locked   bool    `json:"discussion_locked"`
	Labels   []label `json:"labels"`
	Assignee struct {
		Name      string      `json:"name"`
		Login     string      `json:"login"`
		AvatarUrl null.String `json:"avatar_url"`
	} `json:"user"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
}

type label struct {
	Id           int       `json:"id"`
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	RepositoryId int       `json:"repository_id"`
	Url          string    `json:"url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type issueComment struct {
	ID   int `json:"id"`
	User struct {
		Username  string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		Name      string `json:"name"`
	} `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type issueCommentInput struct {
	Body string `json:"body"`
}
type issueCommentEdit struct {
	Repo          string `json:"repo"`
	State         string `json:"state"`
	Title         string `json:"title"`
	Body          string `json:"body"`
	Assignee      string `json:"assignee"`
	Collaborators string `json:"collaborators"`
	MileStone     string `json:"milestone"`
	Labels        string `json:"labels"` //用逗号分开的标签，名称要求长度在2-20之间且非特殊字符。
	Program       string `json:"program"`
	SecurityHole  string `json:"security_hole"`
}

// helper function to convert from the gogs issue list to
// the common issue structure.
func convertIssueList(from []*issue) []*scm.Issue {
	to := []*scm.Issue{}
	for _, v := range from {
		to = append(to, convertIssue(v))
	}
	return to
}

// helper function to convert from the gogs issue structure to
// the common issue structure.
func convertIssue(from *issue) *scm.Issue {
	var labels = make([]string, 0)
	for _, item := range from.Labels {
		labels = append(labels, item.Name)
	}
	return &scm.Issue{
		Number: from.Number,
		Title:  from.Title,
		Body:   from.Body,
		Link:   from.Link,
		Labels: labels,
		Locked: from.Locked,
		Closed: from.State == "closed",
		Author: scm.User{
			Name:   from.Assignee.Name,
			Login:  from.Assignee.Login,
			Avatar: from.Assignee.AvatarUrl.String,
		},
		Created: from.Created,
		Updated: from.Updated,
	}
}

// helper function to convert from the gogs issue comment list
// to the common issue structure.
func convertIssueCommentList(from []*issueComment) []*scm.Comment {
	to := []*scm.Comment{}
	for _, v := range from {
		to = append(to, convertIssueComment(v))
	}
	return to
}

// helper function to convert from the gogs issue comment to
// the common issue comment structure.
func convertIssueComment(from *issueComment) *scm.Comment {
	return &scm.Comment{
		ID:   from.ID,
		Body: from.Body,
		Author: scm.User{
			Name:   from.User.Name,
			Login:  from.User.Username,
			Avatar: from.User.AvatarURL,
		},
		Created: from.CreatedAt,
		Updated: from.UpdatedAt,
	}
}
