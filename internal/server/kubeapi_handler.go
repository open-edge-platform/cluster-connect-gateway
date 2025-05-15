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
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/metrics"
)

const (
	kubeApiEndpoint = "https://kubernetes.default.svc"
	UpgradeHeader   = "Upgrade"
	SpdyPrefix      = "spdy/"
	Websocket       = "websocket"
)

var (
	log     = dazl.GetPackageLogger()
	clients = sync.Map{}
	er      = &errorResponder{}
)

type Client struct {
	httpClient *http.Client
	restCfg    *rest.Config
}

type LoggingTransport struct {
	Transport http.RoundTripper
}

func (lt *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Debugf("Request: Method=%s, URL=%s, ProtoMajor: %v", req.Method, req.URL.String(), req.ProtoMajor)

	// Perform the actual request
	resp, err := lt.Transport.RoundTrip(req)
	if err != nil {
		log.Errorf("Transport error: %v", err)
		return nil, err
	}

	log.Debugf("Response: Status=%s, Headers=%v", resp.Status, resp.Header)
	return resp, nil
}

func (s *Server) KubeapiHandler(rw http.ResponseWriter, req *http.Request) {
	start := time.Now()
	timeout := req.URL.Query().Get("timeout")
	if timeout == "" {
		timeout = defaultTimeout
	}

	vars := mux.Vars(req)
	tunnelID := vars["tunnel_id"]

	// Parse the target URL
	target, err := url.Parse(fmt.Sprintf("%s/%s", kubeApiEndpoint, vars["kubernetes_uri"]))
	if err != nil {
		log.Errorf("[%s] Failed to parse target URL: %v", tunnelID, err)
		http.Error(rw, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	client, cfg, err := s.GetClientFromKubeconfig(tunnelID, timeout)
	if err != nil {
		log.Errorf("[%s] Failed to get client: %v", tunnelID, err)
		http.Error(rw, "Failed to get client", http.StatusInternalServerError)
		return
	}

	// Set common request fields
	setRequestURL(req, target)

	upgradeHeader := strings.ToLower(req.Header.Get(UpgradeHeader))
	if upgradeHeader == Websocket || upgradeHeader == "" {
		s.handleWebSocketOrHTTP(rw, req, target, client, tunnelID)
	} else if strings.HasPrefix(upgradeHeader, SpdyPrefix) {
		s.handleSPDY(rw, req, target, client, cfg, tunnelID)
	} else {
		log.Warnf("[%s] Unsupported Upgrade header: %s", tunnelID, upgradeHeader)
		http.Error(rw, "Unsupported Upgrade header", http.StatusBadRequest)
		return
	}

	// Record metrics
	recordMetrics(rw, start)
}

func (s *Server) handleWebSocketOrHTTP(rw http.ResponseWriter, req *http.Request, target *url.URL, client *http.Client, tunnelID string) {
	proxyHandler := httputil.NewSingleHostReverseProxy(target)
	proxyHandler.Transport = &LoggingTransport{Transport: client.Transport}

	proxyHandler.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.Host = target.Host
		req.URL.Path = target.Path

		if req.ProtoMajor == 1 {
			if upgrade := req.Header.Get(UpgradeHeader); upgrade != "" {
				log.Debugf("[%s] Preserving Upgrade header: %s", tunnelID, upgrade)
			}
		} else {
			// Remove the Upgrade header for HTTP/2 as HTTP/2 does not support it
			req.Header.Del(UpgradeHeader)
		}
	}

	proxyHandler.ModifyResponse = func(r *http.Response) error {
		if r != nil {
			log.Debugf("[%s] Response received: Status=%d", tunnelID, r.StatusCode)
		}
		return nil
	}

	proxyHandler.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Errorf("[%s] Proxy error: %v", tunnelID, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxyHandler.ServeHTTP(rw, req)
}

func (s *Server) handleSPDY(rw http.ResponseWriter, req *http.Request, target *url.URL, client *http.Client, cfg *rest.Config, tunnelID string) {
	proxyHandler := proxy.NewUpgradeAwareHandler(target, client.Transport, false, false, er)

	upgradeTransport, err := makeUpgradeTransport(cfg, client.Transport)
	if err != nil {
		log.Errorf("[%s] Failed to create upgrade transport: %v", tunnelID, err)
		http.Error(rw, "Failed to create upgrade transport", http.StatusInternalServerError)
		return
	}
	proxyHandler.UpgradeTransport = upgradeTransport

	proxyHandler.ServeHTTP(rw, req)
}

func setRequestURL(req *http.Request, target *url.URL) {
	req.URL.Scheme = target.Scheme
	req.Host = target.Host
	req.URL.Path = target.Path
}

func recordMetrics(rw http.ResponseWriter, start time.Time) {
	code := rw.Header().Get("Status")
	if code == "" {
		code = fmt.Sprintf("%d", http.StatusOK)
	}
	metrics.ProxiedHttpResponseCounter.WithLabelValues(code).Inc()

	duration := time.Since(start).Seconds()
	metrics.RequestLatency.Observe(duration)
}
func (s *Server) GetClientFromKubeconfig(tunnelID, timeout string) (*http.Client, *rest.Config, error) {
	// Check if the client is already cached
	key := fmt.Sprintf("%s/%s", tunnelID, timeout)
	client, ok := clients.Load(key)
	if ok {
		return client.(*Client).httpClient, client.(*Client).restCfg, nil
	}

	start := time.Now() // TODO: refactor
	// Get kubeconfig from the Secret if not
	cfg, err := s.kubeclient.GetKubeconfig(tunnelID)
	duration := time.Since(start).Seconds()
	metrics.KubeconfigRetrievalDuration.Observe(duration)
	if err != nil {
		log.Errorf("Unable to get kubeconfig for %s: %v", tunnelID, err)
		return nil, nil, err
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
		return nil, nil, err
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(bytesCfg)
	if err != nil || restCfg == nil {
		log.Errorf("Unable to create client config %s: %v", tunnelID, err)
		return nil, nil, err
	}

	restCfg.Dial = s.remotedialer.Dialer(tunnelID)
	// Now create a new HTTP client with the rest config
	httpClient, err := rest.HTTPClientFor(restCfg)
	if err != nil {
		log.Errorf("Unable to create HTTP client for %s: %v", tunnelID, err)
		return nil, nil, err
	}

	newClient := &Client{
		httpClient: httpClient,
		restCfg:    restCfg,
	}
	clients.Store(key, newClient)
	return httpClient, restCfg, nil
}

func makeUpgradeTransport(config *rest.Config, rt http.RoundTripper) (proxy.UpgradeRequestRoundTripper, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}

	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}

	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader), nil
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
