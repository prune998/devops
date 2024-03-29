// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// [START functions_helloworld_http]

// Package helloworld provides a set of Cloud Functions samples.
package helloworld

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

func init() {
	functions.HTTP("HelloHTTP", HelloHTTP)
}

// HelloHTTP is an HTTP Cloud Function with a request parameter.
func HelloHTTP(w http.ResponseWriter, r *http.Request) {

	// Set the Kubernetes cluster where metrics are pushed to
	// pushUrl := "<pushgateway-url>:4278"
	pushUrl := "http://gravel-gateway.wk-qa-us-central-cluster.qa.wk.internal:4278"
	// Set the Kubernetes namespace the alerting rules are sent to.
	kubernetesNamespace := "prune"
	// Mandatory metric `job` label. Use the service name
	jobName := "HelloHTTP"
	// Prometheus namespace is the metric prefix, use the service name in snake case
	metricPrefix := "hellohttp"

	// Setup metrics
	messagesTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricPrefix,
		Name:      "messages_total",
	}, []string{"type", "clearmode"})

	// manage the HTTP request
	var d struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		fmt.Fprint(w, "ERROR decoding body")
		return
	}

	// Run job
	handleMessage(messagesTotal, len(d.Name))

	// Push on exit
	err := push.New(pushUrl, jobName).
		Format(expfmt.FmtText).
		Grouping("namespace", kubernetesNamespace).
		Collector(messagesTotal).
		// Gatherer(prometheus.DefaultGatherer).
		Add()
	if err != nil {
		fmt.Println(err)
	}

	if d.Name == "" {
		fmt.Fprint(w, "Hello, World!")
		return
	}
	fmt.Fprintf(w, "Hello, %s!", html.EscapeString(d.Name))
}

func handleMessage(messagesTotal *prometheus.CounterVec, val int) {
	// ...
	messagesTotal.With(prometheus.Labels{"type": "test", "clearmode": "family"}).Add(float64(val))
}

// [END functions_helloworld_http]
