package controller

import (
	"fmt"
	"time"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions/azuregaming/v1alpha1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/listers/azuregaming/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	record "k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const dgsControllerAgentName = "dedigated-game-server-controller"

type DedicatedGameServerController struct {
	dgsClient  dgsclientset.Interface
	podClient  kubernetes.Interface
	nodeClient kubernetes.Interface

	dgsLister  listerdgs.DedicatedGameServerLister
	podLister  listercorev1.PodLister
	nodeLister listercorev1.NodeLister

	dgsListerSynced  cache.InformerSynced
	podListerSynced  cache.InformerSynced
	nodeListerSynced cache.InformerSynced
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

func NewDedicatedGameServerController(client kubernetes.Interface, dgsclient dgsclientset.Interface,
	dgsInformer informerdgs.DedicatedGameServerInformer,
	podInformer informercorev1.PodInformer, nodeInformer informercorev1.NodeInformer) *DedicatedGameServerController {
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
		dgsClient:        dgsclient,
		podClient:        client, //getter hits the live API server (can also create/update objects)
		nodeClient:       client,
		dgsLister:        dgsInformer.Lister(),
		podLister:        podInformer.Lister(), //lister hits the cache
		nodeLister:       nodeInformer.Lister(),
		dgsListerSynced:  dgsInformer.Informer().HasSynced,
		podListerSynced:  podInformer.Informer().HasSynced,
		nodeListerSynced: nodeInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DedicatedGameServerSync"),
		recorder:         recorder,
	}

	log.Info("Setting up event handlers for DedicatedGameServer controller")

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("DedicatedGameServer controller - add DGS")
				c.handleDedicatedGameServer(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("DedicatedGameServer controller - update DGS")
				oldDGS := oldObj.(*dgsv1alpha1.DedicatedGameServer)
				newDGS := newObj.(*dgsv1alpha1.DedicatedGameServer)

				if oldDGS.ResourceVersion == newDGS.ResourceVersion {
					return
				}

				if shared.HasDedicatedGameServerChanged(oldDGS, newDGS) {
					c.handleDedicatedGameServer(newObj)
				}

			},
		},
	)
	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Print("DedicatedGameServer controller - add pod")
				c.handlePod(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Print("DedicatedGameServer controller - update pod")
				oldPod := oldObj.(*corev1.Pod)
				newPod := newObj.(*corev1.Pod)

				if oldPod.ResourceVersion == newPod.ResourceVersion {
					return
				}
				c.handlePod(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				log.Print("DedicatedGameServer controller - delete pod")
				c.handlePod(obj)
			},
		},
	)
	return c
}

func (c *DedicatedGameServerController) handlePod(obj interface{}) {
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
		log.Infof("Recovered deleted Pod object '%s' from tombstone", object.GetName())
	}

	//if this Pod has a parent DGS
	if len(object.GetOwnerReferences()) > 0 && object.GetOwnerReferences()[0].Kind == shared.DedicatedGameServerKind {
		//find it
		dgs, err := c.dgsLister.DedicatedGameServers(object.GetNamespace()).Get(object.GetOwnerReferences()[0].Name)

		if err != nil {
			runtime.HandleError(fmt.Errorf("error getting DedicatedGameServer for Pod %s. Maybe it has been deleted?", object.GetName()))
			return
		}
		//and enqueue it
		c.enqueueDedicatedGameServer(dgs)
	}
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

	c.enqueueDedicatedGameServer(object)
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
	dgsTemp, err := c.dgsLister.DedicatedGameServers(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// DedicatedGameServer not found
			runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' in work queue no longer exists", key))
			return nil
		}
		log.Error(err.Error())
		return err
	}

	//check if DGS is markedForDeletion and has zero players connected to it
	//if this is the case, then it's time to delete the DGS
	if c.isDGSMarkedForDeletionWithZeroPlayers(dgsTemp) {
		err = c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Delete(dgsTemp.Name, &metav1.DeleteOptions{})
		if err != nil {
			log.WithFields(log.Fields{
				"Name":  name,
				"Error": err.Error(),
			}).Error("Cannot delete DedicatedGameServer")
			runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' cannot be deleted", name))
			return err
		}
		log.WithField("Name", dgsTemp.Name).Info("DedicatedGameServer with state MarkedForDeletion and with 0 ActivePlayers was deleted")
		c.recorder.Event(dgsTemp, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageMarkedForDeletionDedicatedGameServerDeleted, dgsTemp.Name))
		return nil
	}

	// find the pod(s) the have this DGS as parent
	pod, err := c.getPodForDGS(dgsTemp)

	if err != nil {
		log.WithField("Name", dgsTemp.Name).Error("Error in listing pod(s) for this Dedicated Game Server")
		return err
	}

	if pod == nil {
		// no pod found, so create one
		err = c.createNewPod(dgsTemp)
		if err != nil {
			log.WithField("Name", dgsTemp.Name).Error("Error creating Pod for DedicatedGame Server")
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
			log.WithField("Node", pod.Spec.NodeName).Error("Error in getting Public IP for Node")
			c.recorder.Event(pod, corev1.EventTypeWarning, "Error in getting Public IP for the Node", err.Error())
			return err
		}
	}

	// let's update the DGS
	dgsToUpdate := dgsTemp.DeepCopy()
	log.WithFields(log.Fields{
		"serverName":      dgsTemp.Name,
		"currentPodState": dgsTemp.Status.PodState,
		"currentPublicIP": dgsTemp.Spec.PublicIP,
		"currentNodeName": dgsTemp.Spec.NodeName,
		"updatedPodState": pod.Status.Phase,
		"updatedPublicIP": ip,
		"updatedNodeName": pod.Spec.NodeName,
	}).Info("Updating DedicatedGameServer")

	dgsToUpdate.Status.PodState = pod.Status.Phase
	dgsToUpdate.Labels[shared.LabelPodState] = string(pod.Status.Phase)

	dgsToUpdate.Spec.PublicIP = ip
	dgsToUpdate.Spec.NodeName = pod.Spec.NodeName

	_, err = c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Update(dgsToUpdate)

	if err != nil {
		log.WithFields(log.Fields{
			"Name":  name,
			"Error": err.Error(),
		}).Error("Error in updating DedicatedGameServer")
		c.recorder.Event(dgsTemp, corev1.EventTypeWarning, fmt.Sprintf("Error in updating the DedicatedGameServer for pod %s", name), err.Error())
		return err
	}

	// if all goes well, record an event that everything went great
	c.recorder.Event(dgsTemp, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageResourceSynced, "DedicatedGameServer", dgsTemp.Name))
	return nil
}

func (c *DedicatedGameServerController) getPodForDGS(dgs *dgsv1alpha1.DedicatedGameServer) (*corev1.Pod, error) {
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

	//TODO: check if len(pods) > 1 ???
	//maybe there's a case that one pod is Terminating and another one is being created?
	return pods[0], nil
}

func (c *DedicatedGameServerController) isDGSMarkedForDeletionWithZeroPlayers(dgs *dgsv1alpha1.DedicatedGameServer) bool {
	//check its state and active players
	return dgs.Spec.ActivePlayers == 0 && dgs.Status.DedicatedGameServerState == dgsv1alpha1.DedicatedGameServerStateMarkedForDeletion
}

func (c *DedicatedGameServerController) getPublicIPForNode(nodeName string) (string, error) {
	node, err := c.nodeLister.Get(nodeName)
	if err != nil {
		return "", err
	}

	for _, x := range node.Status.Addresses {
		if x.Type == corev1.NodeExternalIP {
			return x.Address, nil
		}
	}

	return "", fmt.Errorf("node with name %s does not have a Public IP", nodeName)
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

func (c *DedicatedGameServerController) createNewPod(dgs *dgsv1alpha1.DedicatedGameServer) error {

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
func (c *DedicatedGameServerController) Run(controllerThreadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting DedicatedGameServer controller")

	// Wait for the caches for all controllers to be synced before starting workers
	log.Info("Waiting for informer caches to sync for DedicatedGameServer controller")
	if ok := cache.WaitForCacheSync(stopCh, c.nodeListerSynced, c.dgsListerSynced, c.podListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Info("Starting workers for DedicatedGameServer controller")

	// Launch a number of workers to process resources
	for i := 0; i < controllerThreadiness; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will
		// then rekick the worker after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers for DedicatedGameServer Controller")
	<-stopCh
	log.Info("Shutting down workers for DedicatedGameServer Controller")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *DedicatedGameServerController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	log.Info("Starting loop for DedicatedGameServer controller")
	for c.processNextWorkItem() {
	}
}
