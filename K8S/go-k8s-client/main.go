package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		// pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		// if err != nil {
		// 	panic(err.Error())
		// }
		// fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		// ingress, err := clientset.ExtensionsV1beta1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
		ingress, err := clientset.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		// fmt.Printf("There are %d Ingress in the cluster\n", len(ingress.Items))
		for _, ing := range ingress.Items {
			infos := ing.GetObjectMeta()
			// a, _ := json.Marshal(infos)
			// fmt.Println(string(a))
			// fmt.Printf("%s\n", ingress.Items[0].GetObjectKind())
			// fmt.Printf("%s\n", ingress.Items[0].GetObjectMeta())

			fields := infos.GetManagedFields()

			data := fields[0]

			for _, j := range fields {

				// if j.Time.Nanosecond() > data.Time.Nanosecond() {
				// 	data = j
				// }
				var t1, t2 metav1.Time
				t1 = *j.Time
				t2 = *data.Time
				if t1.Time.IsZero() || t2.Time.IsZero() {
					fmt.Println("no time defined")
					continue
				}
				if t1.After(t2.Time) {
					data = j
				} else if t1.Equal(&t2) {
					if j.Manager != "glbc" {
						// fmt.Println("replacing glbc manager")
						data = j
					}
				}
				// if *j.Time.After(data.Time) {
				// 	fmt.Println(data.Time)
				// 	fmt.Println(j.Time)
				// }
			}
			// a, _ := json.Marshal(data)
			// fmt.Println(string(a))
			if data.APIVersion != "networking.k8s.io/v1" {
				fmt.Printf("%s/%s: %s\n", infos.GetNamespace(), infos.GetName(), data.APIVersion)
			}

		}

		os.Exit(0)
		// Examples for error handling:
		// - Use helper functions like e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		// namespace := "default"
		// pod := "example-xxxxx"
		// _, err = clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod, metav1.GetOptions{})
		// if errors.IsNotFound(err) {
		// 	fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
		// } else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		// 	fmt.Printf("Error getting pod %s in namespace %s: %v\n",
		// 		pod, namespace, statusError.ErrStatus.Message)
		// } else if err != nil {
		// 	panic(err.Error())
		// } else {
		// 	fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
		// }

		time.Sleep(10 * time.Second)
	}
}
