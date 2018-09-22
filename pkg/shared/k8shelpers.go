package shared

import (
	"strconv"

	"k8s.io/apimachinery/pkg/runtime/schema"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NewDedicatedGameServerCollection creates a new DedicatedGameServerCollection with the specified parameters
// Initial state is DGSState: creating and PodState: pending
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
			DedicatedGameServerCollectionState: dgsv1alpha1.DedicatedGameServerCollectionStateCreating,
			PodCollectionState:                 corev1.PodPending,
		},
	}
	return dedicatedgameservercollection
}

// NewDedicatedGameServer returns a new DedicatedGameServer object that belongs to the specified
// DedicatedGameServerCollection and has the designated PodSpec
func NewDedicatedGameServer(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, template corev1.PodSpec, namegenerator RandomNameGenerator) *dgsv1alpha1.DedicatedGameServer {
	dgsName := namegenerator.GenerateName(dgsCol.Name)
	initialState := dgsv1alpha1.DedicatedGameServerStateCreating // dgsv1alpha1.DedicatedGameServerStateRunning //TODO: change to Creating
	dedicatedgameserver := &dgsv1alpha1.DedicatedGameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dgsName,
			Namespace: dgsCol.Namespace,
			Labels: map[string]string{LabelServerName: dgsName,
				LabelDedicatedGameServerCollectionName: dgsCol.Name,
				LabelActivePlayers:                     "0",
				LabelDedicatedGameServerState:          string(initialState)},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dgsCol, schema.GroupVersionKind{
					Group:   dgsv1alpha1.SchemeGroupVersion.Group,
					Version: dgsv1alpha1.SchemeGroupVersion.Version,
					Kind:    "DedicatedGameServerCollection",
				}),
			},
		},
		Spec: dgsv1alpha1.DedicatedGameServerSpec{
			Template: *template.DeepCopy(),
		},
		Status: dgsv1alpha1.DedicatedGameServerStatus{
			DedicatedGameServerState: initialState,
			ActivePlayers:            0,
		},
	}

	return dedicatedgameserver
}

// NewDedicatedGameServerWithNoParent creates a new DedicatedGameServer that is not part of a DedicatedGameServerCollection
func NewDedicatedGameServerWithNoParent(namespace string, namePrefix string, template corev1.PodSpec, namegenerator RandomNameGenerator) *dgsv1alpha1.DedicatedGameServer {
	dgsName := namegenerator.GenerateName(namePrefix)
	initialState := dgsv1alpha1.DedicatedGameServerStateCreating // dgsv1alpha1.DedicatedGameServerStateRunning //TODO: change to Creating
	dedicatedgameserver := &dgsv1alpha1.DedicatedGameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dgsName,
			Namespace: namespace,
			Labels: map[string]string{LabelServerName: dgsName,
				LabelActivePlayers:            "0",
				LabelDedicatedGameServerState: string(initialState)},
		},
		Spec: dgsv1alpha1.DedicatedGameServerSpec{
			Template: *template.DeepCopy(),
		},
		Status: dgsv1alpha1.DedicatedGameServerStatus{
			DedicatedGameServerState: initialState,
			ActivePlayers:            0,
		},
	}

	return dedicatedgameserver
}

// APIDetails contains the information that allows our DedicatedGameServer to communicate with the API Server
type APIDetails struct {
	SetActivePlayersURL string
	SetServerStatusURL  string
}

// NewPod returns a Kubernetes Pod struct
// It also sets a label called "DedicatedGameServer" with the value of the corresponding DedicatedGameServer resource
func NewPod(dgs *dgsv1alpha1.DedicatedGameServer, apiDetails APIDetails, namegenerator RandomNameGenerator) *corev1.Pod {
	podName := namegenerator.GenerateName(dgs.Name)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
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
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: "SET_ACTIVE_PLAYERS_URL", Value: apiDetails.SetActivePlayersURL})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: "SET_SERVER_STATUS_URL", Value: apiDetails.SetServerStatusURL})
	}

	pod.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet //https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	return pod
}

// UpdateActivePlayers updates the active players count for the server with name serverName
func UpdateActivePlayers(serverName string, activePlayers int) error {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return err
	}
	dgs, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Get(serverName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	activePlayersString := strconv.Itoa(activePlayers)

	dgs.Status.ActivePlayers = activePlayers
	dgs.Labels[LabelActivePlayers] = activePlayersString

	_, err = dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Update(dgs)
	if err != nil {
		return err
	}
	return nil
}

// UpdateGameServerStatus updates the DedicatedGameServer with the serverName status
func UpdateGameServerStatus(serverName string, serverStatus dgsv1alpha1.DedicatedGameServerState) error {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return err
	}
	dgs, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Get(serverName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	dgs.Status.DedicatedGameServerState = serverStatus
	dgs.Labels[LabelDedicatedGameServerState] = string(serverStatus)

	_, err = dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).Update(dgs)
	if err != nil {
		return err
	}
	return nil
}

func GetDedicatedGameServersPodStateRunning() ([]dgsv1alpha1.DedicatedGameServer, error) {
	_, dgsClient, err := GetClientSet()
	if err != nil {
		return nil, err
	}

	set := labels.Set{
		LabelDedicatedGameServerState: string(dgsv1alpha1.DedicatedGameServerStateRunning),
		LabelPodState:                 string(corev1.PodRunning),
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsPodStateRunning, err := dgsClient.AzuregamingV1alpha1().DedicatedGameServers(GameNamespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	return dgsPodStateRunning.Items, nil
}
