// Run export GOOGLE_APPLICATION_CREDENTIALS="[PATH]" before you run this program.
// Replace [PATH] with the JSON file that contains your service account key.
// See https://cloud.google.com/docs/authentication/production?hl=en#providing_service_account_credentials.

package main

import (
	"github.com/nevill/gcp/compute"
)

/*
Usage: ./gce \
		--container-image "docker.io/nginx:1.17" \
		--container-args "..." \
		--machine-type n1-highmem-8 \
		--project your-project \
		--zone europe-west3-b \
		--auth your_auth.json \
*/
// This program will create:
// 1. a sole-tenant template named 'dedicated'.
// 2. a sole-tenant nodegroup named 'dedicated' with initialCount = 1.
// 3. a running container on a compute instance named 'dedicated' in nodegroup.
func main() {
	computeManager, err := compute.New()
	if err != nil {
		panic(err)
	}
	containerImage := "docker.io/nginx:1.17"
	if err = computeManager.CreateContainer(containerImage); err != nil {
		panic(err)
	}
}
