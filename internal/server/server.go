// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	"github.com/atomix/dazl"
	_ "github.com/atomix/dazl/zap"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rancher/remotedialer"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/metrics"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/middleware"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/opa"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/utils/kubeutil"
	orchlibraryauth "github.com/open-edge-platform/orch-library/go/pkg/auth"
)

const (
	defaultTimeout   = "15"
	maxBodySizeLimit = 100 // mega-bytes
)

type Server struct {
	router                 *mux.Router
	remotedialer           *remotedialer.Server
	listenAddr             string
	enableAuth             bool
	enableMetrics          bool
	authorizer             remotedialer.Authorizer
	errorWriter            remotedialer.ErrorWriter
	kubeclient             kubeutil.Kubeclient
	externalHost           string
	oidcIssuerURL          string
	oidcInsecureSkipVerify bool
	tlsInsecureSkipVerify  bool
	opaAddress             string
	opaPort                int
	cleanupTicker          *time.Ticker
	connectionProbeTicker  *time.Ticker
}

type ServerOptions func(*Server)

func WithListenAddr(addr string) ServerOptions {
	return func(s *Server) {
		s.listenAddr = addr
	}
}

func WithAuth(auth bool, opaAddress string, opaPort int) ServerOptions {
	return func(s *Server) {
		s.enableAuth = auth
		s.opaAddress = opaAddress
		s.opaPort = opaPort
	}
}

func WithAuthorizer(authorizer remotedialer.Authorizer, enableMetrics bool) ServerOptions {
	finalAuthorizer := authorizer

	if enableMetrics {
		finalAuthorizer = func(req *http.Request) (clientKey string, authed bool, err error) {
			clientKey, authed, err = authorizer(req)
			// TODO: check if the connection request is from the existing client.
			// if so, we should not increment the connection counter.
			if err == nil && authed {
				metrics.ConnectionCounter.WithLabelValues("succeeded").Inc()
			}
			return clientKey, authed, err
		}
	}

	return func(s *Server) {
		s.authorizer = finalAuthorizer
		s.enableMetrics = enableMetrics
	}
}

func WithErrorWriter(errorWriter remotedialer.ErrorWriter) ServerOptions {
	return func(s *Server) {
		s.errorWriter = errorWriter
	}
}

func WithKubeClient(kc kubeutil.Kubeclient) ServerOptions {
	return func(s *Server) {
		s.kubeclient = kc
	}
}

func WithExternalHost(host string) ServerOptions {
	return func(s *Server) {
		s.externalHost = host
	}
}

func WithOIDCIssuerURL(issuerURL string) ServerOptions {
	return func(s *Server) {
		s.oidcIssuerURL = issuerURL
	}
}

func WithCleanupTicker(ticker *time.Ticker) ServerOptions {
	return func(s *Server) {
		s.cleanupTicker = ticker
	}
}

func WithConnectionProbeTicker(ticker *time.Ticker) ServerOptions {
	return func(s *Server) {
		s.connectionProbeTicker = ticker
	}
}

func WithOIDCInsecureSkipVerify(insecureSkipVerify bool) ServerOptions {
	return func(s *Server) {
		s.oidcInsecureSkipVerify = insecureSkipVerify
	}
}

func WithTLSInsecureSkipVerify(insecureSkipVerify bool) ServerOptions {
	return func(s *Server) {
		s.tlsInsecureSkipVerify = insecureSkipVerify
	}
}

// Build creates a new Server with the configured options
func NewServer(options ...ServerOptions) (s *Server, err error) {
	server := &Server{
		listenAddr:  "0.0.0.0:8080",
		enableAuth:  false,
		authorizer:  nil,
		errorWriter: remotedialer.DefaultErrorWriter,
	}

	for _, option := range options {
		option(server)
	}

	// Set certManager to a new in-cluster cert manager if not provided
	if server.kubeclient == nil {
		server.kubeclient, err = kubeutil.NewInClusterClient()
		if err != nil {
			log.Fatalf("Failed to create cert manager: %v", err)
		}
	}

	server.remotedialer = remotedialer.New(server.authorizer, server.errorWriter)
	server.router = mux.NewRouter()
	server.initRouter()

	return server, nil
}

func (s *Server) Run() error {
	if s.cleanupTicker != nil {
		go func() {
			log.Debug("starting routine to clean-up unused http clients")
			for range s.cleanupTicker.C {
				s.cleanupUnusedHttpClients()
			}
		}()
	}

	if s.connectionProbeTicker != nil {
		go func() {
			log.Debug("starting routine to check connection of http clients")
			for range s.connectionProbeTicker.C {
				s.checkHttpClientsConnection()
			}
		}()
	}

	log.Infof("Listening on %s", s.listenAddr)
	if err := http.ListenAndServe(s.listenAddr, s.router); err != nil {
		return err
	}

	return nil
}

// This function doesn't work properly with remote kubeapi, getting 403 error
// TODO: fix this later and use instead of GetClientFromKubeconfig
func (s *Server) GetClient(tunnelID string, timeout string) (*http.Client, error) {
	//l.Lock()
	//defer l.Unlock()

	//key := fmt.Sprintf("%s/%s", tunnelID, timeout)
	//client := clients[key]
	//if client != nil {
	//	return client, nil
	//}

	dialer := s.remotedialer.Dialer(tunnelID)
	transport := &http.Transport{
		DialContext: dialer,
	}

	// Get the CA and client certs for remote server
	caPool, cca, err := s.kubeclient.GetCerts(tunnelID)
	if err != nil {
		log.Errorw("Unable to get certs:", dazl.String("tunnel_id", tunnelID), dazl.Error(err))
		return nil, err
	}

	// Set up TLS configuration based on available certs and security settings
	if caPool != nil || len(cca.Certificate) != 0 {
		tlsConfig := &tls.Config{
			RootCAs:            caPool,
			Certificates:       []tls.Certificate{cca},
			InsecureSkipVerify: s.tlsInsecureSkipVerify,
		}
		transport.TLSClientConfig = tlsConfig
	} else {
		// set up basic TLS config
		tlsConfig := &tls.Config{
			InsecureSkipVerify: s.tlsInsecureSkipVerify,
		}
		transport.TLSClientConfig = tlsConfig
	}

	client := &http.Client{
		Transport: transport,
	}

	if timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err == nil {
			client.Timeout = time.Duration(t) * time.Second
		}
	}

	//clients[key] = client
	return client, nil
}

func (s *Server) initRouter() {
	// healthz endpoint that simply returns "Ok" to indicate that the server is running
	s.router.HandleFunc("/healthz", func(rw http.ResponseWriter, req *http.Request) {
		if _, err := rw.Write([]byte("Ok\n")); err != nil {
			return
		}
	}).Methods("GET")

	// metrics endpoint that exposes the prometheus metrics
	if s.enableMetrics {
		s.router.Handle("/metrics", promhttp.Handler())
	}

	// connect endpoint that handles the tunnel connection requests from agents
	s.router.Handle("/connect", s.remotedialer)

	// Setup a subrouter for the external /kubernetes endpoint
	// This subrouter will handle requests to /kubernetes/{tunnel_id}/* from outside the cluster
	// It should perform JWT authorization if enabled
	k := s.router.Host(s.externalHost).PathPrefix("/kubernetes").Subrouter()
	k.HandleFunc("/{tunnel_id}/{kubernetes_uri:.*}", s.KubeapiHandler)
	k.Use(middleware.SizeLimitMiddleware(maxBodySizeLimit * 1024 * 1024)) // 100 MB
	if s.enableAuth {
		opaClient := opa.NewOPAClient(opa.OpaConfig{OpaAddress: s.opaAddress, OpaPort: s.opaPort})
		jAuthorization := middleware.JwtAuthorization{
			JwtAuthenticator: &orchlibraryauth.JwtAuthenticator{}, OpaClient: opaClient, RbacEnabled: true}
		k.Use(jAuthorization.AuthMiddleware)
	}

	// Setup a subrouter for the internal /kubernetes endpoint
	// This subrouter will handle requests to /kubernetes/{tunnel_id}/* from within the cluster
	// No JWT authorization is required
	k = s.router.PathPrefix("/kubernetes").Subrouter()
	k.HandleFunc("/{tunnel_id}/{kubernetes_uri:.*}", s.KubeapiHandler)

	// Add more endpoints and handlers as needed
}
