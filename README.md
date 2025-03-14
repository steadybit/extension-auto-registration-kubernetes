# Extension Auto Registration for Kubernetes

The image provided by this repository is used to discover and register extensions that are installed a Kubernetes cluster.

The image needs to be added as an additional container in the agent deployment. It will then use the kubernetes api to
discover extensions that are installed in the cluster and will sync/register them with the steadybit agent.

## Configuration

| Environment Variable                   | Meaning                                                                 | required | default |
|----------------------------------------|-------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_LOG_LEVEL`                  | The Log Level.                                                          | no       | INFO    |
| `STEADYBIT_EXTENSION_AGENT_KEY`        | The agent key (used to authenticate at the agent api).                  | yes      |         |
| `STEADYBIT_EXTENSION_AGENT_PORT`       | The port where the agent is running.                                    | no       | 42899   |
| `STEADYBIT_EXTENSION_NAMESPACE_FIlTER` | Option to limit the extension lookup to a single namespace.             | no       |         |
| `STEADYBIT_EXTENSION_INITIAL_DELAY`    | The initial delay after startup before reporting extension to the agent | no       | 5       |

## Pre-requisites

### Permissions

The process requires access rights to interact with the Kubernetes API.

The cluster role for the agent requires "read"/"list" and "watch"  permissions for "pods" and "services" in the cluster.
