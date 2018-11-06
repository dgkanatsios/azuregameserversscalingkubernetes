package controller

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	corev1 "k8s.io/api/core/v1"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
)

var fixedTime = time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)

type podAutoScalerFixture struct {
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

func newPodAutoScalerFixture(t *testing.T) *podAutoScalerFixture {

	f := &podAutoScalerFixture{}
	f.t = t

	f.k8sObjects = []runtime.Object{}
	f.dgsObjects = []runtime.Object{}

	f.clock = clockwork.NewFakeClockAt(fixedTime)
	return f
}

func (f *podAutoScalerFixture) newPodAutoScalerController() (*PodAutoScalerController, dgsinformers.SharedInformerFactory) {

	f.k8sClient = k8sfake.NewSimpleClientset(f.k8sObjects...)
	f.dgsClient = fake.NewSimpleClientset(f.dgsObjects...)

	dgsInformers := dgsinformers.NewSharedInformerFactory(f.dgsClient, noResyncPeriodFunc())

	testController := NewPodAutoScalerController(f.k8sClient, f.dgsClient,
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServerCollections(),
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers(), f.clock)

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

func (f *podAutoScalerFixture) run(dgsName string) {
	f.runController(dgsName, true, false)
}

func (f *podAutoScalerFixture) runExpectError(dgsName string) {
	f.runController(dgsName, true, true)
}

func (f *podAutoScalerFixture) runController(dgsName string, startInformers bool, expectError bool) {

	testController, dgsInformers := f.newPodAutoScalerController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		dgsInformers.Start(stopCh)
	}

	err := testController.syncHandler(dgsName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing DGS: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing DGS, got nil")
	}

	actions := filterInformerActionsPodAutoScaler(f.dgsClient.Actions())

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

func (f *podAutoScalerFixture) expectCreateDGSAction(dgs *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *podAutoScalerFixture) expectDeleteDGSAction(dgs *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewDeleteAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs.Name)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *podAutoScalerFixture) expectUpdateDGSAction(dgs *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *podAutoScalerFixture) expectUpdateDGSColAction(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservercollections"}, dgsCol.Namespace, dgsCol)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *podAutoScalerFixture) expectUpdateDGSColActionStatus(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Group: "azuregaming.com", Resource: "dedicatedgameservercollections", Version: "v1alpha1"}, dgsCol.Namespace, dgsCol)
	extAction := extendedAction{action, assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func TestScaleOutDGSCol(t *testing.T) {
	f := newPodAutoScalerFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.DgsAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerDgsAutoScalerDetails{
		MinimumReplicas:     1,
		MaximumReplicas:     5,
		ScaleInThreshold:    60,
		ScaleOutThreshold:   80,
		Enabled:             true,
		CoolDownInMinutes:   5,
		MaxPlayersPerServer: 10,
	}

	dgsCol.Spec.Replicas = 1
	dgsCol.Status.AvailableReplicas = 1
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)

	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DGSRunning

	dgs.Status.PodState = corev1.PodRunning

	dgs.Status.ActivePlayers = 9

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 2
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColCreating

	f.expectUpdateDGSColAction(expDGSCol, nil)
	//f.expectUpdateDGSColActionStatus(expDGSCol)

	f.run(getKeyDGSCol(dgsCol, t))
}

func TestScaleInDGSCol(t *testing.T) {
	f := newPodAutoScalerFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.DgsAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerDgsAutoScalerDetails{
		MinimumReplicas:     1,
		MaximumReplicas:     5,
		ScaleInThreshold:    60,
		ScaleOutThreshold:   80,
		Enabled:             true,
		CoolDownInMinutes:   5,
		MaxPlayersPerServer: 10,
	}

	dgsCol.Spec.Replicas = 2
	dgsCol.Status.AvailableReplicas = 2
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)
	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DGSRunning
	dgs.Status.PodState = corev1.PodRunning
	dgs.Status.ActivePlayers = 5

	dgs2 := shared.NewDedicatedGameServer(dgsCol, podSpec)
	dgs2.Status.DedicatedGameServerState = dgsv1alpha1.DGSRunning
	dgs2.Status.PodState = corev1.PodRunning
	dgs2.Status.ActivePlayers = 5

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.dgsLister = append(f.dgsLister, dgs2)
	f.dgsObjects = append(f.dgsObjects, dgs2)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 1
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColCreating

	f.expectUpdateDGSColAction(expDGSCol, nil)
	//f.expectUpdateDGSColActionStatus(expDGSCol)

	f.run(getKeyDGSCol(dgsCol, t))
}

func TestDoNothingBecauseOfCoolDown(t *testing.T) {
	f := newPodAutoScalerFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.DgsAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerDgsAutoScalerDetails{
		MinimumReplicas:            1,
		MaximumReplicas:            5,
		ScaleInThreshold:           60,
		ScaleOutThreshold:          80,
		Enabled:                    true,
		CoolDownInMinutes:          5,
		MaxPlayersPerServer:        10,
		LastScaleOperationDateTime: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC).String(),
	}

	f.clock.Advance(1 * time.Minute)

	dgsCol.Spec.Replicas = 1
	dgsCol.Status.AvailableReplicas = 1
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)

	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DGSRunning

	dgs.Status.PodState = corev1.PodRunning

	dgs.Status.ActivePlayers = 9

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 2
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColCreating

	//expect nothing

	f.run(getKeyDGSCol(dgsCol, t))
}

func TestWithMalformedLastScaleTime(t *testing.T) {

	f := newPodAutoScalerFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.DgsAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerDgsAutoScalerDetails{
		MinimumReplicas:            1,
		MaximumReplicas:            5,
		ScaleInThreshold:           60,
		ScaleOutThreshold:          80,
		Enabled:                    true,
		CoolDownInMinutes:          5,
		MaxPlayersPerServer:        10,
		LastScaleOperationDateTime: "DEFINITELY NOT GONNA BE PARSED AS DATETIME",
	}

	dgsCol.Spec.Replicas = 1
	dgsCol.Status.AvailableReplicas = 1
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)

	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DGSRunning

	dgs.Status.PodState = corev1.PodRunning

	dgs.Status.ActivePlayers = 9

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 2
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColCreating

	f.expectUpdateDGSColAction(expDGSCol, nil)
	//f.expectUpdateDGSColActionStatus(expDGSCol)

	f.run(getKeyDGSCol(dgsCol, t))
}

// filterInformerActionsDGS filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// noise level in our tests.
func filterInformerActionsPodAutoScaler(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "pods") ||
				action.Matches("watch", "pods") ||
				action.Matches("list", "dedicatedgameservers") ||
				action.Matches("watch", "dedicatedgameservers") ||
				action.Matches("list", "dedicatedgameservercollections") ||
				action.Matches("watch", "dedicatedgameservercollections") ||
				action.Matches("list", "nodes") ||
				action.Matches("watch", "nodes")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}
