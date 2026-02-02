package main

import (
	"context"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/namsral/flag"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/identitytoolkit/v1"
)

var (
	projectID = flag.String("projectID", "", "Google Project ID")
	zone      = flag.String("zone", "us-central1-a", "GCP Zone to search for instances")
	logLevel  = flag.String("logLevel", "info", "one of trace, debug, info, warn, err, none, default to info")
)

func main() {
	flag.Parse()

	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = level.NewFilter(logger, level.AllowInfo()) // <--
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "app", "go-api-compute")
	logger.Log("msg", "app started")

	// scopes := []string{
	// 	compute.DevstorageFullControlScope,
	// 	compute.ComputeScope,
	// }

	ctx2 := context.Background()
	service2, err := identitytoolkit.NewService(ctx2)
	if err != nil {
		logger.Log("msg", "error listing images", "err", err)
		os.Exit(1)
	}

	response, err := service2.Projects.Projects.ServiceAccounts.List("projects/" + projectID).Do()
	if err != nil {
		logger.Log("msg", "error listing images", "err", err)
		os.Exit(1)
	}

	// var client *http.Client
	ctx := context.Background()
	service, err := compute.NewService(ctx)
	if err != nil {
		logger.Log("msg", "Unable to create Compute service", "err", err)
		// log.Fatalf("Unable to create Compute service: %v", err)
	}

	// prefix := "https://www.googleapis.com/compute/v1/projects/" + *projectID
	// imageURL := "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-7-wheezy-v20140606"
	// zone := "us-central1-a"

	// Show the current images that are available.
	res, err := service.Images.List(*projectID).Do()
	if err != nil {
		logger.Log("msg", "error listing images", "err", err)
		os.Exit(1)
	}
	_ = res
	// logger.Log("msg", "Got compute.Images.List", "data", res)

	// trying to list instances
	instanceList, err := service.Instances.List(*projectID, *zone).Do()
	if err != nil {
		logger.Log("msg", "error listing instances", "err", err)
		os.Exit(1)
	}
	// logger.Log("msg", "Got compute.Instances.List", "data", instanceList)
	totalCount := 0
	LabelCount := 0

	for _, instance := range instanceList.Items {
		totalCount++
		if len(instance.Labels) > 0 {
			LabelCount++
		} else {
			logger.Log("msg", "not labeled", "instanceName", instance.Name)
		}

	}
	logger.Log("msg", "Instance count", "total", totalCount, "ladeled", LabelCount)

	// search for specific instance
	testApiGoClientInstance, err := service.Instances.Get(*projectID, *zone, "test-api-go-client").Do()
	if err != nil {
		logger.Log("msg", "error grabbing instance infos", "err", err)
	}

	labels := &compute.InstancesSetLabelsRequest{
		LabelFingerprint: testApiGoClientInstance.LabelFingerprint,
		Labels: map[string]string{
			"owner":   "prune",
			"creator": "go-client",
		},
	}
	_, err = service.Instances.SetLabels(*projectID, *zone, "test-api-go-client", labels).Do()
	if err != nil {
		logger.Log("msg", "error tagging instance", "err", err)
	}

	// log.Printf("Got compute.Instance, err: %#v, %v", inst, err)
	// if googleapi.IsNotModified(err) {
	// 	log.Printf("Instance not modified since insert.")
	// } else {
	// 	log.Printf("Instance modified since insert.")
	// }

	// instance := &compute.Instance{
	// 	Name:        instanceName,
	// 	Description: "compute sample instance",
	// 	MachineType: prefix + "/zones/" + zone + "/machineTypes/n1-standard-1",
	// 	Disks: []*compute.AttachedDisk{
	// 		{
	// 			AutoDelete: true,
	// 			Boot:       true,
	// 			Type:       "PERSISTENT",
	// 			InitializeParams: &compute.AttachedDiskInitializeParams{
	// 				DiskName:    "my-root-pd",
	// 				SourceImage: imageURL,
	// 			},
	// 		},
	// 	},
	// 	NetworkInterfaces: []*compute.NetworkInterface{
	// 		{
	// 			AccessConfigs: []*compute.AccessConfig{
	// 				{
	// 					Type: "ONE_TO_ONE_NAT",
	// 					Name: "External NAT",
	// 				},
	// 			},
	// 			Network: prefix + "/global/networks/default",
	// 		},
	// 	},
	// 	ServiceAccounts: []*compute.ServiceAccount{
	// 		{
	// 			Email:  "default",
	// 			Scopes: scopes,
	// 		},
	// 	},
	// }

	// op, err := service.Instances.Insert(*projectID, zone, instance).Do()
	// log.Printf("Got compute.Operation, err: %#v, %v", op, err)
	// etag := op.Header.Get("Etag")
	// log.Printf("Etag=%v", etag)

	// inst, err := service.Instances.Get(*projectID, zone, instanceName).IfNoneMatch(etag).Do()
	// log.Printf("Got compute.Instance, err: %#v, %v", inst, err)
	// if googleapi.IsNotModified(err) {
	// 	log.Printf("Instance not modified since insert.")
	// } else {
	// 	log.Printf("Instance modified since insert.")
	// }
}
