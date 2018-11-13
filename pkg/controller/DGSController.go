package controller

import (
	"fmt"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	logrus "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

// DGSController is the struct that contains necessary fields for the DGS controller
type DGSController struct {
	dgsClient  dgsclientset.Interface
	podClient  kubernetes.Interface
	nodeClient kubernetes.Interface

	dgsLister  listerdgs.DedicatedGameServerLister
	podLister  listercorev1.PodLister
	nodeLister listercorev1.NodeLister

	dgsListerSynced  cache.InformerSynced
	podListerSynced  cache.InformerSynced
	nodeListerSynced cache.InformerSynced

	logger *logrus.Logger

	portRegistry *PortRegistry
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	controllerHelper *controllerHelper
}

// NewDedicatedGameServerController creates a new DedicatedGameServerController
func NewDedicatedGameServerController(client kubernetes.Interface, dgsclient dgsclientset.Interface,
	dgsInformer informerdgs.DedicatedGameServerInformer,
	podInformer informercorev1.PodInformer, nodeInformer informercorev1.NodeInformer, portRegistry *PortRegistry) *DGSController {

	c := &DGSController{
		dgsClient:        dgsclient,
		podClient:        client, //getter hits the live API server (can also create/update objects)
		nodeClient:       client,
		dgsLister:        dgsInformer.Lister(),
		podLister:        podInformer.Lister(), //lister hits the cache
		nodeLister:       nodeInformer.Lister(),
		dgsListerSynced:  dgsInformer.Informer().HasSynced,
		podListerSynced:  podInformer.Informer().HasSynced,
		nodeListerSynced: nodeInformer.Informer().HasSynced,
		portRegistry:     portRegistry,
		logger:           shared.Logger(),
	}

	c.controllerHelper = &controllerHelper{
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DedicatedGameServerSync"),
		logger:         c.logger,
		syncHandler:    c.syncHandler,
		cacheSyncs:     []cache.InformerSynced{c.nodeListerSynced, c.dgsListerSynced, c.dgsListerSynced, c.podListerSynced},
		controllerType: "DedicatedGameServerController",
	}

	// Create event broadcaster
	// Add DedicatedGameServerController types to the default Kubernetes Scheme so Events can be
	// logged for DedicatedGameServerController types.
	dgsscheme.AddToScheme(dgsscheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(dgsscheme.Scheme, corev1.EventSource{Component: dgsControllerAgentName})

	c.logger.Info("Setting up event handlers for DedicatedGameServer controller")

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.logger.Info("DedicatedGameServer controller - add DGS")
				c.handleDedicatedGameServer(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Info("DedicatedGameServer controller - update DGS")
				oldDGS := oldObj.(*dgsv1alpha1.DedicatedGameServer)
				newDGS := newObj.(*dgsv1alpha1.DedicatedGameServer)

				if oldDGS.ResourceVersion == newDGS.ResourceVersion {
					return
				}

				if c.hasDGSChanged(oldDGS, newDGS) {
					c.handleDedicatedGameServer(newObj)
				}

			},
			DeleteFunc: func(obj interface{}) {
				c.logger.Info("DedicatedGameServer controller - delete DGS")
				c.handleDedicatedGameServerDelete(obj)
			},
		},
	)
	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.logger.Info("DedicatedGameServer controller - add pod")
				c.handlePod(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.logger.Info("DedicatedGameServer controller - update pod")
				oldPod := oldObj.(*corev1.Pod)
				newPod := newObj.(*corev1.Pod)

				if oldPod.ResourceVersion == newPod.ResourceVersion {
					return
				}
				c.handlePod(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				c.logger.Info("DedicatedGameServer controller - delete pod")
				c.handlePod(obj)
			},
		},
	)
	return c
}

func (c *DGSController) handlePod(obj interface{}) {
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
		c.logger.Infof("Recovered deleted Pod object '%s' from tombstone", object.GetName())
	}

	//if this Pod has a parent DGS
	if len(object.GetOwnerReferences()) > 0 && object.GetOwnerReferences()[0].Kind == shared.DedicatedGameServerKind {
		//find it
		dgs, err := c.dgsLister.DedicatedGameServers(object.GetNamespace()).Get(object.GetOwnerReferences()[0].Name)

		if err != nil {
			runtime.HandleError(fmt.Errorf("Warning: cannot get DedicatedGameServer for Pod %s because of %s. Maybe it has been deleted?", object.GetName(), err.Error()))
			return
		}
		//and enqueue it
		c.enqueueDedicatedGameServer(dgs)
	}
}

func (c *DGSController) handleDedicatedGameServer(obj interface{}) {
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
		c.logger.Infof("Recovered deleted DedicatedGameServer object '%s' from tombstone", object.GetName())
	}

	c.enqueueDedicatedGameServer(object)
}

func (c *DGSController) handleDedicatedGameServerDelete(obj interface{}) {
	dgs, ok := obj.(*dgsv1alpha1.DedicatedGameServer)
	if ok {
		//make sure all ports are deleted from the registry
		c.portRegistry.DeregisterServerPorts(dgs.Spec.PortsToExpose)
	}
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DedicatedGameServer resource
// with the current status of the resource.
func (c *DGSController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, dgsName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// try to get the DedicatedGameServer
	dgsTemp, err := c.dgsLister.DedicatedGameServers(namespace).Get(dgsName)
	if err != nil {
		if errors.IsNotFound(err) {
			// DedicatedGameServer not found
			runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' in work queue no longer exists", key))
			return nil
		}
		c.logger.Error(err.Error())
		return err
	}

	// DGS is being terminated
	if !dgsTemp.DeletionTimestamp.IsZero() {
		c.logger.WithField("DedicatedGameServerName", dgsTemp.Name).Info("DedicatedGameServer is being terminated")
		return nil
	}

	//check if DGS is markedForDeletion and has zero players connected to it
	//if this is the case, then it's time to delete the DGS
	if c.isDGSMarkedForDeletionWithZeroPlayers(dgsTemp) {
		return c.handleDGSMarkedForDeletionWithZeroPlayers(dgsTemp)
	}

	// find the pod that belongs to this DGS
	pod, err := c.getPodForDGS(dgsTemp)
	if err != nil {
		c.logger.WithField("Name", dgsTemp.Name).Error("Error in listing pod(s) for this Dedicated Game Server")
		return err
	}

	if pod == nil {
		// no pod found, so create one
		err = c.createNewPod(dgsTemp)
		if err != nil {
			c.logger.WithField("Name", dgsTemp.Name).Error("Error creating Pod for DedicatedGame Server")
			c.recorder.Event(dgsTemp, corev1.EventTypeWarning, "Error creating Pod for DedicatedGameServer", err.Error())
			return err
		}
		return nil //exiting, will get pod details on pod.Update event (will arrive later in the workqueue)
	}

	// pod found

	// try to update DGS with Node's Public IP
	// get the Node/Public IP for this Pod
	var ip string
	if pod.Spec.NodeName != "" { //no-empty string => pod has been scheduled
		ip, err = c.getPublicIPForNode(pod.Spec.NodeName)
		if err != nil {
			c.logger.WithField("Node", pod.Spec.NodeName).Error("Error in getting Public IP for Node")
			c.recorder.Event(pod, corev1.EventTypeWarning, "Error in getting Public IP for the Node", err.Error())
			return err
		}
	}

	// let's update the DGS
	dgsToUpdate := dgsTemp.DeepCopy()
	c.logger.WithFields(logrus.Fields{
		"serverName":      dgsTemp.Name,
		"currentDGSState": dgsTemp.Status.DedicatedGameServerState,
		"currentPodState": dgsTemp.Status.PodState,
		"currentPublicIP": dgsTemp.Status.PublicIP,
		"currentNodeName": dgsTemp.Status.NodeName,
		"updatedPodState": pod.Status.Phase,
		"updatedPublicIP": ip,
		"updatedNodeName": pod.Spec.NodeName,
	}).Info("Updating DedicatedGameServer")

	dgsToUpdate.Status.PodState = pod.Status.Phase

	dgsToUpdate.Status.PublicIP = ip
	dgsToUpdate.Status.NodeName = pod.Spec.NodeName

	_, err = c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Update(dgsToUpdate)

	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"Name":  dgsName,
			"Error": err.Error(),
		}).Error("Error in updating DedicatedGameServer")
		c.recorder.Event(dgsTemp, corev1.EventTypeWarning, fmt.Sprintf("Error in updating the DedicatedGameServer %s", dgsName), err.Error())
		return err
	}

	// if all goes well, record an event that everything went great
	c.recorder.Event(dgsTemp, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageResourceSynced, "DedicatedGameServer", dgsTemp.Name))
	return nil
}

func (c *DGSController) getPodForDGS(dgs *dgsv1alpha1.DedicatedGameServer) (*corev1.Pod, error) {
	// Let's see if the corresponding pod for this DGS exists
	// grab all the DedicatedGameServers that belong to this DedicatedGameServerCollection
	set := labels.Set{
		shared.LabelDedicatedGameServerName: dgs.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	pods, err := c.podLister.Pods(dgs.Namespace).List(selector)
	if err != nil {
		return nil, err //pods cannot be listed
	}
	// if no pod that belong to this DGS are found
	if len(pods) == 0 {
		return nil, nil
	}
	// if there's only one, then return it
	if len(pods) == 1 {
		return pods[0], nil
	}
	// if there's more than one, then probably there's one being terminated
	if len(pods) > 1 {
		// so we will return the first one that is running
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				return pod, nil
			}
		}
	}
	// not sure why we reached here, just return the first one
	return pods[0], nil
}

// enqueueDedicatedGameServer takes a DedicatedGameServer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DedicatedGameServer.
func (c *DGSController) enqueueDedicatedGameServer(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.controllerHelper.workqueue.AddRateLimited(key)
}

func (c *DGSController) createNewPod(dgs *dgsv1alpha1.DedicatedGameServer) error {
	pod := shared.NewPod(dgs,
		shared.APIDetails{
			SetActivePlayersURL: shared.GetActivePlayersSetURL(),
			SetServerStatusURL:  shared.GetServerStatusSetURL(),
		})
	_, err := c.podClient.CoreV1().Pods(dgs.Namespace).Create(pod)
	if err != nil {
		return err
	}
	return nil
}

// Run initiates the DedicatedGameServer controller
func (c *DGSController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	return c.controllerHelper.Run(controllerThreadiness, stopCh)
}
