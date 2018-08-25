package controller

import (
	"fmt"
	"testing"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	informers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type dgsFixture struct {
	t *testing.T

	k8sClient *k8sfake.Clientset
	dgsClient *fake.Clientset
	// Objects to put in the store.
	dgsColLister []*dgsv1alpha1.DedicatedGameServerCollection
	dgsLister    []*dgsv1alpha1.DedicatedGameServer
	// Actions expected to happen on the client.
	dgsActions []core.Action
	// Objects from here preloaded into NewSimpleFake.
	k8sObjects []runtime.Object
	dgsObjects []runtime.Object
}

func newDGSFixture(t *testing.T) *dgsFixture {

	//stupid hack
	//currently, DGS names are generated randomly
	//however, we can't compare random names using deepEqual tests
	//so, we'll override the method that generates the names
	i := 0
	shared.GenerateRandomName = func(prefix string) string {
		i++
		return fmt.Sprintf("%s%d", prefix, i)
	}

	f := &dgsFixture{}
	f.t = t
	f.dgsObjects = []runtime.Object{}
	return f
}

func newDedicatedGameServer(name string, replicas int32, ports []dgsv1alpha1.PortInfo, image string, startMap string) *dgsv1alpha1.DedicatedGameServerCollection {
	return &dgsv1alpha1.DedicatedGameServerCollection{
		TypeMeta: metav1.TypeMeta{APIVersion: dgsv1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: dgsv1alpha1.DedicatedGameServerCollectionSpec{
			Replicas: replicas,
			Ports:    ports,
			Image:    image,
			StartMap: startMap,
		},
	}
}

func (f *dgsFixture) newDedicatedGameServerController() (*DedicatedGameServerCollectionController, informers.SharedInformerFactory) {
	f.k8sClient = k8sfake.NewSimpleClientset(f.k8sObjects...)
	f.dgsClient = fake.NewSimpleClientset(f.dgsObjects...)

	crdInformers := informers.NewSharedInformerFactory(f.dgsClient, noResyncPeriodFunc())

	testController := NewDedicatedGameServerCollectionController(f.k8sClient, f.dgsClient,
		crdInformers.Azuregaming().V1alpha1().DedicatedGameServerCollections(),
		crdInformers.Azuregaming().V1alpha1().DedicatedGameServers())

	testController.dgsColListerSynced = alwaysReady
	testController.dgsListerSynced = alwaysReady
	testController.recorder = &record.FakeRecorder{}

	for _, dgsCol := range f.dgsColLister {
		crdInformers.Azuregaming().V1alpha1().DedicatedGameServerCollections().Informer().GetIndexer().Add(dgsCol)
	}

	for _, dgs := range f.dgsLister {
		crdInformers.Azuregaming().V1alpha1().DedicatedGameServers().Informer().GetIndexer().Add(dgs)
	}

	return testController, crdInformers
}

func (f *dgsFixture) run(dgsColName string) {
	f.runController(dgsColName, true, false)
}

func (f *dgsFixture) runExpectError(dgsColName string) {
	f.runController(dgsColName, true, true)
}

func (f *dgsFixture) runController(dgsColName string, startInformers bool, expectError bool) {
	testController, crdInformers := f.newDedicatedGameServerController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		crdInformers.Start(stopCh)
	}

	err := testController.syncHandler(dgsColName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing DGSCol: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing DGSCol, got nil")
	}

	//for this controller, we're getting only the actions on dgsClient
	actions := filterInformerActionsDGSCol(f.dgsClient.Actions())

	for i, action := range actions {
		if len(f.dgsActions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.dgsActions), actions[i:])
			break
		}

		expectedAction := f.dgsActions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.dgsActions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.dgsActions)-len(actions), f.dgsActions[len(actions):])
	}
}

func (f *dgsFixture) expectCreateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *dgsFixture) expectUpdateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *dgsFixture) expectUpdateDedicatedGameServerCollectionAction(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservercollections"}, dgsCol.Namespace, dgsCol)
	f.dgsActions = append(f.dgsActions, action)
}

func getKeyDGS(dgs *dgsv1alpha1.DedicatedGameServer, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(dgs)
	if err != nil {
		t.Errorf("Unexpected error getting key for DGS %v: %v", dgs.Name, err)
		return ""
	}
	return key
}
