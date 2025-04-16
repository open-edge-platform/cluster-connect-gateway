# Welcome to the Cluster Connect Gateway

## Overview

Cluster Connect Gateway helps solve the challenge of accessing Kubernetes API and Services on edge clusters that are behind NAT or firewalls. By establishing a secure tunnel between a gateway in the management cluster and agents on edge clusters, users can seamlessly access these services without exposing the edge clusters to the external network.

Key features include:

- **Secure Tunnel**: Establishes a websocket connection between the management cluster and edge clusters.
- **Agent-Initiated Connection**: The connection is initiated by the Connect Agent on the edge, working seamlessly with edge clusters behind NAT or firewalls.
- **Service Access**: Allows users to access Kubernetes API and Services running on multiple edge clusters through a centralized gateway.
- **OIDC integration**: Supports OIDC integration, ensuring only authenticated users can access Kubernetes APIs and Services on the edges.
- **Cluster API integration**: Seamlessly work with Cluster API.
- **Intel Open Edge Platform integration**: Seamlessly work with Intel® Open Edge Platform.

## Get Started

Install Cluster Connect Gateway operator.

```
helm install -n orch-system --create-namespace deployment/charts/cluster-connect-gateway-crd
helm install -n orch-system --create-namespace deployment/charts/cluster-connect-gateway
```

Create a ClusterConnect for the edge cluster.

```
cat > example.yaml << "EOF"
apiVersion: cluster.edge-orchestrator.intel.com/v1alpha1
kind: ClusterConnect
metadata:
  name: example-cluster-connect
spec:
  serverCertRef:
    apiVersion: v1
    kind: Secret
    name: example-cluster-ca
    namespace: default
  clientCertRef:
    apiVersion: v1
    kind: Secret
    name: example-cluster-cca
    namespace: default

kubectl apply -f example.yaml
```

Get connect-agent pod manifest.

```
kubectl get clusterconnect example-cluster-connect \
    -o go-template="{{ if .status.ready }}{{ .status.agentManifest }}{{ end }}" \
    > connect-agent.yaml
```

Deploy connect-agent to the edge cluster.

```
kubectl apply -f connect-agent.yaml
```

Another way to try out Cluster Connect Gateway is by using the Intel® Open Edge Platform. Refer to the [Documentation](https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/get_started_guide/gsg_content.html) to get started with Intel® Open Edge Platform.

## Develop

If you are interested in contributing to the development of Cluster Connect Gateway, follow these steps to get started:

```
make run-e2e-and-keep
```

This command creates a KinD cluster, deploy cert-manager, Cluster API operator and providers, and build and deploy Cluster Connect Gateway from your local repository.

After making changes, rebuild and deploy the updated code.

```
make redeploy
```

## Contribute

We welcome contributions from the community! To contribute, please open a pull request to have your changes reviewed and merged into the main. We encourage you to add appropriate unit tests and e2e tests if your contribution introduces a new feature.

Additionally, ensure the following commands are successful:

```
make test
make lint
make license
```

## Community and Support

To learn more about the project, its community, and governance, visit the Edge Orchestrator Community. 
For support, start with Troubleshooting or contact us. 

## License

Cluster Connect Gateway is licensed under [Apache 2.0 License](LICENSES/Apache-2.0.txt)

