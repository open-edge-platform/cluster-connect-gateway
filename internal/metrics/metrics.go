// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ConnectionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_connections_total",
			Help: "Total number of WebSocket connections, partitioned by status.",
		},
		[]string{"status"},
	)
	RequestLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "request_latency_seconds",
		Help:    "Request latency of /kubernetes endpoint in seconds",
		Buckets: prometheus.DefBuckets,
	})
	KubeconfigRetrievalDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "kubernetes",
		Subsystem: "secret",
		Name:      "kubeconfig_retrieval_duration_seconds",
		Help:      "Duration in seconds to retrieve kubeconfig from Kubernetes secret",
		Buckets:   prometheus.DefBuckets,
	})
	ProxiedHttpResponseCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxied_http_response_codes",
			Help: "Count of HTTP response codes for proxied requests",
		},
		[]string{"code"},
	)
)

func init() {
	prometheus.MustRegister(ConnectionCounter)
	prometheus.MustRegister(RequestLatency) //TODO: refactor
	prometheus.MustRegister(KubeconfigRetrievalDuration)
	prometheus.MustRegister(ProxiedHttpResponseCounter)
}
