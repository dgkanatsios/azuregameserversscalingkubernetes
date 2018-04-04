package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		//kubeconfigPath = "/.kube/config"
		kubeconfigPath = "C:\\users\\dgkanatsios\\.kube\\config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetClientOutOfCluster returns a k8s clientset to the request from outside of cluster
func GetClientOutOfCluster() kubernetes.Interface {
	config, err := buildOutOfClusterConfig()
	if err != nil {
		logrus.Fatalf("Can not get kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	return clientset
}

func main() {

	namespace := apiv1.NamespaceDefault
	// config, err := rest.InClusterConfig()

	// if err != nil {
	// 	fmt.Println(err)
	// }

	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	clientset := GetClientOutOfCluster()

	watchlist := cache.NewListWatchFromClient(clientset.Core().RESTClient(), "pods", namespace, fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&apiv1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Println("Pod added : %s\n", obj)
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Println("Pod deleted: %s \n", obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Println("Pod changed \n")
			},
		},
	)
	stop := make(chan struct{})
	go controller.Run(stop)
	for {
		time.Sleep(time.Second)
	}
}
