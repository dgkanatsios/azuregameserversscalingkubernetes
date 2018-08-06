package v1alpha1

type DedicatedGameServerState string
type DedicatedGameServerCollectionState string

const (
	DedicatedGameServerStateCreating          DedicatedGameServerState = "Creating"
	DedicatedGameServerStateRunning           DedicatedGameServerState = "Running"
	DedicatedGameServerStateMarkedForDeletion DedicatedGameServerState = "MarkedForDeletion"
	DedicatedGameServerStateFailed            DedicatedGameServerState = "Failed"
)

const (
	DedicatedGameServerCollectionStateRunning  DedicatedGameServerCollectionState = "Running"
	DedicatedGameServerCollectionStateCreating DedicatedGameServerCollectionState = "Creating"
	DedicatedGameServerCollectionFailed        DedicatedGameServerCollectionState = "Failed"
)
