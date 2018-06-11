package v1alpha1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DedicatedGameServerCollection describes a DedicatedGameServerCollection resource
type DedicatedGameServerCollection struct {
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
	Spec DedicatedGameServerCollectionSpec `json:"spec"`
}

// DedicatedGameServerCollectionSpec is the spec for a DedicatedGameServerCollection resource
type DedicatedGameServerCollectionSpec struct {
	// Message and SomeValue are example custom spec fields
	//
	// this is where you would put your custom resource data
	Replicas int32                               `json:"replicas"`
	Image    string                              `json:"image"`
	StartMap string                              `json:"startmap"`
	Status   DedicatedGameServerCollectionStatus `json:"status"`
}

// DedicatedGameServerCollectionStatus is the status for a DedicatedGameServerCollection resource
type DedicatedGameServerCollectionStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DedicatedGameServerCollectionList is a list of DedicatedGameServerCollectionList resources
type DedicatedGameServerCollectionList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []DedicatedGameServerCollection `json:"items"`
}
