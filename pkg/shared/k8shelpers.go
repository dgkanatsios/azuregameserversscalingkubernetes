package shared

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDedicatedGameServerCollection creates a new DedicatedGameServerCollection with the specified parameters
// Initial state is DGSHealth: creating, DGSState: Idle and PodPhase: pending
func NewDedicatedGameServerCollection(name string, namespace string, replicas int32, template corev1.PodSpec) *dgsv1alpha1.DedicatedGameServerCollection {
	dedicatedgameservercollection := &dgsv1alpha1.DedicatedGameServerCollection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: dgsv1alpha1.DedicatedGameServerCollectionSpec{
			Replicas: replicas,
			Template: template,
		},
		Status: dgsv1alpha1.DedicatedGameServerCollectionStatus{
			DGSCollectionHealth: dgsv1alpha1.DGSColCreating,
			PodCollectionState:  corev1.PodPending,
		},
	}
	return dedicatedgameservercollection
}

// NewDedicatedGameServer returns a new DedicatedGameServer object that belongs to the specified
// DedicatedGameServerCollection and has the designated PodSpec
func NewDedicatedGameServer(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, template corev1.PodSpec) *dgsv1alpha1.DedicatedGameServer {
	initialHealth := dgsv1alpha1.DGSCreating // dgsv1alpha1.DedicatedGameServerStateRunning //TODO: change to Creating
	initialState := dgsv1alpha1.DGSIdle
	dedicatedgameserver := &dgsv1alpha1.DedicatedGameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateName(dgsCol.Name),
			Namespace: dgsCol.Namespace,
			Labels:    map[string]string{LabelDedicatedGameServerCollectionName: dgsCol.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dgsCol, schema.GroupVersionKind{
					Group:   dgsv1alpha1.SchemeGroupVersion.Group,
					Version: dgsv1alpha1.SchemeGroupVersion.Version,
					Kind:    "DedicatedGameServerCollection",
				}),
			},
		},
		Spec: dgsv1alpha1.DedicatedGameServerSpec{
			Template:      *template.DeepCopy(),
			PortsToExpose: dgsCol.Spec.PortsToExpose,
		},
		Status: dgsv1alpha1.DedicatedGameServerStatus{
			Health:        initialHealth,
			DGSState:      initialState,
			ActivePlayers: 0,
		},
	}

	return dedicatedgameserver
}

// NewDedicatedGameServerWithNoParent creates a new DedicatedGameServer that is not part of a DedicatedGameServerCollection
func NewDedicatedGameServerWithNoParent(namespace string, name string, template corev1.PodSpec, portsToExpose []int32) *dgsv1alpha1.DedicatedGameServer {
	initialHealth := dgsv1alpha1.DGSCreating // dgsv1alpha1.DedicatedGameServerStateRunning //TODO: change to Creating
	initialState := dgsv1alpha1.DGSIdle
	dedicatedgameserver := &dgsv1alpha1.DedicatedGameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: dgsv1alpha1.DedicatedGameServerSpec{
			Template:      *template.DeepCopy(),
			PortsToExpose: portsToExpose,
		},
		Status: dgsv1alpha1.DedicatedGameServerStatus{
			Health:        initialHealth,
			DGSState:      initialState,
			ActivePlayers: 0,
		},
	}

	return dedicatedgameserver
}

// APIDetails contains the information that allows our DedicatedGameServer to communicate with the API Server
type APIDetails struct {
	APIServerURL string
	Code         string
}

// NewPod returns a Kubernetes Pod struct
// It also sets a label called "DedicatedGameServer" with the value of the corresponding DedicatedGameServer resource
func NewPod(dgs *dgsv1alpha1.DedicatedGameServer, apiDetails APIDetails) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			//GenerateName: dgs.Name + "-",
			Name:      generateName(dgs.Name),
			Namespace: dgs.Namespace,
			Labels:    map[string]string{LabelDedicatedGameServerName: dgs.Name, LabelIsDedicatedGameServer: "true"},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dgs, schema.GroupVersionKind{
					Group:   dgsv1alpha1.SchemeGroupVersion.Group,
					Version: dgsv1alpha1.SchemeGroupVersion.Version,
					Kind:    DedicatedGameServerKind,
				}),
			},
		},
		Spec: dgs.Spec.Template,
	}

	for i := 0; i < len(pod.Spec.Containers); i++ {
		// assign special ENV
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: "SERVER_NAME", Value: dgs.Name})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: "SERVER_NAMESPACE", Value: dgs.Namespace})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: "API_SERVER_URL", Value: apiDetails.APIServerURL})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: "API_SERVER_CODE", Value: apiDetails.Code})
	}

	pod.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet //https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	return pod
}

// UpdateActivePlayers updates the active players count for the server with name serverName
func UpdateActivePlayers(serverName string, namespace string, activePlayers int) error {
	return UpdateDGSStatus(serverName, namespace, DGSStatusFields{
		ActivePlayers: &activePlayers,
	})
}

func UpdateGameServerMarkedForDeletion(serverName string, namespace string, markedForDeletion bool) error {
	return UpdateDGSStatus(serverName, namespace, DGSStatusFields{
		MarkedForDeletion: &markedForDeletion,
	})
}

// UpdateGameServerState updates the DedicatedGameServer with the serverName state
func UpdateGameServerState(serverName string, namespace string, serverState dgsv1alpha1.DGSState) error {
	return UpdateDGSStatus(serverName, namespace, DGSStatusFields{
		DGSState: &serverState,
	})
}

// UpdateGameServerHealth updates the DedicatedGameServer with the serverName health
func UpdateGameServerHealth(serverName string, namespace string, serverHealth dgsv1alpha1.DGSHealth) error {
	return UpdateDGSStatus(serverName, namespace, DGSStatusFields{
		DGSHealth: &serverHealth,
	})
}

type DGSStatusFields struct {
	MarkedForDeletion *bool
	DGSHealth         *dgsv1alpha1.DGSHealth
	PodPhase          *corev1.PodPhase
	DGSState          *dgsv1alpha1.DGSState
	ActivePlayers     *int
}

func UpdateDGSStatus(serverName string, namespace string, fields DGSStatusFields) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, dgsClient, err := GetClientSet()
		if err != nil {
			return err
		}
		dgs, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Get(serverName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if fields.DGSHealth != nil {
			dgs.Status.Health = *fields.DGSHealth
		}
		if fields.MarkedForDeletion != nil {
			dgs.Status.MarkedForDeletion = *fields.MarkedForDeletion
		}
		if fields.PodPhase != nil {
			dgs.Status.PodPhase = *fields.PodPhase
		}

		_, err = dgsClient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Update(dgs)
		if err != nil {
			return err
		}
		return nil
	})
	return retryErr
}

// GetReadyDGSs returns a list of DGS that are "PodRunning", "Healthy" and not "MarkedForDeletion"
func GetReadyDGSs() ([]dgsv1alpha1.DedicatedGameServer, error) {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return nil, err
	}

	dgss, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	dgsToReturn := make([]dgsv1alpha1.DedicatedGameServer, 0)

	for _, dgs := range dgss.Items {
		if dgs.Status.Health == dgsv1alpha1.DGSHealthy &&
			dgs.Status.PodPhase == corev1.PodRunning &&
			!dgs.Status.MarkedForDeletion {
			dgsToReturn = append(dgsToReturn, dgs)
		}
	}

	return dgsToReturn, nil
}
