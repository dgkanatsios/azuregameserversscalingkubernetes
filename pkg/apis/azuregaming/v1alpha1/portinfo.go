package v1alpha1

type PortInfo struct {
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
	Name          string `json:"name"`
}

type PortInfoExtended struct {
	PortInfo
	HostPort int32 `json:"hostPort"`
}
