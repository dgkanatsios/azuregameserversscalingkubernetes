package autoscale

import (
	"fmt"
	"time"

	"github.com/jonboulle/clockwork"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/controller"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	logrus "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const autoscalerControllerAgentName = "auto-scaler-controller"
const timeformat = "2006-01-02 15:04:05.999999999 -0700 MST"

// DGSAutoScalerController is the struct that represents the DGSAutoScalerController
type DGSAutoScalerController struct {
	dgsColClient       dgsclientset.Interface
	dgsClient          dgsclientset.Interface
	dgsColLister       listerdgs.DedicatedGameServerCollectionLister
	dgsLister          listerdgs.DedicatedGameServerLister
	dgsColListerSynced cache.InformerSynced
	dgsListerSynced    cache.InformerSynced

	logger *logrus.Logger
	clock  clockwork.Clock

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	controllerHelper *controllers.ControllerHelper
}

// NewDGSAutoScalerController creates a new DGSAutoScalerController
func NewDGSAutoScalerController(client kubernetes.Interface, dgsclient dgsclientset.Interface,
	dgsColInformer informerdgs.DedicatedGameServerCollectionInformer,
	dgsInformer informerdgs.DedicatedGameServerInformer, clockImpl clockwork.Clock) *DGSAutoScalerController {

	c := &DGSAutoScalerController{
		dgsColClient:       dgsclient,
		dgsColLister:       dgsColInformer.Lister(),
		dgsColListerSynced: dgsColInformer.Informer().HasSynced,
		dgsClient:          dgsclient,
		dgsLister:          dgsInformer.Lister(),
		dgsListerSynced:    dgsInformer.Informer().HasSynced,
		clock:              clockImpl,
		logger:             shared.Logger(),
	}

	c.controllerHelper = controllers.NewControllerHelper(
		workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DGSAutoScalerSync"),
		c.logger,
		c.syncHandler,
		"DGSAutoScalerController",
		[]cache.InformerSynced{c.dgsColListerSynced, c.dgsListerSynced},
	)

	// Create event broadcaster
	// Add DedicatedGameServerController types to the default Kubernetes Scheme so Events can be
	// logged for DedicatedGameServerController types.
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	c.logger.Info("Creating event broadcaster for DGSAutoScaler controller")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: autoscalerControllerAgentName})

	c.logger.Info("Setting up event handlers for DGSAutoScaler controller")

	dgsColInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.logger.Print("DGSAutoScaler controller - add DedicatedGameServerCollection")
				c.handleDedicatedGameServerCollection(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Print("DGSAutoScaler controller - update DedicatedGameServerCollection")
				oldDGSCol := oldObj.(*dgsv1alpha1.DedicatedGameServerCollection)
				newDGSCol := newObj.(*dgsv1alpha1.DedicatedGameServerCollection)
				if oldDGSCol.ResourceVersion == newDGSCol.ResourceVersion {
					return
				}
				c.handleDedicatedGameServerCollection(newObj)
			},
		},
	)

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// we're doing nothing on add
				// the logic will run either on DGSCol add/update or DGS update
				c.logger.Print("DGSAutoScaler controller - add DedicatedGameServer")
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Print("DGSAutoScaler controller - update DedicatedGameServer")
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

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DedicatedGameServer resource
// with the current status of the resource.
func (c *DGSAutoScalerController) syncHandler(key string) error {
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
	if dgsColTemp.Spec.DgsAutoScalerDetails == nil || dgsColTemp.Spec.DgsAutoScalerDetails.Enabled == false {
		return nil
	}

	// check if both DGS and Pod status != Running
	if dgsColTemp.Status.DGSCollectionHealth != dgsv1alpha1.DGSColHealthy ||
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
	if dgsColTemp.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime != "" {
		lastScaleOperation, err := time.ParseInLocation(timeformat, dgsColTemp.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime, loc)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"DGSColName":                 dgsColTemp.Name,
				"LastScaleOperationDateTime": dgsColTemp.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime,
				"Error":                      err.Error(),
			}).Info("Cannot parse LastScaleOperationDateTime string. Will ignore cooldown duration")
		} else {
			currentTime := c.clock.Now().In(loc)
			durationSinceLastScaleOperation := currentTime.Sub(lastScaleOperation)
			// cooldown period has not passed
			if durationSinceLastScaleOperation.Minutes() <= float64(dgsColTemp.Spec.DgsAutoScalerDetails.CoolDownInMinutes) {
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
	scalerDetails := dgsColTemp.Spec.DgsAutoScalerDetails

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

	if len(dgsRunningList) < scalerDetails.MaximumReplicas && currentLoad > scaleOutThresholdPercent {
		//scale out
		dgsColToUpdate := dgsColTemp.DeepCopy()
		dgsColToUpdate.Spec.Replicas++
		dgsColToUpdate.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime = c.clock.Now().In(loc).String()
		dgsColToUpdate.Status.DGSCollectionHealth = dgsv1alpha1.DGSColCreating

		_, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgsColToUpdate)
		if err != nil {
			c.logger.WithFields(logrus.Fields{"DedicatedGameServerCollectionName": dgsColTemp.Name, "totalActivePlayers": totalActivePlayers, "totalPlayerCapacity": totalPlayerCapacity, "Error": err.Error()}).Error("Cannot scale out")
			return err
		}

		c.logger.WithField("DedicatedGameServerCollectionName", dgsColToUpdate.Name).Info("Scale out occurred")

		return nil
	}

	if len(dgsRunningList) > scalerDetails.MinimumReplicas && currentLoad < scaleInThresholdPercent {
		//scale in
		dgsColToUpdate := dgsColTemp.DeepCopy()
		dgsColToUpdate.Spec.Replicas--
		dgsColToUpdate.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime = c.clock.Now().In(loc).String()
		dgsColToUpdate.Status.DGSCollectionHealth = dgsv1alpha1.DGSColCreating

		_, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgsColToUpdate)
		if err != nil {
			c.logger.WithFields(logrus.Fields{"DedicatedGameServerCollectionName": dgsColTemp.Name, "totalActivePlayers": totalActivePlayers, "totalPlayerCapacity": totalPlayerCapacity, "Error": err.Error()}).Error("Cannot scale in")
			return err
		}

		c.logger.WithField("DedicatedGameServerCollectionName", dgsColToUpdate.Name).Info("Scale in occurred")

		return nil
	}

	return nil
}

// enqueueDedicatedGameServer takes a DedicatedGameServer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServer.
func (c *DGSAutoScalerController) enqueueDedicatedGameServerCollection(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.controllerHelper.Workqueue.AddRateLimited(key)
}

// Run initiates the AutoScalerController
func (c *DGSAutoScalerController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	return c.controllerHelper.Run(controllerThreadiness, stopCh)
}

func (c *DGSAutoScalerController) handleDedicatedGameServerCollection(obj interface{}) {
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

func (c *DGSAutoScalerController) handleDedicatedGameServer(obj interface{}) {
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
