package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/dgkanatsios/AzureGameServersScalingKubernetes/shared"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func main() {

	namespace := apiv1.NamespaceDefault

	clientset := shared.GetClientSet()

	controllerPods := createPodController(clientset, namespace)
	controllerServices := createServiceController(clientset, namespace)

	stop := make(chan struct{})

	go controllerPods.Run(stop)
	go controllerServices.Run(stop)

	fmt.Println("Listening for Kubernetes events...")

	for {
		time.Sleep(time.Second)
	}
}

func createPodController(clientset kubernetes.Interface, namespace string) cache.Controller {
	watchlist := cache.NewListWatchFromClient(clientset.Core().RESTClient(), "pods", namespace, fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&apiv1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handlePodAdd(obj)
			},
			DeleteFunc: func(obj interface{}) {
				handlePodDelete(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handlePodUpdate(newObj)
			},
		},
	)
	return controller
}

func handlePodAdd(obj interface{}) {
	fmt.Println("Pod added:\n", obj)
	pod := obj.(*apiv1.Pod)
	name := pod.ObjectMeta.Name
	if !isOpenArena(name) {
		return
	}
	status := pod.Status.Phase
	shared.UpsertEntity(&shared.StorageEntity{
		Name:     name,
		NodeName: pod.Spec.NodeName,
		Status:   string(status),
	})
}

func handlePodDelete(obj interface{}) {
	fmt.Println("Pod deleted:\n", obj)
	pod := obj.(*apiv1.Pod)
	name := pod.ObjectMeta.Name
	if !isOpenArena(name) {
		return
	}
	shared.DeleteEntity(name)
}

func handlePodUpdate(obj interface{}) {
	fmt.Println("Pod updated: \n", obj)
	pod := obj.(*apiv1.Pod)
	name := pod.ObjectMeta.Name
	if !strings.HasPrefix(name, "openarena") {
		return
	}
	status := pod.Status.Phase
	shared.UpsertEntity(&shared.StorageEntity{
		Name:     name,
		NodeName: pod.Spec.NodeName,
		Status:   string(status),
	})

	if !isOpenArena(name) {
		fmt.Println("Pod ", name, " is now ", status)
	}
}

/*

Pod added:
 &Pod{ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{Name:openarena-xvlbzg,GenerateName:,Namespace:default,SelfLink:/api/v1/namespaces/default/pods/openarena-xvlbzg,UID:766b05ff-3820-11e8-96b8-00155d9f6611,ResourceVersion:488496,Generation:0,CreationTimestamp:2018-04-04 18:54:31 +0300 EEST,DeletionTimestamp:<nil>,DeletionGracePeriodSeconds:nil,Labels:map[string]string{},Annotations:map[string]string{},OwnerReferences:[],Finalizers:[],ClusterName:,Initializers:nil,},Spec:PodSpec{Volumes:[{default-token-ngjtm {nil nil nil nil nil SecretVolumeSource{SecretName:default-token-ngjtm,Items:[],DefaultMode:*420,Optional:nil,} nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil}}],Containers:[{web nginx:1.12 [] []  [{http 0 80 TCP }] [] [] {map[] map[]} [{default-token-ngjtm true /var/run/secrets/kubernetes.io/serviceaccount  <nil>}] [] nil nil nil /dev/termination-log File IfNotPresent nil false false false}],RestartPolicy:Always,TerminationGracePeriodSeconds:*30,ActiveDeadlineSeconds:nil,DNSPolicy:ClusterFirst,NodeSelector:map[string]string{},ServiceAccountName:default,DeprecatedServiceAccount:default,NodeName:,HostNetwork:false,HostPID:false,HostIPC:false,SecurityContext:&PodSecurityContext{SELinuxOptions:nil,RunAsUser:nil,RunAsNonRoot:nil,SupplementalGroups:[],FSGroup:nil,RunAsGroup:nil,},ImagePullSecrets:[],Hostname:,Subdomain:,Affinity:nil,SchedulerName:default-scheduler,InitContainers:[],AutomountServiceAccountToken:nil,Tolerations:[{node.kubernetes.io/not-ready Exists  NoExecute 0xc04203ac60} {node.kubernetes.io/unreachable Exists  NoExecute 0xc04203ac80}],HostAliases:[],PriorityClassName:,Priority:nil,DNSConfig:nil,ShareProcessNamespace:nil,},Status:PodStatus{Phase:Pending,Conditions:[],Message:,Reason:,HostIP:,PodIP:,StartTime:<nil>,ContainerStatuses:[],QOSClass:BestEffort,InitContainerStatuses:[],NominatedNodeName:,},}
Pod changed:
 &Pod{ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{Name:openarena-xvlbzg,GenerateName:,Namespace:default,SelfLink:/api/v1/namespaces/default/pods/openarena-xvlbzg,UID:766b05ff-3820-11e8-96b8-00155d9f6611,ResourceVersion:488497,Generation:0,CreationTimestamp:2018-04-04 18:54:31 +0300 EEST,DeletionTimestamp:<nil>,DeletionGracePeriodSeconds:nil,Labels:map[string]string{},Annotations:map[string]string{},OwnerReferences:[],Finalizers:[],ClusterName:,Initializers:nil,},Spec:PodSpec{Volumes:[{default-token-ngjtm {nil nil nil nil nil SecretVolumeSource{SecretName:default-token-ngjtm,Items:[],DefaultMode:*420,Optional:nil,} nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil}}],Containers:[{web nginx:1.12 [] []  [{http 0 80 TCP }] [] [] {map[] map[]} [{default-token-ngjtm true /var/run/secrets/kubernetes.io/serviceaccount  <nil>}] [] nil nil nil /dev/termination-log File IfNotPresent nil false false false}],RestartPolicy:Always,TerminationGracePeriodSeconds:*30,ActiveDeadlineSeconds:nil,DNSPolicy:ClusterFirst,NodeSelector:map[string]string{},ServiceAccountName:default,DeprecatedServiceAccount:default,NodeName:docker-for-desktop,HostNetwork:false,HostPID:false,HostIPC:false,SecurityContext:&PodSecurityContext{SELinuxOptions:nil,RunAsUser:nil,RunAsNonRoot:nil,SupplementalGroups:[],FSGroup:nil,RunAsGroup:nil,},ImagePullSecrets:[],Hostname:,Subdomain:,Affinity:nil,SchedulerName:default-scheduler,InitContainers:[],AutomountServiceAccountToken:nil,Tolerations:[{node.kubernetes.io/not-ready Exists  NoExecute 0xc04203b420} {node.kubernetes.io/unreachable Exists  NoExecute 0xc04203b440}],HostAliases:[],PriorityClassName:,Priority:nil,DNSConfig:nil,ShareProcessNamespace:nil,},Status:PodStatus{Phase:Pending,Conditions:[{PodScheduled True 0001-01-01 00:00:00 +0000 UTC 2018-04-04 18:54:31 +0300 EEST  }],Message:,Reason:,HostIP:,PodIP:,StartTime:<nil>,ContainerStatuses:[],QOSClass:BestEffort,InitContainerStatuses:[],NominatedNodeName:,},}
Pod changed:
 &Pod{ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{Name:openarena-xvlbzg,GenerateName:,Namespace:default,SelfLink:/api/v1/namespaces/default/pods/openarena-xvlbzg,UID:766b05ff-3820-11e8-96b8-00155d9f6611,ResourceVersion:488499,Generation:0,CreationTimestamp:2018-04-04 18:54:31 +0300 EEST,DeletionTimestamp:<nil>,DeletionGracePeriodSeconds:nil,Labels:map[string]string{},Annotations:map[string]string{},OwnerReferences:[],Finalizers:[],ClusterName:,Initializers:nil,},Spec:PodSpec{Volumes:[{default-token-ngjtm {nil nil nil nil nil SecretVolumeSource{SecretName:default-token-ngjtm,Items:[],DefaultMode:*420,Optional:nil,} nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil}}],Containers:[{web nginx:1.12 [] []  [{http 0 80 TCP }] [] [] {map[] map[]} [{default-token-ngjtm true /var/run/secrets/kubernetes.io/serviceaccount  <nil>}] [] nil nil nil /dev/termination-log File IfNotPresent nil false false false}],RestartPolicy:Always,TerminationGracePeriodSeconds:*30,ActiveDeadlineSeconds:nil,DNSPolicy:ClusterFirst,NodeSelector:map[string]string{},ServiceAccountName:default,DeprecatedServiceAccount:default,NodeName:docker-for-desktop,HostNetwork:false,HostPID:false,HostIPC:false,SecurityContext:&PodSecurityContext{SELinuxOptions:nil,RunAsUser:nil,RunAsNonRoot:nil,SupplementalGroups:[],FSGroup:nil,RunAsGroup:nil,},ImagePullSecrets:[],Hostname:,Subdomain:,Affinity:nil,SchedulerName:default-scheduler,InitContainers:[],AutomountServiceAccountToken:nil,Tolerations:[{node.kubernetes.io/not-ready Exists  NoExecute 0xc04203bc70} {node.kubernetes.io/unreachable Exists  NoExecute 0xc04203bc90}],HostAliases:[],PriorityClassName:,Priority:nil,DNSConfig:nil,ShareProcessNamespace:nil,},Status:PodStatus{Phase:Pending,Conditions:[{Initialized True 0001-01-01 00:00:00 +0000 UTC 2018-04-04 18:54:31 +0300 EEST  } {Ready False 0001-01-01 00:00:00 +0000 UTC 2018-04-04 18:54:31 +0300 EEST ContainersNotReady containers with unready status: [web]} {PodScheduled True 0001-01-01 00:00:00 +0000 UTC 2018-04-04 18:54:31 +0300 EEST  }],Message:,Reason:,HostIP:192.168.65.3,PodIP:,StartTime:2018-04-04 18:54:31 +0300 EEST,ContainerStatuses:[{web {ContainerStateWaiting{Reason:ContainerCreating,Message:,} nil nil} {nil nil nil} false 0 nginx:1.12  }],QOSClass:BestEffort,InitContainerStatuses:[],NominatedNodeName:,},}
Pod changed:
 &Pod{ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{Name:openarena-xvlbzg,GenerateName:,Namespace:default,SelfLink:/api/v1/namespaces/default/pods/openarena-xvlbzg,UID:766b05ff-3820-11e8-96b8-00155d9f6611,ResourceVersion:488507,Generation:0,CreationTimestamp:2018-04-04 18:54:31 +0300 EEST,DeletionTimestamp:<nil>,DeletionGracePeriodSeconds:nil,Labels:map[string]string{},Annotations:map[string]string{},OwnerReferences:[],Finalizers:[],ClusterName:,Initializers:nil,},Spec:PodSpec{Volumes:[{default-token-ngjtm {nil nil nil nil nil SecretVolumeSource{SecretName:default-token-ngjtm,Items:[],DefaultMode:*420,Optional:nil,} nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil nil}}],Containers:[{web nginx:1.12 [] []  [{http 0 80 TCP }] [] [] {map[] map[]} [{default-token-ngjtm true /var/run/secrets/kubernetes.io/serviceaccount  <nil>}] [] nil nil nil /dev/termination-log File IfNotPresent nil false false false}],RestartPolicy:Always,TerminationGracePeriodSeconds:*30,ActiveDeadlineSeconds:nil,DNSPolicy:ClusterFirst,NodeSelector:map[string]string{},ServiceAccountName:default,DeprecatedServiceAccount:default,NodeName:docker-for-desktop,HostNetwork:false,HostPID:false,HostIPC:false,SecurityContext:&PodSecurityContext{SELinuxOptions:nil,RunAsUser:nil,RunAsNonRoot:nil,SupplementalGroups:[],FSGroup:nil,RunAsGroup:nil,},ImagePullSecrets:[],Hostname:,Subdomain:,Affinity:nil,SchedulerName:default-scheduler,InitContainers:[],AutomountServiceAccountToken:nil,Tolerations:[{node.kubernetes.io/not-ready Exists  NoExecute 0xc042360540} {node.kubernetes.io/unreachable Exists  NoExecute 0xc042360560}],HostAliases:[],PriorityClassName:,Priority:nil,DNSConfig:nil,ShareProcessNamespace:nil,},Status:PodStatus{Phase:Running,Conditions:[{Initialized True 0001-01-01 00:00:00 +0000 UTC 2018-04-04 18:54:31 +0300 EEST  } {Ready True 0001-01-01 00:00:00 +0000 UTC 2018-04-04 18:54:33 +0300 EEST
 } {PodScheduled True 0001-01-01 00:00:00 +0000 UTC 2018-04-04 18:54:31 +0300 EEST  }],Message:,Reason:,HostIP:192.168.65.3,PodIP:10.1.0.43,StartTime:2018-04-04 18:54:31 +0300 EEST,ContainerStatuses:[{web {nil ContainerStateRunning{StartedAt:2018-04-04 18:54:32 +0300 EEST,} nil} {nil nil nil} true 0 nginx:1.12 docker-pullable://nginx@sha256:416134fd8b36457ee5dfdc08eb7271a30aa0ce0d8a1b55a6bcb9852f8f362630 docker://a4b855994078d81fae60b764ad1215fbf0c043ad8b3fd9e85b986ff1f09850af}],QOSClass:BestEffort,InitContainerStatuses:[],NominatedNodeName:,},}

*/

func createServiceController(clientset kubernetes.Interface, namespace string) cache.Controller {
	watchlist := cache.NewListWatchFromClient(clientset.Core().RESTClient(), "services", namespace, fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&apiv1.Service{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleServiceAdd(obj)
			},
			DeleteFunc: func(obj interface{}) {
				handleServiceDelete(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handleServiceUpdate(newObj)
			},
		},
	)
	return controller
}

func handleServiceAdd(obj interface{}) {
	fmt.Println("Service added:\n", obj)
	service := obj.(*apiv1.Service)
	name := service.ObjectMeta.Name
	if !isOpenArena(name) {
		return
	}
}

func handleServiceDelete(obj interface{}) {
	fmt.Println("Service deleted:\n", obj)
	service := obj.(*apiv1.Service)
	name := service.ObjectMeta.Name
	if !isOpenArena(name) {
		return
	}
}

func handleServiceUpdate(obj interface{}) {
	service := obj.(*apiv1.Service)
	name := service.ObjectMeta.Name
	if !isOpenArena(name) {
		return
	}
	var externalIP string
	if len(service.Status.LoadBalancer.Ingress) > 0 {
		externalIP = service.Status.LoadBalancer.Ingress[0].IP
		shared.UpsertEntity(&shared.StorageEntity{
			Name:     shared.GetPodNameFromServiceName(name),
			PublicIP: externalIP,
		})
	}

	fmt.Println("Service updated:\n", name, externalIP)
}

func isOpenArena(name string) bool {
	return strings.HasPrefix(name, "openarena")
}

/*

Service added:
 &Service{ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{Name:openarena-ywtirc,GenerateName:,Namespace:default,SelfLink:/api/v1/namespaces/default/services/openarena-yw
tirc,UID:09c17c33-3d8e-11e8-9630-0a58ac1f0661,ResourceVersion:267936,Generation:0,CreationTimestamp:2018-04-11 13:41:29 +0000 UTC,DeletionTimestamp:<nil>,DeletionGracePeriodSeconds
:nil,Labels:map[string]string{},Annotations:map[string]string{},OwnerReferences:[],Finalizers:[],ClusterName:,Initializers:nil,},Spec:ServiceSpec{Ports:[{port1 UDP 26105 {0 26105 }
 31961}],Selector:map[string]string{server: openarena-ywtirc,},ClusterIP:10.0.180.157,Type:LoadBalancer,ExternalIPs:[],SessionAffinity:None,LoadBalancerIP:,LoadBalancerSourceRanges
:[],ExternalName:,ExternalTrafficPolicy:Cluster,HealthCheckNodePort:0,PublishNotReadyAddresses:false,SessionAffinityConfig:nil,},Status:ServiceStatus{LoadBalancer:LoadBalancerStatu
s{Ingress:[],},},}


Service updated:
 openarena-ywtirc
&Service{ObjectMeta:k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta{Name:openarena-ywtirc,GenerateName:,Namespace:default,SelfLink:/api/v1/namespaces/default/services/openarena-ywt
irc,UID:09c17c33-3d8e-11e8-9630-0a58ac1f0661,ResourceVersion:267998,Generation:0,CreationTimestamp:2018-04-11 13:41:29 +0000 UTC,DeletionTimestamp:<nil>,DeletionGracePeriodSeconds:
nil,Labels:map[string]string{},Annotations:map[string]string{},OwnerReferences:[],Finalizers:[],ClusterName:,Initializers:nil,},Spec:ServiceSpec{Ports:[{port1 UDP 26105 {0 26105 }
31961}],Selector:map[string]string{server: openarena-ywtirc,},ClusterIP:10.0.180.157,Type:LoadBalancer,ExternalIPs:[],SessionAffinity:None,LoadBalancerIP:,LoadBalancerSourceRanges:
[],ExternalName:,ExternalTrafficPolicy:Cluster,HealthCheckNodePort:0,PublishNotReadyAddresses:false,SessionAffinityConfig:nil,},Status:ServiceStatus{LoadBalancer:LoadBalancerStatus
{Ingress:[{52.170.255.33 }],},},}

*/
