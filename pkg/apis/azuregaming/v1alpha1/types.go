package v1alpha1

type DedicatedGameServerState string
type DedicatedGameServerCollectionState string

type DedicatedGameServerFailBehavior string

const (
	DGSCreating          DedicatedGameServerState = "Creating"
	DGSRunning           DedicatedGameServerState = "Running"
	DGSMarkedForDeletion DedicatedGameServerState = "MarkedForDeletion"
	DGSFailed            DedicatedGameServerState = "Failed"
)

const (
	DGSColRunning           DedicatedGameServerCollectionState = "Running"
	DGSColCreating          DedicatedGameServerCollectionState = "Creating"
	DGSColFailed            DedicatedGameServerCollectionState = "Failed"
	DGSColNeedsIntervention DedicatedGameServerCollectionState = "NeedsIntervention"
)

const (
	Delete DedicatedGameServerFailBehavior = "Delete"
	Remove DedicatedGameServerFailBehavior = "Remove"
)
