package shared

const (
	// TableName is the name for the Azure Storage Table
	TableName = "aksdetails"
	// Timeout for Azure Table Storage operations
	Timeout = 30
	// MinPort is minimum Port Number
	MinPort = 20000
	// MaxPort is maximum Port Number
	MaxPort = 30000
	// CreatingState = "Creating"
	CreatingState = "Creating"
	// TerminatingState = "Terminating"
	TerminatingState = "Terminating"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a CRD is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a CRD fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by CRD"
	// MessageResourceSynced is the message used for an Event fired when a CRD
	// is synced successfully
	MessageResourceSynced = "CRD synced successfully"
)
