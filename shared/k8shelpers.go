package shared

import (
	"strconv"

	"k8s.io/apimachinery/pkg/runtime/schema"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDedicatedGameServerCollection(name string, startmap string, image string, replicas int32) *dgsv1alpha1.DedicatedGameServerCollection {
	dedicatedgameservercollection := &dgsv1alpha1.DedicatedGameServerCollection{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{DedicatedGameServerCollectionNameLabel: name},
		},
		Spec: dgsv1alpha1.DedicatedGameServerCollectionSpec{
			Image:    image,
			Replicas: replicas,
			StartMap: startmap,
		},
	}
	return dedicatedgameservercollection
}

func NewDedicatedGameServer(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, name string, port int, startmap string, image string) *dgsv1alpha1.DedicatedGameServer {
	dedicatedgameserver := &dgsv1alpha1.DedicatedGameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{ServerNameLabel: name, DedicatedGameServerCollectionNameLabel: dgsCol.Name},
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
			Port:     port,
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
			Labels: map[string]string{DedicatedGameServerLabel: dgs.Name},
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
					Name:  "dedicatedgameserver",
					Image: dgs.Spec.Image,
					Ports: []core.ContainerPort{
						{
							Name:          "port1",
							Protocol:      core.ProtocolUDP,
							ContainerPort: int32(dgs.Spec.Port),
						},
					},
					Env: []core.EnvVar{
						{
							Name:  "OA_STARTMAP",
							Value: dgs.Spec.StartMap,
						},
						{
							Name:  "OA_PORT",
							Value: strconv.Itoa(int(dgs.Spec.Port)),
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
			HostNetwork:   true,
			DNSPolicy:     core.DNSClusterFirstWithHostNet, //https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
			RestartPolicy: core.RestartPolicyNever,
		},
	}
	return pod
}
