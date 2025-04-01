// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"net/http"
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
	log.Debugf("[%s] REQ OK t=%s %+v", tunnelID, timeout, req)

	client, cfg, err := s.GetClientFromKubeconfig(tunnelID, timeout)
	if err != nil {
		log.Errorf("Error getting client for tunnel %s: %s", tunnelID, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create proxy and set the transport to remotedialer client
	proxy := proxy.NewUpgradeAwareHandler(target, client.Transport, false, false, er)

	upgradeTransport, err := makeUpgradeTransport(cfg, client.Transport)
	if err != nil {
		return
	}
	proxy.UpgradeTransport = upgradeTransport

	req.URL.Scheme = target.Scheme
	req.Host = target.Host
	req.URL.Path = target.Path
	log.Debugf("[%s] REQ DONE: %+v", tunnelID, req)

	proxy.ServeHTTP(rw, req)
	log.Debugf("[%s] RESP RECEIVED: %+v", tunnelID, rw)

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
			log.Debugf("session %s doesn't exist anymore, will proceed to remove client %s", tunnelId, clientName)
			clients.Delete(clientName)
			log.Debug("finished removing unused http client")
		}
		return true
	})
}
