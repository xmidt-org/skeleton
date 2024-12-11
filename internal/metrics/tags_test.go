// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUnknownTagIfEmpty(t *testing.T) {
	assert.Equal(t, "unknown", GetUnknownTagIfEmpty(""))
	assert.Equal(t, "some-tag", GetUnknownTagIfEmpty("some-tag"))
}
