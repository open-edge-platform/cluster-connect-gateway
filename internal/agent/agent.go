// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rancher/remotedialer"
	"go.uber.org/zap"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/utils/certutil"
)

type ConnectAgent struct {
	AuthToken          string
	Closed             chan struct{}
	GatewayUrl         string
	InsecureSkipVerify bool
	TunnelId           string
	TokenPath          string
	TunnelAuthMode     string
}

const (
	TunnelIdHeader = "X-Tunnel-Id"        // #nosec G101
	TokenHeader    = "X-API-Tunnel-Token" // #nosec G101
)

func (c *ConnectAgent) Run(ctx context.Context) {
	// Use Go's built-in resolver to resolve DNS names.  We may want to revisit this.
	resolver := &net.Resolver{
		PreferGo: true,
	}

	// Create a new dialer with the resolver and the TLS configuration
	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Minute,
		NetDialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver:  resolver,
		}).DialContext,
		TLSClientConfig: certutil.GetTLSConfigs(c.InsecureSkipVerify),
	}
	headers := http.Header{
		TunnelIdHeader: {c.TunnelId},
	}
	switch c.TunnelAuthMode {
	case "token":
		zap.L().Info("Token auth to gateway enabled")
		headers.Add(TokenHeader, c.AuthToken)
	case "jwt":
		zap.L().Info("Jwt auth to gateway enabled")
		// Read jwt token provided by node-agent
		token, err := os.ReadFile(c.TokenPath)
		if err != nil {
			zap.L().Fatal("Error reading token file", zap.Error(err))
		}

		// Convert the token to a string and trim any whitespace
		jwtToken := string(token)
		jwtToken = strings.TrimSpace(jwtToken)

		headers.Add("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	}

	connAuthorizer := func(proto, address string) bool {
		switch proto {
		case "tcp":
			return true
		case "unix":
			return address == "/var/run/docker.sock"
		case "npipe":
			return address == "//./pipe/docker_engine"
		}
		return false
	}

	onConnect := func(ctx context.Context, _ *remotedialer.Session) error {
		// Do nothing on successful connection now
		// Periodic checks can be added here later
		zap.L().Info("Connected to gateway")
		return nil
	}

	if err := remotedialer.ClientConnect(ctx,
		c.GatewayUrl,
		headers,
		dialer,
		connAuthorizer,
		onConnect); err != nil {
		errMsg := fmt.Errorf("Unable to connect to %s (%s): %v", c.GatewayUrl, c.TunnelId, err)
		zap.L().Error(errMsg.Error())
		panic(errMsg)
	}
}
