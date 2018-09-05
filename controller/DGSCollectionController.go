package controller

import (
	"fmt"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"

	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	cache "k8s.io/client-go/tools/cache"

	record "k8s.io/client-go/tools/record"
	workqueue "k8s.io/client-go/util/workqueue"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	kubernetes "k8s.io/client-go/kubernetes"

	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"

	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	log "github.com/sirupsen/logrus"

	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	dgsColControllerAgentName = "dedigated-game-server-collection-controller"
)

// DedicatedGameServerCollectionController represents a controller for a Dedicated Game Server Collection
type DedicatedGameServerCollectionController struct {
	dgsColClient dgsclientset.Interface
	dgsClient    dgsclientset.Interface

	dgsColLister listerdgs.DedicatedGameServerCollectionLister
	dgsLister    listerdgs.DedicatedGameServerLister

	dgsColListerSynced cache.InformerSynced
	dgsListerSynced    cache.InformerSynced

	clock         clockwork.Clock
	namegenerator shared.RandomNameGenerator

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// NewDedicatedGameServerCollectionController initializes and returns a new DedicatedGameServerCollectionController instance
func NewDedicatedGameServerCollectionController(client kubernetes.Interface, dgsclient dgsclientset.Interface,
	dgsColInformer informerdgs.DedicatedGameServerCollectionInformer, dgsInformer informerdgs.DedicatedGameServerInformer,
	randomNameGenerator shared.RandomNameGenerator, clockImpl clockwork.Clock) *DedicatedGameServerCollectionController {
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	log.Info("Creating Event broadcaster for DedicatedGameServerCollection controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: dgsColControllerAgentName})

	c := &DedicatedGameServerCollectionController{
		dgsColClient:       dgsclient,
		dgsClient:          dgsclient,
		dgsColLister:       dgsColInformer.Lister(),
		dgsLister:          dgsInformer.Lister(),
		dgsColListerSynced: dgsColInformer.Informer().HasSynced,
		dgsListerSynced:    dgsInformer.Informer().HasSynced,
		namegenerator:      randomNameGenerator,
		clock:              clockImpl,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DedicatedGameServerCollectionSync"),
		recorder:           recorder,
	}

	log.Info("Setting up event handlers for DedicatedGameServerCollection controller")

	dgsColInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("DedicatedGameServerCollection controller - add DGSCol")
				c.handleDedicatedGameServerCollection(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("DedicatedGameServerCollection controller - update DGSCol")
				oldDGSCol := oldObj.(*dgsv1alpha1.DedicatedGameServerCollection)
				newDGSCol := newObj.(*dgsv1alpha1.DedicatedGameServerCollection)

				if oldDGSCol.ResourceVersion == newDGSCol.ResourceVersion {
					return
				}

				//only enqueue if if the number of requested replicas has changed
				//TODO: should we allow other updates?
				if c.hasDedicatedGameServerCollectionChanged(oldDGSCol, newDGSCol) {
					c.handleDedicatedGameServerCollection(newObj)
				}

			},
		},
	)

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("DedicatedGameServerCollection controller - add DGS")
				c.handleDedicatedGameServer(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("DedicatedGameServerCollection controller - update DGS")
				oldDGS := oldObj.(*dgsv1alpha1.DedicatedGameServer)
				newDGS := newObj.(*dgsv1alpha1.DedicatedGameServer)

				if oldDGS.ResourceVersion == newDGS.ResourceVersion {
					return
				}

				if shared.HasDedicatedGameServerChanged(oldDGS, newDGS) {
					c.handleDedicatedGameServer(newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				log.Print("DedicatedGameServerCollection controller - delete DGS")
				c.handleDedicatedGameServer(obj)
			},
		},
	)

	return c
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *DedicatedGameServerCollectionController) processNextWorkItem() bool {
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

func (c *DedicatedGameServerCollectionController) getDGSForDGSCol(dgsColTemp *dgsv1alpha1.DedicatedGameServerCollection) ([]*dgsv1alpha1.DedicatedGameServer, error) {
	// Find out how many DedicatedGameServer replicas exist for this DedicatedGameServerCollection
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsColTemp.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsExisting, err := c.dgsLister.DedicatedGameServers(dgsColTemp.Namespace).List(selector)

	return dgsExisting, err
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DedicatedGameServer resource
// with the current status of the resource.
func (c *DedicatedGameServerCollectionController) syncHandler(key string) error {

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the DedicatedGameServerCollection resource with this namespace/name
	dgsColTemp, err := c.dgsColLister.DedicatedGameServerCollections(namespace).Get(name)
	if err != nil {
		// The DedicatedGameServerCollection resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("DedicatedGameServerCollection '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	dgsExisting, err := c.getDGSForDGSCol(dgsColTemp)

	if err != nil {
		log.WithFields(log.Fields{
			"DGSColName": dgsColTemp.Name,
			"Error":      err.Error(),
		}).Error("Cannot get DedicatedGameServers for DedicatedGameServerCollection via Label selector")
		return err
	}

	dgsExistingCount := len(dgsExisting)

	// if there are less DedicatedGameServers than the ones we requested
	if dgsExistingCount < int(dgsColTemp.Spec.Replicas) {
		err = c.increaseDGSReplicas(dgsColTemp, dgsExistingCount)

		if err != nil {
			c.recorder.Event(dgsColTemp, corev1.EventTypeWarning, "Cannot increase dedicated game servers", err.Error())
			log.WithFields(log.Fields{
				"DGSColName": dgsColTemp.Name,
				"Error":      err.Error(),
			}).Error("Cannot increase dedicated game servers")
			return err
		}

		c.recorder.Event(dgsColTemp, corev1.EventTypeNormal, shared.DedicatedGameServerReplicasChanged, fmt.Sprintf(shared.MessageReplicasIncreased, "DedicatedGameServerCollection", dgsColTemp.Name, int(dgsColTemp.Spec.Replicas)-dgsExistingCount))
		return nil //exiting sync handler, further DGS updates will propagate here as well via another item in the workqueue

	} else if dgsExistingCount > int(dgsColTemp.Spec.Replicas) { //if there are more DGS than the ones we requested

		err = c.decreaseDGSReplicas(dgsColTemp, dgsExisting)

		if err != nil {
			c.recorder.Event(dgsColTemp, corev1.EventTypeWarning, "Cannot decrease dedicated game servers", err.Error())
			log.WithFields(log.Fields{
				"DGSColName": dgsColTemp.Name,
				"Error":      err.Error(),
			}).Error("Cannot decrease dedicated game servers")
			return err
		}

		c.recorder.Event(dgsColTemp, corev1.EventTypeNormal, shared.DedicatedGameServerReplicasChanged, fmt.Sprintf(shared.MessageReplicasDecreased, "DedicatedGameServerCollection", dgsColTemp.Name, dgsExistingCount-int(dgsColTemp.Spec.Replicas)))
		return nil //exiting sync handler, further DGS updates will propagate here as well via another item in the workqueue
	}

	// if we reach this point, this means that the number of requested replicas is equal to the number of the available ones
	// next step would be to modify the various DGSCol statuses

	dgsColToUpdate := dgsColTemp.DeepCopy()

	err = c.modifyAvailableReplicasStatus(dgsColToUpdate)
	if err != nil {
		c.recorder.Event(dgsColTemp, corev1.EventTypeWarning, "Error setting DedicatedGameServerCollection - GameServer status", err.Error())
		return err
	}

	//modify DGSCol.Status.DGSState
	err = c.assignGameServerCollectionState(dgsColToUpdate)
	if err != nil {
		c.recorder.Event(dgsColTemp, corev1.EventTypeWarning, "Error setting DedicatedGameServerCollection - GameServer status", err.Error())
		return err
	}

	//modify DGSCol.Status.PodState
	err = c.assignPodCollectionState(dgsColToUpdate)
	if err != nil {
		c.recorder.Event(dgsColTemp, corev1.EventTypeWarning, "Error setting DedicatedGameServerCollection - Pod status", err.Error())
		return err
	}

	_, err = c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgsColToUpdate)

	if err != nil {
		log.WithFields(log.Fields{
			"Name":  dgsColTemp.Name,
			"Error": err.Error(),
		}).Error("Error in updating DedicatedGameServerCollection")
		c.recorder.Event(dgsColTemp, corev1.EventTypeWarning, "Error updating DedicatedGameServerCollection", err.Error())
		return err
	}

	c.recorder.Event(dgsColTemp, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageResourceSynced, "DedicatedGameServerCollection", dgsColTemp.Name))
	return nil
}

func (c *DedicatedGameServerCollectionController) increaseDGSReplicas(dgsColTemp *dgsv1alpha1.DedicatedGameServerCollection, dgsExistingCount int) error {
	//create them
	increaseCount := int(dgsColTemp.Spec.Replicas) - dgsExistingCount

	log.WithFields(log.Fields{
		"DGSColName":    dgsColTemp.Name,
		"IncreaseCount": increaseCount,
	}).Printf("Scaling out")

	for i := 0; i < increaseCount; i++ {
		dgsName := c.namegenerator.GenerateName(dgsColTemp.Name)
		// first, get random ports
		for j := 0; j < len(dgsColTemp.Spec.Template.Containers[0].Ports); j++ {
			//get a random port
			hostport, errPort := portRegistry.GetNewPort(dgsName)
			if errPort != nil {
				return errPort
			}
			dgsColTemp.Spec.Template.Containers[0].Ports[j].HostPort = hostport
		}

		// each dedicated game server will have a name of
		// DedicatedGameServerCollectioName + "-" + random name
		dgs := shared.NewDedicatedGameServer(dgsColTemp, dgsColTemp.Spec.Template, c.namegenerator)
		_, err := c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(dgsColTemp.Namespace).Create(dgs)

		if err != nil {
			return err
		}
	}

	// add/update an annotation with the latest scale info
	dgsColToUpdate := dgsColTemp.DeepCopy()
	if dgsColToUpdate.Annotations == nil {
		dgsColToUpdate.Annotations = make(map[string]string)
	}
	dgsColToUpdate.Annotations["LastScaleOutDateTime"] = c.clock.Now().In(time.UTC).String()
	//update the DGS CRD
	_, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsColToUpdate.Namespace).Update(dgsColToUpdate)
	if err != nil {
		return err
	}

	return nil
}

func (c *DedicatedGameServerCollectionController) decreaseDGSReplicas(dgsColTemp *dgsv1alpha1.DedicatedGameServerCollection, dgsExisting []*dgsv1alpha1.DedicatedGameServer) error {
	dgsExistingCount := len(dgsExisting)
	// we need to decrease our DGS for this collection
	// to accomplish this, we'll first find the number of DGS we need to decrease
	decreaseCount := dgsExistingCount - int(dgsColTemp.Spec.Replicas)
	// we'll remove random instances of DGS from our DGSCol
	indexesToDecrease := shared.GetRandomIndexes(dgsExistingCount, decreaseCount)

	log.WithFields(log.Fields{
		"DGSColName":    dgsColTemp.Name,
		"DecreaseCount": decreaseCount,
	}).Printf("Scaling in")

	for i := 0; i < len(indexesToDecrease); i++ {
		dgsToMarkForDeletionTemp, err := c.dgsLister.DedicatedGameServers(dgsColTemp.Namespace).Get(dgsExisting[indexesToDecrease[i]].Name)

		if err != nil {
			return err
		}
		dgsToMarkForDeletionToUpdate := dgsToMarkForDeletionTemp.DeepCopy()
		// update the DGS so it has no owners
		dgsToMarkForDeletionToUpdate.ObjectMeta.OwnerReferences = nil
		//remove the DGSCol name from the DGS labels
		delete(dgsToMarkForDeletionToUpdate.ObjectMeta.Labels, shared.LabelDedicatedGameServerCollectionName)
		//set its state as marked for deletion
		dgsToMarkForDeletionToUpdate.Status.DedicatedGameServerState = dgsv1alpha1.DedicatedGameServerStateMarkedForDeletion
		dgsToMarkForDeletionToUpdate.Labels[shared.LabelDedicatedGameServerState] = string(dgsv1alpha1.DedicatedGameServerStateMarkedForDeletion)
		//mark its previous owner
		dgsToMarkForDeletionToUpdate.ObjectMeta.Labels[shared.LabelOriginalDedicatedGameServerCollectionName] = dgsColTemp.Name
		//update the DGS CRD
		_, err = c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(dgsColTemp.Namespace).Update(dgsToMarkForDeletionToUpdate)
		if err != nil {
			return err
		}

	}

	// add/update an annotation with the latest scale info
	dgsColToUpdate := dgsColTemp.DeepCopy()
	if dgsColToUpdate.Annotations == nil {
		dgsColToUpdate.Annotations = make(map[string]string)
	}
	dgsColToUpdate.Annotations["LastScaleInDateTime"] = c.clock.Now().In(time.UTC).String()
	//update the DGS CRD
	_, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsColToUpdate.Namespace).Update(dgsColToUpdate)
	if err != nil {
		return err
	}

	return nil //exiting, DGS updates will propagate here as well via another item in the workqueue
}

func (c *DedicatedGameServerCollectionController) modifyAvailableReplicasStatus(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {

	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsInstances, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)

	if err != nil {
		return err
	}

	dgsCol.Status.AvailableReplicas = 0

	for _, dgs := range dgsInstances {
		if dgs.Status.DedicatedGameServerState == dgsv1alpha1.DedicatedGameServerStateRunning && dgs.Status.PodState == corev1.PodRunning {
			dgsCol.Status.AvailableReplicas++
		}
	}

	return nil
}

func (c *DedicatedGameServerCollectionController) assignGameServerCollectionState(
	dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {

	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsInstances, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)

	if err != nil {
		return err
	}

	for _, dgs := range dgsInstances {
		//at least one of the DGS is not running
		if dgs.Status.DedicatedGameServerState != dgsv1alpha1.DedicatedGameServerStateRunning {
			//so set the overall collection state as the state of this one
			dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionState(dgs.Status.DedicatedGameServerState)
			return nil
		}
	}
	//all of the DGS are running, so set the DGSCol state as running
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateRunning
	return nil

}

func (c *DedicatedGameServerCollectionController) assignPodCollectionState(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsInstances, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)

	if err != nil {
		return err
	}

	for _, dgs := range dgsInstances {
		//one pod is not running
		if dgs.Status.PodState != corev1.PodRunning {
			// so set the collection's Pod State to this one Pod's value
			dgsCol.Status.PodCollectionState = dgs.Status.PodState
			return nil
		}
	}
	//all pods are running, so set the collection Pod State with the running value
	dgsCol.Status.PodCollectionState = corev1.PodRunning
	return nil

}

func (c *DedicatedGameServerCollectionController) handleDedicatedGameServerCollection(obj interface{}) {
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

func (c *DedicatedGameServerCollectionController) handleDedicatedGameServer(obj interface{}) {
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
		log.Infof("Recovered deleted DedicatedGameServerCollection object '%s' from tombstone", object.GetName())
	}

	// when we get a DGS, we will enqueue the DGSCol only if
	// DGS has a parent DGSCol, so that DGSCol get updated with new status etc.

	//if this DGS has a parent DGSCol
	if len(object.GetOwnerReferences()) > 0 {
		dgsCol, err := c.dgsColLister.DedicatedGameServerCollections(object.GetNamespace()).Get(object.GetOwnerReferences()[0].Name)
		if err != nil {
			runtime.HandleError(fmt.Errorf("error getting a DedicatedGameServer Collection from the Dedicated Game Server with Name %s", object.GetName()))
			return
		}

		c.enqueueDedicatedGameServerCollection(dgsCol)
	}

}

// enqueueDedicatedGameServerCollection takes a DedicatedGameServerCollection resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServerCollection.
func (c *DedicatedGameServerCollectionController) enqueueDedicatedGameServerCollection(obj interface{}) {
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
func (c *DedicatedGameServerCollectionController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	log.Info("Starting loop for DedicatedGameServerCollection controller")
	for c.processNextWorkItem() {
	}
}

func (c *DedicatedGameServerCollectionController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting DedicatedGameServerCollection controller")

	// Wait for the caches for all controllers to be synced before starting workers
	log.Info("Waiting for informer caches to sync for DedicatedGameServerCollection controller")
	if ok := cache.WaitForCacheSync(stopCh, c.dgsColListerSynced, c.dgsListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Info("Starting workers for DedicatedGameServerCollection controller")

	// Launch a number of workers to process resources
	for i := 0; i < controllerThreadiness; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will
		// then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers for DedicatedGameServerCollection controller")
	<-stopCh
	log.Info("Shutting down workers for DedicatedGameServerCollection controller")

	return nil
}

func (c *DedicatedGameServerCollectionController) hasDedicatedGameServerCollectionChanged(oldDGSCol, newDGSCol *dgsv1alpha1.DedicatedGameServerCollection) bool {
	return oldDGSCol.Spec.Replicas != newDGSCol.Spec.Replicas || !shared.AreMapsSame(oldDGSCol.Annotations, newDGSCol.Annotations)
}
