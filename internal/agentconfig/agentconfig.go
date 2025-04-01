// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package agentconfig

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"os"
)

const (
	agentTemplateText = `apiVersion: v1
kind: Pod
metadata:
  name: connect-agent
  namespace: kube-system
spec:
  containers:
  - name: connect-agent
    image: "{{.Image}}"
{{- if ne .HttpProxy "" }}
    env:
    - name: HTTP_PROXY
      value: {{.HttpProxy}}
{{- end }}
{{- if ne .HttpsProxy "" }}
    - name: HTTPS_PROXY
      value: {{.HttpsProxy}}
{{- end }}
{{- if ne .NoProxy "" }}
    - name: NO_PROXY
      value: {{.NoProxy}}
{{- end }}
    command: [ "/connect-agent" ]
    args:
    - "--gateway-url={{.GatewayURL}}"
    - "--tunnel-id={{.TunnelID}}"
    - "--auth-token={{.Token}}"
    - "--insecure-skip-verify={{.InsecureSkipVerify}}"
    - "--log-level={{.LogLevel}}"
    - "--token-path={{.TokenPath}}"
    - "--tunnel-auth-mode={{.AgentAuthMode}}"
    securityContext:
{{- if eq .AgentAuthMode "jwt" }}
      runAsUser: 501
      runAsGroup: 500
{{- end }}
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      readOnlyRootFilesystem: true
      seccompProfile:
        type: RuntimeDefault
    resources:
      limits: {}
      requests:
        cpu: 100m
        memory: 128Mi
    volumeMounts:
{{- if eq .TLSMode "system-store" }}		
    - name: server-ca
      mountPath: /etc/secrets/ca/cert
      readOnly: true
{{- end }}
{{- if eq .AgentAuthMode "jwt" }}
    - name: jwt-token
      mountPath: {{.TokenPath}}
      readOnly: true
{{- end }}
  volumes:
{{- if eq .TLSMode "system-store" }}	
  - name: server-ca
    hostPath:
      path: /usr/local/share/ca-certificates/ca.crt
      type: File
{{- end }}
{{- if eq .AgentAuthMode "jwt" }}
  - name: jwt-token
    hostPath:
      path: {{.TokenPath}}
      type: File
{{- end }}`
)

var (
	agentconfig   config
	agentTemplate = template.Must(template.New("agentTemplate").Parse(agentTemplateText))
)

type config struct {
	Image              string
	GatewayURL         string
	GatewayCA          string
	InsecureSkipVerify string
	TunnelID           string
	LogLevel           string
	HttpProxy          string
	HttpsProxy         string
	NoProxy            string
	Token              string
	TLSMode            string
	TokenPath          string
	AgentAuthMode      string
}

// InitAgentConfig initializes the agent configuration by reading environment variables.
// It returns an error if mandatory configurations are not set.
func InitAgentConfig() error {
	agentconfig = config{}

	// Mandatory configs, return error if not set
	agentconfig.Image = os.Getenv("AGENT_IMAGE")
	if agentconfig.Image == "" {
		return fmt.Errorf("AGENT_IMAGE is not set")
	}

	gatewayURL := os.Getenv("GATEWAY_EXTERNAL_URL")
	if gatewayURL == "" {
		return fmt.Errorf("GATEWAY_EXTERNAL_URL is not set")
	}
	parsedURL, err := url.Parse(gatewayURL)
	if err != nil {
		return fmt.Errorf("GATEWAY_EXTERNAL_URL is invalid")
	}
	switch parsedURL.Scheme {
	case "http":
		parsedURL.Scheme = "ws"
	case "https":
		parsedURL.Scheme = "wss"
	case "ws", "wss":
		// Do nothing
	default:
		return fmt.Errorf("GATEWAY_EXTERNAL_URL has unsupported scheme")
	}
	parsedURL.Path = "/connect"
	agentconfig.GatewayURL = parsedURL.String()

	agentconfig.TokenPath = os.Getenv("AGENT_JWT_TOKEN_PATH")
	if agentconfig.TokenPath == "" {
		return fmt.Errorf("AGENT_JWT_TOKEN_PATH is not set")
	}

	// Optional configs, can be empty
	agentconfig.GatewayCA = os.Getenv("GATEWAY_CA")
	agentconfig.InsecureSkipVerify = getEnv("INSECURE_SKIP_VERIFY", "true") // TODO: enable by default
	agentconfig.LogLevel = getEnv("AGENT_LOG_LEVEL", "info")
	agentconfig.HttpProxy = os.Getenv("HTTP_PROXY")
	agentconfig.HttpsProxy = os.Getenv("HTTPS_PROXY")
	agentconfig.NoProxy = os.Getenv("NO_PROXY")
	agentconfig.TLSMode = getEnv("TLS_MODE", "strict")
	agentconfig.AgentAuthMode = getEnv("AGENT_AUTH_MODE", "token")

	return nil
}

// GenerateAgentConfig generates the connect-agent pod manifest in YAML for a given tunnel ID and token.
// It returns the generated manifest as a string and any error encountered during template execution.
func GenerateAgentConfig(tunnelId, token string) (string, error) {
	// Do not modify the original config
	config := agentconfig
	config.TunnelID = tunnelId
	config.Token = token

	buf := new(bytes.Buffer)
	err := agentTemplate.Execute(buf, &config)
	return buf.String(), err
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
