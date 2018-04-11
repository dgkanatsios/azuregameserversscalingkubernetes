package main

import (
	"encoding/json"
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
	router.HandleFunc("/running", getRunningHandler).Methods("GET")

	port := 8000

	fmt.Printf("Waiting for requests at port %s\n", strconv.Itoa(port))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(port)), router))
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")
	podname, _ := createStuff()
	w.Write([]byte(podname + " was created"))
}

func createStuff() (podName string, serviceName string) {

	name := "openarena-" + shared.RandString(6)

	clientset := shared.GetClientSet()

	podsClient := clientset.Core().Pods(namespace)
	servicesClient := clientset.Core().Services(namespace)

	var port int
	port = shared.GetRandomInt(shared.MinPort, shared.MaxPort)
	for {
		if shared.IsPortUsed(port) {
			port = shared.GetRandomInt(shared.MinPort, shared.MaxPort)
		} else {
			break
		}
	}

	fmt.Println("Creating pod...")
	shared.UpsertEntity(&shared.StorageEntity{
		Name:   name,
		Status: shared.CreatingState,
		Port:   strconv.Itoa(port),
	})
	pod := shared.CreatePod(name, int32(port))
	service := shared.CreateService(shared.GetServiceNameFromPodName(name), int32(port))

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

	shared.UpsertEntity(&shared.StorageEntity{
		Name:   name,
		Status: shared.TerminatingState,
	})
	err = servicesClient.Delete(name, nil)
	if err != nil {
		log.Fatal("Cannot delete service due to ", err)
	}

	w.Write([]byte(name + " were deleted"))
}

func getRunningHandler(w http.ResponseWriter, r *http.Request) {
	entities := shared.GetRunningEntities()
	result, err := json.Marshal(entities)
	if err != nil {
		w.Write([]byte("Error in marshaling to JSON: " + err.Error()))
	}
	w.Write(result)
}
