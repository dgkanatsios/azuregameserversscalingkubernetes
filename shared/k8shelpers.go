package shared

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func NewDedicatedGameServerCollection(name string, startmap string, image string, replicas int32, ports []dgsv1alpha1.PortInfo) *dgsv1alpha1.DedicatedGameServerCollection {
	dedicatedgameservercollection := &dgsv1alpha1.DedicatedGameServerCollection{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{LabelDedicatedGameServerCollectionName: name},
		},
		Spec: dgsv1alpha1.DedicatedGameServerCollectionSpec{
			Image:    image,
			Replicas: replicas,
			StartMap: startmap,
			Ports:    ports,
		},
	}
	return dedicatedgameservercollection
}

func NewDedicatedGameServer(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, name string, ports []dgsv1alpha1.PortInfoExtended, startmap string, image string) *dgsv1alpha1.DedicatedGameServer {
	dedicatedgameserver := &dgsv1alpha1.DedicatedGameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{LabelServerName: name, LabelDedicatedGameServerCollectionName: dgsCol.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dgsCol, schema.GroupVersionKind{
					Group:   dgsv1alpha1.SchemeGroupVersion.Group,
					Version: dgsv1alpha1.SchemeGroupVersion.Version,
					Kind:    "DedicatedGameServerCollection",
				}),
			},
		},
		Spec: dgsv1alpha1.DedicatedGameServerSpec{
			Image:    image,
			Ports:    ports,
			StartMap: startmap,
		},
	}
	return dedicatedgameserver
}

// NewPod returns a Kubernetes Pod struct
// It also sets a label called "DedicatedGameServer" with the value of the corresponding DedicatedGameServer resource
func NewPod(dgs *dgsv1alpha1.DedicatedGameServer, setActivePlayersURL string, setServerStatusURL string) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   dgs.Name,
			Labels: map[string]string{LabelDedicatedGameServer: dgs.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dgs, schema.GroupVersionKind{
					Group:   dgsv1alpha1.SchemeGroupVersion.Group,
					Version: dgsv1alpha1.SchemeGroupVersion.Version,
					Kind:    "DedicatedGameServer",
				}),
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "dedicatedgameserver-" + dgs.Name,
					Image: dgs.Spec.Image,
					Env: []core.EnvVar{
						{
							Name:  "OA_STARTMAP",
							Value: dgs.Spec.StartMap,
						},
						{
							Name:  "OA_PORT",
							Value: "27960", //TODO: works only for openarena
						},
						{
							Name: "STORAGE_ACCOUNT_NAME",
							ValueFrom: &core.EnvVarSource{
								SecretKeyRef: &core.SecretKeySelector{
									LocalObjectReference: core.LocalObjectReference{
										Name: "openarena-storage-secret",
									},
									Key: "azurestorageaccountname",
								},
							},
						},
						{
							Name: "STORAGE_ACCOUNT_KEY",
							ValueFrom: &core.EnvVarSource{
								SecretKeyRef: &core.SecretKeySelector{
									LocalObjectReference: core.LocalObjectReference{
										Name: "openarena-storage-secret",
									},
									Key: "azurestorageaccountkey",
								},
							},
						},
						{
							Name:  "SERVER_NAME",
							Value: dgs.Name,
						},
						{
							Name:  "SET_ACTIVE_PLAYERS_URL",
							Value: setActivePlayersURL,
						},
						{
							Name:  "SET_SERVER_STATUS_URL",
							Value: setServerStatusURL,
						},
						{
							Name: "POD_NAMESPACE",
							ValueFrom: &core.EnvVarSource{
								FieldRef: &core.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
					},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      "openarenavolume",
							MountPath: "/data",
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: "openarenavolume",
					VolumeSource: core.VolumeSource{
						AzureFile: &core.AzureFileVolumeSource{
							SecretName: "openarena-storage-secret",
							ShareName:  "openarenadata",
							ReadOnly:   false,
						},
					},
				},
			},
			DNSPolicy:     core.DNSClusterFirstWithHostNet, //https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
			RestartPolicy: core.RestartPolicyNever,
		},
	}

	var ports []core.ContainerPort
	for _, portInfo := range dgs.Spec.Ports {
		ports = append(ports, core.ContainerPort{
			Name:          portInfo.Name,
			Protocol:      core.Protocol(portInfo.Protocol),
			ContainerPort: portInfo.ContainerPort,
			HostPort:      portInfo.HostPort,
		})

	}

	// assign the ports
	pod.Spec.Containers[0].Ports = ports

	return pod
}

func UpdateActivePlayers(serverName string, activePlayers int) error {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return err
	}
	dgs, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Get(serverName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	//dgsCopy := dgs.DeepCopy()
	dgs.Spec.ActivePlayers = string(activePlayers)
	dgs.Labels[LabelActivePlayers] = string(activePlayers)

	_, err = dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Update(dgs)
	if err != nil {
		return err
	}
	return nil
}

func UpdateGameServerStatus(serverName string, serverStatus string) error {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return err
	}
	dgs, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Get(serverName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	dgsCopy := dgs.DeepCopy()
	dgsCopy.Status.GameServerState = serverStatus
	dgsCopy.Labels[LabelGameServerState] = serverStatus

	_, err = dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Update(dgsCopy)
	if err != nil {
		return err
	}
	return nil
}

func GetDedicatedGameServersMarkedForDeletionWithZeroPlayers() ([]dgsv1alpha1.DedicatedGameServer, error) {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return nil, err
	}

	set := labels.Set{
		LabelGameServerState: GameServerStateMarkedForDeletion,
		LabelActivePlayers:   "0",
	}
	// we seach via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsToDelete, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})

	fmt.Printf("LALA2:%v\n", dgsToDelete)

	if err != nil {
		return nil, err
	}

	return dgsToDelete.Items, nil
}

func GetDedicatedGameServersPodStateRunning() ([]dgsv1alpha1.DedicatedGameServer, error) {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return nil, err
	}

	set := labels.Set{
		//LabelGameServerState: GameServerStateRunning,
		LabelPodState: PodStateRunning,
	}
	// we seach via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsPodStateRunning, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	return dgsPodStateRunning.Items, nil
}
