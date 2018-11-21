package dgscollection

import (
	"testing"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/controller/testhelpers"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type dgsColFixture struct {
	t *testing.T

	k8sClient *k8sfake.Clientset
	dgsClient *fake.Clientset
	// Objects to put in the store.
	dgsColLister []*dgsv1alpha1.DedicatedGameServerCollection
	dgsLister    []*dgsv1alpha1.DedicatedGameServer
	// Actions expected to happen on the client.
	dgsActions []testhelpers.ExtendedAction
	// Objects from here preloaded into NewSimpleFake.
	k8sObjects []runtime.Object
	dgsObjects []runtime.Object

	clock clockwork.FakeClock
}

func newDGSColFixture(t *testing.T) *dgsColFixture {

	f := &dgsColFixture{}
	f.t = t
	f.dgsObjects = []runtime.Object{}
	f.clock = clockwork.NewFakeClockAt(testhelpers.FixedTime)
	return f
}

func (f *dgsColFixture) newDedicatedGameServerCollectionController() (*DGSCollectionController, dgsinformers.SharedInformerFactory) {
	f.k8sClient = k8sfake.NewSimpleClientset(f.k8sObjects...)
	f.dgsClient = fake.NewSimpleClientset(f.dgsObjects...)

	dgsInformers := dgsinformers.NewSharedInformerFactory(f.dgsClient, testhelpers.NoResyncPeriodFunc())

	testController, err := NewDedicatedGameServerCollectionController(f.k8sClient, f.dgsClient,
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServerCollections(),
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers(), nil)

	if err != nil {
		f.t.Fatalf("Error in initializing DGSCol: %s", err.Error())
	}

	testController.dgsColListerSynced = testhelpers.AlwaysReady
	testController.dgsListerSynced = testhelpers.AlwaysReady
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

	//for this controller, we do care only the actions on dgsClient
	actions := filterInformerActionsDGSCol(f.dgsClient.Actions())

	for i, action := range actions {
		if len(f.dgsActions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.dgsActions), actions[i:])
			break
		}

		expectedAction := f.dgsActions[i]

		testhelpers.CheckAction(expectedAction, action, f.t)
	}

	if len(f.dgsActions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.dgsActions)-len(actions), f.dgsActions[len(actions):])
	}
}

func (f *dgsColFixture) expectCreateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	extAction := testhelpers.ExtendedAction{Action: action, Assertions: assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *dgsColFixture) expectUpdateDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d)
	extAction := testhelpers.ExtendedAction{Action: action, Assertions: assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *dgsColFixture) expectDeleteDedicatedGameServerAction(d *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewDeleteAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, d.Namespace, d.Name)
	extAction := testhelpers.ExtendedAction{Action: action, Assertions: assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *dgsColFixture) expectUpdateDedicatedGameServerCollectionAction(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservercollections"}, dgsCol.Namespace, dgsCol)
	extAction := testhelpers.ExtendedAction{Action: action, Assertions: assertions}
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

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, testhelpers.PodSpec)

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	expDGS := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)

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

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, testhelpers.PodSpec)

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	for i := 0; i < 5; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)
		f.dgsLister = append(f.dgsLister, dgs)
		f.dgsObjects = append(f.dgsObjects, dgs)
	}

	//Update replicas
	dgsCol.Spec.Replicas = 10
	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)

	for i := 0; i < 5; i++ {
		dgsExpected := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)
		f.expectCreateDedicatedGameServerAction(dgsExpected, nil)
	}

	f.run(getKeyDGSCol(dgsCol, t))

	dgss, err := f.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assertDGSList(t, dgss.Items, 10)
}

func TestDecreaseReplicasOnDedicatedGameServerCollection(t *testing.T) {
	f := newDGSColFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, testhelpers.PodSpec)

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	for i := 0; i < 5; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)
		f.dgsLister = append(f.dgsLister, dgs)
		f.dgsObjects = append(f.dgsObjects, dgs)
	}

	//Update replicas
	dgsCol.Spec.Replicas = 3
	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)

	for i := 0; i < 2; i++ {
		dgsExpected := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)
		f.expectUpdateDedicatedGameServerAction(dgsExpected, func(actual runtime.Object) {
			dgs := actual.(*dgsv1alpha1.DedicatedGameServer)
			assert.Equal(t, true, dgs.Status.MarkedForDeletion)
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
		} else if _, ok := dgs.Labels[shared.LabelOriginalDedicatedGameServerCollectionName]; ok && dgs.Status.MarkedForDeletion {
			countNotInCollection++
		} else {
			t.Error("we should not be here")
		}
	}
	assert.Equal(t, 3, countInCollection)
	assert.Equal(t, 2, countNotInCollection)
}

func TestFailDedicatedGameServerCollectionForFirstTimeRemove(t *testing.T) {
	f := newDGSColFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, testhelpers.PodSpec)
	dgsCol.Status.DGSCollectionHealth = dgsv1alpha1.DGSColHealthy
	dgsCol.Status.PodCollectionState = corev1.PodRunning
	dgsCol.Spec.DGSMaxFailures = 2
	dgsCol.Spec.DGSFailBehavior = dgsv1alpha1.Remove

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)
	var failedDGS *dgsv1alpha1.DedicatedGameServer
	for i := 0; i < 5; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)

		dgs.Status.Health = dgsv1alpha1.DGSHealthy
		dgs.Status.PodPhase = corev1.PodRunning

		//set one to failed
		if i == 3 {
			dgs.Status.Health = dgsv1alpha1.DGSFailed
			failedDGS = dgs
		}
		f.dgsLister = append(f.dgsLister, dgs)
		f.dgsObjects = append(f.dgsObjects, dgs)
	}

	f.expectUpdateDedicatedGameServerAction(failedDGS, nil) //DGS that's removed from the collection
	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, func(actual runtime.Object) {
		dgsCol := actual.(*dgsv1alpha1.DedicatedGameServerCollection)
		assert.Equal(t, dgsv1alpha1.DGSColFailed, dgsCol.Status.DGSCollectionHealth)
		assert.Equal(t, int32(1), dgsCol.Status.DGSTimesFailed)
	})

	f.run(getKeyDGSCol(dgsCol, t))

	dgss, err := f.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assertDGSList(t, dgss.Items, 5) //5 -> 4 in the collection, 1 out. 1 more will be created in next controller loop
	for i := 0; i < 5; i++ {
		dgs := dgss.Items[i]
		if val, ok := dgs.Labels[shared.LabelDedicatedGameServerCollectionName]; ok && val == dgsCol.Name && len(dgs.OwnerReferences) > 0 {
			assert.Equal(t, dgsv1alpha1.DGSHealthy, dgs.Status.Health)
		} else {
			assert.Equal(t, dgsv1alpha1.DGSFailed, dgs.Status.Health)
		}
	}
}

func TestFailDedicatedGameServerCollectionPassedThresholdRemove(t *testing.T) {
	testFailSurpassThreshold(t, dgsv1alpha1.Remove)
}

func TestFailDedicatedGameServerCollectionForFirstTimeDelete(t *testing.T) {
	f := newDGSColFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, testhelpers.PodSpec)
	dgsCol.Status.DGSCollectionHealth = dgsv1alpha1.DGSColHealthy
	dgsCol.Status.PodCollectionState = corev1.PodRunning
	dgsCol.Spec.DGSMaxFailures = 2
	dgsCol.Spec.DGSFailBehavior = dgsv1alpha1.Delete

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)
	var failedDGS *dgsv1alpha1.DedicatedGameServer
	for i := 0; i < 5; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)

		dgs.Status.Health = dgsv1alpha1.DGSHealthy
		dgs.Status.PodPhase = corev1.PodRunning

		//set one to failed
		if i == 3 {
			dgs.Status.Health = dgsv1alpha1.DGSFailed
			failedDGS = dgs
		}
		f.dgsLister = append(f.dgsLister, dgs)
		f.dgsObjects = append(f.dgsObjects, dgs)
	}

	f.expectDeleteDedicatedGameServerAction(failedDGS, nil) //DGS that's removed from the collection
	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, func(actual runtime.Object) {
		dgsCol := actual.(*dgsv1alpha1.DedicatedGameServerCollection)
		assert.Equal(t, dgsv1alpha1.DGSColFailed, dgsCol.Status.DGSCollectionHealth)
		assert.Equal(t, int32(1), dgsCol.Status.DGSTimesFailed)
	})

	f.run(getKeyDGSCol(dgsCol, t))

	dgss, err := f.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assertDGSList(t, dgss.Items, 4)
	for i := 0; i < 4; i++ {
		dgs := dgss.Items[i]
		if val, ok := dgs.Labels[shared.LabelDedicatedGameServerCollectionName]; ok && val == dgsCol.Name && len(dgs.OwnerReferences) > 0 {
			assert.Equal(t, dgsv1alpha1.DGSHealthy, dgs.Status.Health)
		} else {
			t.Error("There shouldn't be a DGS without a parent")
		}
	}
}

func TestFailDedicatedGameServerCollectionPassedThresholdDelete(t *testing.T) {
	testFailSurpassThreshold(t, dgsv1alpha1.Delete)
}

func testFailSurpassThreshold(t *testing.T, fb dgsv1alpha1.DedicatedGameServerFailBehavior) {
	f := newDGSColFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 5, testhelpers.PodSpec)
	dgsCol.Status.DGSCollectionHealth = dgsv1alpha1.DGSColHealthy
	dgsCol.Status.PodCollectionState = corev1.PodRunning
	dgsCol.Spec.DGSMaxFailures = 2
	dgsCol.Status.DGSTimesFailed = 2
	dgsCol.Spec.DGSFailBehavior = fb

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, nil)

	for i := 0; i < 5; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)

		dgs.Status.Health = dgsv1alpha1.DGSHealthy
		dgs.Status.PodPhase = corev1.PodRunning

		//set one to failed
		if i == 3 {
			dgs.Status.Health = dgsv1alpha1.DGSFailed
		}
		f.dgsLister = append(f.dgsLister, dgs)
		f.dgsObjects = append(f.dgsObjects, dgs)
	}

	f.expectUpdateDedicatedGameServerCollectionAction(dgsCol, func(actual runtime.Object) {
		dgsCol := actual.(*dgsv1alpha1.DedicatedGameServerCollection)
		assert.Equal(t, dgsv1alpha1.DGSColNeedsIntervention, dgsCol.Status.DGSCollectionHealth)
		assert.Equal(t, int32(2), dgsCol.Status.DGSTimesFailed)
	})

	f.run(getKeyDGSCol(dgsCol, t))

	dgss, err := f.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(shared.GameNamespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assertDGSList(t, dgss.Items, 5)
	var runningCount, failedCount int
	for i := 0; i < 5; i++ {
		dgs := dgss.Items[i]
		if val, ok := dgs.Labels[shared.LabelDedicatedGameServerCollectionName]; ok && val == dgsCol.Name && len(dgs.OwnerReferences) > 0 {
			if dgs.Status.Health == dgsv1alpha1.DGSHealthy {
				runningCount++
			} else if dgs.Status.Health == dgsv1alpha1.DGSFailed {
				failedCount++
			}
		}
	}
	assert.Equal(t, 4, runningCount)
	assert.Equal(t, 1, failedCount)
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
