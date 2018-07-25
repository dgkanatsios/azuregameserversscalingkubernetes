package controller

import (
	"fmt"
	"strconv"

	dgstypes "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/typed/azuregaming/v1alpha1"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const dgsControllerAgentName = "dedigated-game-server-controller"
const setSessionsURL = "lala"

type DedicatedGameServerController struct {
	dgsColClient       dgsv1alpha1.DedicatedGameServerCollectionsGetter
	dgsClient          dgsv1alpha1.DedicatedGameServersGetter
	podClient          typedcorev1.PodsGetter
	dgsColLister       listerdgs.DedicatedGameServerCollectionLister
	dgsLister          listerdgs.DedicatedGameServerLister
	podLister          listercorev1.PodLister
	dgsColListerSynced cache.InformerSynced
	dgsListerSynced    cache.InformerSynced
	podListerSynced    cache.InformerSynced
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
	dgsColInformer informerdgs.DedicatedGameServerCollectionInformer, dgsInformer informerdgs.DedicatedGameServerInformer,
	podInformer informercorev1.PodInformer) *DedicatedGameServerController {
	// Create event broadcaster
	// Add DedicatedGameServerController types to the default Kubernetes Scheme so Events can be
	// logged for DedicatedGameServerController types.
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	log.Info("Creating event broadcaster for DedicatedGameServer controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: dgsControllerAgentName})

	c := &DedicatedGameServerController{
		dgsColClient:       dgsclient.AzuregamingV1alpha1(),
		dgsClient:          dgsclient.AzuregamingV1alpha1(),
		podClient:          client.CoreV1(), //getter hits the live API server (can also create/update objects)
		dgsColLister:       dgsColInformer.Lister(),
		dgsLister:          dgsInformer.Lister(),
		podLister:          podInformer.Lister(), //lister hits the cache
		dgsColListerSynced: dgsColInformer.Informer().HasSynced,
		dgsListerSynced:    dgsInformer.Informer().HasSynced,
		podListerSynced:    podInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DedicatedGameServerSync"),
		recorder:           recorder,
	}

	log.Info("Setting up event handlers for DedicatedGameServer controller")

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("DedicatedGameServer controller - add")
				c.handleDedicatedGameServer(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("DedicatedGameServer controller - update")

				// get the previous state
				newObj.(*dgstypes.DedicatedGameServer).Status.PreviousState = oldObj.(*dgstypes.DedicatedGameServer).Status.State

				c.handleDedicatedGameServer(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta nodeQueue, therefore for deletes we have to use this
				// key function.
				//key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				log.Print("DedicatedGameServer controller - delete")
			},
		},
	)

	return c
}

func (c *DedicatedGameServerController) handleDedicatedGameServer(obj interface{}) {
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

	c.enqueueDedicatedGameServer(obj)
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *DedicatedGameServerController) RunWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	log.Info("Starting loop for DedicatedGameServer controller")
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

	// check the dedicated game server status
	// if it's "Running", we need to increase

	// Let's see if the corresponding pod exists
	_, err = c.podLister.Pods(namespace).Get(name)
	if err != nil {
		// if pod does not exist
		if errors.IsNotFound(err) {
			pod := shared.NewPod(dgs, setSessionsURL)
			// we'll create it
			createdPod, err2 := c.podClient.Pods(namespace).Create(pod)

			if err2 != nil {
				return err2
			}
			// and notify table storage accordingly
			err2 = shared.UpsertGameServerEntity(&shared.GameServerEntity{
				Name:      createdPod.Name,
				Namespace: createdPod.Namespace,
				NodeName:  createdPod.Spec.NodeName,
				Port:      strconv.Itoa(dgs.Spec.Port),
			})

			if err2 != nil {
				return err2
			}

			return nil
		} else {
			return err
		}
	}

	// if pod exists, let's see if the DedicatedGameServerCollection (if one exists and the pod isn't orphan) State needs updating
	if len(dgs.OwnerReferences) > 0 && dgs.Status.State != dgs.Status.PreviousState {
		// state changed, so let's update DedicatedGameServerCollection
		dgsColName := dgs.OwnerReferences[0].Name
		dgsCol, err := c.dgsColClient.DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
		if err != nil {
			c.recorder.Event(dgs, corev1.EventTypeWarning, "Error retrieving DedicatedGameServerCollection", err.Error())
			return err
		}
		dgsColCopy := dgsCol.DeepCopy()

		// if current state is Running, then we have a new running DedicatedGameServer
		// else, we lost one
		if dgs.Status.State == "Running" {
			dgsColCopy.Status.AvailableReplicas++
		} else if dgs.Status.PreviousState == "Running" {
			dgsColCopy.Status.AvailableReplicas--
		}

		_, err = c.dgsColClient.DedicatedGameServerCollections(namespace).Update(dgsColCopy)
		if err != nil {
			c.recorder.Event(dgs, corev1.EventTypeWarning, "Error updating DedicatedGameServerCollection", err.Error())
			return err
		}

	}

	c.recorder.Event(dgs, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageResourceSynced, "DedicatedGameServer", dgs.Name))
	return nil
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

func (c *DedicatedGameServerController) Workqueue() workqueue.RateLimitingInterface {
	return c.workqueue
}

func (c *DedicatedGameServerController) ListersSynced() []cache.InformerSynced {
	return []cache.InformerSynced{c.dgsColListerSynced, c.dgsListerSynced, c.podListerSynced}
}
