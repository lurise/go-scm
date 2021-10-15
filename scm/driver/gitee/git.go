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

type gitService struct {
	client *wrapper
}

func (s *gitService) CreateBranch(ctx context.Context, repo string, params *scm.CreateBranch) (*scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/branches", encode(repo))
	in := &createBranch{
		BranchName: params.Name,
		Refs:       params.Sha,
	}
	return s.client.do(ctx, "POST", path, in, nil)
}

func (s *gitService) FindBranch(ctx context.Context, repo, name string) (*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/branches/%s", encode(repo), name)
	out := new(branch)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertBranch(out), res, err
}

func (s *gitService) FindCommit(ctx context.Context, repo, ref string) (*scm.Commit, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/commits/%s", encode(repo), scm.TrimRef(ref))
	out := new(commit)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertCommit(out), res, err
}

func (s *gitService) FindTag(ctx context.Context, repo, name string) (*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/tags/%s", encode(repo), name)
	out := new(branch)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertTag(out), res, err
}

func (s *gitService) ListBranches(ctx context.Context, repo string, opts scm.ListOptions) ([]*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/branches?%s", encode(repo), encodeListOptions(opts))
	out := []*branch{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertBranchList(out), res, err
}

func (s *gitService) ListCommits(ctx context.Context, repo string, opts scm.CommitListOptions) ([]*scm.Commit, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/commits?%s", encode(repo), encodeCommitListOptions(opts))
	out := []*commit{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertCommitList(out), res, err
}

func (s *gitService) ListTags(ctx context.Context, repo string, opts scm.ListOptions) ([]*scm.Reference, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/tags?%s", encode(repo), encodeListOptions(opts))
	out := []*branch{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertTagList(out), res, err
}

func (s *gitService) ListChanges(ctx context.Context, repo, ref string, opts scm.ListOptions) ([]*scm.Change, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/commits/%s", encode(repo), ref)
	outTemp := &commit{}
	res, err := s.client.do(ctx, "GET", path, nil, &outTemp)
	out := outTemp.Files
	return convertChangeList(out), res, err
}

func (s *gitService) CompareChanges(ctx context.Context, repo, source, target string, _ scm.ListOptions) ([]*scm.Change, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/compare/%s...%s", encode(repo), source, target)
	out := new(compare)
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertChangeList(out.Files), res, err
}

type branch struct {
	Name   string `json:"name"`
	Commit struct {
		Sha string `json:"sha"`
	}
}

type createBranch struct {
	BranchName string `json:"branch_name"`
	Refs       string `json:"refs"`
}

type commit struct {
	Sha         string      `json:"sha"`
	Url         string      `json:"url"`
	HtmlUrl     string      `json:"html_url"`
	CommentsUrl string      `json:"conmments_url"`
	Commit      giteeCommit `json:"commit"`
	Files       []*change   `json:"files"`
}

type giteeCommit struct {
	Author    Author    `json:"author"`
	Committer Committer `json:"commiter"`
	Message   string    `json:"message"`
	Tree      string    `json:"tree"`
}
type Author struct {
	Name  string    `json:"name"`
	Date  time.Time `json:"date"`
	Email string    `json:"email"`
}

type Committer struct {
	Name     string    `json:"name"`
	Date     time.Time `json:"date"`
	Username string    `json:"username"`
	UserName string    `json:"user_name"`
	Email    string    `json:"email"`
}

type tree struct {
	Sha string `json:"sha"`
	Url string `json:"url"`
}

type compare struct {
	Files []*change `json:"files"`
}

func convertCommitList(from []*commit) []*scm.Commit {
	to := []*scm.Commit{}
	for _, v := range from {
		to = append(to, convertCommit(v))
	}
	return to
}

func convertCommit(from *commit) *scm.Commit {
	return &scm.Commit{
		Message: from.Commit.Message,
		Sha:     from.Sha,
		Author: scm.Signature{
			Login: from.Commit.Author.Name,
			Name:  from.Commit.Author.Name,
			Email: from.Commit.Author.Email,
			Date:  from.Commit.Author.Date,
		},
		Committer: scm.Signature{
			Login: from.Commit.Author.Name,
			Name:  from.Commit.Author.Name,
			Email: from.Commit.Author.Email,
			Date:  from.Commit.Author.Date,
		},
	}
}

func convertBranchList(from []*branch) []*scm.Reference {
	to := []*scm.Reference{}
	for _, v := range from {
		to = append(to, convertBranch(v))
	}
	return to
}

func convertBranch(from *branch) *scm.Reference {
	return &scm.Reference{
		Name: scm.TrimRef(from.Name),
		Path: scm.ExpandRef(from.Name, "refs/heads/"),
		Sha:  from.Commit.Sha,
	}
}

func convertTagList(from []*branch) []*scm.Reference {
	to := []*scm.Reference{}
	for _, v := range from {
		to = append(to, convertTag(v))
	}
	return to
}

func convertTag(from *branch) *scm.Reference {
	return &scm.Reference{
		Name: scm.TrimRef(from.Name),
		Path: scm.ExpandRef(from.Name, "refs/tags/"),
		Sha:  from.Commit.Sha,
	}
}
