// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package opa

import (
	"fmt"

	"github.com/atomix/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
)

var log = dazl.GetPackageLogger()

type OpaConfig struct {
	OpaAddress string
	OpaPort    int
}

func NewOPAClient(opaConfig OpaConfig) openpolicyagent.ClientWithResponsesInterface {
	opaServerAddr := fmt.Sprintf("%s:%d", opaConfig.OpaAddress, opaConfig.OpaPort)
	log.Infow("OPA is enabled, creating an OPA client", dazl.String("OPA server addr", opaServerAddr))

	opaClient, err := openpolicyagent.NewClientWithResponses(opaServerAddr)
	if err != nil {
		log.Fatalw("OPA client cannot be created", dazl.Error(err))
		return nil
	}
	return opaClient
}
