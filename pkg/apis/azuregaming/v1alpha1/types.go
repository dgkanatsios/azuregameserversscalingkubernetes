package v1alpha1

// DGSState represents the DGS State
type DGSState string

const (
	// DGSIdle represents the Idle state for a DGS
	DGSIdle DGSState = "Idle"
	// DGSAssigned represents the Assigned state for a DGS
	DGSAssigned DGSState = "Assigned"
	// DGSRunning represents the Running state for a DGS
	DGSRunning DGSState = "Running"
	// DGSPostMatch represents the PostMatch state for a DGS
	DGSPostMatch DGSState = "PostMatch"
)

// DGSHealth represents the DGS Health
type DGSHealth string

const (
	// DGSCreating represents Creating DGS
	DGSCreating DGSHealth = "Creating"
	// DGSHealthy represents a Healthy DGS
	DGSHealthy DGSHealth = "Healthy"
	// DGSFailed represents a Failed DGS
	DGSFailed DGSHealth = "Failed"
)

type DGSColHealth string

const (
	DGSColHealthy           DGSColHealth = "Healthy"
	DGSColCreating          DGSColHealth = "Creating"
	DGSColFailed            DGSColHealth = "Failed"
	DGSColNeedsIntervention DGSColHealth = "NeedsIntervention"
)

type DedicatedGameServerFailBehavior string

const (
	Delete DedicatedGameServerFailBehavior = "Delete"
	Remove DedicatedGameServerFailBehavior = "Remove"
)
