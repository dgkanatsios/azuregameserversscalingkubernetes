package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	"github.com/gorilla/mux"

	helpers "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/helpers"
)

var podsClient = helpers.Clientset.Core().Pods(helpers.Namespace)
var endpointsClient = helpers.Clientset.Core().Endpoints(helpers.Namespace)

var (
	startmap    string
	dockerImage string
	port        int
)

func Run(startmap_, dockerImage_ string, port_ int) error {

	startmap = startmap_
	dockerImage = dockerImage_
	port = port_

	router := mux.NewRouter()
	router.HandleFunc("/create", createDGSHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/createcollection", createDGSColHandler).Queries("code", "{code}").Methods("POST")
	router.HandleFunc("/delete", deleteDGSHandler).Queries("name", "{name}", "code", "{code}").Methods("GET")
	router.HandleFunc("/running", getRunningDGSHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/setsessions", setActiveSessionsHandler).Methods("POST")

	log.Printf("Waiting for requests at port %s\n", strconv.Itoa(port))

	return http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(port)), router)
}

func createDGSHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")

	if !helpers.IsAPICallAuthorized(w, r) {
		return
	}

	podname, err := createDedicatedGameServerCRD()

	if err != nil {
		log.Printf("error encountered: %s", err.Error())
		w.Write([]byte(fmt.Sprintf("Error %s encountered", err.Error())))
	} else {
		w.Write([]byte("DedicatedGameServer " + podname + " was created"))
	}
}

func createDGSColHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("createcollection was called")

	if !helpers.IsAPICallAuthorized(w, r) {
		return
	}

	colname, err := createDedicatedGameServerCollectionCRD()

	if err != nil {
		log.Printf("error encountered: %s", err.Error())
		w.Write([]byte(fmt.Sprintf("Error %s encountered", err.Error())))
	} else {
		w.Write([]byte("DedicatedGameServerCollection " + colname + " was created"))
	}
}

func createDedicatedGameServerCRD() (string, error) {
	name := "openarena-" + shared.RandString(6)

	//get a random port
	port, err := shared.GetRandomPort()

	if err != nil {
		return "", err
	}

	log.Println("Creating DedicatedGameServer...")

	if helpers.SetSessionsURL == "" {
		helpers.InitializeSetSessionsURL()
	}

	dgs := shared.NewDedicatedGameServer(nil, name, port, helpers.SetSessionsURL, startmap, dockerImage)

	dgsInstance, err := helpers.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServers(helpers.Namespace).Create(dgs)

	if err != nil {
		return "", err
	}
	return dgsInstance.ObjectMeta.Name, nil

}

func createDedicatedGameServerCollectionCRD() (string, error) {
	name := "openarenacollection-" + shared.RandString(6)

	log.Println("Creating DedicatedGameServerCollection...")

	if helpers.SetSessionsURL == "" {
		helpers.InitializeSetSessionsURL()
	}

	dgsCol := shared.NewDedicatedGameServerCollection(name, startmap, dockerImage, 5)

	dgsColInstance, err := helpers.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServerCollections(helpers.Namespace).Create(dgsCol)

	if err != nil {
		return "", err
	}
	return dgsColInstance.ObjectMeta.Name, nil

}

func deleteDGSHandler(w http.ResponseWriter, r *http.Request) {

	if !helpers.IsAPICallAuthorized(w, r) {
		return
	}

	name := r.FormValue("name")

	var err error
	err = helpers.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServers(helpers.Namespace).Delete(name, nil)
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
