// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone/touchkit"
	"go.uber.org/fx"
)

type metricType int

const (
	COUNTER   metricType = 1
	GAUGE     metricType = 2
	HISTOGRAM metricType = 3
)

type metricDefinition struct {
	Type    metricType
	Name    string // the metric name (prometheus.CounterOpts.Name, etc)
	Help    string // the metric help (prometheus.CounterOpts.Help, etc)
	Labels  string // a comma separated list of labels that are whitespace trimmed
	Buckets string // a comma separated list of labels that are whitespace trimmed
}

var fxMetrics = []metricDefinition{
	{
		Type:   COUNTER,
		Name:   "oking_request_count",
		Help:   "The number of times the oking request has been attempted.",
		Labels: "outcome, partnerid, status_code",
	},

	{
		Type:    HISTOGRAM,
		Name:    "oking_call_duration",
		Help:    "The duration of call.",
		Labels:  "outcome, partnerid, status_code",
		Buckets: "10, 100, 1000, 5000, 10000, 100000, 500000, 1000000, 2000000",
	},
}

func Provide() fx.Option {
	var opts []fx.Option // nolint: prealloc

	for _, m := range fxMetrics {
		labels := strings.Split(m.Labels, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}

		var opt fx.Option

		switch m.Type {
		case COUNTER:
			opt = touchkit.Counter(
				prometheus.CounterOpts{
					Name: m.Name,
					Help: m.Help,
				},
				labels...)
		case GAUGE:
			opt = touchkit.Gauge(
				prometheus.GaugeOpts{
					Name: m.Name,
					Help: m.Help,
				},
				labels...)
		case HISTOGRAM:
			buckets := strings.Split(m.Buckets, ",")
			bucketLimits := make([]float64, len(buckets))
			for i := range buckets {
				bucketLimit, err := strconv.ParseFloat(strings.TrimSpace(buckets[i]), 64)
				if err != nil {
					panic(fmt.Sprintf("bucket has non-numeric value '%s'", buckets[i]))
				}
				bucketLimits[i] = bucketLimit
			}
			opt = touchkit.Histogram(
				prometheus.HistogramOpts{
					Name:    m.Name,
					Help:    m.Help,
					Buckets: bucketLimits,
				},
				labels...)
		default:
			panic(fmt.Sprintf("unknown metric type %d for '%s'", m.Type, m.Name))
		}

		if opt == nil {
			panic(fmt.Sprintf("failed to create metric '%s'", m.Name))
		}

		opts = append(opts, opt)
	}

	return fx.Options(opts...)
}
