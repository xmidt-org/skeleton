// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package oker

func WithConfig(c Config) Option {
	return optionFunc(func(s *Server) error {
		s.config = c
		return nil
	})
}

// AddOkListener adds a listener for oking events.  If the optional cancel
// parameter is provided, it is set to a function that can be used to cancel
// the listener.
func AddOkEventListener(listener OkEventListener, cancel ...*func()) Option {
	return optionFunc(func(s *Server) error {
		cncl := s.okEventListeners.Add(listener)
		if len(cancel) > 0 && cancel[0] != nil {
			*cancel[0] = cncl
		}
		return nil
	})
}
