// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/cluster-connect-gateway/test/utils"
)

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	// This test suite validates the basic deployment and health of the cluster-connect-gateway components.
	// It verifies that both the controller and gateway pods are successfully deployed via Helm chart
	// and are running in the expected namespace with correct status.
	Context("Manager", func() {
		// Test: Deployment Validation
		// Purpose: Ensures that both controller and gateway components are properly deployed and healthy
		// This is a smoke test that validates the basic deployment succeeded before running integration tests
		It("should run successfully", func() {
			// Phase 1: Validate Controller Pod Deployment
			// The controller is responsible for watching ClusterConnect CRDs and injecting
			// connect-agent configurations into K3s control plane specifications
			By("validating that the connect-controller pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod using component label selector
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "app.kubernetes.io/component=controller",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve connect-controller pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller"))

				// Validate the pod's status - must be Running for tests to proceed
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect connect-controller pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())

			// Phase 2: Validate Gateway Pod Deployment
			// The gateway provides the HTTP API endpoint for agent registration and communication
			By("validating that the connect-gateway pod is running as expected")
			verifyGatewayUp := func(g Gomega) {
				// Get the name of the gateway pod using component label selector
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "app.kubernetes.io/component=gateway",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("gateway"))

				// Validate the pod's status - must be Running for full functionality
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect connect-gateway pod status")
			}
			Eventually(verifyGatewayUp).Should(Succeed())
		})
	})

	// NOTE: Connect Agent Integration Test (Currently Disabled)
	// The following test was designed to validate end-to-end agent connectivity by:
	// 1. Extracting the agent manifest from ClusterConnect status.agentManifest
	// 2. Deploying the connect-agent pod directly to the test cluster
	// 3. Verifying the agent runs without restarts and can communicate with the gateway
	//
	// This test is commented out because:
	// - It requires a running gateway endpoint for agent registration
	// - The agent needs proper network connectivity to reach the gateway service
	// - The test environment may not have the required networking setup
	//
	// In production, the agent would be deployed automatically during cluster bootstrap
	// via the static pod manifest injected by the controller into K3s control plane nodes.
	//Context("Connect Agent", Ordered, func() {
	//	It("should connect successfully", func() {
	//		By("deploying agent manifest from ClusterConnect")
	//		var agentManifestOutput string
	//		Eventually(func() error {
	//			cmdGetClusterConnect := exec.Command("kubectl", "get", "clusterconnect", "capd-k3s-test", "-o", "jsonpath={.status.agentManifest}")
	//			var err error
	//			agentManifestOutput, err = utils.Run(cmdGetClusterConnect)
	//			return err
	//		}).Should(Succeed(), "Failed to retrieve cluster connect agent pod manifest from ClusterConnect")

	//		cmdDeploy := exec.Command("kubectl", "apply", "-f", "-")
	//		cmdDeploy.Stdin = strings.NewReader(agentManifestOutput)
	//		output, err := utils.Run(cmdDeploy)
	//		Expect(err).NotTo(HaveOccurred(), "Failed to deploy cluster connect agent pod manifest from ClusterConnect")
	//		Expect(output).To(Equal("pod/connect-agent created\n"), "Failed to deploy cluster connect agent pod manifest from ClusterConnect")

	//		By("validating that agent is not restarting")
	//		podName := "connect-agent"
	//		cmd := exec.Command("kubectl", "get", "pod", podName, "-n", "kube-system", "-o", "go-template={{range .status.containerStatuses}}{{if eq .name \""+podName+"\"}}{{.restartCount}}{{end}}{{end}}")
	//		restartCount, err := utils.Run(cmd)
	//		Expect(err).NotTo(HaveOccurred(), "Failed to get cluster connect agent pod restart count")
	//		if restartCount != "0" {
	//			Fail("is restarting")
	//		}
	//	})
	//})
})
