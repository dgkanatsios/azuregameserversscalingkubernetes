package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

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

	fmt.Printf("Waiting for requests at port %s\n", strconv.Itoa(port))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(port)), router))
}

func create(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")
	podname, servicename := createStuff()
	w.Write([]byte(podname + " " + servicename + " was created"))
}

func createStuff() (podName string, serviceName string) {

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
	servicesClient := clientset.Core().Services(namespace)

	name := "openarena-" + shared.RandString(6)

	pod := createPod(name, 80)
	service := createService(name, 80)

	fmt.Println("Creating pod...")
	result, err := podsClient.Create(pod)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created pod %q.\n", result.GetObjectMeta().GetName())

	result2, err := servicesClient.Create(service)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created service %q.\n", result2.GetObjectMeta().GetName())

	return pod.ObjectMeta.Name, service.ObjectMeta.Name
}

func createPod(name string, port int32) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"server": name},
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
							ContainerPort: port,
						},
					},
				},
			},
		},
	}
	return pod
}

func createService(name string, port int32) *core.Service {
	service := &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-service",
		},
		Spec: core.ServiceSpec{
			Ports: []apiv1.ServicePort{{
				Name:     "port",
				Protocol: "UDP",
				Port:     port,
			}},
			Selector: map[string]string{"server": name},
			Type:     "LoadBalancer",
		},
	}
	return service
}
