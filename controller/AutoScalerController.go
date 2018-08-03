package controller

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	typeddgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/typed/azuregaming/v1alpha1"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const autoscalerControllerAgentName = "auto-scaler-controller"

type AutoScalerController struct {
	dgsColClient       typeddgsv1alpha1.DedicatedGameServerCollectionsGetter
	dgsClient          typeddgsv1alpha1.DedicatedGameServersGetter
	dgsColLister       listerdgs.DedicatedGameServerCollectionLister
	dgsLister          listerdgs.DedicatedGameServerLister
	dgsColListerSynced cache.InformerSynced
	dgsListerSynced    cache.InformerSynced

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

func NewAutoScalerControllerController(client *kubernetes.Clientset, dgsclient *dgsclientset.Clientset,
	dgsColInformer informerdgs.DedicatedGameServerCollectionInformer,
	dgsInformer informerdgs.DedicatedGameServerInformer) *AutoScalerController {
	// Create event broadcaster
	// Add DedicatedGameServerController types to the default Kubernetes Scheme so Events can be
	// logged for DedicatedGameServerController types.
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	log.Info("Creating event broadcaster for AutoScaler controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: dgsControllerAgentName})

	c := &AutoScalerController{
		dgsClient:          dgsclient.AzuregamingV1alpha1(),
		dgsColLister:       dgsColInformer.Lister(),
		dgsLister:          dgsInformer.Lister(),
		dgsColListerSynced: dgsColInformer.Informer().HasSynced,
		dgsListerSynced:    dgsInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "AutoScalerSync"),
		recorder:           recorder,
	}

	log.Info("Setting up event handlers for AutoScaler controller")

	dgsColInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("AutoScaler controller - add")
				c.handleDedicatedGameServerCol(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("AutoScaler controller - update")

				oldDGSCol := oldObj.(*dgsv1alpha1.DedicatedGameServerCollection)
				newDGSCol := newObj.(*dgsv1alpha1.DedicatedGameServerCollection)

				if oldDGSCol.ResourceVersion == newDGSCol.ResourceVersion {
					return
				}

				c.handleDedicatedGameServerCol(newObj)
			},
		},
	)

	return c
}

func (c *AutoScalerController) handleDedicatedGameServerCol(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding DedicatedGameServerCollection object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding DedicatedGameServerCollection object tombstone, invalid type"))
			return
		}
		log.Infof("Recovered deleted DedicatedGameServerCollection object '%s' from tombstone", object.GetName())
	}

	c.enqueueDedicatedGameServerCollection(object)
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *AutoScalerController) RunWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	log.Info("Starting loop for AutoScaler controller")
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *AutoScalerController) processNextWorkItem() bool {
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
func (c *AutoScalerController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// try to get the DedicatedGameServerCollection
	dgsCol, err := c.dgsColLister.DedicatedGameServerCollections(namespace).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			// DedicatedGameServerCollection not found
			runtime.HandleError(fmt.Errorf("DedicatedGameServerCollection '%s' in work queue no longer exists", key))
			return nil
		}
		log.Print(err.Error())
		return err
	}

	// check if it has autoscaling enabled
	if dgsCol.Spec.AutoScalerDetails.Enabled {
		// grab all the DedicatedGameServers that belong to this DedicatedGameServerCollection
		set := labels.Set{
			shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
		}
		// we seach via Labels, each DGS will have the DGSCol name as a Label
		selector := labels.SelectorFromSet(set)
		dgsList, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)
		log.Print(dgsList, err)
	}

	// //check its state and active players
	// if dgs.Spec.ActivePlayers == "0" && dgs.Status.GameServerState == shared.GameServerStateMarkedForDeletion {
	// 	dgsToDelete, err := c.dgsClient.DedicatedGameServers(namespace).Get(name, metav1.GetOptions{})
	// 	if err != nil {
	// 		log.Errorf("Cannot fetch DedicatedGameServer %s", name)
	// 		// DedicatedGameServer not found
	// 		runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' cannot be retrieved", name))
	// 		return err
	// 	}

	// 	err = c.dgsClient.DedicatedGameServers(namespace).Delete(dgsToDelete.Name, &metav1.DeleteOptions{})

	// 	if err != nil {
	// 		log.Errorf("Cannot delete DedicatedGameServer %s", name)
	// 		// DedicatedGameServer not found
	// 		runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' cannot be deleted", name))
	// 		return err
	// 	}

	// 	c.recorder.Event(dgs, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageMarkedForDeletionDedicatedGameServerDeleted, dgs.Name))
	// 	return nil
	// }

	return nil
}

// enqueueDedicatedGameServer takes a DedicatedGameServer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServer.
func (c *AutoScalerController) enqueueDedicatedGameServerCollection(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *AutoScalerController) Workqueue() workqueue.RateLimitingInterface {
	return c.workqueue
}

func (c *AutoScalerController) ListersSynced() []cache.InformerSynced {
	return []cache.InformerSynced{c.dgsColListerSynced, c.dgsListerSynced}
}
