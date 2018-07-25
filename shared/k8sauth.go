package shared

import (
	"flag"
	"os"
	"path/filepath"
	goruntime "runtime"

	log "github.com/sirupsen/logrus"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetClientSet returns a Kubernetes interface object that will allow us to give commands to the K8s API
func GetClientSet() (*kubernetes.Clientset, *dgsclientset.Clientset) {
	//if we're running inside the cluster
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
		//kubeconfigPath = "/home/dgkanatsios/.kube/config"
		//kubeconfigPath = "C:\\users\\dgkanatsios\\.kube\\config"
		kubeconfigPath = filepath.Join(userHomeDir(), ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// GetClientOutOfCluster returns a k8s clientset to allow requests from outside of cluster
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

// get user home directory depending on OS
func userHomeDir() string {
	if goruntime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
