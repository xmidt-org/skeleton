// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package apiauth

import "errors"

var (
	// ErrInvalidConfig is returned when the config is invalid.
	ErrInvalidConfig = errors.New("invalid config")
)
