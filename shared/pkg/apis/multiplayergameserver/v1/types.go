package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiplayerGameServer describes a MultiplayerGameServer resource
type MultiplayerGameServer struct {
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
	Spec MultiplayerGameServerSpec `json:"spec"`
}

// MultiplayerGameServerSpec is the spec for a MultiplayerGameServer resource
type MultiplayerGameServerSpec struct {
	// Message and SomeValue are example custom spec fields
	//
	// this is where you would put your custom resource data
	Image    string `json:"image"`
	Port     *int32 `json:"port"`
	StartMap string `json:startmap`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiplayerGameServerList is a list of MultiplayerGameServer resources
type MultiplayerGameServerList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []MultiplayerGameServer `json:"items"`
}
