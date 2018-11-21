package helpers

// ServerMarkedForDeletion represents the markedForDeletion status of the dedicated game server
type ServerMarkedForDeletion struct {
	ServerName        string `json:"serverName"`
	Namespace         string `json:"namespace"`
	MarkedForDeletion bool   `json:"markedForDeletion"`
}

// ServerState represents the status of the dedicated game server
type ServerState struct {
	ServerName string `json:"serverName"`
	Namespace  string `json:"namespace"`
	State      string `json:"state"`
}

// ServerHealth represents the status of the dedicated game server
type ServerHealth struct {
	ServerName string `json:"serverName"`
	Namespace  string `json:"namespace"`
	Health     string `json:"health"`
}

// ServerActivePlayers is a struct that represents active sessions (connected players) per pod
type ServerActivePlayers struct {
	ServerName  string `json:"serverName"`
	Namespace   string `json:"namespace"`
	PlayerCount int    `json:"playerCount"`
}
