package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	helpers "github.com/dgkanatsios/azuregameserversscalingkubernetes/api/helpers"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	"github.com/gorilla/mux"
)

var podsClient = helpers.Clientset.Core().Pods(helpers.Namespace)
var endpointsClient = helpers.Clientset.Core().Endpoints(helpers.Namespace)

const startmap = "dm4ish"
const dockerImage = "docker.io/dgkanatsios/docker_openarena_k8s:latest"
const port = 8000

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/create", createDGSHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/delete", deleteDGSHandler).Queries("name", "{name}", "code", "{code}").Methods("GET")
	router.HandleFunc("/running", getRunningDGSHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/setsessions", setActiveSessionsHandler).Methods("POST")

	log.Printf("Waiting for requests at port %s\n", strconv.Itoa(port))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(port)), router))
}

func createDGSHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")

	if !helpers.IsAPICallAuthorized(w, r) {
		return
	}

	podname := createDedicatedGameServer()
	w.Write([]byte("DedicatedGameServer " + podname + " was created"))
}

func createDedicatedGameServer() string {
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

	log.Println("Creating DedicatedGameServer...")

	if helpers.SetSessionsURL == "" {
		helpers.InitializeSetSessionsURL()
	}

	dgs := shared.NewDedicatedGameServer(name, int32(port), helpers.SetSessionsURL, startmap, dockerImage)

	dgsInstance, err := helpers.Dedicatedgameserverclientset.Azure().DedicatedGameServers(helpers.Namespace).Create(dgs)

	if err != nil {
		log.Printf("error encountered: %s", err.Error())
	}
	return dgsInstance.ObjectMeta.Name

}

func deleteDGSHandler(w http.ResponseWriter, r *http.Request) {

	if !helpers.IsAPICallAuthorized(w, r) {
		return
	}

	name := r.FormValue("name")

	var err error
	err = helpers.Dedicatedgameserverclientset.Azure().DedicatedGameServers(helpers.Namespace).Delete(name, nil)
	if err != nil {
		msg := fmt.Sprintf("Cannot delete DedicatedGameServer due to %s", err.Error())
		log.Print(msg)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte(name + " was deleted"))
}

func getRunningDGSHandler(w http.ResponseWriter, r *http.Request) {
	if !helpers.IsAPICallAuthorized(w, r) {
		return
	}

	entities := ""
	result, err := json.Marshal(entities)
	if err != nil {
		w.Write([]byte("Error in marshaling to JSON: " + err.Error()))
	}
	w.Write(result)
}

func setActiveSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if !helpers.IsAPICallAuthorized(w, r) {
		return
	}

	var serverSessions helpers.ServerSessions
	err := json.NewDecoder(r.Body).Decode(&serverSessions)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting sessions: " + err.Error()))
		return
	}

	w.Write([]byte(fmt.Sprintf("Set sessions OK for pod: %s\n", serverSessions.Name)))
}
