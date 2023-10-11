package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

func main() {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		fmt.Println("GOOGLE_CLOUD_PROJECT environment variable must be set.")
		os.Exit(1)
	}

	dataset := os.Getenv("DATASET")
	if dataset == "" {
		fmt.Println("DATASET environment variable must be set.")
		os.Exit(1)
	}

	table := os.Getenv("TABLE")
	if dataset == "" {
		fmt.Println("TABLE environment variable must be set.")
		os.Exit(1)
	}

	ctx := context.Background()

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("bigquery.NewClient: %v", err)
	}
	defer client.Close()

	// test dataset
	// loader := client.DatasetInProject(projectID, dataset) // .Table(options.tableID).LoaderFrom(gcsRef)

	fmt.Println("Listing Datasets")
	it := client.Datasets(ctx)
	for {
		dataset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(dataset.DatasetID)
	}

	fmt.Println("Describe Datasets")
	meta, err := client.Dataset(dataset).Metadata(ctx)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Dataset ID: %s\n", dataset)
	fmt.Printf("Description: %s\n", meta.Description)
	fmt.Println("Labels:")
	for k, v := range meta.Labels {
		fmt.Printf("\t%s: %s", k, v)
	}

	fmt.Println("Tables:")
	it2 := client.Dataset(dataset).Tables(ctx)

	cnt := 0
	for {
		t, err := it2.Next()
		if err == iterator.Done {
			break
		}
		cnt++
		fmt.Printf("\t%s\n", t.TableID)
	}
	if cnt == 0 {
		fmt.Println("\tThis dataset does not contain any tables.")
	}

	fmt.Println("running query")
	rows, err := query(ctx, client, projectID, dataset, table)
	if err != nil {
		log.Fatal(err)
	}
	if err := printResults(os.Stdout, rows); err != nil {
		log.Fatal(err)
	}

}

// query returns a row iterator suitable for reading query results.
func query(ctx context.Context, client *bigquery.Client, projectID, dataset, table string) (*bigquery.RowIterator, error) {

	selectQuery := `
	SELECT  distinct site_id
	FROM ` + projectID + `.` + dataset + `.` + table + `
	ORDER BY site_id DESC
	LIMIT 10;
`
	fmt.Printf("running query: %s \n", selectQuery)

	query := client.Query(selectQuery)

	// query := client.Query(
	// 	`SELECT
	// 				CONCAT(
	// 						'https://stackoverflow.com/questions/',
	// 						CAST(id as STRING)) as url,
	// 				view_count
	// 		FROM ` + "`bigquery-public-data.stackoverflow.posts_questions`" + `
	// 		WHERE tags like '%google-bigquery%'
	// 		ORDER BY view_count DESC
	// 		LIMIT 10;`)
	return query.Read(ctx)
}

type StackOverflowRow struct {
	WebsiteID int64 `bigquery:"site_id"`
}

// printResults prints results from a query to the Stack Overflow public dataset.
func printResults(w io.Writer, iter *bigquery.RowIterator) error {
	for {
		var row StackOverflowRow
		err := iter.Next(&row)
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error iterating through results: %w", err)
		}

		fmt.Fprintf(w, "WebsiteID: %d\n", row.WebsiteID)
	}
}

// // queryBasic demonstrates issuing a query and reading results.
// func queryBasic(w io.Writer, projectID string) error {
// 	// projectID := "my-project-id"
// 	ctx := context.Background()
// 	client, err := bigquery.NewClient(ctx, projectID)
// 	if err != nil {
// 		return fmt.Errorf("bigquery.NewClient: %v", err)
// 	}
// 	defer client.Close()

// 	q := client.Query(
// 		"SELECT name FROM `bigquery-public-data.usa_names.usa_1910_2013` " +
// 			"WHERE state = \"TX\" " +
// 			"LIMIT 100")
// 	// Location must match that of the dataset(s) referenced in the query.
// 	q.Location = "US"
// 	// Run the query and print results when the query job is completed.
// 	job, err := q.Run(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	status, err := job.Wait(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	if err := status.Err(); err != nil {
// 		return err
// 	}
// 	it, err := job.Read(ctx)
// 	for {
// 		var row []bigquery.Value
// 		err := it.Next(&row)
// 		if err == iterator.Done {
// 			break
// 		}
// 		if err != nil {
// 			return err
// 		}
// 		fmt.Fprintln(w, row)
// 	}
// 	return nil
// }
