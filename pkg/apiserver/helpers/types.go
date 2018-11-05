package helpers

// ServerStatus represents the status of the dedicated game server
type ServerStatus struct {
	ServerName string `json:"serverName"`
	Status     string `json:"status"`
}

// ServerActivePlayers is a struct that represents active sessions (connected players) per pod
type ServerActivePlayers struct {
	ServerName  string `json:"serverName"`
	PlayerCount int    `json:"playerCount"`
}
