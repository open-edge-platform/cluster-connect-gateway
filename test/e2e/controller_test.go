// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/base64"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-edge-platform/cluster-connect-gateway/test/utils"
)

const resourceName = "capd-k3s-test"

// Test Suite: ClusterConnect Controller Integration
// This test suite validates the core functionality of the cluster-connect-gateway controller
// by testing the complete lifecycle of ClusterConnect resources and their interaction with
// Cluster API K3s providers (KThreesControlPlane and KThreesConfig).
//
// Test Scenario Overview:
// 1. Create a K3s Cluster and KThreesControlPlane (without infrastructure)
// 2. Create a ClusterConnect resource that references the cluster
// 3. Verify that the controller injects connect-agent configuration into the KThreesControlPlane
// 4. Create infrastructure resources (DockerCluster) to trigger bootstrap config creation
// 5. Verify that bootstrap secrets are created with the injected connect-agent configuration
//
// This test validates:
// - ClusterConnect CRD functionality and controller reconciliation
// - Integration with Cluster API K3s providers (KThreesControlPlane/KThreesConfig)
// - Proper injection of connect-agent static pod manifests into control plane files
// - Bootstrap secret generation containing the connect-agent configuration
var _ = Describe("Create ClusterConnect with ClusterRef", Ordered, func() {
	BeforeEach(func() {
		// Ensure clean test state by verifying that test resources don't already exist
		// This prevents interference between test runs and ensures predictable test outcomes
		By("ensuring the test Cluster and ClusterConnect do not exist")
		cmd := exec.Command("kubectl", "get", "cluster", resourceName, "-n", namespace)
		output, err := utils.Run(cmd)
		Expect(err).To(HaveOccurred())
		Expect(output).To(ContainSubstring("NotFound"))

		cmd = exec.Command("kubectl", "get", "clusterconnect", resourceName)
		output, err = utils.Run(cmd)
		Expect(err).To(HaveOccurred())
		Expect(output).To(ContainSubstring("NotFound"))
	})

	AfterEach(func() {
		// TODO: enable clean up later
		//By("cleanup the test Cluster and ClusterConnect")
		//Eventually(func(g Gomega) {
		//	cmd := exec.Command("kubectl", "delete", "-f", testDataPath+"test-cluster-connect.yaml")
		//	_, err := utils.Run(cmd)
		//	g.Expect(err).NotTo(HaveOccurred(), "Failed to cleanup test ClusterConnect")
		//}, timeout, interval).Should(Succeed())

		//Eventually(func(g Gomega) {
		//	cmd := exec.Command("kubectl", "delete", "-f", testDataPath+"test-cluster-infra-docker.yaml")
		//	_, err := utils.Run(cmd)
		//	g.Expect(err).NotTo(HaveOccurred(), "Failed to cleanup test Cluster")
		//}, timeout, interval).Should(Succeed())

		//Eventually(func(g Gomega) {
		//	cmd := exec.Command("kubectl", "delete", "-f", testDataPath+"test-cluster-controlplane-k3s.yaml")
		//	_, err := utils.Run(cmd)
		//	g.Expect(err).NotTo(HaveOccurred(), "Failed to cleanup test Cluster")
		//}, timeout, interval).Should(Succeed())
	})

	It("should successfully update ControlPlane object and Secret for the bootstrap data", func() {
		// Test Phase 1: Create Cluster and KThreesControlPlane
		// This emulates the real-world scenario where cluster resources are created
		// before the infrastructure is ready, which is common in GitOps workflows
		By("creating the test Cluster and ControlPlane")
		cmd := exec.Command("kubectl", "apply", "-f", testDataPath+"test-cluster-controlplane-k3s.yaml")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test Cluster and ControlPlane")

		// Validate that the KThreesControlPlane object is created but initially empty
		// At this point, no KThreesConfig or bootstrap Secret exists because the
		// KThreesControlPlane controller waits for infrastructure readiness before
		// proceeding to create Machines and bootstrap configurations
		By("validationg that ControlPlane object is created as expected")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"kthreescontrolplane", resourceName,
				"-o", "go-template={{ if .spec.kthreesConfigSpec.files }}{{ .spec.kthreesConfigSpec.files }}"+
					"{{ else }}{{ \"\" }}{{ end }}",
				"-n", namespace)
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			//g.Expect(output).To(BeEmpty()) // Initially no files should be present
		}, timeout, interval).Should(Succeed())

		// Test Phase 2: Create ClusterConnect Resource
		// This triggers the cluster-connect-gateway controller to process the cluster
		// and inject connect-agent configuration into the KThreesControlPlane
		By("creating ClusterConnect resource")
		cmd = exec.Command("kubectl", "apply", "-f", testDataPath+"test-cluster-connect.yaml")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterConnect")

		// Validate that the ClusterConnect resource is properly reconciled
		// The controller should set the status.ready field to true after successful processing
		By("validationg that ClusterConnect is created as expected")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"clusterconnect", resourceName,
				"-o", "go-template={{ .status.ready }}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("true"))
		}, timeout, interval).Should(Succeed())

		// Test Phase 3: Verify Connect-Agent Injection
		// This is the core test - verify that the cluster-connect-gateway controller
		// successfully injected the connect-agent static pod manifest into the
		// KThreesControlPlane spec.kthreesConfigSpec.files field
		By("validationg that ControlPlane object is updated as expected")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"kthreescontrolplane", resourceName,
				"-o", "go-template={{ if .spec.kthreesConfigSpec.files }}{{ .spec.kthreesConfigSpec.files }}"+
					"{{ else }}{{ \"\" }}{{ end }}",
				"-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).NotTo(BeEmpty())                           // Files should now be present
			g.Expect(output).To(ContainSubstring("connect-agent.yaml")) // Should contain connect-agent manifest
		}, timeout, interval).Should(Succeed())

		// Test Phase 4: Create Infrastructure Resources
		// Create DockerCluster and DockerMachineTemplate to make the infrastructure "ready"
		// This triggers the KThreesControlPlane controller to create bootstrap configurations
		// TODO: Replace Docker with IntelCluster and IntelMachineTemplate in future versions
		By("creating InfraCluster and InfraMachineTemplate so that bootstrap config to be created")
		cmd = exec.Command("kubectl", "apply", "-f", testDataPath+"test-cluster-infra-docker.yaml")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test InfraCluster and InfraMachineTemplate")

		// Verify that infrastructure resources are created and ready
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "dockercluster", resourceName, "-n", namespace)
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}, timeout, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "dockermachinetemplate", resourceName, "-n", namespace)
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
		}, timeout, interval).Should(Succeed())

		// Test Phase 5: Verify Bootstrap Configuration Creation
		// Now that infrastructure is ready, the KThreesControlPlane controller should create
		// a KThreesConfig bootstrap configuration that inherits the connect-agent files
		// from the KThreesControlPlane specification
		var bootstrapConfigName string
		By("validationg that BootstrapConfig is created as expected")
		Eventually(func(g Gomega) {
			// Get the name of the bootstrap config created by KThreesControlPlane controller
			cmd := exec.Command("kubectl", "get",
				"kthreesconfig", "-l", "cluster.x-k8s.io/cluster-name="+resourceName,
				"-o", "go-template={{ range .items }}"+
					"{{ if not .metadata.deletionTimestamp }}"+
					"{{ .metadata.name }}"+
					"{{ \"\\n\" }}{{ end }}{{ end }}",
				"-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve bootstrap config")
			configNames := utils.GetNonEmptyLines(output)
			g.Expect(configNames).To(HaveLen(1), "expected 1 bootstrap config created")
			bootstrapConfigName = configNames[0]

			// Verify that the KThreesConfig has inherited the connect-agent files
			// from the parent KThreesControlPlane, proving that the injection worked
			cmd = exec.Command("kubectl", "get",
				"kthreesconfig", bootstrapConfigName,
				"-o", "go-template={{ if .spec.files }}{{ .spec.files }}"+
					"{{ else }}{{ \"\" }}{{ end }}",
				"-n", namespace)
			output, err = utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).ToNot(BeEmpty())                           // Files should be inherited from ControlPlane
			g.Expect(output).To(ContainSubstring("connect-agent.yaml")) // Should contain connect-agent manifest
		}, timeout, interval).Should(Succeed())

		// Test Phase 6: Verify Bootstrap Secret Creation
		// The final validation - ensure that the bootstrap Secret contains the
		// connect-agent configuration that will be used during node initialization
		By("validationg that Secret for the bootstrap data is created as expected")
		Eventually(func(g Gomega) {
			// Get bootstrap secret name from KThreesConfig status
			cmd := exec.Command("kubectl", "get",
				"kthreesconfig", bootstrapConfigName,
				"-o", "go-template={{ .status.dataSecretName }}",
				"-n", namespace)
			bootstrapSecretName, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(bootstrapSecretName).NotTo(BeEmpty())

			// Retrieve and decode the bootstrap secret data
			cmd = exec.Command("kubectl", "get",
				"secret", bootstrapSecretName,
				"-o", "go-template={{ .data.value }}",
				"-n", namespace)
			dataEncoded, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(dataEncoded).NotTo(BeEmpty())

			// Verify that the decoded secret contains the connect-agent configuration
			// This proves end-to-end functionality: ClusterConnect -> KThreesControlPlane -> KThreesConfig -> Secret
			dataDecoded, err := base64.StdEncoding.DecodeString(dataEncoded)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(dataDecoded).NotTo(BeEmpty())
			g.Expect(dataDecoded).To(ContainSubstring("connect-agent.yaml")) // Final verification of connect-agent presence
		}, timeout, interval).Should(Succeed())
	})
})
