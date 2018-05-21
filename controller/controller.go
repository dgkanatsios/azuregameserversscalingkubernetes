package controller

//based on Kubernetes sample-controller sample
//https://github.com/kubernetes/sample-controller/blob/master/controller.go

import (
	"fmt"
	"strconv"

	"time"

	log "github.com/Sirupsen/logrus"

	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"

	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	apidgsv1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/apis/dedicatedgameserver/v1"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned"
	dgsv1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned/typed/dedicatedgameserver/v1"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/informers/externalversions/dedicatedgameserver/v1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/listers/dedicatedgameserver/v1"

	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned/scheme"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const namespace = apiv1.NamespaceDefault
const controllerAgentName = "dedigated-game-server-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a DedicatedGameServer is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a DedicatedGameServer fails
	// to sync due to a Pod of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Pod already existing
	MessageResourceExists = "Resource %q already exists and is not managed by DedicatedGameServer"
	// MessageResourceSynced is the message used for an Event fired when a DedicatedGameServer
	// is synced successfully
	MessageResourceSynced = "DedicatedGameServer synced successfully"
)

type DedicatedGameServerController struct {
	nodeGetter            typedcorev1.NodesGetter
	podGetter             typedcorev1.PodsGetter
	dgsGetter             dgsv1.DedicatedGameServersGetter
	nodeLister            listercorev1.NodeLister
	podLister             listercorev1.PodLister
	dgsLister             listerdgs.DedicatedGameServerLister
	podListerSynced       cache.InformerSynced
	dgsListerSynced       cache.InformerSynced
	namespaceGetter       typedcorev1.NamespacesGetter
	namespaceLister       listercorev1.NamespaceLister
	namespaceListerSynced cache.InformerSynced

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

func NewDedicatedGameServerController(client *kubernetes.Clientset, dgsclient *dgsclientset.Clientset,
	podInformer informercorev1.PodInformer, nodeInformer informercorev1.NodeInformer, dgsInformer informerdgs.DedicatedGameServerInformer) *DedicatedGameServerController {

	// Create event broadcaster
	// Add DedicatedGameServerController types to the default Kubernetes Scheme so Events can be
	// logged for DedicatedGameServerController types.
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	log.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(dgsscheme.Scheme, apiv1.EventSource{Component: controllerAgentName})

	c := &DedicatedGameServerController{
		nodeGetter:      client.CoreV1(),
		podGetter:       client.CoreV1(), //getter hits the live API server (can also create/update objects)
		dgsGetter:       dgsclient.AzureV1(),
		nodeLister:      nodeInformer.Lister(),
		podLister:       podInformer.Lister(), //lister hits the cache
		dgsLister:       dgsInformer.Lister(),
		podListerSynced: podInformer.Informer().HasSynced,
		dgsListerSynced: dgsInformer.Informer().HasSynced,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DedicatedGameServerSync"),
		recorder:        recorder,
	}

	log.Info("Setting up event handlers")

	nodeInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				//do nothing
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newNode := newObj.(*apiv1.Node)
				oldNode := oldObj.(*apiv1.Node)
				if newNode.ResourceVersion == oldNode.ResourceVersion {
					// Periodic resync will send update events for all known objects.
					// Two different versions of the same Deployment will always have different RVs.
					return
				}
				//do nothing
			},
			DeleteFunc: func(obj interface{}) {
				//do nothing
			},
		},
	)

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.handlePod(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newPod := newObj.(*apiv1.Pod)
				oldPod := oldObj.(*apiv1.Pod)
				if newPod.ResourceVersion == oldPod.ResourceVersion {
					// Periodic resync will send update events for all known Pods.
					// Two different versions of the same Deployment will always have different RVs.
					return
				}
				c.handlePod(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				c.handlePod(obj)
			},
		},
	)

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.enqueueDedicatedGameServer(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.enqueueDedicatedGameServer(newObj)
			},
		},
	)

	return c
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *DedicatedGameServerController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting DedicatedGameServer controller")

	// Wait for the caches to be synced before starting workers
	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.dgsListerSynced, c.podListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Info("Starting workers")
	// Launch two workers to process DedicatedGameServer resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *DedicatedGameServerController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *DedicatedGameServerController) processNextWorkItem() bool {
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

// handleObject will take any resource implementing metav1.Object and attempt
// to find the DedicatedGameServer resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that DedicatedGameServer resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *DedicatedGameServerController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		log.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	log.Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a DedicatedGameServer, we should not do anything more
		// with it.
		if ownerRef.Kind != "DedicatedGameServer" {
			return
		}

		dgs, err := c.dgsLister.DedicatedGameServers(namespace).Get(ownerRef.Name)
		if err != nil {
			log.Infof("ignoring orphaned object '%s' of DedicatedGameServer '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueDedicatedGameServer(dgs)
		return
	}
}

// enqueueDedicatedGameServer takes a DedicatedGameServer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServer.
func (c *DedicatedGameServerController) enqueueDedicatedGameServer(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *DedicatedGameServerController) handlePod(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		log.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	log.Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a DedicatedGameServer, we should not do anything more
		// with it.
		if ownerRef.Kind != "DedicatedGameServer" {
			return
		}

		_, err := c.dgsLister.DedicatedGameServers(namespace).Get(ownerRef.Name)
		if err != nil {
			log.Infof("Ignoring orphaned pod '%s' of DedicatedGameServer '%s'.Deleting it from table storage", object.GetSelfLink(), ownerRef.Name)
			err := shared.DeleteEntity(object.GetName())
			if err != nil {
				log.Errorf("Cannot delete pod %s entry from table storage due to %s", object.GetName(), err.Error())
			}
			return
		}

		c.enqueuePod(obj.(*apiv1.Pod))
		return
	}
}

func (c *DedicatedGameServerController) enqueuePod(pod *apiv1.Pod) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(pod); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DedicatedGameServer resource
// with the current status of the resource.
func (c *DedicatedGameServerController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	dgs, err := c.dgsLister.DedicatedGameServers(namespace).Get(name)
	if err != nil {
		//DedicatedGameServer no longer exists, no need to process any more
		if errors.IsNotFound(err) {

			runtime.HandleError(fmt.Errorf("Dedicated Game Server '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	pod, err := c.podLister.Pods(namespace).Get(dgs.Name)

	// If the pod doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		// Get the Dedicated Game Server with the same name as the Pod
		newPod := shared.NewPod(dgs, "TODO:SETSESSIONSURL")
		pod, err = c.podGetter.Pods(namespace).Create(newPod)

		//add stuff to table storage for the newly created Pod
		podEntity := &shared.StorageEntity{
			Name:   dgs.Name,
			Status: string(shared.CreatingState),
			Port:   strconv.Itoa(int(dgs.Spec.Port)),
		}
		shared.UpsertEntity(podEntity)
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Pod is not controlled by this DedicatedGameServer resource, we should log
	// a warning to the event recorder and return
	if !metav1.IsControlledBy(pod, dgs) {
		msg := fmt.Sprintf(MessageResourceExists, pod.Name)
		c.recorder.Event(dgs, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. THis could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the DedicatedGameServer resource to reflect the
	// current state of the world
	err = c.updatePodStatus(dgs, pod)
	if err != nil {
		return err
	}

	c.recorder.Event(dgs, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *DedicatedGameServerController) updatePodStatus(dgs *apidgsv1.DedicatedGameServer, pod *apiv1.Pod) error {

	//not sure if this is needed, but just in case
	if pod.Status.Phase == "Terminating" || pod.Status.Phase == "Failed" {
		return shared.DeleteEntity(pod.Name)
	}

	podEntity := &shared.StorageEntity{
		Name:     pod.Name,
		NodeName: pod.Spec.NodeName,
		Status:   string(pod.Status.Phase),
	}

	if pod.Status.Phase == "Running" { //only update PublicIP for a running Node
		podEntity.PublicIP = c.getNodeExternalIP(pod.Spec.NodeName)
	}

	var err error
	if err = shared.UpsertEntity(podEntity); err != nil {
		log.Errorf("Cannot upsert entity due to %s", err.Error())
	}
	return err
}

func (c *DedicatedGameServerController) getNodeExternalIP(nodename string) string {

	node, err := c.nodeLister.Get(nodename)
	if err != nil {
		log.Errorf("Error in getting IP for node %s because of error %s", nodename, err.Error())
		return ""
	}
	for _, address := range node.Status.Addresses {
		if address.Type == apiv1.NodeExternalIP {
			return address.Address
		}
	}
	return ""
}
