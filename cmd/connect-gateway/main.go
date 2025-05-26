// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/atomix/dazl"
	_ "github.com/atomix/dazl/zap"
	"github.com/sirupsen/logrus"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/auth"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/server"
	orchlibraryauth "github.com/open-edge-platform/orch-library/go/pkg/auth"
)

var (
	log = dazl.GetPackageLogger()
)

func main() {
	var gatewayAddress, logLevel, opaAddress, oidcIssuerURL, externalHost, tunnelAuthMode string
	var gatewayPort, opaPort int
	var enableAuth, enableMetrics, oidcInsecureSkipVerify bool
	flag.StringVar(&gatewayAddress, "address", "0.0.0.0", "Address to listen on for edge connection gateway")
	flag.IntVar(&gatewayPort, "port", 8080, "Port to listen on for edge connection gateway")
	flag.BoolVar(&enableAuth, "enable-auth", false, "Enable OIDC authentication")
	flag.BoolVar(&enableMetrics, "enable-metrics", false, "Enable metrics")
	flag.StringVar(&logLevel, "log-level", "info", "Log levels: info, debug, trace, warn")
	flag.StringVar(&oidcIssuerURL, "oidc-issuer-url", "", "OIDC Issuer URL")
	flag.BoolVar(&oidcInsecureSkipVerify, "oidc-insecure-skip-verify", false, "OIDC Insecure Skip Verify")
	flag.StringVar(&externalHost, "external-host", "", "External host for the gateway")

	flag.StringVar(&opaAddress, "opa-address", "http://localhost", "Address to opa")
	flag.IntVar(&opaPort, "opa-port", 8181, "Port to opa")
	flag.StringVar(&tunnelAuthMode, "tunnel-auth-mode", "token", "Specify the authentication mode for tunnel connections: 'token' or 'jwt'")
	flag.Parse()

	setLogLevel(logLevel)
	log.Infof("Agent authentication mode for tunnel connections %s", tunnelAuthMode)
	var tunnelAuth func(req *http.Request) (clientKey string, authed bool, err error)
	switch tunnelAuthMode {
	case "token":
		tokenManager, err := auth.NewTokenManager()
		if err != nil {
			log.Fatal(err)
		}
		secretTokenAuth := auth.SecretTokenAuthorizer{TokenManager: tokenManager}
		tunnelAuth = secretTokenAuth.Authorizer
	case "jwt":
		jwtAuth := auth.JwtTokenAuthorizer{JwtAuth: &orchlibraryauth.JwtAuthenticator{}}
		tunnelAuth = jwtAuth.Authorizer
	}

	listenAddr := fmt.Sprintf("%s:%d", gatewayAddress, gatewayPort)
	// TODO: make the # of hours configurable via helm chart. Will use minutes so we can trigger it quick if needed
	// hours seems too much
	clientCleanupTicker := time.NewTicker(480 * time.Minute)
	defer clientCleanupTicker.Stop()

	connectionProbeTicker := time.NewTicker(1 * time.Minute)
	defer connectionProbeTicker.Stop()

	server, err := server.NewServer(
		server.WithListenAddr(listenAddr),
		server.WithAuth(enableAuth, opaAddress, opaPort),
		server.WithAuthorizer(tunnelAuth, enableMetrics),
		server.WithExternalHost(externalHost),
		server.WithOIDCIssuerURL(oidcIssuerURL),
		server.WithOIDCInsecureSkipVerify(oidcInsecureSkipVerify),
		server.WithCleanupTicker(clientCleanupTicker),
		server.WithConnectionProbeTicker(connectionProbeTicker),
	)
	if err != nil {
		log.Fatalf("Failed to create gateway server: %v", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Create an error channel
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runServer(ctx, server, errChan)

	// Wait for either an error or an OS signal
	select {
	case err := <-errChan:
		// Handle the error
		log.Errorf("Error encountered: %s", err)
	case sig := <-c:
		// Handle the signal
		log.Infof("Got %s signal. Aborting...", sig)
	}
}

func setLogLevel(logLevel string) {
	var level dazl.Level
	switch logLevel {
	case "debug":
		level = dazl.DebugLevel
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		level = dazl.InfoLevel
	case "warn":
		level = dazl.WarnLevel
	default:
		level = dazl.InfoLevel
		log.Warnf("Unknown log level '%s', defaulting to 'info'", logLevel)
	}
	dazl.GetRootLogger().SetLevel(level)
}

func runServer(ctx context.Context, server *server.Server, errChan chan error) {
	select {
	case <-ctx.Done():
		// Handle the context being canceled
		log.Info("Context canceled")
	default:
		// Catch any error from Run and send it to the error channel
		if err := server.Run(); err != nil {
			errChan <- err
		}
	}
}
