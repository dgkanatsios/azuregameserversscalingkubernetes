package controller

import (
	"reflect"
	"testing"
	"time"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/shared"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	informers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
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

func newFixture(t *testing.T) *fixture {

	//stupid hack
	//currently, DGS names are generated randomly
	//however, we can't compare random names using deepEqual tests
	//so, we'll override the method that generates the names
	shared.GenerateRandomName = func(prefix string) string {
		return prefix
	}

	f := &fixture{}
	f.t = t
	f.dgsObjects = []runtime.Object{}
	return f
}

func newDedicatedGameServerCollection(name string, replicas int32, ports []dgsv1alpha1.PortInfo, image string, startMap string) *dgsv1alpha1.DedicatedGameServerCollection {
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

func (f *fixture) newDedicatedGameServerCollectionController() (*DedicatedGameServerCollectionController, informers.SharedInformerFactory) {
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

func (f *fixture) run(dgsColName string) {
	f.runController(dgsColName, true, false)
}

func (f *fixture) runExpectError(dgsColName string) {
	f.runController(dgsColName, true, true)
}

func (f *fixture) runController(dgsColName string, startInformers bool, expectError bool) {
	testController, crdInformers := f.newDedicatedGameServerCollectionController()
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
	actions := filterInformerActions(f.dgsClient.Actions())

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

func (f *fixture) expectCreateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *fixture) expectUpdateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *fixture) expectUpdateDedicatedGameServerCollectionStatusAction(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservercollections"}, dgsCol.Namespace, dgsCol)
	f.dgsActions = append(f.dgsActions, action)
}

func getKey(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(dgsCol)
	if err != nil {
		t.Errorf("Unexpected error getting key for DGSCol %v: %v", dgsCol.Name, err)
		return ""
	}
	return key
}

func TestCreatesDedicatedGameServerCollection(t *testing.T) {
	f := newFixture(t)
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, "startMap", "myimage", 1, nil)

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	expDGS := shared.NewDedicatedGameServer(dgsCol, "test", nil, "startMap", "myimage")

	f.expectCreateDedicatedGameServerAction(expDGS)

	f.run(getKey(dgsCol, t))
}

func TestUpdateDedicatedGameServerCollectionStatus(t *testing.T) {
	f := newFixture(t)
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, "startMap", "myimage", 1, nil)
	dgs := shared.NewDedicatedGameServer(dgsCol, "test", nil, "startMap", "myimage")

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.expectUpdateDedicatedGameServerCollectionStatusAction(dgsCol)
	f.run(getKey(dgsCol, t))
}

// func TestUpdateDeployment(t *testing.T) {
// 	f := newFixture(t)
// 	foo := newFoo("test", int32Ptr(1))
// 	d := newDeployment(foo)

// 	Update replicas
// 	foo.Spec.Replicas = int32Ptr(2)
// 	expDeployment := newDeployment(foo)

// 	f.fooLister = append(f.fooLister, foo)
// 	f.objects = append(f.objects, foo)
// 	f.deploymentLister = append(f.deploymentLister, d)
// 	f.kubeobjects = append(f.kubeobjects, d)

// 	f.expectUpdateFooStatusAction(foo)
// 	f.expectUpdateDeploymentAction(expDeployment)
// 	f.run(getKey(foo, t))
// }

// func TestNotControlledByUs(t *testing.T) {
// 	f := newFixture(t)
// 	foo := newFoo("test", int32Ptr(1))
// 	d := newDeployment(foo)

// 	d.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}

// 	f.fooLister = append(f.fooLister, foo)
// 	f.objects = append(f.objects, foo)
// 	f.deploymentLister = append(f.deploymentLister, d)
// 	f.kubeobjects = append(f.kubeobjects, d)

// 	f.runExpectError(getKey(foo, t))
// }

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// noise level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "dedicatedgameservercollections") ||
				action.Matches("watch", "dedicatedgameservercollections") ||
				action.Matches("list", "dedicatedgameservers") ||
				action.Matches("watch", "dedicatedgameservers")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateAction:
		e, _ := expected.(core.CreateAction)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintDiff(expObject, object))
		}
	case core.UpdateAction:
		e, _ := expected.(core.UpdateAction)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintDiff(expObject, object))
		}
	case core.PatchAction:
		e, _ := expected.(core.PatchAction)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, expPatch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintDiff(expPatch, patch))
		}
	}
}
