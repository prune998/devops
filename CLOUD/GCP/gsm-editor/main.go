package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/namsral/flag"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/yaml.v3"

	"github.com/totherme/unstructured"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var (
	projectName = flag.String("projectName", "", "Google Project ID")
	secretName  = flag.String("secretName", "my-secret", "name of the Google secret")
	secretKey   = flag.String("secretKey", "my-key", "key of the Google secret to update")
	secretValue = flag.String("secretValue", "my-value", "value to use for the new secret")
	logLevel    = flag.String("logLevel", "info", "one of trace, debug, info, warn, err, none, default to info")
	quiet       = flag.Bool("quiet", false, "automatically apply the change without prompting (dangerous)")
)

func main() {

	// init go stuff
	flag.Parse()

	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = level.NewFilter(logger, level.Allow(level.ParseDefault(*logLevel, level.InfoValue())))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "app", "gsm-editor")
	level.Info(logger).Log("msg", "app started")

	ctx := context.Background()

	// Connect to Gcloud
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		level.Error(logger).Log("Failed to create client: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	// Get the GSM Secret's data
	// Secret name is like name := "projects/my-project/secrets/my-secret/versions/latest"
	reqData := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", *projectName, *secretName),
	}

	resp, err := client.AccessSecretVersion(ctx, reqData)
	if err != nil {
		level.Error(logger).Log("failed to get secret version: %v", err)
		os.Exit(1)
	}

	// Split the key with a path + the object to change
	path, key := filepath.Split(*secretKey)

	// Parse the data, which is unstructured
	myData, err := unstructured.ParseYAML(string(resp.Payload.Data))
	if err != nil {
		level.Error(logger).Log("err", "Couldn't parse my own yaml")
		os.Exit(1)
	}

	myPayloadData, err := myData.GetByPointer(filepath.Clean(path))
	if err != nil {
		level.Error(logger).Log("Couldn't address into my own yaml", filepath.Clean(path))
		os.Exit(1)
	}

	// update the secret with new value
	err = myPayloadData.SetField(key, *secretValue)
	if err != nil {
		level.Error(logger).Log("Key should be the key name to the Object to change", key)
	}

	// print the diff of the change
	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2) // this is what you're looking for
	yamlEncoder.Encode(myData.UnsafeObValue())

	// Compute the change diff
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(resp.Payload.Data), b.String(), false)

	// print some debug
	if *logLevel == "debug" {
		fmt.Printf("-----------\noriginal data:\n\n")
		fmt.Println(string(resp.Payload.Data))
		fmt.Printf("-----------\nupdated data:\n\n")
		fmt.Println(b.String())

	}

	if len(diffs) > 1 {

		// Print info
		if *logLevel == "info" || *logLevel == "debug" {
			fmt.Printf("-----------\ndiff data:\n\n")
			// do the diff

			// wait for user consent to change the secret
			fmt.Println(dmp.DiffPrettyText(diffs))
		}

		if diffs[1].Type == 0 {
			level.Error(logger).Log("err", "Nothing to change in the Secret", "len", len(diffs))
			os.Exit(1)
		}

		if !*quiet {
			// Prompt the user to confirm before continuing
			fmt.Println("\nDo you want to continue? type 'Y' to continue and update the secret (creates new version): ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				level.Error(logger).Log("error reading user input", err)
				os.Exit(1)
			}
			response = strings.TrimSpace(response)

			if response != "Y" {
				level.Info(logger).Log("changes", "nothing")
				os.Exit(0)
			}
		}

		// apply the change in the secret
		updateReq := &secretmanagerpb.AddSecretVersionRequest{
			Parent: fmt.Sprintf("projects/%s/secrets/%s", *projectName, *secretName),
			Payload: &secretmanagerpb.SecretPayload{
				Data: b.Bytes(),
			},
		}
		// Call the API.
		updateResult, err := client.AddSecretVersion(ctx, updateReq)
		if err != nil {
			level.Error(logger).Log("failed to update secret", err)
		}
		level.Info(logger).Log("msg", "Updated secret", "secret", updateResult.Name, "version", updateResult.State.String())
	} else {
		level.Error(logger).Log("err", "Nothing to change in the Secret", "len", len(diffs))
	}
}
