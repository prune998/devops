package main

import (
"context"
"log"
"os"
"time"

"cloud.google.com/go/pubsub"
)

// main is the entry point of the application.
// It connects to Google Pub/Sub and publishes a single message.
// Authentication is handled automatically via Workload Identity when running in a
// configured GKE cluster.
func main() {
  // The project ID and topic ID should be provided as environment variables.
  // This is a standard practice for containerized applications.
  // Example in Kubernetes Pod spec:
  // env:
  // - name: GCP_PROJECT_ID
  //   value: "your-gcp-project-id"
  // - name: PUBSUB_TOPIC_ID
  //   value: "your-pubsub-topic"
  projectID := os.Getenv("GCP_PROJECT_ID")
  if projectID == "" {
    log.Fatal("FATAL: GCP_PROJECT_ID environment variable must be set.")
  }

  topicID := os.Getenv("PUBSUB_TOPIC_ID")
  if topicID == "" {
    log.Fatal("FATAL: PUBSUB_TOPIC_ID environment variable must be set.")
  }

  log.Printf("Attempting to publish to topic '%s' in project '%s'", topicID, projectID)

  ctx := context.Background()

  // When no credentials are provided, the client library will automatically
  // use the GKE metadata server to obtain credentials. This is the magic
  // of Workload Identity.
  client, err := pubsub.NewClient(ctx, projectID)
  if err != nil {
    log.Fatalf("Failed to create Pub/Sub client: %v", err)
  }
  defer client.Close()

  // Get a handle to the specific topic.
  topic := client.Topic(topicID)

  // Create the message payload.
  message := &pubsub.Message{
  Data: []byte("Hello from Kubernetes with Workload Identity!"),
  Attributes: map[string]string{
    "origin": "gke-workload-identity-app",
    },
  }

  // The Publish call is asynchronous. It returns a PublishResult which can be
  // used to wait for the message to be acknowledged by the Pub/Sub server.
  result := topic.Publish(ctx, message)

  // Use a context with a timeout to wait for the publish operation to complete.
  // This is crucial to ensure the message is sent before the application exits.
  publishCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
  defer cancel()

  // Get blocks until the message is acknowledged by the server or the context times out.
  // It returns the server-assigned message ID.
  msgID, err := result.Get(publishCtx)
  if err != nil {
    log.Fatalf("Failed to publish message: %v", err)
  }

  log.Printf("Successfully published a message with ID: %s\n", msgID)
  log.Println("use ctrc-c to stop or if in a pod, kubectl exec to start a new command")

  // Block forever
  select{}
}
