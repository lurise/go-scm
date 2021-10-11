package gitee

import (
	"context"
	"fmt"

	"github.com/drone/go-scm/scm"
)

type releaseService struct {
	client *wrapper
}

type release struct {
	ID          int    `json:"id"`
	Title       string `json:"name"`
	Description string `json:"description"`
	Tag         string `json:"tag_name"`
	Assets      []struct {
		BrowerDownloadUrl string `json:"browser_download_url"`
	}
	TargetCommitish string `json:"target_commitish"`
	Prerelease      bool   `json:"prerelease"`
}

type releaseInput struct {
	Title       string `json:"name"`
	Description string `json:"description"`
	Tag         string `json:"tag_name"`
}

func (s *releaseService) Find(ctx context.Context, repo string, id int) (*scm.Release, *scm.Response, error) {
	url := fmt.Sprintf("api/v5/repos/%s/releases/%s", repo, id)
	out := new(release)
	res, err := s.client.do(ctx, "GET", url, nil, out)
	return convertRelease(out), res, err
}

func (s *releaseService) FindByTag(ctx context.Context, repo string, tag string) (*scm.Release, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/releases/tags/%s", encode(repo), tag)
	out := new(release)
	res, err := s.client.do(ctx, "GET", path, nil, out)
	return convertRelease(out), res, err
}

func (s *releaseService) List(ctx context.Context, repo string, opts scm.ReleaseListOptions) ([]*scm.Release, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/releases", encode(repo))
	out := []*release{}
	res, err := s.client.do(ctx, "GET", path, nil, &out)
	return convertReleaseList(out), res, err
}

func (s *releaseService) Create(ctx context.Context, repo string, input *scm.ReleaseInput) (*scm.Release, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/releases", encode(repo))
	in := &releaseInput{
		Title:       input.Title,
		Description: input.Description,
		Tag:         input.Tag,
	}
	out := new(release)
	res, err := s.client.do(ctx, "POST", path, in, out)
	return convertRelease(out), res, err
}

func (s *releaseService) Delete(ctx context.Context, repo string, id int) (*scm.Response, error) {
	return nil, scm.ErrNotSupported
}

func (s *releaseService) DeleteByTag(ctx context.Context, repo string, tag string) (*scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/releases/%s", encode(repo), tag)
	return s.client.do(ctx, "DELETE", path, nil, nil)
}

func (s *releaseService) Update(ctx context.Context, repo string, id int, input *scm.ReleaseInput) (*scm.Release, *scm.Response, error) {
	// this could be implemented by List and filter but would be to expensive
	panic("gitlab only allows to update a release by tag")
}

func (s *releaseService) UpdateByTag(ctx context.Context, repo string, tag string, input *scm.ReleaseInput) (*scm.Release, *scm.Response, error) {
	path := fmt.Sprintf("api/v5/repos/%s/releases/%s", encode(repo), tag)
	in := &releaseInput{}
	if input.Title != "" {
		in.Title = input.Title
	}
	if input.Description != "" {
		in.Description = input.Description
	}
	if input.Tag != "" {
		in.Tag = input.Tag
	}
	out := new(release)
	res, err := s.client.do(ctx, "PUT", path, in, out)
	return convertRelease(out), res, err
}

func convertReleaseList(from []*release) []*scm.Release {
	var to []*scm.Release
	for _, m := range from {
		to = append(to, convertRelease(m))
	}
	return to
}

func convertRelease(from *release) *scm.Release {
	return &scm.Release{
		ID:          from.ID,
		Title:       from.Title,
		Description: from.Description,
		Link:        from.Assets[0].BrowerDownloadUrl,
		Tag:         from.Tag,
		Commitish:   from.TargetCommitish,
		Draft:       false, // not supported by gitee
		Prerelease:  from.Prerelease,
	}
}
