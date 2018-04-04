package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	apiv1 "k8s.io/api/core/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// our main function
func main() {
	router := mux.NewRouter()
	router.HandleFunc("/create", create).Methods("GET")

	log.Fatal(http.ListenAndServe(":8000", router))
}

func create(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")
	createPod()
	w.Write([]byte("OK"))
}

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
		log.Fatalf("Can not get kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	return clientset
}

func createPod() {

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
	podsClient := clientset.Core().Pods(namespace)

	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openarena-" + randString(6),
		},
		Spec: core.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:  "web",
					Image: "nginx:1.12",
					Ports: []apiv1.ContainerPort{
						{
							Name:          "http",
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	fmt.Println("Creating pod...")
	result, err := podsClient.Create(pod)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())
}

//CreateKubeConfig authenticates to the local cluster
func createKubeConfig() *string {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	return kubeconfig
}

func int32Ptr(i int32) *int32 { return &i }

func randString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
