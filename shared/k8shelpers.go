package shared

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// CreatePod returns a Kubernetes Pod struct
func CreatePod(name string, port int32) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"server": name},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "gameserver",
					Image: "docker.io/dgkanatsios/docker_openarena_k8s:latest",
					Ports: []core.ContainerPort{
						{
							Name:          "port1",
							Protocol:      core.ProtocolUDP,
							ContainerPort: port,
						},
					},
					Env: []core.EnvVar{
						{
							Name:  "OA_STARTMAP",
							Value: "dm4ish",
						},
						{
							Name:  "OA_PORT",
							Value: "27960",
						},
					},
					EnvFrom: []core.EnvFromSource{
						{
							SecretRef: &core.SecretEnvSource{
								LocalObjectReference: core.LocalObjectReference{
									Name: "openarena-storage-secret",
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
		},
	}
	return pod
}

// CreateService returns a Kubernetes Service struct
func CreateService(name string, port int32) *core.Service {
	service := &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{{
				Name:     "port1",
				Protocol: core.ProtocolUDP,
				Port:     port,
			}},
			Selector: map[string]string{"server": name},
			Type:     "LoadBalancer",
		},
	}
	return service
}

// GetClientSet returns a Kubernetes interface object that will allow us to give commands to the K8s API
func GetClientSet() kubernetes.Interface {
	if runInK8s := os.Getenv("RUN_IN_K8S"); runInK8s == "" || runInK8s == "true" {
		config, err := rest.InClusterConfig()

		if err != nil {
			fmt.Println(err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Println(err)
		}
		return clientset
	}
	//else...
	clientset := GetClientOutOfCluster()
	return clientset

}

func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		//kubeconfigPath = "/.kube/config"
		kubeconfigPath = "C:\\users\\dgkanatsios\\.kube\\config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetClientOutOfCluster returns a k8s clientset to the request from outside of cluster
func GetClientOutOfCluster() kubernetes.Interface {
	config, err := buildOutOfClusterConfig()
	if err != nil {
		log.Fatalf("Can not get kubernetes config: %v", err)
	}

	clientset, err2 := kubernetes.NewForConfig(config)

	if err2 != nil {
		log.Fatalf("Can not create clientset for config: %v", err2)
	}

	return clientset
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
