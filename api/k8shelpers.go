package main

import (
	"github.com/dgkanatsios/AzureGameServersScalingKubernetes/shared"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func createPod(name string, port int32) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"server": name},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:  "web",
					Image: "nginx:1.12",
					Ports: []core.ContainerPort{
						{
							Name:          "http",
							Protocol:      core.ProtocolTCP,
							ContainerPort: port,
						},
					},
				},
			},
		},
	}
	return pod
}

func createService(name string, port int32) *core.Service {
	service := &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{{
				Name:     "port",
				Protocol: "UDP",
				Port:     port,
			}},
			Selector: map[string]string{"server": name},
			Type:     "LoadBalancer",
		},
	}
	return service
}

func getClientSet() kubernetes.Interface {
	// config, err := rest.InClusterConfig()

	// if err != nil {
	// 	fmt.Println(err)
	// }

	// clientset, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	clientset := shared.GetClientOutOfCluster()
	return clientset
}
