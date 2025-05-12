// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/atomix/dazl"
	"github.com/gorilla/mux"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/metrics"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubeApiEndpoint = "https://kubernetes.default.svc"
)

var (
	log     = dazl.GetPackageLogger()
	clients = sync.Map{}
)

type LoggingTransport struct {
	Transport http.RoundTripper
}

func (lt *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Debugf("Request: Method=%s, URL=%s, Headers=%+v, ProtoMajor: %v", req.Method, req.URL.String(), req.Header, req.ProtoMajor)

	// Perform the actual request
	resp, err := lt.Transport.RoundTrip(req)
	if err != nil {
		log.Errorf("Transport error: %v", err)
		return nil, err
	}

	log.Infof("Response: Status=%s, Headers=%v", resp.Status, resp.Header)
	return resp, nil
}
func (s *Server) KubeapiHandler(rw http.ResponseWriter, req *http.Request) {
	start := time.Now() // TODO: refactor
	timeout := req.URL.Query().Get("timeout")
	if timeout == "" {
		timeout = defaultTimeout
	}

	vars := mux.Vars(req)
	tunnelID := vars["tunnel_id"]

	// Parse the target URL
	target, err := url.Parse(fmt.Sprintf("%s/%s", kubeApiEndpoint, vars["kubernetes_uri"]))
	if err != nil {
		log.Errorf("Error parsing URL %s: %s", target, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Debugf("[%s] REQ OK t=%s %s", tunnelID, timeout, target.String())

	client, err := s.GetClientFromKubeconfig(tunnelID, timeout)
	if err != nil {
		log.Errorf("Error getting client for tunnel %s: %s", tunnelID, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create proxy and set the transport to remotedialer client
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Wrap the transport with LoggingTransport
	proxy.Transport = &LoggingTransport{
		Transport: client.Transport, // Use the existing transport
	}

	// Log transport details
	if httpTransport, ok := client.Transport.(*http.Transport); ok {
		log.Infof("Transport details: ForceAttemptHTTP2=%v, MaxIdleConns=%d, IdleConnTimeout=%s",
			httpTransport.ForceAttemptHTTP2, httpTransport.MaxIdleConns, httpTransport.IdleConnTimeout)
	} else {
		log.Warn("Transport is not of type *http.Transport")
	}

	// Modify the Director function to change the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Scheme = target.Scheme
		req.Host = target.Host
		req.URL.Path = target.Path

		// Preserve the Upgrade header for HTTP/1.1 requests
		if req.ProtoMajor == 1 {
			if upgrade := req.Header.Get("Upgrade"); upgrade != "" {
				log.Infof("[%s] Preserving Upgrade header: %s", tunnelID, upgrade)
			}
		} else {
			// Remove the Upgrade header for HTTP/2 requests
			req.Header.Del("Upgrade")
		}

		log.Debugf("[%s] REQ DONE: %v", tunnelID, req)
	}

	proxy.ModifyResponse = func(r *http.Response) error {
		if r != nil {
			code := fmt.Sprintf("%d", r.StatusCode)
			metrics.ProxiedHttpResponseCounter.WithLabelValues(code).Inc()
		}
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		metrics.ProxiedHttpResponseCounter.WithLabelValues("502").Inc()
		log.Debugf("[%s] REQ failed: %v", tunnelID, err)
	}

	proxy.ServeHTTP(rw, req)
	log.Debugf("[%s] RESP RECEIVED: %v", tunnelID, rw)
	duration := time.Since(start).Seconds()
	metrics.RequestLatency.Observe(duration)
}

func (s *Server) GetClientFromKubeconfig(tunnelID string, timeout string) (*http.Client, error) {
	// Check if the client is already cached
	key := fmt.Sprintf("%s/%s", tunnelID, timeout)
	client, ok := clients.Load(key)
	if ok {
		return client.(*http.Client), nil
	}

	start := time.Now() // TODO: refactor
	// Get kubeconfig from the Secret if not
	cfg, err := s.kubeclient.GetKubeconfig(tunnelID)
	duration := time.Since(start).Seconds()
	metrics.KubeconfigRetrievalDuration.Observe(duration)
	if err != nil {
		log.Errorf("Unable to get kubeconfig for %s: %v", tunnelID, err)
		return nil, err
	}

	// Set the server URL to the default kubeapi service URL
	// This is needed because rest.HTTPClientFor will not set transport config properly,
	// such as client certificates, when the scheme is http
	for i := range cfg.Clusters {
		cfg.Clusters[i].Server = kubeApiEndpoint
	}

	bytesCfg, err := clientcmd.Write(*cfg)
	if err != nil {
		log.Errorf("Unable to write kubeconfig for %s: %v", tunnelID, err)
		return nil, err
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(bytesCfg)
	if err != nil || restCfg == nil {
		log.Errorf("Unable to create client config %s: %v", tunnelID, err)
		return nil, err
	}

	restCfg.Dial = s.remotedialer.Dialer(tunnelID)

	// Disable HTTP/2 in the transport
	transport, err := rest.TransportFor(restCfg)
	if err != nil {
		log.Errorf("Unable to create transport for %s: %v", tunnelID, err)
		return nil, err
	}
	if httpTransport, ok := transport.(*http.Transport); ok {
		httpTransport.ForceAttemptHTTP2 = false
	}

	// Now create a new HTTP client with the rest config
	httpClient := &http.Client{Transport: transport}

	clients.Store(key, httpClient)
	return httpClient, nil
}

func (s *Server) cleanupUnusedHttpClients() {
	log.Debug("cleaning unused http clients")
	clients.Range(func(key, value any) bool {
		clientName := key.(string)
		// remove the timeout from the key to get the tunnel ID
		tunnelId := strings.Split(clientName, "/")[0]
		if !s.remotedialer.HasSession(tunnelId) {
			log.Infof("session %s doesn't exist anymore, will proceed to remove client %s", tunnelId, clientName)
			clients.Delete(clientName)
			log.Info("finished removing unused http client")
		}
		return true
	})
}
