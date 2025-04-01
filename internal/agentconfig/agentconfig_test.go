// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package agentconfig

import (
	"os"
	"testing"
)

//nolint:errcheck
func TestGenerateAgentConfig(t *testing.T) {
	expected := `apiVersion: v1
kind: Pod
metadata:
  name: connect-agent
  namespace: kube-system
spec:
  containers:
  - name: connect-agent
    image: "connect-gateway:latest"
    command: [ "/connect-agent" ]
    args:
    - "--gateway-url=wss://connect-gateway.kind.internal/connect"
    - "--tunnel-id=test-tunnel-id"
    - "--auth-token=test-token"
    - "--insecure-skip-verify=true"
    - "--log-level=info"
    - "--token-path=/testpath"
    - "--tunnel-auth-mode=token"
    securityContext:
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
  volumes:`

	os.Setenv("AGENT_IMAGE", "connect-gateway:latest")
	os.Setenv("GATEWAY_EXTERNAL_URL", "https://connect-gateway.kind.internal")
	os.Setenv("AGENT_JWT_TOKEN_PATH", "/testpath")

	// Save original values
	originalHTTPProxy := os.Getenv("HTTP_PROXY")
	originalHTTPSProxy := os.Getenv("HTTPS_PROXY")
	originalNoProxy := os.Getenv("NO_PROXY")

	// Unset the variables
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("NO_PROXY")

	defer func() {
		// Reset the variables to their original values
		os.Setenv("HTTP_PROXY", originalHTTPProxy)
		os.Setenv("HTTPS_PROXY", originalHTTPSProxy)
		os.Setenv("NO_PROXY", originalNoProxy)
	}()

	err := InitAgentConfig()
	if err != nil {
		t.Fatalf("InitAgentConfig() error = %v", err)
	}

	tunnelId := "test-tunnel-id"
	token := "test-token"
	configStr, err := GenerateAgentConfig(tunnelId, token)

	if err != nil {
		t.Fatalf("GenerateAgentConfig() error = %v", err)
	}

	if configStr != expected {
		t.Errorf("expected \n%s\ngot \n%s", expected, configStr)
	}
}
