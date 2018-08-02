package shared

const GameDockerImage = "docker.io/dgkanatsios/docker_openarena_k8s:0.0.4"
const APIAccessCodeSecretName = "apiaccesscode"

const (
	setActivePlayersURLPrefix = "http://docker-openarena-k8s-apiserver/setactiveplayers?code="
	setServerStatusURLPrefix  = "http://docker-openarena-k8s-apiserver/setserverstatus?code="
)

const (
	GameNamespace = "default"

	// Timeout for Azure Table Storage operations
	Timeout = 30
	// MinPort is minimum Port Number
	MinPort = 20000
	// MaxPort is maximum Port Number
	MaxPort = 30000
	// PodStateTerminating = "Terminating"
	PodStateTerminating = "Terminating"
	PodStatePending     = "Pending"
	PodStateRunning     = "Running"
)

const (
	LabelDedicatedGameServer               = "DedicatedGameServer"
	LabelServerName                        = "ServerName"
	LabelDedicatedGameServerCollectionName = "DedicatedGameServerCollectionName"
	LabelActivePlayers                     = "ActivePlayers"
	LabelGameServerState                   = "GameServerState"
	LabelPodState                          = "PodState"
)

const (
	GameServerStateCreating          = "Creating"
	GameServerStateRunning           = "Running"
	GameServerStateMarkedForDeletion = "MarkedForDeletion"
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
	// MessageResourceSynced is the message used for an Event fired when a resource
	// is synced successfully
	MessageResourceSynced = "%s with name %s synced successfully"

	DedicatedGameServerReplicasChanged = "Dedicated Game Server Replicas Changed"

	MessageReplicasDecreased = "%s with name %s decreased by %d replicas successfully"
	MessageReplicasIncreased = "%s with name %s increased by %d replicas successfully"
)
