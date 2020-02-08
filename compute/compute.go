package compute

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
	project     string = "macro-mile-203600"
	region      string = "europe-west3"
	zone        string = "europe-west3-b"
	machineType string = "n1-highmem-8"
)

const (
	// The key name used in metadata will be used by konlet.
	ContainerDeclarationMetadata string = "gce-container-declaration"

	// Used as the name of node template / node group / instance.
	SoleTenantName        string = "dedicated"
	NodeGroupInitialCount int64  = 1
)

type Manager struct {
	computeService *compute.Service

	nodeTemplate *compute.NodeTemplate
	nodeGroup    *compute.NodeGroup
	instance     *compute.Instance
}

type EnvVars []struct {
	Name  string
	Value string
}

func New() (*Manager, error) {
	ctx := context.Background()
	svc, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	manager := Manager{
		computeService: svc,
	}

	return &manager, nil
}

func (m *Manager) CreateContainer(image string, cmd []string, args []string, env EnvVars) error {

	if m.nodeGroup == nil {
		if err := m.createNodeGroup(); err != nil {
			return err
		}
	}

	policy := konlet.RestartPolicyNever
	containerSpec := konlet.ContainerSpec{
		Spec: konlet.ContainerSpecStruct{
			Containers: []konlet.Container{
				konlet.Container{
					Image:   image,
					Args:    args,
					Env:     env,
					Command: cmd,
				},
			},
			RestartPolicy: &policy,
		},
	}

	metadata, _ := yaml.Marshal(containerSpec)
	metadataString := string(metadata)

	return m.createInstance(&metadataString)
}

// This function will look for a node template named 'dedicated'
func (m *Manager) readNodeTemplate() error {
	api := m.computeService.NodeTemplates
	template, err := api.Get(project, region, SoleTenantName).Do()

	if err != nil {
		return err
	}

	m.nodeTemplate = template

	return nil
}

func (m *Manager) createNodeTemplate() error {
	nodeTemplate := compute.NodeTemplate{
		Name:     SoleTenantName,
		NodeType: "n1-node-96-624",
		NodeAffinityLabels: map[string]string{
			"workload": "benchmark",
		},
	}
	api := m.computeService
	op, err := api.NodeTemplates.Insert(project, region, &nodeTemplate).Do()
	if err != nil {
		return err
	}

	err = waitFor(fmt.Sprintf("Creating new node template %s ... ", nodeTemplate.Name), op, api)

	if err != nil {
		return err
	}

	fmt.Println("Node template", nodeTemplate.Name, "has been created.")
	return m.readNodeTemplate()
}

func (m *Manager) readNodeGroup() error {
	api := m.computeService.NodeGroups
	group, err := api.Get(project, zone, SoleTenantName).Do()

	if err != nil {
		return err
	}

	m.nodeGroup = group

	return nil
}

func (m *Manager) createNodeGroup() error {
	if m.nodeTemplate == nil {
		if err := m.createNodeTemplate(); err != nil {
			return err
		}
	}

	api := m.computeService
	nodeGroup := compute.NodeGroup{
		Name:         SoleTenantName,
		NodeTemplate: m.nodeTemplate.SelfLink,
	}
	op, err := api.NodeGroups.Insert(project, zone, NodeGroupInitialCount, &nodeGroup).Do()

	if err != nil {
		return err
	}

	err = waitFor(fmt.Sprintf("Creating new node group %s ... ", nodeGroup.Name), op, api)

	if err != nil {
		return err
	}

	fmt.Println("Node group", nodeGroup.Name, "has been created.")
	return m.readNodeGroup()
}

func (m *Manager) readInstance() error {
	api := m.computeService.Instances
	instance, err := api.Get(project, zone, SoleTenantName).Do()

	if err != nil {
		return err
	}

	m.instance = instance

	return nil
}

func (m *Manager) createInstance(metadata *string) error {
	instance := compute.Instance{
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
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
					Value: metadata,
				},
			},
		},
		Name: SoleTenantName,
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

	api := m.computeService
	op, err := api.Instances.Insert(project, zone, &instance).Do()
	if err != nil {
		return err
	}

	err = waitFor(fmt.Sprintf("Creating new compute instance %s ...", instance.Name), op, api)
	if err != nil {
		return err
	}

	fmt.Println("New instance", instance.Name, "has been created")
	return m.readInstance()
}

func (m *Manager) DeleteInstance() error {
	if m.instance == nil {
		if err := m.readInstance(); err != nil {
			return err
		}
	}

	api := m.computeService
	op, err := api.Instances.Delete(project, zone, m.instance.Name).Do()
	if err != nil {
		return err
	}

	err = waitFor(fmt.Sprintf("Deleting compute instance %s ...", m.instance.Name), op, api)
	if err != nil {
		return err
	}

	fmt.Println("Instance", m.instance.Name, "has been deleted")
	return nil
}

func (m *Manager) DeleteNodeGroup() error {
	if m.nodeGroup == nil {
		if err := m.readNodeGroup(); err != nil {
			return err
		}
	}

	api := m.computeService
	op, err := api.NodeGroups.Delete(project, zone, m.nodeGroup.Name).Do()
	if err != nil {
		return err
	}

	err = waitFor(fmt.Sprintf("Deleting node group %s ...", m.nodeGroup.Name), op, api)
	if err != nil {
		return err
	}

	fmt.Println("Node group", m.nodeGroup.Name, "has been deleted")
	return nil
}

func (m *Manager) DeleteNodeTemplate() error {
	if m.nodeTemplate == nil {
		if err := m.readNodeTemplate(); err != nil {
			return err
		}
	}

	api := m.computeService
	op, err := api.NodeTemplates.Delete(project, region, m.nodeTemplate.Name).Do()
	if err != nil {
		return err
	}

	err = waitFor(fmt.Sprintf("Deleting node template %s ...", m.nodeTemplate.Name), op, api)
	if err != nil {
		return err
	}

	fmt.Println("Node template", m.nodeTemplate.Name, "has been deleted")
	return nil
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
