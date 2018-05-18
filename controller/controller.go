package controller

import (
	"fmt"
	"log"
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"

	dgsclientset "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned"
	dgsv1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/clientset/versioned/typed/dedicatedgameserver/v1"
	informerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/informers/externalversions/dedicatedgameserver/v1"
	listerdgs "github.com/dgkanatsios/azuregameserversscalingkubernetes/shared/pkg/client/listers/dedicatedgameserver/v1"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const namespace = apiv1.NamespaceDefault

type DedicatedGameServerController struct {
	podGetter             corev1.PodsGetter
	dgsGetter             dgsv1.DedicatedGameServersGetter
	podLister             listercorev1.PodLister
	dgsLister             listerdgs.DedicatedGameServerLister
	podListerSynced       cache.InformerSynced
	dgsListerSynced       cache.InformerSynced
	namespaceGetter       corev1.NamespacesGetter
	namespaceLister       listercorev1.NamespaceLister
	namespaceListerSynced cache.InformerSynced

	queue workqueue.RateLimitingInterface
}

func NewDedicatedGameServerController(client *kubernetes.Clientset, dgsclient *dgsclientset.Clientset,
	podInformer informercorev1.PodInformer, dgsInformer informerdgs.DedicatedGameServerInformer) *DedicatedGameServerController {
	c := &DedicatedGameServerController{
		podGetter:       client.CoreV1(),
		dgsGetter:       dgsclient.AzureV1(),
		podLister:       podInformer.Lister(),
		dgsLister:       dgsInformer.Lister(),
		podListerSynced: podInformer.Informer().HasSynced,
		dgsListerSynced: dgsInformer.Informer().HasSynced,
		queue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DedicatedGameServerSync"),
	}

	dgsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleDedicatedGameServerAdd(obj, c.podLister, client)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handleDedicatedGameServerUpdate(oldObj, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				handleDedicatedGameServerDelete(obj, client)
			},
		},
	)

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handlePodAdd(obj, client, dgsclient)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handlePodUpdate(oldObj, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				handlePodDelete(obj)
			},
		},
	)

	return c
}

func (c *DedicatedGameServerController) Run(stop <-chan struct{}) {
	var wg sync.WaitGroup

	defer func() {
		// make sure the work queue is shut down which will trigger workers to end
		log.Print("shutting down queue")
		c.queue.ShutDown()

		// wait on the workers
		log.Print("shutting down workers")
		wg.Wait()

		log.Print("workers are all done")
	}()

	log.Print("waiting for cache sync")
	if !cache.WaitForCacheSync(
		stop,
		c.podListerSynced,
		c.dgsListerSynced) {
		log.Print("timed out waiting for cache sync")
		return
	}
	log.Print("caches are synced")

	go func() {
		// runWorker will loop until "something bad" happens. wait.Until will
		// then rekick the worker after one second.
		wait.Until(c.runWorker, time.Second, stop)
		// tell the WaitGroup this worker is done
		wg.Done()
	}()

	// wait until we're told to stop
	log.Print("waiting for stop signal")
	<-stop
	log.Print("received stop signal")
}

func (c *DedicatedGameServerController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will
	// automatically wait until there's work available, so we don't worry
	// about secondary waits
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false
// when it's time to quit.
func (c *DedicatedGameServerController) processNextWorkItem() bool {
	// pull the next work item from queue.  It should be a key we use to lookup
	// something in a cache
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// you always have to indicate to the queue that you've completed a piece of
	// work
	defer c.queue.Done(key)

	// do your work on the key.  This method will contains your "do stuff" logic
	err := c.doSync()
	if err == nil {
		// if you had no error, tell the queue to stop tracking history for your
		// key. This will reset things like failure counts for per-item rate
		// limiting
		c.queue.Forget(key)
		return true
	}

	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring
	runtime.HandleError(fmt.Errorf("doSync failed with: %v", err))

	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	c.queue.AddRateLimited(key)

	return true
}

// func (c *DedicatedGameServerController) getSecretsInNS(ns string) ([]*core.Secret, error) {
// 	rawSecrets, err := c.secretLister.Secrets(ns).List(labels.Everything())
// 	if err != nil {
// 		return nil, err
// 	}

// 	var secrets []*core.Secret
// 	for _, secret := range rawSecrets {
// 		if _, ok := secret.Annotations[secretSyncAnnotation]; ok {
// 			secrets = append(secrets, secret)
// 		}
// 	}
// 	return secrets, nil
// }

func (c *DedicatedGameServerController) doSync() error {
	log.Printf("Starting doSync")

	//get DedicatedGameServers
	c.dgsLister.DedicatedGameServers(namespace)

	log.Print("Finishing doSync")
	return nil
}

// func (c *DedicatedGameServerController) SyncNamespace(secrets []*core.Secret, ns string) {
// 	// 1. Create/Update all of the secrets in this namespace
// 	for _, secret := range secrets {
// 		newSecretInf, _ := scheme.Scheme.DeepCopy(secret)
// 		newSecret := newSecretInf.(*core.Secret)
// 		newSecret.Namespace = ns
// 		newSecret.ResourceVersion = ""
// 		newSecret.UID = ""

// 		log.Printf("Creating %v/%v", ns, secret.Name)
// 		_, err := c.secretGetter.Secrets(ns).Create(newSecret)
// 		if apierrors.IsAlreadyExists(err) {
// 			log.Printf("Scratch that, updating %v/%v", ns, secret.Name)
// 			_, err = c.secretGetter.Secrets(ns).Update(newSecret)
// 		}
// 		if err != nil {
// 			log.Printf("Error adding secret %v/%v: %v", ns, secret.Name, err)
// 		}
// 	}

// 	// 2. Delete secrets that have annotation but are not in our src list
// 	srcSecrets := sets.String{}
// 	targetSecrets := sets.String{}

// 	for _, secret := range secrets {
// 		srcSecrets.Insert(secret.Name)
// 	}

// 	targetSecretList, err := c.getSecretsInNS(ns)
// 	if err != nil {
// 		log.Printf("Error listing secrets in %v: %v", ns, err)
// 	}
// 	for _, secret := range targetSecretList {
// 		targetSecrets.Insert(secret.Name)
// 	}

// 	deleteSet := targetSecrets.Difference(srcSecrets)
// 	for secretName, _ := range deleteSet {
// 		log.Printf("Delete %v/%v", ns, secretName)
// 		err = c.secretGetter.Secrets(ns).Delete(secretName, nil)
// 		if err != nil {
// 			log.Printf("Error deleting %v/%v: %v", ns, secretName, err)
// 		}
// 	}
// }
