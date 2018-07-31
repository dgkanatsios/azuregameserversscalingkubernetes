package shared

import (
	"flag"
	"os"
	"path/filepath"
	goruntime "runtime"

	log "github.com/sirupsen/logrus"

	dgsclientsetversioned "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var clientSet *kubernetes.Clientset
var dgsClientSet *dgsclientsetversioned.Clientset

// GetClientSet returns a Kubernetes interface object that will allow us to give commands to the K8s API
func GetClientSet() (*kubernetes.Clientset, *dgsclientsetversioned.Clientset, error) {

	//do they exist in cache? if yes, return them
	if clientSet != nil && dgsClientSet != nil {
		return clientSet, dgsClientSet, nil
	}

	//if we're running inside the cluster
	if runInK8s := os.Getenv("RUN_IN_K8S"); runInK8s == "" || runInK8s == "true" {
		config, err := rest.InClusterConfig()

		if err != nil {
			log.Println(err)
			return nil, nil, err
		}

		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Println(err)
			return nil, nil, err
		}

		dgsClientSet, err := dgsclientsetversioned.NewForConfig(config)
		if err != nil {
			log.Fatalf("getClusterConfig: %v", err)
		}

		return clientSet, dgsClientSet, nil
	}

	//else...
	var err error
	clientSet, dgsClientSet, err = getClientOutOfCluster()
	if err != nil {
		return nil, nil, err
	}
	return clientSet, dgsClientSet, nil

}

func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join(userHomeDir(), ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

func getClientOutOfCluster() (*kubernetes.Clientset, *dgsclientsetversioned.Clientset, error) {
	config, err := buildOutOfClusterConfig()
	if err != nil {
		log.Fatalf("Can not get kubernetes config: %v", err)
		return nil, nil, err
	}

	clientSet, err = kubernetes.NewForConfig(config)

	if err != nil {
		log.Fatalf("Can not create clientset for config: %v", err)
		return nil, nil, err
	}

	dgsClientSet, err = dgsclientsetversioned.NewForConfig(config)
	if err != nil {
		log.Fatalf("GetClientOutOfCluster: %v", err)
		return nil, nil, err
	}

	return clientSet, dgsClientSet, nil
}

func createKubeConfig() *string {
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
