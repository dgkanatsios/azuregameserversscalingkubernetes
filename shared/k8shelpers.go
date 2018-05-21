package shared

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime/schema"

	dgs_v1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/apis/dedicatedgameserver/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func NewDedicatedGameServer(name string, port int32, setSessionsURL string, startmap string, image string) *dgs_v1.DedicatedGameServer {
	dedicatedgameserver := &dgs_v1.DedicatedGameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"server": name},
		},
		Spec: dgs_v1.DedicatedGameServerSpec{
			Image:    image,
			Port:     port,
			StartMap: startmap,
		},
	}
	return dedicatedgameserver
}

// NewPod returns a Kubernetes Pod struct
func NewPod(dgs *dgs_v1.DedicatedGameServer, setSessionsURL string) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   dgs.Name,
			Labels: map[string]string{"server": dgs.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dgs, schema.GroupVersionKind{
					Group:   dgs_v1.SchemeGroupVersion.Group,
					Version: dgs_v1.SchemeGroupVersion.Version,
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
							ContainerPort: dgs.Spec.Port,
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
							Name:  "SET_SESSIONS_URL",
							Value: setSessionsURL,
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

// GetClientSet returns a Kubernetes interface object that will allow us to give commands to the K8s API
func GetClientSet() (*kubernetes.Clientset, *dgsclientset.Clientset) {
	if runInK8s := os.Getenv("RUN_IN_K8S"); runInK8s == "" || runInK8s == "true" {
		config, err := rest.InClusterConfig()

		if err != nil {
			log.Println(err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Println(err)
		}

		dedicatedgameserverclientset, err := dgsclientset.NewForConfig(config)
		if err != nil {
			log.Fatalf("getClusterConfig: %v", err)
		}

		return clientset, dedicatedgameserverclientset
	}

	//else...
	clientset, dedicatedgameserverclientset := GetClientOutOfCluster()
	return clientset, dedicatedgameserverclientset

}

func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = "/home/dgkanatsios/.kube/config"
		//kubeconfigPath = "C:\\users\\dgkanatsios\\.kube\\config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetClientOutOfCluster returns a k8s clientset to the request from outside of cluster
func GetClientOutOfCluster() (*kubernetes.Clientset, *dgsclientset.Clientset) {
	config, err := buildOutOfClusterConfig()
	if err != nil {
		log.Fatalf("Can not get kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Fatalf("Can not create clientset for config: %v", err)
	}

	dedicatedgameserverclientset, err := dgsclientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("GetClientOutOfCluster: %v", err)
	}

	return clientset, dedicatedgameserverclientset
}

//CreateKubeConfig authenticates to the local cluster
func CreateKubeConfig() *string {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	return kubeconfig
}
