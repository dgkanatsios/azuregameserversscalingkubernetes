package helpers

type ServerStatus struct {
	ServerName   string `json:"serverName"`
	PodNamespace string `json:"podNamespace"`
	Status       string `json:"status"`
}

// ServerActivePlayers is a struct that represents active sessions (connected players) per pod
type ServerActivePlayers struct {
	ServerName   string `json:"serverName"`
	PodNamespace string `json:"podNamespace"`
	PlayerCount  int    `json:"playerCount"`
}
