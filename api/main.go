package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	"github.com/gorilla/mux"
	core "k8s.io/api/core/v1"
)

const namespace string = core.NamespaceDefault

var clientset, dedicatedgameserverclientset = shared.GetClientSet()
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

	podname := createDedicatedGameServer()
	w.Write([]byte("DedicatedGameServer " + podname + " was created"))
}

func createDedicatedGameServer() string {
	name := "openarena-" + shared.RandString(6)

	var port int
	//get a random port - FIX
	port = shared.GetRandomInt(shared.MinPort, shared.MaxPort)

	fmt.Println("Creating DedicatedGameServer...")

	if setSessionsURL == "" {
		initializeSetSessionsURL()
	}

	dgs := shared.NewDedicatedGameServer(name, int32(port), setSessionsURL)

	dgsInstance, err := dedicatedgameserverclientset.Azure().DedicatedGameServers(namespace).Create(dgs)

	if err != nil {
		fmt.Println("error")
	}
	return dgsInstance.ObjectMeta.Name

}

func deleteHandler(w http.ResponseWriter, r *http.Request) {

	if !isAPICallAuthorized(w, r) {
		return
	}

	name := r.FormValue("name")

	var err error
	err = podsClient.Delete(name, nil)
	if err != nil {
		output := fmt.Sprintf("Cannot delete pod due to %s", err.Error())
		fmt.Println(output)
		w.Write([]byte(output))
		return
	}

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

	entities := ""
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

	w.Write([]byte(fmt.Sprintf("Set sessions OK for pod: %s\n", serverSessions.Name)))
}
