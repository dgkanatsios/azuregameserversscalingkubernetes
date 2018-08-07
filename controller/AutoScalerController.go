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
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const autoscalerControllerAgentName = "auto-scaler-controller"
const timeformat = "Jan 2, 2006 at 3:04pm (MST)"

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
				log.Print("AutoScaler controller - add DedicatedGameServerCollection")
				c.handleDedicatedGameServerCol(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("AutoScaler controller - update DedicatedGameServerCollection")

				oldDGSCol := oldObj.(*dgsv1alpha1.DedicatedGameServerCollection)
				newDGSCol := newObj.(*dgsv1alpha1.DedicatedGameServerCollection)

				if oldDGSCol.ResourceVersion == newDGSCol.ResourceVersion {
					return
				}

				c.handleDedicatedGameServerCol(newObj)
			},
		},
	)

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("AutoScaler controller - add DedicatedGameServer")
				c.handleDedicatedGameServerCol(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("AutoScaler controller - update DedicatedGameServer")

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
	dgsColTemp, err := c.dgsColLister.DedicatedGameServerCollections(namespace).Get(name)

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
	if dgsColTemp.Spec.AutoScalerDetails == nil || dgsColTemp.Spec.AutoScalerDetails.Enabled == false {
		return nil
	}

	// check if both DGS and Pod status != Running
	if dgsColTemp.Status.DedicatedGameServerCollectionState != dgsv1alpha1.DedicatedGameServerCollectionStateRunning ||
		dgsColTemp.Status.PodCollectionState != corev1.PodRunning {
		return nil
	}

	loc, err := time.LoadLocation("UTC")
	if err != nil {
		log.Error("Cannot load UTC time")
		return err
	}

	// lastScaleOperationDateTime != "" => scale in/out has happened before, at least once
	// let's see if time has passed since then is more than the cooldown threshold
	if dgsColTemp.Spec.AutoScalerDetails.LastScaleOperationDateTime != "" {

		lastScaleOperation, err := time.ParseInLocation(timeformat, dgsColTemp.Spec.AutoScalerDetails.LastScaleOperationDateTime, loc)

		if err != nil {
			log.WithFields(log.Fields{
				"DGSColName":                 dgsColTemp.Name,
				"LastScaleOperationDateTime": dgsColTemp.Spec.AutoScalerDetails.LastScaleOperationDateTime,
				"Error": err.Error(),
			}).Info("Cannot parse LastScaleOperationDateTime string. Will ignore potential cooldown duration")
		} else {

			currentTime := time.Now().In(loc)

			durationSinceLastScaleOperation := currentTime.Sub(lastScaleOperation)

			// cooldown period has not passed
			if durationSinceLastScaleOperation.Minutes() <= float64(dgsColTemp.Spec.AutoScalerDetails.CooldownInMinutes) {
				return nil
			}
		}
	}

	// grab all the DedicatedGameServers that belong to this DedicatedGameServerCollection
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsColTemp.Name,
	}
	// we seach via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsRunningList, err := c.dgsLister.DedicatedGameServers(dgsColTemp.Namespace).List(selector)

	// measure current load, i.e. total Active Players
	totalActivePlayers := 0
	for _, dgs := range dgsRunningList {
		totalActivePlayers += dgs.Spec.ActivePlayers
	}

	// get scaler information
	scalerDetails := dgsColTemp.Spec.AutoScalerDetails

	// measure total player capacity
	totalPlayerCapacity := scalerDetails.MaxPlayersPerServer * len(dgsRunningList)

	if len(dgsRunningList) < scalerDetails.MinimumReplicas ||
		(len(dgsRunningList) < scalerDetails.MaximumReplicas && totalActivePlayers/totalPlayerCapacity > scalerDetails.ScaleOutThreshold) {

		//scale out
		dgsColToUpdate := dgsColTemp.DeepCopy()
		dgsColToUpdate.Spec.Replicas++

		_, err := c.dgsColClient.DedicatedGameServerCollections(namespace).Update(dgsColToUpdate)

		if err != nil {
			log.WithFields(log.Fields{
				"DGSColName":          dgsColTemp.Name,
				"totalActivePlayers":  totalActivePlayers,
				"totalPlayerCapacity": totalPlayerCapacity,
				"Error":               err.Error(),
			}).Error("Cannot scale out")
			return err
		}
		scalerDetails.LastScaleOperationDateTime = time.Now().In(loc).String()
	}

	if (len(dgsRunningList) > scalerDetails.MinimumReplicas && totalActivePlayers/totalPlayerCapacity < scalerDetails.ScaleInThreshold) ||
		len(dgsRunningList) > scalerDetails.MaximumReplicas {
		//scale in
		dgsColToUpdate := dgsColTemp.DeepCopy()
		dgsColToUpdate.Spec.Replicas--

		_, err := c.dgsColClient.DedicatedGameServerCollections(namespace).Update(dgsColToUpdate)

		if err != nil {
			log.WithFields(log.Fields{
				"DGSColName":          dgsColTemp.Name,
				"totalActivePlayers":  totalActivePlayers,
				"totalPlayerCapacity": totalPlayerCapacity,
				"Error":               err.Error(),
			}).Error("Cannot scale in")
			return err
		}
		scalerDetails.LastScaleOperationDateTime = time.Now().In(loc).String()
	}

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

// Run initiates the AutoScalerController
func (c *AutoScalerController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting AutoScaler controller")

	// Wait for the caches for all controllers to be synced before starting workers
	log.Info("Waiting for informer caches to sync for AutoScaler controller")
	if ok := cache.WaitForCacheSync(stopCh, c.dgsColListerSynced, c.dgsListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Info("Starting workers for AutoScaler controller")

	// Launch a number of workers to process resources
	for i := 0; i < controllerThreadiness; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will
		// then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers for AutoScaler controller")
	<-stopCh
	log.Info("Shutting down workers for AutoScaler controller")

	return nil
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *AutoScalerController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	log.Info("Starting loop for AutoScaler controller")
	for c.processNextWorkItem() {
	}
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

func (c *AutoScalerController) handleDedicatedGameServer(obj interface{}) {
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
