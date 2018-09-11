package controller

import (
	"fmt"
	"time"

	"github.com/jonboulle/clockwork"

	"k8s.io/client-go/kubernetes"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	logrus "github.com/sirupsen/logrus"
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
const timeformat = "2006-01-02 15:04:05.999999999 -0700 MST"

// PodAutoScalerController is the struct that represents the PodAutoScalerController
type PodAutoScalerController struct {
	dgsColClient       dgsclientset.Interface
	dgsClient          dgsclientset.Interface
	dgsColLister       listerdgs.DedicatedGameServerCollectionLister
	dgsLister          listerdgs.DedicatedGameServerLister
	dgsColListerSynced cache.InformerSynced
	dgsListerSynced    cache.InformerSynced

	logger *logrus.Logger
	clock  clockwork.Clock
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

// NewPodAutoScalerController creates a new PodAutoScalerController
func NewPodAutoScalerController(client kubernetes.Interface, dgsclient dgsclientset.Interface,
	dgsColInformer informerdgs.DedicatedGameServerCollectionInformer,
	dgsInformer informerdgs.DedicatedGameServerInformer, clockImpl clockwork.Clock) *PodAutoScalerController {

	c := &PodAutoScalerController{
		dgsColClient:       dgsclient,
		dgsColLister:       dgsColInformer.Lister(),
		dgsColListerSynced: dgsColInformer.Informer().HasSynced,
		dgsClient:          dgsclient,
		dgsLister:          dgsInformer.Lister(),
		dgsListerSynced:    dgsInformer.Informer().HasSynced,
		clock:              clockImpl,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "PodAutoScalerSync"),
	}

	c.logger = shared.Logger()

	// Create event broadcaster
	// Add DedicatedGameServerController types to the default Kubernetes Scheme so Events can be
	// logged for DedicatedGameServerController types.
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	c.logger.Info("Creating event broadcaster for PodAutoScaler controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: dgsControllerAgentName})

	c.logger.Info("Setting up event handlers for PodAutoScaler controller")

	dgsColInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.logger.Print("PodAutoScaler controller - add DedicatedGameServerCollection")
				c.handleDedicatedGameServerCol(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Print("PodAutoScaler controller - update DedicatedGameServerCollection")

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
				// we're doing nothing on add
				// the logic will run either on DGSCol add/update or DGS update
				c.logger.Print("PodAutoScaler controller - add DedicatedGameServer")
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Print("PodAutoScaler controller - update DedicatedGameServer")

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
func (c *PodAutoScalerController) processNextWorkItem() bool {
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
		c.logger.Infof("Successfully synced '%s'", key)
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
func (c *PodAutoScalerController) syncHandler(key string) error {
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
		c.logger.WithField("DGSColName", dgsColTemp.Name).Errorf("Error listing DGSCol: %s", err.Error())
		return err
	}

	// DGSCol is being terminated
	if !dgsColTemp.DeletionTimestamp.IsZero() {
		c.logger.WithField("DGSColName", dgsColTemp.Name).Info("DGSCol is being terminated")
		return nil
	}

	// check if it has autoscaling enabled
	if dgsColTemp.Spec.PodAutoScalerDetails == nil || dgsColTemp.Spec.PodAutoScalerDetails.Enabled == false {
		return nil
	}

	// check if both DGS and Pod status != Running
	if dgsColTemp.Status.DedicatedGameServerCollectionState != dgsv1alpha1.DedicatedGameServerCollectionStateRunning ||
		dgsColTemp.Status.PodCollectionState != corev1.PodRunning {
		c.logger.WithField("DGSColName", dgsColTemp.Name).Info("Not checking about autoscaling because DedicatedGameServer and/or Pod states are not Running")
		return nil
	}

	loc, err := time.LoadLocation("UTC")
	if err != nil {
		c.logger.Error("Cannot load UTC time")
		return err
	}

	// lastScaleOperationDateTime != "" => scale in/out has happened before, at least once
	// let's see if time has passed since then is more than the cooldown threshold
	if dgsColTemp.Spec.PodAutoScalerDetails.LastScaleOperationDateTime != "" {

		lastScaleOperation, err := time.ParseInLocation(timeformat, dgsColTemp.Spec.PodAutoScalerDetails.LastScaleOperationDateTime, loc)

		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"DGSColName":                 dgsColTemp.Name,
				"LastScaleOperationDateTime": dgsColTemp.Spec.PodAutoScalerDetails.LastScaleOperationDateTime,
				"Error":                      err.Error(),
			}).Info("Cannot parse LastScaleOperationDateTime string. Will ignore cooldown duration")
		} else {

			currentTime := c.clock.Now().In(loc)

			durationSinceLastScaleOperation := currentTime.Sub(lastScaleOperation)

			// cooldown period has not passed
			if durationSinceLastScaleOperation.Minutes() <= float64(dgsColTemp.Spec.PodAutoScalerDetails.CoolDownInMinutes) {
				c.logger.WithField("DGSColName", dgsColTemp.Name).Info("Not checking about autoscaling because coolDownPeriod has not passed")
				return nil
			}
		}
	}

	// grab all the DedicatedGameServers that belong to this DedicatedGameServerCollection
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsColTemp.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsRunningList, err := c.dgsLister.DedicatedGameServers(dgsColTemp.Namespace).List(selector)

	// measure current load, i.e. total Active Players
	totalActivePlayers := 0
	for _, dgs := range dgsRunningList {
		totalActivePlayers += dgs.Status.ActivePlayers
	}

	// get scaler information
	scalerDetails := dgsColTemp.Spec.PodAutoScalerDetails

	// measure total player capacity
	totalPlayerCapacity := scalerDetails.MaxPlayersPerServer * len(dgsRunningList)

	currentLoad := float32(totalActivePlayers) / float32(totalPlayerCapacity)
	scaleOutThresholdPercent := float32(scalerDetails.ScaleOutThreshold) / float32(100)
	scaleInThresholdPercent := float32(scalerDetails.ScaleInThreshold) / float32(100)

	// c.logger.WithFields(c.logger.Fields{
	// 	"DGSCol":                   dgsColTemp.Name,
	// 	"currentLoad":              currentLoad,
	// 	"scaleOutThresholdPercent": scaleOutThresholdPercent,
	// 	"scaleInThresholdPercent":  scaleInThresholdPercent,
	// 	"len(dgsRunningList)":      len(dgsRunningList),
	// 	"minReplicas":              scalerDetails.MinimumReplicas,
	// 	"maxReplicas":              scalerDetails.MaximumReplicas,
	// }).Info("Scaler details")

	if len(dgsRunningList) < scalerDetails.MinimumReplicas ||
		(len(dgsRunningList) < scalerDetails.MaximumReplicas && currentLoad > scaleOutThresholdPercent) {

		//scale out
		dgsColToUpdate := dgsColTemp.DeepCopy()
		dgsColToUpdate.Spec.Replicas++
		dgsColToUpdate.Spec.PodAutoScalerDetails.LastScaleOperationDateTime = c.clock.Now().In(loc).String()
		dgsColToUpdate.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateCreating

		_, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgsColToUpdate)

		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"DGSColName":          dgsColTemp.Name,
				"totalActivePlayers":  totalActivePlayers,
				"totalPlayerCapacity": totalPlayerCapacity,
				"Error":               err.Error(),
			}).Error("Cannot scale out")
			return err
		}

		c.logger.WithField("DGSColName", dgsColToUpdate.Name).Info("Scale out occurred")

		return nil
	}

	if (len(dgsRunningList) > scalerDetails.MinimumReplicas && currentLoad < scaleInThresholdPercent) ||
		len(dgsRunningList) > scalerDetails.MaximumReplicas {
		//scale in
		dgsColToUpdate := dgsColTemp.DeepCopy()
		dgsColToUpdate.Spec.Replicas--
		dgsColToUpdate.Spec.PodAutoScalerDetails.LastScaleOperationDateTime = c.clock.Now().In(loc).String()
		dgsColToUpdate.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateCreating

		_, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgsColToUpdate)

		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"DGSColName":          dgsColTemp.Name,
				"totalActivePlayers":  totalActivePlayers,
				"totalPlayerCapacity": totalPlayerCapacity,
				"Error":               err.Error(),
			}).Error("Cannot scale in")
			return err
		}

		c.logger.WithField("DGSColName", dgsColToUpdate.Name).Info("Scale in occurred")

		return nil
	}

	return nil
}

// enqueueDedicatedGameServer takes a DedicatedGameServer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServer.
func (c *PodAutoScalerController) enqueueDedicatedGameServerCollection(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// Run initiates the AutoScalerController
func (c *PodAutoScalerController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	c.logger.Info("Starting PodAutoScaler controller")

	// Wait for the caches for all controllers to be synced before starting workers
	c.logger.Info("Waiting for informer caches to sync for PodAutoScaler controller")
	if ok := cache.WaitForCacheSync(stopCh, c.dgsColListerSynced, c.dgsListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	c.logger.Info("Starting workers for PodAutoScaler controller")

	// Launch a number of workers to process resources
	for i := 0; i < controllerThreadiness; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will
		// then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	c.logger.Info("Started workers for PodAutoScaler controller")
	<-stopCh
	c.logger.Info("Shutting down workers for PodAutoScaler controller")

	return nil
}

// RunWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *PodAutoScalerController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	c.logger.Info("Starting loop for PodAutoScaler controller")
	for c.processNextWorkItem() {
	}
}

func (c *PodAutoScalerController) handleDedicatedGameServerCol(obj interface{}) {
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
		c.logger.Infof("Recovered deleted DedicatedGameServerCollection object '%s' from tombstone", object.GetName())
	}

	c.enqueueDedicatedGameServerCollection(object)
}

func (c *PodAutoScalerController) handleDedicatedGameServer(obj interface{}) {
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
		c.logger.Infof("Recovered deleted DedicatedGameServerCollection object '%s' from tombstone", object.GetName())
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
