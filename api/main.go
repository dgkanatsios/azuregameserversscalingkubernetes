package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dgkanatsios/AzureGameServersScalingKubernetes/shared"
	"github.com/gorilla/mux"
	apiv1 "k8s.io/api/core/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// our main function
func main() {
	router := mux.NewRouter()
	router.HandleFunc("/create", create).Methods("GET")

	port := 8000

	fmt.Println("Waiting for requests at port ", port)

	log.Fatal(http.ListenAndServe(":"+string(port), router))
}

func create(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")
	podname := createPod()
	w.Write([]byte(podname + " was created"))
}

func createPod() string {

	namespace := apiv1.NamespaceDefault
	// config, err := rest.InClusterConfig()

	// if err != nil {
	// 	fmt.Println(err)
	// }

	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	clientset := shared.GetClientOutOfCluster()
	podsClient := clientset.Core().Pods(namespace)

	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openarena-" + shared.RandString(6),
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
	return pod.ObjectMeta.Name
}
