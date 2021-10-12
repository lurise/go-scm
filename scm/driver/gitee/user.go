// Copyright 2017 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gitee

import (
	"context"
	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/internal/null"
)

type userService struct {
	client *wrapper
}

func (s *userService) Find(ctx context.Context) (*scm.User, *scm.Response, error) {
	out := new(user)
	res, err := s.client.do(ctx, "GET", "api/v5/user", nil, out)
	return convertUser(out), res, err
}

func (s *userService) FindLogin(ctx context.Context, login string) (*scm.User, *scm.Response, error) {

	return nil, nil, scm.ErrNotSupported

}

func (s *userService) FindEmail(ctx context.Context) (string, *scm.Response, error) {
	user, res, err := s.Find(ctx)
	return user.Email, res, err
}

type user struct {
	Username string      `json:"login"`
	Name     string      `json:"name"`
	Email    null.String `json:"email"`
	Avatar   string      `json:"avatar_url"`
}

func convertUser(from *user) *scm.User {
	return &scm.User{
		Avatar: from.Avatar,
		Email:  from.Email.String,
		Login:  from.Username,
		Name:   from.Name,
	}
}
