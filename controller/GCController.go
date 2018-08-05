package controller

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"

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
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const gcControllerAgentName = "garbage-collection-controller"

type GarbageCollectionController struct {
	dgsClient       typeddgsv1alpha1.DedicatedGameServersGetter
	dgsLister       listerdgs.DedicatedGameServerLister
	dgsListerSynced cache.InformerSynced

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

func NewGarbageCollectionController(client *kubernetes.Clientset, dgsclient *dgsclientset.Clientset,
	dgsInformer informerdgs.DedicatedGameServerInformer) *GarbageCollectionController {
	// Create event broadcaster
	// Add DedicatedGameServerController types to the default Kubernetes Scheme so Events can be
	// logged for DedicatedGameServerController types.
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	log.Info("Creating event broadcaster for DedicatedGameServer controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: dgsControllerAgentName})

	c := &GarbageCollectionController{
		dgsClient:       dgsclient.AzuregamingV1alpha1(),
		dgsLister:       dgsInformer.Lister(),
		dgsListerSynced: dgsInformer.Informer().HasSynced,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "GarbageCollectionSync"),
		recorder:        recorder,
	}

	log.Info("Setting up event handlers for GarbageCollection controller")

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("GarbageCollection controller - add")
				c.handleDedicatedGameServer(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("GarbageCollection controller - update")

				oldDGS := oldObj.(*dgsv1alpha1.DedicatedGameServer)
				newDGS := newObj.(*dgsv1alpha1.DedicatedGameServer)

				if oldDGS.ResourceVersion == newDGS.ResourceVersion {
					return
				}

				c.handleDedicatedGameServer(newObj)
			},
		},
	)

	return c
}

func (c *GarbageCollectionController) handleDedicatedGameServer(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding DedicatedGameServer object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding DedicatedGameServer object tombstone, invalid type"))
			return
		}
		log.Infof("Recovered deleted DedicatedGameServer object '%s' from tombstone", object.GetName())
	}

	c.enqueueDedicatedGameServer(object)
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *GarbageCollectionController) processNextWorkItem() bool {
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

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DedicatedGameServer resource
// with the current status of the resource.
func (c *GarbageCollectionController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// try to get the DedicatedGameServer
	dgs, err := c.dgsLister.DedicatedGameServers(namespace).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			// DedicatedGameServer not found
			runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' in work queue no longer exists", key))
			return nil
		}
		log.Print(err.Error())
		return err
	}

	//check its state and active players
	if dgs.Spec.ActivePlayers == "0" && dgs.Status.GameServerState == shared.GameServerStateMarkedForDeletion {
		dgsToDelete, err := c.dgsClient.DedicatedGameServers(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Cannot fetch DedicatedGameServer %s", name)
			// DedicatedGameServer not found
			runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' cannot be retrieved", name))
			return err
		}

		err = c.dgsClient.DedicatedGameServers(namespace).Delete(dgsToDelete.Name, &metav1.DeleteOptions{})

		if err != nil {
			log.Errorf("Cannot delete DedicatedGameServer %s", name)
			// DedicatedGameServer not found
			runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' cannot be deleted", name))
			return err
		}

		c.recorder.Event(dgs, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageMarkedForDeletionDedicatedGameServerDeleted, dgs.Name))
		return nil
	}

	return nil
}

// enqueueDedicatedGameServer takes a DedicatedGameServer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServer.
func (c *GarbageCollectionController) enqueueDedicatedGameServer(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *GarbageCollectionController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	log.Info("Starting loop for GarbageCollection controller")
	for c.processNextWorkItem() {
	}
}

func (c *GarbageCollectionController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting GarbageCollection controller")

	// Wait for the caches for all controllers to be synced before starting workers
	log.Info("Waiting for informer caches to sync for GarbageCollection controller")
	if ok := cache.WaitForCacheSync(stopCh, c.dgsListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Info("Starting workers for GarbageCollection controller")

	// Launch a number of workers to process resources
	for i := 0; i < controllerThreadiness; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will
		// then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers for GarbageCollection Controller")
	<-stopCh
	log.Info("Shutting down workers for GarbageCollection Controller")

	return nil
}
