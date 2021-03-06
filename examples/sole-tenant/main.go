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

type Variable struct {
	Name  string
	Value string
}

func main() {
	app := kingpin.New(
		filepath.Base(os.Args[0]),
		"Operate on sole-tenant node",
	)

	app.HelpFlag.Short('h')

	createCommand := app.Command("create", "Create an instance with running container on sole-tenant node.")
	image := createCommand.Flag("image", "The image will be run in the container").Required().String()
	command := createCommand.Flag("command", "Command to run in the container").Required().String()
	args := createCommand.Flag("arg", "Arguments will be passed to container").Strings()
	env := createCommand.Flag("env", "Environment variables will be passed to container").Strings()

	deleteCommand := app.Command("delete", "Delete the instance and corresponding node group, node template")

	parsedCmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	computeManager, err := compute.New()
	if err != nil {
		panic(err)
	}

	switch parsedCmd {
	case createCommand.FullCommand():
		var envVars compute.EnvVars
		for _, val := range *env {
			parts := strings.Split(val, "=")
			envVars = append(envVars, Variable{parts[0], parts[1]})
		}

		if err = computeManager.CreateContainer(*image, []string{*command}, *args, envVars); err != nil {
			panic(err)
		}
	case deleteCommand.FullCommand():
		if err := computeManager.DeleteInstance(); err != nil {
			panic(err)
		}
		if err := computeManager.DeleteNodeGroup(); err != nil {
			panic(err)
		}
		if err := computeManager.DeleteNodeTemplate(); err != nil {
			panic(err)
		}
	}
}
