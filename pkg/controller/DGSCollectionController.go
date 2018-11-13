package controller

import (
	"fmt"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	"github.com/jonboulle/clockwork"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubernetes "k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	cache "k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	workqueue "k8s.io/client-go/util/workqueue"
)

const (
	dgsColControllerAgentName = "dedigated-game-server-collection-controller"
)

// DGSCollectionController represents a controller for a Dedicated Game Server Collection
type DGSCollectionController struct {
	dgsColClient       dgsclientset.Interface
	dgsClient          dgsclientset.Interface
	dgsColLister       listerdgs.DedicatedGameServerCollectionLister
	dgsLister          listerdgs.DedicatedGameServerLister
	dgsColListerSynced cache.InformerSynced
	dgsListerSynced    cache.InformerSynced
	clock              clockwork.Clock
	logger             *logrus.Logger
	portRegistry       *PortRegistry
	recorder           record.EventRecorder
	controllerHelper   *controllerHelper
}

// NewDedicatedGameServerCollectionController initializes and returns a new DedicatedGameServerCollectionController instance
func NewDedicatedGameServerCollectionController(client kubernetes.Interface, dgsclient dgsclientset.Interface,
	dgsColInformer informerdgs.DedicatedGameServerCollectionInformer, dgsInformer informerdgs.DedicatedGameServerInformer,
	portRegistry *PortRegistry) (*DGSCollectionController, error) {
	dgsscheme.AddToScheme(dgsscheme.Scheme)

	c := &DGSCollectionController{
		dgsColClient:       dgsclient,
		dgsClient:          dgsclient,
		dgsColLister:       dgsColInformer.Lister(),
		dgsLister:          dgsInformer.Lister(),
		dgsColListerSynced: dgsColInformer.Informer().HasSynced,
		dgsListerSynced:    dgsInformer.Informer().HasSynced,
		portRegistry:       portRegistry,
		logger:             shared.Logger(),
	}

	c.controllerHelper = &controllerHelper{
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DedicatedGameServerCollectionSync"),
		logger:         c.logger,
		syncHandler:    c.syncHandler,
		cacheSyncs:     []cache.InformerSynced{c.dgsColListerSynced, c.dgsListerSynced},
		controllerType: "DedicatedGameServerCollectionController",
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: dgsColControllerAgentName})

	c.logger.Info("Setting up event handlers for DedicatedGameServerCollection controller")

	dgsColInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.logger.Print("DedicatedGameServerCollection controller - add DGSCol")
				c.handleDedicatedGameServerCollection(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Print("DedicatedGameServerCollection controller - update DGSCol")
				oldDGSCol := oldObj.(*dgsv1alpha1.DedicatedGameServerCollection)
				newDGSCol := newObj.(*dgsv1alpha1.DedicatedGameServerCollection)

				if oldDGSCol.ResourceVersion == newDGSCol.ResourceVersion {
					return
				}
				if c.hasSpecChanged(oldDGSCol, newDGSCol) {
					c.handleDedicatedGameServerCollection(newObj)
				}

			},
		},
	)

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.logger.Print("DedicatedGameServerCollection controller - add DGS")
				c.handleDedicatedGameServer(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Print("DedicatedGameServerCollection controller - update DGS")
				oldDGS := oldObj.(*dgsv1alpha1.DedicatedGameServer)
				newDGS := newObj.(*dgsv1alpha1.DedicatedGameServer)
				if oldDGS.ResourceVersion == newDGS.ResourceVersion {
					return
				}
				if c.hasDGSStatusChanged(oldDGS, newDGS) {
					c.handleDedicatedGameServer(newObj)
				}
			},
			DeleteFunc: func(obj interface{}) {
				c.logger.Print("DedicatedGameServerCollection controller - delete DGS")
				c.handleDedicatedGameServer(obj)
			},
		},
	)

	return c, nil
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DedicatedGameServer resource
// with the current status of the resource.
func (c *DGSCollectionController) syncHandler(key string) error {

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	dgsCol, err := c.dgsColLister.DedicatedGameServerCollections(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) { // The DedicatedGameServerCollection resource may no longer exist, in which case we stop processing
			runtime.HandleError(fmt.Errorf("DedicatedGameServerCollection '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	if !dgsCol.DeletionTimestamp.IsZero() { // DGSCol is being terminated
		c.logger.WithField("DGSColName", dgsCol.Name).Info("DGSCol is being terminated")
		return nil
	}

	err = c.reconcileStatuses(dgsCol)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"DedicatedGameServerCollection": dgsCol.Name, "Error": err.Error()}).Errorf("Failed to reconcile statuses of the DGSCol because of %s", err.Error())
		return err
	}

	// get the DGSs in the collection that have failed
	dgsFailed, err := c.getFailedDGSForDGSCol(dgsCol)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"DGSColName": dgsCol.Name, "Error": err.Error()}).Error("Cannot get Failed DedicatedGameServers for DedicatedGameServerCollection via Label selector")
		return err
	}
	// if there are DGS that have failed, handle them
	if len(dgsFailed) > 0 {
		err = c.handleDGSFailed(dgsCol, dgsFailed)
		if err != nil {
			c.logger.WithFields(logrus.Fields{"DGSColName": dgsCol.Name, "Error": err.Error()}).Error("Error in handling DGSFailed")
			return err
		}
		return nil
	}

	// DGSCol needs intervention
	if dgsCol.Status.DedicatedGameServerCollectionState == dgsv1alpha1.DGSColNeedsIntervention {
		c.logger.WithField("DGSColName", dgsCol.Name).Info("DGSCol is in NeedsIntervention state, will not proceed")
		return nil
	}

	// DGSCol is NOT in NeedsIntervention state, so we will proceed in checking available DGSs

	dgsExisting, err := c.getNotFailedDGSForDGSCol(dgsCol)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"DGSColName": dgsCol.Name, "Error": err.Error()}).Error("Cannot get non-Failed DedicatedGameServers for DedicatedGameServerCollection via Label selector")
		return err
	}

	dgsExistingCount := len(dgsExisting)

	// if there are less DedicatedGameServers than the ones we requested
	if dgsExistingCount < int(dgsCol.Spec.Replicas) {
		err = c.addDGSColReplicas(dgsCol, dgsExistingCount)
		if err != nil {
			c.recorder.Event(dgsCol, corev1.EventTypeWarning, "Cannot increase dedicated game servers", err.Error())
			c.logger.WithFields(logrus.Fields{"DGSColName": dgsCol.Name, "Error": err.Error()}).Error("Cannot increase dedicated game servers")
			return err
		}
		c.recorder.Event(dgsCol, corev1.EventTypeNormal, shared.DedicatedGameServerReplicasChanged, fmt.Sprintf(shared.MessageReplicasIncreased, "DedicatedGameServerCollection", dgsCol.Name, int(dgsCol.Spec.Replicas)-dgsExistingCount))
		return nil //exiting sync handler, further DGS updates will propagate here as well via another item in the workqueue

	} else if dgsExistingCount > int(dgsCol.Spec.Replicas) { //if there are more DGS than the ones we requested
		err = c.removeDGSColReplicas(dgsCol, dgsExisting)
		if err != nil {
			c.recorder.Event(dgsCol, corev1.EventTypeWarning, "Cannot decrease dedicated game servers", err.Error())
			c.logger.WithFields(logrus.Fields{"DGSColName": dgsCol.Name, "Error": err.Error()}).Error("Cannot decrease dedicated game servers")
			return err
		}
		c.recorder.Event(dgsCol, corev1.EventTypeNormal, shared.DedicatedGameServerReplicasChanged, fmt.Sprintf(shared.MessageReplicasDecreased, "DedicatedGameServerCollection", dgsCol.Name, dgsExistingCount-int(dgsCol.Spec.Replicas)))
		return nil //exiting sync handler, further DGS updates will propagate here as well via another item in the workqueue
	}

	c.recorder.Event(dgsCol, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageResourceSynced, "DedicatedGameServerCollection", dgsCol.Name))
	return nil
}

func (c *DGSCollectionController) handleDGSFailed(dgsCol *dgsv1alpha1.DedicatedGameServerCollection,
	failedDGSs []*dgsv1alpha1.DedicatedGameServer) error {

	if dgsCol.Status.DedicatedGameServerCollectionState == dgsv1alpha1.DGSColNeedsIntervention {
		return nil
	}

	if dgsCol.Status.DGSTimesFailed == dgsCol.Spec.DGSMaxFailures {
		err := c.setDGSColToNeedsIntervention(dgsCol)
		if err != nil {
			c.logger.WithFields(logrus.Fields{"Name": dgsCol.Name, "Error": err.Error()}).Errorf("Error in updating DedicatedGameServerCollection: %s", err.Error())
			return err
		}
		return nil
	}

	if dgsCol.Spec.DGSFailBehavior == "" || dgsCol.Spec.DGSFailBehavior == dgsv1alpha1.Remove {
		c.logger.WithFields(logrus.Fields{"DedicatedGameServerCollection": dgsCol.Name}).Info("Failed DGS - removing from DGSCol")

		for _, dgs := range failedDGSs {
			dgsToRemove := dgs.DeepCopy()                                                        // we remove the DGS from the DGSCol
			dgsToRemove.ObjectMeta.OwnerReferences = nil                                         // update the DGS so it has no owners
			delete(dgsToRemove.ObjectMeta.Labels, shared.LabelDedicatedGameServerCollectionName) //remove the DGSCol name from the DGS labels

			dgsToRemove.ObjectMeta.Labels[shared.LabelOriginalDedicatedGameServerCollectionName] = dgsCol.Name //mark its previous owner

			_, err := c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(dgs.Namespace).Update(dgsToRemove) //update the DGS CRD
			if err != nil {
				return err
			}
		}
	} else if dgsCol.Spec.DGSFailBehavior == dgsv1alpha1.Delete {
		c.logger.WithFields(logrus.Fields{"DedicatedGameServerCollection": dgsCol.Name}).Info("Failed DGS - deleting from the cluster")
		for _, dgs := range failedDGSs { // delete the DGS CRDs
			err := c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(dgs.Namespace).Delete(dgs.Name, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	c.increaseTimesFailed(dgsCol, len(failedDGSs))
	return nil
}

func (c *DGSCollectionController) handleDedicatedGameServerCollection(obj interface{}) {
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

func (c *DGSCollectionController) handleDedicatedGameServer(obj interface{}) {
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

	// DGS is being terminated
	if !object.GetDeletionTimestamp().IsZero() {
		c.logger.WithField("DedicatedGameServerName", object.GetName()).Info("DedicatedGameServer is being terminated")
		return
	}

	// when we get a DGS, we will enqueue the DGSCol only if
	// 1. DGS has a parent DGSCol, so that DGSCol get updated with new status etc.
	// 2. or DGS's old parent was a DGSCol and DGS has Failed or MarkedForDeletion

	if len(object.GetOwnerReferences()) > 0 { // it belongs to a collection
		dgsCol, err := c.dgsColLister.DedicatedGameServerCollections(object.GetNamespace()).Get(object.GetLabels()[shared.LabelDedicatedGameServerCollectionName])
		if err != nil {
			runtime.HandleError(fmt.Errorf("error getting a DedicatedGameServerCollection from the DedicatedGameServer with Name %s, err: %s", object.GetName(), err.Error()))
			return
		}
		c.enqueueDedicatedGameServerCollection(dgsCol)
	} else if value, ok := object.GetLabels()[shared.LabelOriginalDedicatedGameServerCollectionName]; ok { // did belong to a collection
		dgsCol, err := c.dgsColLister.DedicatedGameServerCollections(object.GetNamespace()).Get(value)
		if err != nil {
			runtime.HandleError(fmt.Errorf("error getting a DedicatedGameServerCollection from the DedicatedGameServer with Name %s, err: %s", object.GetName(), err.Error()))
			return
		}
		// its state is Failed or MarkedForDeletion
		// this check may be redundant (Why would a Running DGS be out of the collection?) but anyway
		if dgs := object.(*dgsv1alpha1.DedicatedGameServer); dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSFailed || dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSMarkedForDeletion {
			c.enqueueDedicatedGameServerCollection(dgsCol)
		}
	}
}

// enqueueDedicatedGameServerCollection takes a DedicatedGameServerCollection resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServerCollection.
func (c *DGSCollectionController) enqueueDedicatedGameServerCollection(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.controllerHelper.workqueue.AddRateLimited(key)
}

// Run starts the DedicatedGameServerCollectionController
func (c *DGSCollectionController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	return c.controllerHelper.Run(controllerThreadiness, stopCh)
}
