// Run export GOOGLE_APPLICATION_CREDENTIALS="[PATH]" before you run this program.
// Replace [PATH] with the JSON file that contains your service account key.
// See https://cloud.google.com/docs/authentication/production?hl=en#providing_service_account_credentials.

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nevill/gcp/compute"
	"gopkg.in/alecthomas/kingpin.v2"
)

/*
Usage: ./sole-tenant \
		--image "docker.io/nginx:1.17" \
		--args "..." \
		--command "..." \
		--machine-type n1-highmem-8 \
		--auth your_auth.json \
*/
// This program will create:
// 1. a sole-tenant template named 'dedicated'.
// 2. a sole-tenant nodegroup named 'dedicated' with initialCount = 1.
// 3. a running container on a compute instance named 'dedicated' in nodegroup.

type Variable struct{
	Name string
	Value string
}

func main() {
	app := kingpin.New(
		filepath.Base(os.Args[0]),
		"create an instance with running container on sole-tenant node.",
	)
	image := app.Flag("image", "The image will be run in the container").Required().String()
	command := app.Flag("command", "Command to run in the container").Required().String()
	args := app.Flag("arg", "Arguments will be passed to container").Strings()
	env := app.Flag("env", "Environment variables will be passed to container").Strings()
	app.HelpFlag.Short('h')

	kingpin.MustParse(app.Parse(os.Args[1:]))

	var envVars compute.EnvVars
	for _, val := range *env {
		parts := strings.Split(val, "=")
		envVars = append(envVars, Variable{parts[0], parts[1]})
	}

	computeManager, err := compute.New()
	if err != nil {
		panic(err)
	}
	if err = computeManager.CreateContainer(*image, []string{*command}, *args, envVars); err != nil {
		panic(err)
	}
}
