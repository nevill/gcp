// Run export GOOGLE_APPLICATION_CREDENTIALS="[PATH]" before you run this program.
// Replace [PATH] with the JSON file that contains your service account key.
// See https://cloud.google.com/docs/authentication/production?hl=en#providing_service_account_credentials.

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	konlet "github.com/GoogleCloudPlatform/konlet/gce-containers-startup/types"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	yaml "gopkg.in/yaml.v2"
)

// All the parameters input by user
const (
	project                      string = "macro-mile-203600"
	region                       string = "europe-west3"
	zone                         string = "europe-west3-b"
	instanceName                 string = "funcbench-by-client"
	containerImage               string = "docker.io/nginx:1.17"
	machineType                  string = "n1-highmem-8"
	soleTenantNodeTemplateName   string = "benchmark-tmpl-sole-tenancy"
	soleTenantNodeGroupName      string = "benchmark-nodegroup"
)

// The key name used in metadata will be used by konlet.
const ContainerDeclarationMetadata string = "gce-container-declaration"

func createNodeTemplate(svc *compute.Service) string {
	nodeTemplate := compute.NodeTemplate{
		Name:     soleTenantNodeTemplateName,
		NodeType: "n1-node-96-624",
		NodeAffinityLabels: map[string]string{
			"workload": "benchmark",
		},
	}
	op, err := svc.NodeTemplates.Insert(project, region, &nodeTemplate).Do()
	if err != nil {
		panic(err)
	}

	err = waitFor(fmt.Sprintf("Creating new node template %s ... ", soleTenantNodeTemplateName), op, svc)

	if err != nil {
		panic(err)
	}

	fmt.Println("Created node template:", nodeTemplate.Name)
	return op.TargetLink
}

func createNodeGroup(svc *compute.Service, nodeTemplateUrl string) {
	var count int64 = 1
	nodeGroup := compute.NodeGroup{
		Name: soleTenantNodeGroupName,
		NodeTemplate: nodeTemplateUrl,
	}
	op, err := svc.NodeGroups.Insert(project, zone, count, &nodeGroup).Do()

	if err != nil {
		panic(err)
	}

	err = waitFor(fmt.Sprintf("Creating new node group %s ... ", soleTenantNodeGroupName), op, svc)

	if err != nil {
		panic(err)
	}

	fmt.Println("Created new node group:", nodeGroup.Name)
}

func createInstance(svc *compute.Service) {
	policy := konlet.RestartPolicyNever
	containerSpec := konlet.ContainerSpec{
		Spec: konlet.ContainerSpecStruct{
			Containers: []konlet.Container{
				konlet.Container{
					Image: containerImage,
				},
			},
			RestartPolicy: &policy,
		},
	}

	metadata, _ := yaml.Marshal(containerSpec)
	metadataString := string(metadata)

	instance := compute.Instance{
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				Boot: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskSizeGb:  100,
					SourceImage: "projects/cos-cloud/global/images/family/cos-stable",
				},
			},
		},
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType),
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{},
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				&compute.MetadataItems{
					Key:   ContainerDeclarationMetadata,
					Value: &metadataString,
				},
			},
		},
		Name: instanceName,
		Scheduling: &compute.Scheduling{
			NodeAffinities: []*compute.SchedulingNodeAffinity{
				&compute.SchedulingNodeAffinity{
					Key:      "workload",
					Operator: "IN",
					Values: []string{
						"benchmark",
					},
				},
			},
		},
	}

	op, err := svc.Instances.Insert(project, zone, &instance).Do()
	if err != nil {
		panic(err)
	}

	err = waitFor(fmt.Sprintf("Creating new compute instance %s ...", instanceName), op, svc)
	if err != nil {
		panic(err)
	}

	fmt.Println("New instance", instanceName, "has been created")
}

// Get the last part of URL, split by '/'.
func nameFromResourceUrl(link string) string {
	parts := strings.Split(link, "/")
	return parts[len(parts)-1]
}

func getResourceCheckFunc(op *compute.Operation, svc *compute.Service) func(...googleapi.CallOption) (*compute.Operation, error) {
	if op.Zone != "" {
		zone := nameFromResourceUrl(op.Zone)
		return svc.ZoneOperations.Get(project, zone, op.Name).Do
	} else if op.Region != "" {
		region := nameFromResourceUrl(op.Region)
		return svc.RegionOperations.Get(project, region, op.Name).Do
	}
	return svc.GlobalOperations.Get(project, op.Name).Do
}

func waitFor(activity string, op *compute.Operation, svc *compute.Service) (result error) {
	timeout := time.Duration(60) * time.Second
	waitChan := time.After(timeout)
	workChan := make(chan bool)

	defer close(workChan)

	go func() {
		check := getResourceCheckFunc(op, svc)
		rop := op
		var err error
		for {
			fmt.Printf("%s %s\n", activity, rop.Status)

			if rop.Status == "DONE" {
				workChan <- true
				return
			}

			rop, err = check()

			if err != nil {
				out, _ := rop.MarshalJSON()
				fmt.Printf("Error with operation: %s\n", out)
				panic(err)
			}

			time.Sleep(time.Duration(1) * time.Second)
		}
	}()

	for {
		select {
		case <-waitChan:
			return fmt.Errorf("Operation timeout")
		case <-workChan:
			return nil
		}
	}
}

func main() {
	ctx := context.Background()
	svc, err := compute.NewService(ctx)
	if err != nil {
		panic(err)
	}

	nodeTemplate :=	createNodeTemplate(svc)
	createNodeGroup(svc, nodeTemplate)
	createInstance(svc)
}
