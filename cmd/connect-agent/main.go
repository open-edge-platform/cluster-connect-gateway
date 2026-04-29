// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/agent"
	"github.com/sirupsen/logrus"
)

const (
	defaultAgentHealthListenAddr = "0.0.0.0:8082"
	defaultAgentHealthCheckURL   = "http://127.0.0.1:8082/healthz"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthCheck())
	}

	var gatewayUrl, tunnelId, logLevel, tokenPath, authToken, tunnelAuthMode string
	var insecureSkipVerify bool
	flag.StringVar(&gatewayUrl, "gateway-url", "", "The URL of the gateway")
	// TODO: set this to false by default once CA mount is implemented
	flag.BoolVar(&insecureSkipVerify, "insecure-skip-verify", true, "Skip TLS verification")
	flag.StringVar(&tunnelAuthMode, "tunnel-auth-mode", "token", "Specify the authentication mode for tunnel connections: 'token' or 'jwt'")
	flag.StringVar(&tunnelId, "tunnel-id", "", "The tunnel ID")
	flag.StringVar(&authToken, "auth-token", "", "The authentication token")
	flag.StringVar(&logLevel, "log-level", "info", "Log levels: info, debug, trace")
	flag.StringVar(&tokenPath, "token-path", "./access_token", "path to jwt token")
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

	startHealthServer(ctx, logger)

	agent := &agent.ConnectAgent{
		GatewayUrl:         gatewayUrl,
		InsecureSkipVerify: insecureSkipVerify,
		TunnelId:           tunnelId,
		TokenPath:          tokenPath,
		TunnelAuthMode:     tunnelAuthMode,
		AuthToken:          authToken,
	}

	agent.Run(ctx)
}

func runHealthCheck() int {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(defaultAgentHealthCheckURL)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return 1
	}
	return 0
}

func startHealthServer(ctx context.Context, logger *zap.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write([]byte("Ok\n"))
	})

	srv := &http.Server{
		Addr:              defaultAgentHealthListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Warn("Failed to shut down health server", zap.Error(err))
		}
	}()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Health server failed", zap.Error(err))
		}
	}()
}
