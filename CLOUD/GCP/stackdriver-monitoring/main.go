// Sample monitoring-quickstart writes a data point to Stackdriver Monitoring.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/namsral/flag"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
)

var (
	version        = "no version set"
	displayVersion = flag.Bool("version", false, "Show version and quit")
	logLevel       = flag.String("logLevel", "warn", "log level from debug, info, warning, error. When debug, genetate 100% Tracing")
	projectID      = flag.String("projectID", "", "Google GCP Project name")
	metricName     = flag.String("metricName", "storage.googleapis.com/storage/total_bytes", "Full name of the metric to gather")
)

func printVersion() {
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("App Version: %v\n", version)
}

func main() {
	// parse flags
	flag.Parse()
	if *displayVersion {
		printVersion()
		os.Exit(0)
	}

	if *projectID == "" {
		fmt.Println("ERROR: You need to set --projectID <GCP PRoject name>")
		os.Exit(1)
	}
	ctx := context.Background()

	// Creates a client.
	c, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// list metrics typres you can query for this project
	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name: "projects/" + *projectID,
	}
	iter := c.ListMetricDescriptors(ctx, req)

	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Could not list metrics: %v", err)
		}
		fmt.Printf("%v\n", resp.GetType())
	}

	// get some infos on the metric
	fmt.Printf("\nnow requesting metric %s:\n\n", *metricName)

	req2 := &monitoringpb.GetMetricDescriptorRequest{
		Name: fmt.Sprintf("projects/%s/metricDescriptors/%s", *projectID, *metricName),
	}
	resp2, err := c.GetMetricDescriptor(ctx, req2)
	if err != nil {
		log.Printf("could not get custom metric: %v", err)
	}

	fmt.Printf("Name: %v\n", resp2.GetName())
	fmt.Printf("Description: %v\n", resp2.GetDescription())
	fmt.Printf("Type: %v\n", resp2.GetType())
	fmt.Printf("Metric Kind: %v\n", resp2.GetMetricKind())
	fmt.Printf("Value Type: %v\n", resp2.GetValueType())
	fmt.Printf("Unit: %v\n", resp2.GetUnit())
	fmt.Printf("Labels:\n")
	for _, l := range resp2.GetLabels() {
		fmt.Printf("\t%s (%s) - %s", l.GetKey(), l.GetValueType(), l.GetDescription())
	}

	fmt.Println("\n\n")

	// query the metric
	startTime := time.Now().UTC().Add(time.Minute * -20)
	endTime := time.Now().UTC()
	req4 := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + *projectID,
		Filter: `metric.type="` + *metricName + `"`,
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamp.Timestamp{
				Seconds: startTime.Unix(),
			},
			EndTime: &timestamp.Timestamp{
				Seconds: endTime.Unix(),
			},
		},
		View: monitoringpb.ListTimeSeriesRequest_FULL,
	}
	fmt.Printf("Found data points for the following instances:\n")
	it := c.ListTimeSeries(ctx, req4)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Printf("could not read time series value: %v", err)
		}
		// fmt.Printf("\t%v\n", resp.GetMetric())
		// fmt.Printf("\t%v\n", resp.GetPoints())
		fmt.Printf("\t%v\n", resp.GetResource().GetLabels()["bucket_name"])
		// fmt.Printf("\t%v\n", resp.GetMetric())
		fmt.Printf("\t%v\n\n", resp.GetPoints())
		// fmt.Printf("\t%v\n", resp.GetMetric().GetLabels()["storage_class"])
	}
	fmt.Println("Done")

	// req3 := &monitoringpb.ListMonitoredResourceDescriptorsRequest{
	// 	Name: "projects/" + *projectID,
	// }
	// iter3 := c.ListMonitoredResourceDescriptors(ctx, req3)

	// for {
	// 	resp, err := iter3.Next()
	// 	if err == iterator.Done {
	// 		break
	// 	}
	// 	if err != nil {
	// 		log.Printf("Could not list time series: %v", err)
	// 	}
	// 	fmt.Printf("%v\n", resp)
	// }

	fmt.Printf("Done writing time series data.\n")
}
