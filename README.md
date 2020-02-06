## About
This project contains a collection of wrapped [API of Google Cloud Services](https://google.golang.org/api).

## Examples

### 1. [sole-tenant](examples/sole-tenant)
This will create a running container in a VM instance, which is run on top of a sole-tenant node.

**⚠️Warning**

This program will create a sole-tenant node, it costs a lot.

## Reference
1. https://cloud.google.com/compute/docs/nodes/create-nodes
1. https://cloud.google.com/compute/docs/containers/configuring-options-to-run-containers
1. https://cloud.google.com/compute/docs/containers/deploying-containers
1. https://cloud.google.com/compute/docs/reference/rest/v1/
1. https://cloud.google.com/sdk/gcloud/reference/compute/instance-templates/create-with-container
