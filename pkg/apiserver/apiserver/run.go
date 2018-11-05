package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	helpers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apiserver/helpers"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	"github.com/gorilla/mux"
)

var listPodStateRunningRequiresAuth = false

// Run begins the WebServer
func Run(port int, listrunningauth bool) *http.Server {

	server := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
	}

	router := mux.NewRouter()

	router.HandleFunc("/create", createDGSColHandler).Queries("code", "{code}").Methods("POST")
	router.HandleFunc("/delete", deleteDGSColHandler).Queries("name", "{name}", "code", "{code}").Methods("GET")
	router.HandleFunc("/healthz", healthHandler).Methods("GET")
	route := router.HandleFunc("/running", getPodStateRunningDGSHandler).Methods("GET")
	if listrunningauth {
		listPodStateRunningRequiresAuth = true
		route.Queries("code", "{code}")
	}

	// Dedicated Game Server API methods
	router.HandleFunc("/setactiveplayers", setActivePlayersHandler).Methods("POST")
	router.HandleFunc("/setserverstatus", setServerStatusHandler).Methods("POST")

	//this should be the last handler
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./html/"))).Methods("GET")

	log.Printf("API Server waiting for requests at port %d", port)

	server.Handler = router

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Errorf("Failed to listen and serve API server: %v", err)
		}
	}()

	return server
}

func createDGSColHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("createcollection was called")

	result, err := helpers.IsAPICallAuthenticated(w, r)
	if err != nil {
		log.Errorf("Error in authentication: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Error"))
		return
	}

	if !result {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	var dgsCol dgsv1alpha1.DedicatedGameServerCollection
	err = json.NewDecoder(r.Body).Decode(&dgsCol)

	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Incorrect arguments: " + err.Error()))
		return
	}

	colname, err := helpers.CreateDedicatedGameServerCollectionCRD(dgsCol.Name, dgsCol.Spec.Replicas, dgsCol.Spec.Template)

	if err != nil {
		log.Printf("error encountered: %s", err.Error())
		w.Write([]byte(fmt.Sprintf("Error %s encountered", err.Error())))
	} else {
		w.Write([]byte("DedicatedGameServerCollection " + colname + " was created"))
	}
}

func deleteDGSColHandler(w http.ResponseWriter, r *http.Request) {

	result, err := helpers.IsAPICallAuthenticated(w, r)
	if err != nil {
		log.Errorf("Error in authentication: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Error"))
		return
	}

	if !result {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	name := r.FormValue("name")

	_, dgsClient, err := shared.GetClientSet()

	if err != nil {
		log.Errorf("Error in getting client set: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Error"))
		return
	}

	err = dgsClient.AzuregamingV1alpha1().DedicatedGameServerCollections(shared.GameNamespace).Delete(name, nil)
	if err != nil {
		msg := fmt.Sprintf("Cannot delete DedicatedGameServerCollection due to %s", err.Error())
		log.Print(msg)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte(name + " was deleted"))
}

func getPodStateRunningDGSHandler(w http.ResponseWriter, r *http.Request) {

	if listPodStateRunningRequiresAuth {
		result, err := helpers.IsAPICallAuthenticated(w, r)
		if err != nil {
			log.Errorf("Error in authentication: %v", err)
			w.WriteHeader(500)
			w.Write([]byte("Error"))
			return
		}

		if !result {
			w.WriteHeader(401)
			w.Write([]byte("Unathorized"))
			return
		}
	}

	entities, err := shared.GetDedicatedGameServersPodStateRunning()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error in marshaling to JSON: " + err.Error()))
	}
	result, err := json.Marshal(entities)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error in marshaling to JSON: " + err.Error()))
	}
	w.Write(result)
}

func setActivePlayersHandler(w http.ResponseWriter, r *http.Request) {
	result, err := helpers.IsAPICallAuthenticated(w, r)
	if err != nil {
		log.Errorf("Error in authentication: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Error"))
		return
	}

	if !result {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	var serverActivePlayers helpers.ServerActivePlayers
	err = json.NewDecoder(r.Body).Decode(&serverActivePlayers)

	if err != nil {
		log.Errorf("Error in deserialization: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Error in deserialization: " + err.Error()))
		return
	}

	err = shared.UpdateActivePlayers(serverActivePlayers.ServerName, serverActivePlayers.PlayerCount)

	if err != nil {
		log.Errorf("Error in setting Active Players: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Error setting Active Players: " + err.Error()))
		return
	}

	w.Write([]byte(fmt.Sprintf("Set active players OK for server: %s\n", serverActivePlayers.ServerName)))
}

func setServerStatusHandler(w http.ResponseWriter, r *http.Request) {
	result, err := helpers.IsAPICallAuthenticated(w, r)
	if err != nil {
		log.Errorf("Error in authentication: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Error"))
		return
	}

	if !result {
		w.WriteHeader(401)
		w.Write([]byte("Unathorized"))
		return
	}

	var serverStatus helpers.ServerStatus
	err = json.NewDecoder(r.Body).Decode(&serverStatus)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting ServerStatus: " + err.Error()))
		log.Errorf("Error setting server status:%s", err.Error())
		return
	}

	status := dgsv1alpha1.DedicatedGameServerState(serverStatus.Status)
	//a very simple validation
	if status != dgsv1alpha1.DGSCreating && status != dgsv1alpha1.DGSMarkedForDeletion && status != dgsv1alpha1.DGSRunning && status != dgsv1alpha1.DGSFailed {
		w.WriteHeader(400)
		w.Write([]byte("Wrong value for serverStatus"))
		log.Errorf("Error in server status, wrong value:%s", status)
		return
	}

	err = shared.UpdateGameServerStatus(serverStatus.ServerName, status)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting ServerStatus: " + err.Error()))
		log.Errorf("Error setting server status:%s", err.Error())
		return
	}

	w.Write([]byte(fmt.Sprintf("Set server status OK for server: %s\n", serverStatus.ServerName)))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
