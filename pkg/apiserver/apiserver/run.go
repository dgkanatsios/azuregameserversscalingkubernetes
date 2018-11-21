package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	helpers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apiserver/helpers"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	"github.com/gorilla/mux"
)

var listPodPhaseRunningRequiresAuth = false

// Run begins the WebServer
func Run(port int, listrunningauth bool) *http.Server {

	server := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
	}

	router := mux.NewRouter()

	router.HandleFunc("/create", createDGSColHandler).Queries("code", "{code}").Methods("POST")
	router.HandleFunc("/delete", deleteDGSColHandler).Queries("name", "{name}", "code", "{code}").Methods("GET")
	router.HandleFunc("/healthz", healthHandler).Methods("GET")
	route := router.HandleFunc("/running", getPodPhaseRunningDGSHandler).Methods("GET")
	if listrunningauth {
		listPodPhaseRunningRequiresAuth = true
		route.Queries("code", "{code}")
	}

	// Dedicated Game Server API methods
	router.HandleFunc("/setactiveplayers", setActivePlayersHandler).Methods("POST")
	router.HandleFunc("/setdgsstate", setServerStateHandler).Methods("POST")
	router.HandleFunc("/setsdgshealth", setServerHealthHandler).Methods("POST")
	router.HandleFunc("/setdgsmarkedfordeletion", setServerMarkedForDeletionHandler).Methods("POST")

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

func getPodPhaseRunningDGSHandler(w http.ResponseWriter, r *http.Request) {

	if listPodPhaseRunningRequiresAuth {
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

	entities, err := shared.GetReadyDGSs()
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
	setDGSStatusHandler(w, r, func(r io.ReadCloser) (interface{}, error) {
		var serverActivePlayers helpers.ServerActivePlayers
		err := json.NewDecoder(r).Decode(&serverActivePlayers)

		//a very simple validation
		if serverActivePlayers.PlayerCount < 0 {
			w.WriteHeader(400)
			w.Write([]byte("Wrong value for activePlayers"))
			return "", fmt.Errorf("Error in active players, wrong value:%d", serverActivePlayers.PlayerCount)
		}
		return serverActivePlayers, err
	})
}

func setServerStateHandler(w http.ResponseWriter, r *http.Request) {
	setDGSStatusHandler(w, r, func(r io.ReadCloser) (interface{}, error) {
		var serverState helpers.ServerState
		err := json.NewDecoder(r).Decode(&serverState)

		//a very simple validation
		state := dgsv1alpha1.DGSState(serverState.State)
		if state != dgsv1alpha1.DGSIdle && state != dgsv1alpha1.DGSAssigned && state != dgsv1alpha1.DGSRunning && state != dgsv1alpha1.DGSPostMatch {
			w.WriteHeader(400)
			w.Write([]byte("Wrong value for serverState"))

			return "", fmt.Errorf("Error in server state, wrong value:%s", state)
		}
		return serverState, err
	})
}

func setServerHealthHandler(w http.ResponseWriter, r *http.Request) {
	setDGSStatusHandler(w, r, func(r io.ReadCloser) (interface{}, error) {
		var serverHealth helpers.ServerHealth
		err := json.NewDecoder(r).Decode(&serverHealth)

		//a very simple validation
		health := dgsv1alpha1.DGSHealth(serverHealth.Health)
		if health != dgsv1alpha1.DGSCreating && health != dgsv1alpha1.DGSHealthy && health != dgsv1alpha1.DGSFailed {
			w.WriteHeader(400)
			w.Write([]byte("Wrong value for serverHealth"))
			return "", fmt.Errorf("Error in server status, wrong value:%s", health)
		}
		return serverHealth, err
	})
}

func setServerMarkedForDeletionHandler(w http.ResponseWriter, r *http.Request) {
	setDGSStatusHandler(w, r, func(r io.ReadCloser) (interface{}, error) {
		var serverMarkedForDeletion helpers.ServerMarkedForDeletion
		err := json.NewDecoder(r).Decode(&serverMarkedForDeletion)
		return serverMarkedForDeletion, err
	})
}

func setDGSStatusHandler(w http.ResponseWriter, r *http.Request, decode func(r io.ReadCloser) (interface{}, error)) {
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

	decoded, err := decode(r.Body)

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting serverMarkedForDeletion: " + err.Error()))
		log.Errorf("Error setting server marked for deletion:%s", err.Error())
		return
	}

	switch v := decoded.(type) {
	case helpers.ServerMarkedForDeletion:
		err = shared.UpdateGameServerMarkedForDeletion(v.ServerName, v.Namespace, v.MarkedForDeletion)
	case helpers.ServerState:
		err = shared.UpdateGameServerState(v.ServerName, v.Namespace, dgsv1alpha1.DGSState(v.State))
	case helpers.ServerHealth:
		err = shared.UpdateGameServerHealth(v.ServerName, v.Namespace, dgsv1alpha1.DGSHealth(v.Health))
	case helpers.ServerActivePlayers:
		err = shared.UpdateActivePlayers(v.ServerName, v.Namespace, v.PlayerCount)
	default:
		err = fmt.Errorf("Cannot recognize type %T", v)
	}

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error setting values: " + err.Error()))
		log.Errorf("Error setting values %v because of %s", decoded, err.Error())
		return
	}

	w.Write([]byte(fmt.Sprintf("Set values %v OK\n", decoded)))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
