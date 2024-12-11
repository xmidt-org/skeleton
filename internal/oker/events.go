// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package oker

import (
	"fmt"
	"strings"
	"time"
)

// OkEvent is the event that is sent about an ok request.
type OkEvent struct {
	// At holds the time when the request was made.
	At time.Time

	// PartnerID is the partner id of the request.
	PartnerID string

	// Duration is the time needed to ok the request.
	Duration time.Duration

	// StatusCode is the resulting http status code.
	StatusCode int

	// Error is the resulting error.
	Err error
}

func (e OkEvent) String() string {
	buf := strings.Builder{}

	buf.WriteString("oker.OkEvent{\n")
	buf.WriteString(fmt.Sprintf("  At:         %s\n", e.At.Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("  Duration:   %s\n", e.Duration.String()))
	buf.WriteString(fmt.Sprintf("  StatusCode: %d\n", e.StatusCode))
	buf.WriteString(fmt.Sprintf("  Err:        %v\n", e.Err))
	buf.WriteString("}")

	return buf.String()
}

// OkEventListener is the interface that must be implemented by types that
// want to receive FetchEvent notifications.
type OkEventListener interface {
	OnOkEvent(OkEvent)
}

// OkEventListenerFunc is a function type that implements OkEventListener.
// It can be used as an adapter for functions that need to implement the
// OkEventListener interface.
type OkEventListenerFunc func(OkEvent)

func (f OkEventListenerFunc) OnEventEvent(e OkEvent) {
	f(e)
}
