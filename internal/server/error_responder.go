// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"net/http"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/metrics"
)

type errorResponder struct {
}

func (e *errorResponder) Error(w http.ResponseWriter, req *http.Request, err error) {
	log.Debugf("Error response: %v", err)

	code := http.StatusInternalServerError
	label := fmt.Sprintf("%d", code)
	metrics.ProxiedHttpResponseCounter.WithLabelValues(label).Inc()

	w.WriteHeader(code)          // nolint: errcheck
	w.Write([]byte(err.Error())) // nolint: errcheck
}
