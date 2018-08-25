package helpers

import (
	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
)

type ServerStatus struct {
	ServerName   string `json:"serverName"`
	PodNamespace string `json:"podNamespace"`
	Status       string `json:"status"`
}

// ServerActivePlayers is a struct that represents active sessions (connected players) per pod
type ServerActivePlayers struct {
	ServerName   string `json:"serverName"`
	PodNamespace string `json:"podNamespace"`
	PlayerCount  int    `json:"playerCount"`
}

type DedicatedGameServerCollectionInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	dgsv1alpha1.DedicatedGameServerCollectionSpec
}

type DedicatedGameServerInfo struct {
	Name string `json:"name"`
	dgsv1alpha1.DedicatedGameServerSpec
}
