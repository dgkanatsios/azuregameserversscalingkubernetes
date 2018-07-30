package v1alpha1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PortRegistry describes a PortRegistry resource
type PortRegistry struct {
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
	Spec PortRegistrySpec `json:"spec"`
}

// PortRegistrySpec is the spec for a PortRegistry resource
type PortRegistrySpec struct {
	Ports           map[int32]bool    `json:"ports"`
	GameServerPorts map[string]string `json:"gameServerPorts"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PortRegistryList is a list of PortRegistryList resources
type PortRegistryList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []PortRegistry `json:"items"`
}
