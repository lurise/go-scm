// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitee

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"

	"github.com/drone/go-scm/scm"
)

type contentService struct {
	client *wrapper
}

func (s *contentService) Find(ctx context.Context, repo, path, ref string) (*scm.Content, *scm.Response, error) {
	endpoint := fmt.Sprintf("api/v5/repos/%s/contents/%s?ref=%s", encode(repo), encodePath(path), ref)
	out := new(content)
	res, err := s.client.do(ctx, "GET", endpoint, nil, out)
	raw, berr := base64.StdEncoding.DecodeString(out.Content)
	if berr != nil {
		// samples in the gitlab documentation use RawStdEncoding
		// so we fallback if StdEncoding returns an error.
		raw, berr = base64.RawStdEncoding.DecodeString(out.Content)
		if berr != nil {
			return nil, res, err
		}
	}
	return &scm.Content{
		Path:   out.Path,
		Data:   raw,
		Sha:    out.Sha,
		BlobID: out.Sha,
	}, res, err
}

func (s *contentService) Create(ctx context.Context, repo, path string, params *scm.ContentParams) (*scm.Response, error) {
	endpoint := fmt.Sprintf("api/v5/repos/%s/contents/%s", encode(repo), encodePath(path))
	in := &createUpdateContent{
		Branch:         params.Branch,
		Content:        params.Data,
		Message:        params.Message,
		AuthorName:     params.Signature.Name,
		AuthorEmail:    params.Signature.Email,
		CommitterName:  params.Signature.Name,
		CommitterEmail: params.Signature.Email,
	}
	res, err := s.client.do(ctx, "POST", endpoint, in, nil)
	return res, err

}

func (s *contentService) Update(ctx context.Context, repo, path string, params *scm.ContentParams) (*scm.Response, error) {
	endpoint := fmt.Sprintf("api/v5/repos/%s/contents/%s", encode(repo), encodePath(path))
	in := &createUpdateContent{
		Branch:         params.Branch,
		Content:        params.Data,
		Message:        params.Message,
		AuthorName:     params.Signature.Name,
		AuthorEmail:    params.Signature.Email,
		CommitterName:  params.Signature.Name,
		CommitterEmail: params.Signature.Email,
	}
	res, err := s.client.do(ctx, "PUT", endpoint, in, nil)
	return res, err
}

func (s *contentService) Delete(ctx context.Context, repo, path string, params *scm.ContentParams) (*scm.Response, error) {
	endpoint := fmt.Sprintf("api/v5/repos/%s/contents/%s", encode(repo), encodePath(path))
	in := &createUpdateContent{
		Branch:         params.Branch,
		Message:        params.Message,
		AuthorName:     params.Signature.Name,
		AuthorEmail:    params.Signature.Email,
		CommitterName:  params.Signature.Name,
		CommitterEmail: params.Signature.Email,
	}
	res, err := s.client.do(ctx, "DELETE", endpoint, in, nil)
	return res, err
}

func (s *contentService) List(ctx context.Context, repo, path, ref string, opts scm.ListOptions) ([]*scm.ContentInfo, *scm.Response, error) {
	endpoint := fmt.Sprintf("api/v5/repos/%s/git/trees/%s", encode(repo), url.QueryEscape(path), ref, encodeListOptions(opts))
	out := []*object{}
	res, err := s.client.do(ctx, "GET", endpoint, nil, &out)
	return convertContentInfoList(out), res, err
}

type content struct {
	Type        string `json:"type"`
	Encoding    string `json:"encoding"`
	Size        int    `json:"size"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	Sha         string `json:"sha"`
	Url         string `json:"url"`
	HtmlUrl     string `json:"html_url"`
	DownloadUrl string `json:"download_url"`
	Links       string `json:"_links"`
}

type Link struct {
	Self string `json:"self"`
	Html string `json:"html"`
}

type createUpdateContent struct {
	Content        []byte `json:"content"`
	Message        string `json:"message"`
	Branch         string `json:"branch"`
	CommitterName  string `json:"committer[name]"`
	CommitterEmail string `json:"committer[email]"`
	AuthorName     string `json:"author[name]"`
	AuthorEmail    string `json:"author[email]"`
}

type object struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

func convertContentInfoList(from []*object) []*scm.ContentInfo {
	to := []*scm.ContentInfo{}
	for _, v := range from {
		to = append(to, convertContentInfo(v))
	}
	return to
}

func convertContentInfo(from *object) *scm.ContentInfo {
	to := &scm.ContentInfo{Path: from.Path}
	// See the following link for supported file modes:
	// https://godoc.org/gopkg.in/src-d/go-git.v4/plumbing/filemode
	switch mode, _ := strconv.ParseInt(from.Mode, 8, 32); mode {
	case 0100644, 0100664, 0100755:
		to.Kind = scm.ContentKindFile
	case 0040000:
		to.Kind = scm.ContentKindDirectory
	case 0120000:
		to.Kind = scm.ContentKindSymlink
	case 0160000:
		to.Kind = scm.ContentKindGitlink
	default:
		to.Kind = scm.ContentKindUnsupported
	}
	return to
}
