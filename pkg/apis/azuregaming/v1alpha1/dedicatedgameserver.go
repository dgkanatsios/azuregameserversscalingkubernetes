package v1alpha1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DedicatedGameServer describes a DedicatedGameServer resource
type DedicatedGameServer struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object, including
	// things like...
	//  - name
	//  - namespace
	//  - self link
	//  - labels
	//  - ... etc ...
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the custom resource spec
	Spec   DedicatedGameServerSpec   `json:"spec"`
	Status DedicatedGameServerStatus `json:"status"`
}

// DedicatedGameServerSpec is the spec for a DedicatedGameServer resource
type DedicatedGameServerSpec struct {
	// Message and SomeValue are example custom spec fields
	//
	// this is where you would put your custom resource data
	Image         string             `json:"image"`
	StartMap      string             `json:"startmap"`
	Ports         []PortInfoExtended `json:"ports"`
	PublicIP      string             `json:"publicIP"`
	NodeName      string             `json:"nodeName"`
	ActivePlayers string             `json:"activePlayers"`
}

// DedicatedGameServerStatus is the status for a DedicatedGameServer resource
type DedicatedGameServerStatus struct {
	PreviousGameServerState string `json:"previousGameServerState"`
	PreviousPodState        string `json:"previousPodState"`
	PodState                string `json:"podState"`
	GameServerState         string `json:"gameServerState"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DedicatedGameServerList is a list of DedicatedGameServerList resources
type DedicatedGameServerList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []DedicatedGameServer `json:"items"`
}
