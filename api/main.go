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

var clientset = shared.GetClientSet()
var podsClient = clientset.Core().Pods(namespace)
var servicesClient = clientset.Core().Services(namespace)
var secretsClient = clientset.Core().Secrets(namespace)
var endpointsClient = clientset.Core().Endpoints(namespace)

var setSessionsURL string

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/create", createHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/delete", deleteHandler).Queries("name", "{name}", "code", "{code}").Methods("GET")
	router.HandleFunc("/running", getRunningPodsHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/setsessions", setSessionsHandler).Methods("POST")

	port := 8000

	fmt.Printf("Waiting for requests at port %s\n", strconv.Itoa(port))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(port)), router))
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")

	if !isAPICallAuthorized(w, r) {
		return
	}

	podname, _ := createPodAndService()
	w.Write([]byte("pod " + podname + " and relevant Service were created"))
}

func createPodAndService() (podName string, serviceName string) {

	name := "openarena-" + shared.RandString(6)

	var port int
	//get a random port
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

	if setSessionsURL == "" {
		initializeSetSessionsURL()
	}

	pod := shared.NewPod(name, int32(port), setSessionsURL)
	service := shared.NewService(shared.GetServiceNameFromPodName(name), int32(port))

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

	if !isAPICallAuthorized(w, r) {
		return
	}

	name := r.FormValue("name")

	var err error
	err = podsClient.Delete(name, nil)
	if err != nil {
		output := "Cannot delete pod due to " + err.Error()
		fmt.Println(output)
		w.Write([]byte(output))
		return
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

func getRunningPodsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPICallAuthorized(w, r) {
		return
	}

	entities := shared.GetRunningEntities()
	result, err := json.Marshal(entities)
	if err != nil {
		w.Write([]byte("Error in marshaling to JSON: " + err.Error()))
	}
	w.Write(result)
}

func setSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPICallAuthorized(w, r) {
		return
	}

	var serverSessions ServerSessions
	err := json.NewDecoder(r.Body).Decode(&serverSessions)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting sessions: " + err.Error()))
		return
	}

	shared.UpsertEntity(&shared.StorageEntity{
		Name:           serverSessions.Name,
		ActiveSessions: &serverSessions.ActiveSessions,
	})

	w.Write([]byte("Set sessions OK for pod: " + serverSessions.Name))
}
