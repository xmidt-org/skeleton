// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

// events
const (
	SigningRequestReceived = "signing_request_received"
)

// errors
const (
	Panic = "panic"
)

func GetUnknownTagIfEmpty(tag string) string {
	if tag == "" {
		return "unknown"
	}
	return tag
}
