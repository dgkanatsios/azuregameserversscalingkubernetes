package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	"github.com/jonboulle/clockwork"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

var podSpec = corev1.PodSpec{
	Containers: []corev1.Container{
		corev1.Container{
			Ports: []corev1.ContainerPort{}, // if we create any ports here we need to initialize port registry before (during??) running the test
		},
	},
}

type dgsColFixture struct {
	t *testing.T

	k8sClient *k8sfake.Clientset
	dgsClient *fake.Clientset
	// Objects to put in the store.
	dgsColLister []*dgsv1alpha1.DedicatedGameServerCollection
	dgsLister    []*dgsv1alpha1.DedicatedGameServer
	// Actions expected to happen on the client.
	dgsActions []extendedAction
	// Objects from here preloaded into NewSimpleFake.
	k8sObjects []runtime.Object
	dgsObjects []runtime.Object

	clock clockwork.FakeClock
}

func newDGSColFixture(t *testing.T) *dgsColFixture {

	f := &dgsColFixture{}
	f.t = t
	f.dgsObjects = []runtime.Object{}
	f.clock = clockwork.NewFakeClockAt(fixedTime)
	return f
}

func (f *dgsColFixture) newDedicatedGameServerCollectionController() (*DGSCollectionController, dgsinformers.SharedInformerFactory) {
	f.k8sClient = k8sfake.NewSimpleClientset(f.k8sObjects...)
	f.dgsClient = fake.NewSimpleClientset(f.dgsObjects...)

	dgsInformers := dgsinformers.NewSharedInformerFactory(f.dgsClient, noResyncPeriodFunc())

	testController, err := NewDedicatedGameServerCollectionController(f.k8sClient, f.dgsClient,
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServerCollections(),
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers(), nil)

	if err != nil {
		f.t.Fatalf("Error in initializing DGSCol: %s", err.Error())
	}

	testController.dgsColListerSynced = alwaysReady
	testController.dgsListerSynced = alwaysReady
	testController.recorder = &record.FakeRecorder{}

	for _, dgsCol := range f.dgsColLister {
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServerCollections().Informer().GetIndexer().Add(dgsCol)
	}

	for _, dgs := range f.dgsLister {
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers().Informer().GetIndexer().Add(dgs)
	}

	return testController, dgsInformers
}

func (f *dgsColFixture) run(dgsColName string) {
	f.runController(dgsColName, true, false)
}

func (f *dgsColFixture) runExpectError(dgsColName string) {
	f.runController(dgsColName, true, true)
}

func (f *dgsColFixture) runController(dgsColName string, startInformers bool, expectError bool) {
	testController, dgsInformers := f.newDedicatedGameServerCollectionController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		dgsInformers.Start(stopCh)
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

func (f *dgsColFixture) expectCreateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *dgsColFixture) expectUpdateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *dgsColFixture) expectUpdateDedicatedGameServerCollectionAction(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservercollections"}, dgsCol.Namespace, dgsCol)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func getKeyDGSCol(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(dgsCol)
	if err != nil {
		t.Errorf("Unexpected error getting key for DGSCol %v: %v", dgsCol.Name, err)
		return ""
	}
	return key
}

func TestCreatesDedicatedGameServerCollection(t *testing.T) {
	f := newDGSColFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, podSpec)

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	expDGS := shared.NewDedicatedGameServer(dgsCol, podSpec)

	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)
	for i := 0; i < 5; i++ {
		f.expectCreateDedicatedGameServerAction(expDGS, nil)
	}
	f.run(getKeyDGSCol(dgsCol, t))

	dgss, err := f.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assertDGSList(t, dgss.Items, 5)

}

func TestIncreaseReplicasOnDedicatedGameServerCollection(t *testing.T) {
	f := newDGSColFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, podSpec)

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	for i := 0; i < 5; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)
		f.dgsLister = append(f.dgsLister, dgs)
		f.dgsObjects = append(f.dgsObjects, dgs)
	}

	//Update replicas
	dgsCol.Spec.Replicas = 10
	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)

	for i := 0; i < 5; i++ {
		dgsExpected := shared.NewDedicatedGameServer(dgsCol, podSpec)
		f.expectCreateDedicatedGameServerAction(dgsExpected, nil)
	}

	f.run(getKeyDGSCol(dgsCol, t))

	dgss, err := f.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assertDGSList(t, dgss.Items, 10)
}

func TestDecreaseReplicasOnDedicatedGameServerCollection(t *testing.T) {
	f := newDGSColFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, podSpec)

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	for i := 0; i < 5; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)
		f.dgsLister = append(f.dgsLister, dgs)
		f.dgsObjects = append(f.dgsObjects, dgs)
	}

	//Update replicas
	dgsCol.Spec.Replicas = 3
	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)

	for i := 0; i < 2; i++ {
		dgsExpected := shared.NewDedicatedGameServer(dgsCol, podSpec)
		f.expectUpdateDedicatedGameServerAction(dgsExpected, func(actual runtime.Object) {
			dgs := actual.(*dgsv1alpha1.DedicatedGameServer)
			assert.Equal(t, dgsv1alpha1.DGSMarkedForDeletion, dgs.Status.DedicatedGameServerState)
		})
	}

	f.run(getKeyDGSCol(dgsCol, t))

	dgss, err := f.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assertDGSList(t, dgss.Items, 5)

	countInCollection := 0
	countNotInCollection := 0

	for _, dgs := range dgss.Items {
		if _, ok := dgs.Labels[shared.LabelDedicatedGameServerCollectionName]; ok && len(dgs.OwnerReferences) > 0 {
			countInCollection++
		} else if _, ok := dgs.Labels[shared.LabelOriginalDedicatedGameServerCollectionName]; ok &&
			dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSMarkedForDeletion {
			countNotInCollection++
		}
	}
	assert.Equal(t, 3, countInCollection)
	assert.Equal(t, 2, countNotInCollection)
}

func assertDGSList(t *testing.T, dgss []dgsv1alpha1.DedicatedGameServer, count int) {
	assert.NotNil(t, dgss)
	assert.Equal(t, count, len(dgss))
}

// filterInformerActionsDGSCol filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// noise level in our tests.
func filterInformerActionsDGSCol(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		// we removed the len(action.GetNamespace()) == 0 because the PortRegistry is initialized in the shared.GameNamespace
		if action.Matches("get", "dedicatedgameservercollections") ||
			action.Matches("get", "dedicatedgameservers") ||
			action.Matches("list", "dedicatedgameservercollections") ||
			action.Matches("watch", "dedicatedgameservercollections") ||
			action.Matches("list", "dedicatedgameservers") ||
			action.Matches("watch", "dedicatedgameservers") {

			continue
		}
		ret = append(ret, action)
	}

	return ret
}
