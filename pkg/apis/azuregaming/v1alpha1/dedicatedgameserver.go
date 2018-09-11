package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
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
	// this is where you would put your custom resource data
	Template corev1.PodSpec `json:"template"`
}

// DedicatedGameServerStatus is the status for a DedicatedGameServer resource
type DedicatedGameServerStatus struct {
	PodState                 corev1.PodPhase          `json:"podState"`
	DedicatedGameServerState DedicatedGameServerState `json:"gameServerState"`
	PublicIP                 string                   `json:"publicIP"`
	NodeName                 string                   `json:"nodeName"`
	ActivePlayers            int                      `json:"activePlayers"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DedicatedGameServerList is a list of DedicatedGameServerList resources
type DedicatedGameServerList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []DedicatedGameServer `json:"items"`
}
