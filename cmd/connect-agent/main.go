// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"go.uber.org/zap"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/agent"
	"github.com/sirupsen/logrus"
)

func main() {
	var gatewayUrl, tunnelId, logLevel, tokenPath, authToken, tunnelAuthMode, staticPodPath string
	var insecureSkipVerify bool
	flag.StringVar(&gatewayUrl, "gateway-url", "", "The URL of the gateway")
	// TODO: set this to false by default once CA mount is implemented
	flag.BoolVar(&insecureSkipVerify, "insecure-skip-verify", true, "Skip TLS verification")
	flag.StringVar(&tunnelAuthMode, "tunnel-auth-mode", "token", "Specify the authentication mode for tunnel connections: 'token' or 'jwt'")
	flag.StringVar(&tunnelId, "tunnel-id", "", "The tunnel ID")
	flag.StringVar(&authToken, "auth-token", "", "The authentication token")
	flag.StringVar(&logLevel, "log-level", "info", "Log levels: info, debug, trace")
	flag.StringVar(&tokenPath, "token-path", "./access_token", "path to jwt token")
	flag.StringVar(&staticPodPath, "static-pod-path", "/var/lib/kubelet/static-pods/connect-agent.yaml", "path to static pod manifests")
	flag.Parse()

	// Set log level for the tunnel data
	if level, err := logrus.ParseLevel(logLevel); err == nil {
		logrus.SetLevel(level)
	} else {
		log.Fatalf("can't initialize logrus logger: %v", err)
	}

	logger, err := zap.NewProduction(zap.Fields(zap.String("tunnel-id", tunnelId)))
	if logLevel == "debug" || logLevel == "trace" {
		logger, err = zap.NewDevelopment(zap.Fields(zap.String("tunnel-id", tunnelId)))
	}
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	zap.ReplaceGlobals(logger)

	// Required parameters
	if gatewayUrl == "" || tunnelId == "" {
		logger.Error("gateway-url and tunnel-id are required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer func() {
		logger.Info("Received interrupt signal, shutting down")
		stop()
	}()

	agent := &agent.ConnectAgent{
		GatewayUrl:         gatewayUrl,
		InsecureSkipVerify: insecureSkipVerify,
		TunnelId:           tunnelId,
		TokenPath:          tokenPath,
		TunnelAuthMode:     tunnelAuthMode,
		AuthToken:          authToken,
		StaticPodPath:      staticPodPath,
	}

	agent.Run(ctx)
}
