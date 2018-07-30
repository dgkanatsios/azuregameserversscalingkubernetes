package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"

	helpers "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/helpers"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var podsClient = shared.Clientset.Core().Pods(shared.GameNamespace)
var dgsClient = shared.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace)

func Run(port int) error {

	router := mux.NewRouter()
	router.HandleFunc("/create", createDGSHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/createcollection", createDGSColHandler).Queries("code", "{code}").Methods("POST")
	router.HandleFunc("/delete", deleteDGSHandler).Queries("name", "{name}", "code", "{code}").Methods("GET")
	router.HandleFunc("/running", getRunningDGSHandler).Queries("code", "{code}").Methods("GET")
	router.HandleFunc("/setactiveplayers", setActivePlayersHandler).Methods("POST")
	router.HandleFunc("/setserverstatus", setServerStatusHandler).Methods("POST")

	log.Printf("Waiting for requests at port %s\n", strconv.Itoa(port))

	return http.ListenAndServe(fmt.Sprintf(":%s", strconv.Itoa(port)), router)
}

func createDGSHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("create was called")

	if !helpers.IsAPICallAuthorized(w, r) {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	var dgsInfo helpers.DedicatedGameServerInfo
	err := json.NewDecoder(r.Body).Decode(&dgsInfo)

	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Incorrect arguments: " + err.Error()))
		return
	}

	podname, err := helpers.CreateDedicatedGameServerCRD(dgsInfo)

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
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	var dgsColInfo helpers.DedicatedGameServerCollectionInfo
	err := json.NewDecoder(r.Body).Decode(&dgsColInfo)

	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Incorrect arguments: " + err.Error()))
		return
	}

	colname, err := helpers.CreateDedicatedGameServerCollectionCRD(dgsColInfo)

	if err != nil {
		log.Printf("error encountered: %s", err.Error())
		w.Write([]byte(fmt.Sprintf("Error %s encountered", err.Error())))
	} else {
		w.Write([]byte("DedicatedGameServerCollection " + colname + " was created"))
	}
}

func deleteDGSHandler(w http.ResponseWriter, r *http.Request) {

	if !helpers.IsAPICallAuthorized(w, r) {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	name := r.FormValue("name")

	var err error
	err = shared.Dedicatedgameserverclientset.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).Delete(name, nil)
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
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	entities := ""
	result, err := json.Marshal(entities)
	if err != nil {
		w.Write([]byte("Error in marshaling to JSON: " + err.Error()))
	}
	w.Write(result)
}

func setActivePlayersHandler(w http.ResponseWriter, r *http.Request) {
	if !helpers.IsAPICallAuthorized(w, r) {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	var serverActivePlayers helpers.ServerActivePlayers
	err := json.NewDecoder(r.Body).Decode(&serverActivePlayers)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting Active Players: " + err.Error()))
		return
	}

	dgs, err := dgsClient.Get(serverActivePlayers.ServerName, metav1.GetOptions{})
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting Active Players: " + err.Error()))
		return
	}

	dgsCopy := dgs.DeepCopy()
	dgsCopy.Spec.ActivePlayers = strconv.Itoa(serverActivePlayers.PlayerCount)

	_, err = dgsClient.Update(dgsCopy)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting Active Players: " + err.Error()))
		return
	}

	w.Write([]byte(fmt.Sprintf("Set active players OK for server: %s\n", serverActivePlayers.ServerName)))
}

func setServerStatusHandler(w http.ResponseWriter, r *http.Request) {
	if !helpers.IsAPICallAuthorized(w, r) {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	var serverStatus helpers.ServerStatus
	err := json.NewDecoder(r.Body).Decode(&serverStatus)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting ServerStatus: " + err.Error()))
		return
	}

	status := serverStatus.Status
	if status != shared.GameServerStatusCreating && status != shared.GameServerStatusMarkedForDeletion && status != shared.GameServerStatusRunning {
		w.WriteHeader(400)
		w.Write([]byte("Wrong value for serverStatus"))
		return
	}

	dgs, err := dgsClient.Get(serverStatus.ServerName, metav1.GetOptions{})
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting ServerStatus: " + err.Error()))
		return
	}

	dgsCopy := dgs.DeepCopy()
	dgsCopy.Status.GameServerState = serverStatus.Status

	_, err = dgsClient.Update(dgsCopy)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting ServerStatus: " + err.Error()))
		return
	}

	w.Write([]byte(fmt.Sprintf("Set server status OK for server: %s\n", serverStatus.ServerName)))
}
