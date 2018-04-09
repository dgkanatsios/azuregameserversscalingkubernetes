package shared

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/joho/godotenv/autoload" // load env variables
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) //randomize name creation
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

// RandString creates a random string with lowercase characters
func RandString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetServiceNameFromPodName converts the name of the pod to a similar name for the service
func GetServiceNameFromPodName(podName string) string {
	return podName
}
