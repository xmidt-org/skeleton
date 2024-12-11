// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package oker

import (
	"net/http"
	"time"

	"github.com/xmidt-org/eventor"
)

type Config struct {
	Name string
}

type Server struct {
	config           Config
	okEventListeners eventor.Eventor[OkEventListener]
}

type Option interface {
	apply(*Server) error
}

type optionFunc func(*Server) error

func (f optionFunc) apply(s *Server) error {
	return f(s)
}

func New(opts ...Option) (*Server, error) {
	var s Server

	for _, opt := range opts {
		if opt != nil {
			if err := opt.apply(&s); err != nil {
				return nil, err
			}
		}
	}

	return &s, nil
}

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var e OkEvent

	e.At = time.Now()
	e.StatusCode = http.StatusOK
	resp.WriteHeader(http.StatusOK)

	s.okEventListeners.Visit(func(listener OkEventListener) {
		listener.OnOkEvent(e)
	})
}
