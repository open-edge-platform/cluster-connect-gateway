// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("Server", func() {
	//var (
	//	testServer *Server
	//	addr       string
	//)

	//BeforeEach(func() {
	//	addr = "127.0.0.1:8123"
	//	cfg = &config.Config{K8sUrl: "https://kubernetes.default.svc", EnableAuth: false}
	//})

	//Describe("New Server", func() {
	//	Context("When a server is created", func() {
	//		It("Should create server successfully", func() {
	//			testServer = NewServer(addr, cfg)
	//			Expect(testServer).ToNot(BeNil())
	//			Expect(testServer.addr).To(Equal(addr))
	//		})
	//	})
	//})
})
