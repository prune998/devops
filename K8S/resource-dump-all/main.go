package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	var label *string
	var namespace *string
	var all *bool
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	label = flag.String("l", "", "label selector")
	namespace = flag.String("n", "", "namespace selector")
	all = flag.Bool("a", false, "include non-namespaced object")
	flag.Parse()

	ctx := context.TODO()
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// Create client set (discovery and dynamic)
	clientsetDynamic, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	clientsetDiscovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	var namespaces []string = []string{*namespace}
	if *namespace == "" {
		namespaces, err = getNamespace(clientsetDynamic, ctx)
		if err != nil {
			panic(err)
		}
	}

	// Search for namespaced resources
	r, err := clientsetDiscovery.ServerPreferredResources()
	if err != nil {
		panic(err)
	}
	for _, v := range r {
		group, _ := schema.ParseGroupVersion(v.GroupVersion)
		for _, a := range v.APIResources {
			// If the resource cannot be listed, there is no way to get information about it.
			if !strings.Contains(a.Verbs.String(), "list") {
				continue
			}
			// Generate a resource id to identify which resource we want
			resourceId := schema.GroupVersionResource{
				Group:    group.Group,
				Version:  group.Version,
				Resource: a.Name,
			}

			// Get resources
			var result *unstructured.UnstructuredList
			if a.Namespaced {
				for _, n := range namespaces {
					result, err = clientsetDynamic.Resource(resourceId).Namespace(n).List(ctx, metav1.ListOptions{LabelSelector: *label})
					if err != nil {
						fmt.Printf("err: '%v' ressource: '%s/%s'\n", err, group.String(), a.Name)
					}
					printResult(result)
				}
			} else if *all {
				result, err = clientsetDynamic.Resource(resourceId).List(ctx, metav1.ListOptions{LabelSelector: *label})
				if err != nil {
					fmt.Printf("err: '%v' ressource: '%s/%s'\n", err, group.String(), a.Name)
				}
				printResult(result)
			}
		}
	}
}

func printResult(result *unstructured.UnstructuredList) {
	if len(result.Items) > 0 {
		for _, item := range result.Items {
			fmt.Printf("%s: %s/%s\n", item.GetKind(), item.GetNamespace(), item.GetName())
		}
	}
}

func getNamespace(c dynamic.Interface, ctx context.Context) ([]string, error) {
	var namespaces []string
	namespaceResourceId := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "namespaces",
	}
	result, err := c.Resource(namespaceResourceId).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, v := range result.Items {
		namespaces = append(namespaces, v.GetName())
	}
	return namespaces, nil
}

