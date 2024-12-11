// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package oker

import (
	"strconv"
	"time"

	kit "github.com/go-kit/kit/metrics"
	"go.uber.org/zap"
)

type telemetry struct {
	counter  kit.Counter
	duration kit.Histogram
	logger   *zap.Logger
}

func (t *telemetry) OnOkEvent(e OkEvent) {
	outcome := "failure"
	if e.Err == nil {
		outcome = "success"
	}

	fields := []zap.Field{
		zap.String("fetched_at", e.At.Format(time.RFC3339)),
		zap.Duration("duration", e.Duration),
		zap.Int("status_code", e.StatusCode),
		zap.String("outcome", outcome),
		zap.Error(e.Err),
	}

	if e.Err == nil {
		t.logger.Info("oking request", fields...)
	} else {
		t.logger.Error("oking request", fields...)
	}

	labels := []string{
		"outcome", outcome,
		"status_code", strconv.Itoa(e.StatusCode),
		"partnerid", e.PartnerID,
	}

	t.counter.With(labels...).Add(1)
	t.duration.With(labels...).Observe(float64(e.Duration.Milliseconds()))
}
