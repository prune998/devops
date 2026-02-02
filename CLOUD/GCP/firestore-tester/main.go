package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

func main() {
          projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
        if projectID == "" {
                fmt.Println("GOOGLE_CLOUD_PROJECT environment variable must be set.")
                os.Exit(1)
        }

        collection := os.Getenv("FIRESTORE")
        if collection == "" {
                fmt.Println("FIRESTORE collection variable must be set to an existing Collection.")
                os.Exit(1)
        }

        creds := os.Getenv("CREDENTIALS")
        if creds == "" {
                fmt.Println("CREDENTIALS environment variable not set, using Worload Identity.")
        }

        ctx := context.Background()
        var client *firestore.Client
        var err error
        if creds != "" {
                client, err = firestore.NewClient(ctx, projectID, option.WithCredentialsFile("/tmp/sa_credentials.json"))
                if err != nil {
                        log.Fatal(err)
                }
        } else {

                client, err = firestore.NewClient(ctx, projectID)
                if err != nil {
                        log.Fatal(err)
                }
        }
        defer client.Close()

        fmt.Println(client.)
        documents := client.Collection(collection).Where("requires_update", "==", true).Select().Documents(ctx)
        snapshots, err := documents.GetAll()
        for _,i := range snapshots {
                fmt.Println(i.Ref.Path)
                fmt.Println(i.Ref.ID)

                if i.Exists() && i.Ref.ID=="237" {
                  doc := client.Collection(collection).Doc("237")
	                docData, err := doc.Get(ctx)
									if err != nil {
                		  fmt.Println(err)
																	}
									fmt.Print(docData.Data())
                }
        }

}
