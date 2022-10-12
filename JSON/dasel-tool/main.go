package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/namsral/flag"
	"github.com/tomwright/dasel"
)

var (
	input         = flag.String("input", "", "path to the json file, use stdin if empty")
	kind          = flag.String("kind", "", "kind of object to parse: pubsub, storage, datasets (bigquery), redis, functions, computedisk, computeinstance, secrets, bigtable")
	fetchProjects = flag.Bool("fetchProjects", false, "set to True to get the project mapping from ID to name, warning: this is slow as there is no cache")
	data          interface{}
)

type Project struct {
	Name          string `json:"name"`
	ProjectId     string `json:"projectId"`
	ProjectNumber string `json:"projectNumber"`
}

func main() {
	flag.Parse()

	projects := make(map[string]string)

	if *fetchProjects {
		projects = getProjects()
	}

	// the path to the resource informations like name and labels
	basePath := ".[*]"
	if *kind == "storage" {
		basePath = ".[*].metadata"
	}

	// the path to the key holding the name of the resource
	namePath := ".name"
	if *kind == "datasets" {
		namePath = ".id"
	}

	if *input != "" {
		// jsonFilePath := "/Users/prune/tmp/pubsub.json"
		jsonFile, err := os.Open(*input)
		if err != nil {
			fmt.Println(err)
		}
		byteValue, _ := ioutil.ReadAll(jsonFile)

		err = json.Unmarshal(byteValue, &data)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		err := json.NewDecoder(os.Stdin).Decode(&data)
		if err != nil {
			log.Fatal(err)
		}
	}

	rootNode := dasel.New(data)
	results, _ := rootNode.QueryMultiple(basePath)

	isLabeled := false
	for _, node := range results {
		nodeNameNode, _ := node.Query(namePath)
		labels, _ := node.QueryMultiple(".labels.-")
		for _, label := range labels {
			labelName := label.InterfaceValue()
			if labelName == "service" {
				isLabeled = true
				break
			}

		}
		if !isLabeled {
			switch *kind {
			case "pubsub":
				// split the node name
				nodeNameParts := strings.Split(nodeNameNode.String(), "/")
				fmt.Printf("pubsub,%s,%s,%s\n", nodeNameParts[2], nodeNameParts[1], nodeNameParts[3])
			case "storage":
				projectNumber, err := node.Query(".projectNumber")
				if err != nil {
					fmt.Println(err)
				}
				proj := projectNumber.String()
				if p, ok := projects[projectNumber.String()]; ok {
					proj = p
				}
				fmt.Printf("storage,bucket,%s,%s\n", proj, nodeNameNode.String())
			case "datasets":
				nodeNameParts := strings.Split(nodeNameNode.String(), ":")
				fmt.Printf("bigquery,datasets,%s,%s\n", nodeNameParts[0], nodeNameParts[1])
			case "redis":
				nodeNameParts := strings.Split(nodeNameNode.String(), "/")
				fmt.Printf("memorystore,redis,%s,%s,%s\n", nodeNameParts[1], nodeNameParts[3], nodeNameParts[5])
			case "functions":
				nodeNameParts := strings.Split(nodeNameNode.String(), "/")
				runtime, err := node.Query(".runtime")
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("functions,%s,%s,%s,%s\n", runtime.String(), nodeNameParts[1], nodeNameParts[3], nodeNameParts[5])
			case "computedisk":
				selflinkParts, err := node.Query(".selfLink")
				if err != nil {
					fmt.Println(err)
				}
				project := strings.Split(selflinkParts.String(), "/")

				fmt.Printf("compute,disk,%s,%s\n", project[6], nodeNameNode.String())
			case "computeinstance":
				selflinkParts, err := node.Query(".selfLink")
				if err != nil {
					fmt.Println(err)
				}
				project := strings.Split(selflinkParts.String(), "/")

				fmt.Printf("compute,instance,%s,%s\n", project[6], nodeNameNode.String())
			case "secrets":
				nameParts := strings.Split(nodeNameNode.String(), "/")

				proj := nameParts[1]
				if p, ok := projects[nameParts[1]]; ok {
					proj = p
				}

				fmt.Printf("secrets,secrets,%s,%s\n", proj, nameParts[3])

			case "bigtable":
				nameParts := strings.Split(nodeNameNode.String(), "/")
				fmt.Printf("bigtable,instance,%s,%s\n", nameParts[1], nameParts[3])

			default:
				fmt.Println("unknown Kind")
			}
		} else {
			// fmt.Printf("true\n")
		}
		isLabeled = false

	}
}

func getProjects() map[string]string {
	// grab project list and build id <-> name map

	cmd := exec.Command("gcloud", "projects", "list", "--format", "json", "--quiet")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	var allProjects []Project
	json.Unmarshal(out.Bytes(), &allProjects)
	if err != nil {
		log.Fatal(err)
	}

	projects := make(map[string]string)
	for _, proj := range allProjects {
		projects[proj.ProjectNumber] = proj.Name
	}

	return projects
}
