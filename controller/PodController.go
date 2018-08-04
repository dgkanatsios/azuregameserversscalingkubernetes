package controller

import (
	"fmt"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	typeddgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/typed/azuregaming/v1alpha1"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const podControllerAgentName = "pod-controller"

type PodController struct {
	dgsColClient       typeddgsv1alpha1.DedicatedGameServerCollectionsGetter
	dgsClient          typeddgsv1alpha1.DedicatedGameServersGetter
	podClient          typedcorev1.PodsGetter
	nodeClient         typedcorev1.NodesGetter
	dgsColLister       listerdgs.DedicatedGameServerCollectionLister
	dgsLister          listerdgs.DedicatedGameServerLister
	podLister          listercorev1.PodLister
	nodeLister         listercorev1.NodeLister
	dgsColListerSynced cache.InformerSynced
	dgsListerSynced    cache.InformerSynced
	podListerSynced    cache.InformerSynced
	nodeListerSynced   cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewPodController(client *kubernetes.Clientset, dgsclient *dgsclientset.Clientset, dgsColInformer informerdgs.DedicatedGameServerCollectionInformer,
	dgsInformer informerdgs.DedicatedGameServerInformer, podInformer informercorev1.PodInformer, nodeInformer informercorev1.NodeInformer) *PodController {
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	// Create event broadcaster
	log.Info("Creating event broadcaster for Pod controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: podControllerAgentName})

	c := &PodController{
		dgsColClient:       dgsclient.AzuregamingV1alpha1(),
		dgsClient:          dgsclient.AzuregamingV1alpha1(),
		podClient:          client.CoreV1(), //getter hits the live API server (can also create/update objects)
		nodeClient:         client.CoreV1(),
		dgsColLister:       dgsColInformer.Lister(),
		dgsLister:          dgsInformer.Lister(),
		podLister:          podInformer.Lister(), //lister hits the cache
		nodeLister:         nodeInformer.Lister(),
		dgsColListerSynced: dgsColInformer.Informer().HasSynced,
		dgsListerSynced:    dgsInformer.Informer().HasSynced,
		podListerSynced:    podInformer.Informer().HasSynced,
		nodeListerSynced:   nodeInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "PodSync"),
		recorder:           recorder,
	}

	log.Info("Setting up event handlers for Pod Controller")

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.handlePod(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newPod := newObj.(*corev1.Pod)
				oldPod := oldObj.(*corev1.Pod)
				if newPod.ResourceVersion == oldPod.ResourceVersion {
					// Periodic resync will send update events for all known Pods.
					// Two different versions of the same Pod will always have different RVs.
					return
				}
				c.handlePod(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				c.handlePod(obj)
			},
		},
	)

	return c
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *PodController) RunWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	log.Info("Starting loop for Pod controller")
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *PodController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// DedicatedGameServer resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		log.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func podBelongsToDedicatedGameServer(pod *corev1.Pod) bool {
	podLabels := pod.ObjectMeta.Labels

	for labelKey := range podLabels {
		if labelKey == shared.LabelDedicatedGameServer {
			return true
		}
	}
	return false
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Pod resource
// with the current status of the resource.
func (c *PodController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)

	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// try to get the Pod
	pod, err := c.podLister.Pods(namespace).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			// Pod not found, already deleted from the cluster
			runtime.HandleError(fmt.Errorf("Pod '%s' in work queue no longer exists", key))

			//unregister ports
			portRegistry.DeregisterServerPorts(name)

			return nil
		}

		log.Printf("Error in pod listing: %s", err.Error())
		c.recorder.Event(pod, corev1.EventTypeWarning, "Error in getting the Pod", err.Error())
		return err
	}

	// get the Node for this Pod
	nodeName := pod.Spec.NodeName
	var ip string
	if nodeName != "" { //no-empty string => pod has been scheduled
		ip, err = c.getPublicIPForNode(nodeName)

		if err != nil {
			log.Print(err.Error())
			c.recorder.Event(pod, corev1.EventTypeWarning, "Error in getting Public IP for the Node", err.Error())
			return err
		}
	}

	//dgsLister was used here but on update, the API server complained that the resource had changed
	//"the object has been modified; please apply your changes to the latest version and try again"
	dgsTemp, err := c.dgsLister.DedicatedGameServers(namespace).Get(name)
	dgsToUpdate := dgsTemp.DeepCopy()

	if err != nil {

		if errors.IsNotFound(err) {
			log.Infof("Dedicated Game Server %s not found, probably deleted already. Exiting PodController syncHandler", name)
			return nil
		}

		c.recorder.Event(pod, corev1.EventTypeWarning, fmt.Sprintf("Error in getting the DedicatedGameServer for pod %s", name), err.Error())
		return err
	}

	//update the DGS

	log.Printf("Updating DGS %s with PodState %s, Public IP %s, NodeName %s. Current PodState %s, Public IP %s, NodeName %s \n", dgsTemp.Name, pod.Status.Phase, ip, nodeName, dgsTemp.Status.PodState, dgsTemp.Spec.PublicIP, dgsTemp.Spec.NodeName)

	dgsToUpdate.Status.PodState = string(pod.Status.Phase)
	dgsToUpdate.Labels[shared.LabelPodState] = string(pod.Status.Phase)

	dgsToUpdate.Spec.PublicIP = ip
	dgsToUpdate.Spec.NodeName = nodeName

	_, err = c.dgsClient.DedicatedGameServers(namespace).Update(dgsToUpdate)

	if err != nil {
		log.Error(err.Error())
		c.recorder.Event(pod, corev1.EventTypeWarning, fmt.Sprintf("Error in updating the DedicatedGameServer for pod %s", name), err.Error())
		return err
	}

	//update the DGSCol
	if len(dgsTemp.OwnerReferences) > 0 {
		dgsCol, err := c.dgsColLister.DedicatedGameServerCollections(namespace).Get(dgsToUpdate.OwnerReferences[0].Name)
		dgsColCopy := dgsCol.DeepCopy()
		if err != nil {
			log.Error(err.Error())
			c.recorder.Event(pod, corev1.EventTypeWarning, fmt.Sprintf("Error in getting the DedicatedGameServerCollection for pod %s", name), err.Error())
			return err
		}
		err = c.assignPodCollectionState(namespace, dgsColCopy, pod)
		if err != nil {
			log.Error(err.Error())
			c.recorder.Event(pod, corev1.EventTypeWarning, fmt.Sprintf("Error in setting the DedicatedGameServerCollection - PodState for pod %s", name), err.Error())
			return err
		}

		_, err = c.dgsColClient.DedicatedGameServerCollections(namespace).Update(dgsColCopy)
		if err != nil {
			log.Error(err.Error())
			c.recorder.Event(pod, corev1.EventTypeWarning, fmt.Sprintf("Error in updating the DedicatedGameServerCollection for pod %s", name), err.Error())
			return err
		}
	}

	c.recorder.Event(pod, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageResourceSynced, "Pod", pod.Name))
	return nil
}

func (c *PodController) assignPodCollectionState(namespace string,
	dgsCol *dgsv1alpha1.DedicatedGameServerCollection, pod *corev1.Pod) error {
	// if this Pod state is running, then we should check whether
	// ALL the Pod in the Collection have the running state
	// if true, then DGSCol Pod state is running
	// else DGSCol Pod state is equal to this Pod State
	if pod.Status.Phase == corev1.PodRunning {

		// get all the DGS instances for this collection
		set := labels.Set{
			shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
		}
		selector := labels.SelectorFromSet(set)
		dgsInstances, err := c.dgsLister.DedicatedGameServers(namespace).List(selector)

		if err != nil {
			log.Error("Cannot get DGS instances")
			return err
		}

		for _, dgsInstance := range dgsInstances {
			// get this DedicatedGameServer's corresponding pod
			podInstance, err := c.podLister.Pods(namespace).Get(dgsInstance.Name)
			if err != nil {
				log.Error("Cannot get Pod instance")
				return err
			}
			if podInstance.Status.Phase != corev1.PodRunning {
				dgsCol.Status.PodCollectionState = string(podInstance.Status.Phase)
				return nil
			}
		}
		// end of the loop, so all Pods are "running"
		dgsCol.Status.PodCollectionState = shared.PodCollectionRunning
		return nil
	}
	dgsCol.Status.PodCollectionState = string(pod.Status.Phase)
	return nil
}

func (c *PodController) handlePod(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding Pod object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding Pod object tombstone, invalid type"))
			return
		}
		log.Infof("Recovered deleted Pod object '%s' from tombstone", object.GetName())
	}

	pod := object.(*corev1.Pod)
	if !podBelongsToDedicatedGameServer(pod) {
		log.Printf("Ignoring non-DedicatedGameServer pod %s", pod.Name)
		return
	}

	c.enqueuePod(pod)
}

// enqueuePod takes a Pod resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Pod.
func (c *PodController) enqueuePod(obj interface{}) {

	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *PodController) Workqueue() workqueue.RateLimitingInterface {
	return c.workqueue
}

func (c *PodController) ListersSynced() []cache.InformerSynced {
	return []cache.InformerSynced{c.podListerSynced, c.dgsListerSynced, c.dgsColListerSynced, c.nodeListerSynced}
}

func (c *PodController) getPublicIPForNode(nodeName string) (string, error) {
	node, err := c.nodeLister.Get(nodeName)
	if err != nil {
		return "", err
	}

	for _, x := range node.Status.Addresses {
		if x.Type == corev1.NodeExternalIP {
			return x.Address, nil
		}
	}

	return "", fmt.Errorf("node with name %s does not have a Public IP", nodeName)
}
