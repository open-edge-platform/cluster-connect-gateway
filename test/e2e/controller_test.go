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

const resourceName = "capd-rke2-test"

var _ = Describe("Create ClusterConnect with ClusterRef", Ordered, func() {
	BeforeEach(func() {
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
		//	cmd := exec.Command("kubectl", "delete", "-f", testDataPath+"test-cluster-controlplane-rke2.yaml")
		//	_, err := utils.Run(cmd)
		//	g.Expect(err).NotTo(HaveOccurred(), "Failed to cleanup test Cluster")
		//}, timeout, interval).Should(Succeed())
	})

	It("should successfully update ControlPlane object and Secret for the bootstrap data", func() {
		// Create Cluster and ControlPlane only now to emulate the real world scenario
		// where the InfraCluster is ready only after ClusterConnect object is ready
		By("creating the test Cluster and ControlPlane")
		cmd := exec.Command("kubectl", "apply", "-f", testDataPath+"test-cluster-controlplane-rke2.yaml")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test Cluster and ControlPlane")

		// Only RKE2ControlPlane object is created and no RKE2Config or Secret for bootstrap data because
		// RKE2ControlPlane manager waits until InfraCluster to be ready
		// before proceeding to create Machines and bootstrap config for the machines.
		By("validationg that ControlPlane object is created as expected")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"rke2controlplane", resourceName,
				"-o", "go-template={{ if .spec.files }}{{ .spec.files }}"+
					"{{ else }}{{ \"\" }}{{ end }}",
				"-n", namespace)
			_, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			//g.Expect(output).To(BeEmpty())
		}, timeout, interval).Should(Succeed())

		By("creating ClusterConnect resource")
		cmd = exec.Command("kubectl", "apply", "-f", testDataPath+"test-cluster-connect.yaml")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterConnect")

		By("validationg that ClusterConnect is created as expected")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"clusterconnect", resourceName,
				"-o", "go-template={{ .status.ready }}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("true"))
		}, timeout, interval).Should(Succeed())

		By("validationg that ControlPlane object is updated as expected")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get",
				"rke2controlplane", resourceName,
				"-o", "go-template={{ if .spec.files }}{{ .spec.files }}"+
					"{{ else }}{{ \"\" }}{{ end }}",
				"-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).NotTo(BeEmpty())
			g.Expect(output).To(ContainSubstring("connect-agent.yaml"))
		}, timeout, interval).Should(Succeed())

		// Now create DockerCluster and DockerMachineTemplate.
		// TODO: Replace Docker with IntelCluster and IntelMachineTemplate.
		By("creating InfraCluster and InfraMachineTemplate so that bootstrap config to be created")
		cmd = exec.Command("kubectl", "apply", "-f", testDataPath+"test-cluster-infra-docker.yaml")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test InfraCluster and InfraMachineTemplate")

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

		// Now, check RKE2Config object is created as expected.
		var bootstrapConfigName string
		By("validationg that BootstrapConfig is created as expected")
		Eventually(func(g Gomega) {
			// Get the name of the bootstrap config
			cmd := exec.Command("kubectl", "get",
				"rke2config", "-l", "cluster.x-k8s.io/cluster-name="+resourceName,
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

			// Check files field is set as expected
			cmd = exec.Command("kubectl", "get",
				"rke2config", bootstrapConfigName,
				"-o", "go-template={{ if .spec.files }}{{ .spec.files }}"+
					"{{ else }}{{ \"\" }}{{ end }}",
				"-n", namespace)
			output, err = utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).ToNot(BeEmpty())
			g.Expect(output).To(ContainSubstring("connect-agent.yaml"))
		}, timeout, interval).Should(Succeed())

		// Also check the contents of bootstrap data Secret is as expected
		By("validationg that Secret for the bootstrap data is created as expected")
		Eventually(func(g Gomega) {
			// Get bootstrap secret name
			cmd := exec.Command("kubectl", "get",
				"rke2config", bootstrapConfigName,
				"-o", "go-template={{ .status.dataSecretName }}",
				"-n", namespace)
			bootstrapSecretName, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(bootstrapSecretName).NotTo(BeEmpty())

			// Retrieve secret data
			cmd = exec.Command("kubectl", "get",
				"secret", bootstrapSecretName,
				"-o", "go-template={{ .data.value }}",
				"-n", namespace)
			dataEncoded, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(dataEncoded).NotTo(BeEmpty())

			// Ensure the secret data has connect-agent pod manifest config
			dataDecoded, err := base64.StdEncoding.DecodeString(dataEncoded)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(dataDecoded).NotTo(BeEmpty())
			g.Expect(dataDecoded).To(ContainSubstring("connect-agent.yaml"))
		}, timeout, interval).Should(Succeed())
	})
})
