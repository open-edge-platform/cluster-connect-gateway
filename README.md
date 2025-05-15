# Cluster Connect Gateway

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/open-edge-platform/cluster-connect-gateway/badge)](https://scorecard.dev/viewer/?uri=github.com/open-edge-platform/cluster-connect-gateway)

## Table of Contents

- [Overview](#overview)
- [Get Started](#get-started)
- [Develop](#develop)
- [Contribute](#contribute)
- [Community and Support](#community-and-support)
- [License](#license)

## Overview

Cluster Connect Gateway helps solve the challenge of accessing Kubernetes API and Services on edge clusters that are behind NAT or firewalls. By establishing a secure tunnel between a gateway in the management cluster and agents on edge clusters, users can seamlessly access these services without exposing the edge clusters to the external network.

Key features include:

- **Secure Tunnel**: Establishes a websocket connection between the management cluster and edge clusters.
- **Agent-Initiated Connection**: The connection is initiated by the Connect Agent on the edge, working seamlessly with edge clusters behind NAT or firewalls.
- **Service Access**: Allows users to access Kubernetes API and Services running on multiple edge clusters through a centralized gateway.
- **OIDC integration**: Supports OIDC integration, ensuring only authenticated users can access Kubernetes APIs and Services on the edges.
- **Cluster API integration**: Seamlessly work with Cluster API.
- **Intel Open Edge Platform integration**: Seamlessly work with Intel® Open Edge Platform.

Read more about Cluster Connect Gateway in the [Edge Cluster Orchestrator Developer Guide][cluster-orch-dev-guide-url] for internals and software architecture.

## Get Started

The recommended way to try out the Cluster Connect Gateway is by using the Edge Orchestrator.
Refer to the [Getting Started Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/get_started_guide/index.html) to get started with the Edge Orchestrator.

## Develop

If you are interested in contributing to the development of Cluster Connect Gateway, you will need an environment where you can use it to create and delete clusters.  

The [cluster-tests](https://github.com/open-edge-platform/cluster-tests) repo provides a lightweight environment for integration testing of Cluster Connect Gateway as well as other Edge Orchestrator components related to cluster management. Clone that repo, change into the cluster-tests directory, and run:

```
make test
```

This command creates a KinD cluster and deploys cert-manager, Cluster API operator, CAPI Provider for Intel, Cluster Manager, and Cluster Connect Gateway. It then creates and deletes a cluster inside a Kubernetes pod. Consult the cluster-tests [README](https://github.com/open-edge-platform/cluster-tests/blob/main/README.md) for details on how to test your code in this environment.

## Contribute

We welcome contributions from the community! To contribute, please open a pull request to have your changes reviewed and merged into the main. To learn how to contribute to the project, see the [contributor's guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html). We encourage you to add appropriate unit tests and e2e tests if your contribution introduces a new feature.

Additionally, ensure the following commands are successful:

```
make test
make lint
make license
```

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator Community](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html). 
For support, start with [Troubleshooting](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html) or contact us. 

## License

Cluster Connect Gateway is licensed under [Apache 2.0 License](LICENSES/Apache-2.0.txt)

Last Updated Date: April 16, 2025

[cluster-orch-dev-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/cluster_orch/index.html
