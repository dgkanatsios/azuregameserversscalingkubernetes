package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dgkanatsios/AzureGameServersScalingKubernetes/shared"
	"github.com/gorilla/mux"
	core "k8s.io/api/core/v1"
)

const namespace string = core.NamespaceDefault

// our main function
func main() {

	router := mux.NewRouter()
	router.HandleFunc("/create", createHandler).Methods("GET")
	router.HandleFunc("/delete", deleteHandler).Queries("name", "{name}").Methods("GET")

	port := 8000

	fmt.Printf("Waiting for requests at port %s\n", strconv.Itoa(port))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(port)), router))
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")
	podname, servicename := createStuff()
	w.Write([]byte(podname + " and " + servicename + " was created"))
}

func createStuff() (podName string, serviceName string) {

	name := "openarena-" + shared.RandString(6)

	clientset := shared.GetClientSet()

	podsClient := clientset.Core().Pods(namespace)
	servicesClient := clientset.Core().Services(namespace)

	pod := shared.CreatePod(name, 27960)
	service := shared.CreateService(shared.GetServiceNameFromPodName(name), 27960)

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

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	clientset := shared.GetClientSet()

	podsClient := clientset.Core().Pods(namespace)
	servicesClient := clientset.Core().Services(namespace)

	name := r.FormValue("name")

	var err error
	err = podsClient.Delete(name, nil)
	if err != nil {
		log.Fatal("Cannot delete pod due to ", err)
	}

	err = servicesClient.Delete(name, nil)
	if err != nil {
		log.Fatal("Cannot delete service due to ", err)
	}

	w.Write([]byte(name + " " + name + " were deleted"))
}
