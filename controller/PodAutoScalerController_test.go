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
	dgsActions []core.Action

	// Objects from here preloaded into NewSimpleFake.
	k8sObjects []runtime.Object
	dgsObjects []runtime.Object

	clock         clockwork.FakeClock
	namegenerator shared.FakeRandomNameGenerator
}

func newPodAutoScalerFixture(t *testing.T) *podAutoScalerFixture {

	f := &podAutoScalerFixture{}
	f.t = t

	f.k8sObjects = []runtime.Object{}
	f.dgsObjects = []runtime.Object{}

	f.clock = clockwork.NewFakeClockAt(fixedTime)
	f.namegenerator = shared.NewFakeRandomNameGenerator()

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

func (f *podAutoScalerFixture) expectCreateDGSAction(dgs *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *podAutoScalerFixture) expectDeleteDGSAction(dgs *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewDeleteAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs.Name)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *podAutoScalerFixture) expectUpdateDGSAction(dgs *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *podAutoScalerFixture) expectUpdateDGSColAction(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservercollections"}, dgsCol.Namespace, dgsCol)
	f.dgsActions = append(f.dgsActions, action)
}

func TestScaleOutDGSCol(t *testing.T) {
	f := newPodAutoScalerFixture(t)

	f.namegenerator.SetName("test0")
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.PodAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerPodAutoScalerDetails{
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
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec, f.namegenerator)

	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DedicatedGameServerStateRunning
	dgs.Labels[shared.LabelDedicatedGameServerState] = string(dgsv1alpha1.DedicatedGameServerStateRunning)

	dgs.Status.PodState = corev1.PodRunning
	dgs.Labels[shared.LabelPodState] = string(corev1.PodRunning)

	dgs.Status.ActivePlayers = 9
	dgs.Labels[shared.LabelActivePlayers] = "9"

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.PodAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 2
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateCreating

	f.expectUpdateDGSColAction(expDGSCol)

	f.run(getKeyDGSCol(dgsCol, t))
}

func TestScaleInDGSCol(t *testing.T) {
	f := newPodAutoScalerFixture(t)

	f.namegenerator.SetName("test0")
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.PodAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerPodAutoScalerDetails{
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
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	f.namegenerator.SetName("test0")
	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec, f.namegenerator)
	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DedicatedGameServerStateRunning
	dgs.Labels[shared.LabelDedicatedGameServerState] = string(dgsv1alpha1.DedicatedGameServerStateRunning)
	dgs.Status.PodState = corev1.PodRunning
	dgs.Labels[shared.LabelPodState] = string(corev1.PodRunning)
	dgs.Status.ActivePlayers = 5
	dgs.Labels[shared.LabelActivePlayers] = "5"

	f.namegenerator.SetName("test1")
	dgs2 := shared.NewDedicatedGameServer(dgsCol, podSpec, f.namegenerator)
	dgs2.Status.DedicatedGameServerState = dgsv1alpha1.DedicatedGameServerStateRunning
	dgs2.Labels[shared.LabelDedicatedGameServerState] = string(dgsv1alpha1.DedicatedGameServerStateRunning)
	dgs2.Status.PodState = corev1.PodRunning
	dgs2.Labels[shared.LabelPodState] = string(corev1.PodRunning)
	dgs2.Status.ActivePlayers = 5
	dgs2.Labels[shared.LabelActivePlayers] = "5"

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.dgsLister = append(f.dgsLister, dgs2)
	f.dgsObjects = append(f.dgsObjects, dgs2)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.PodAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 1
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateCreating

	f.expectUpdateDGSColAction(expDGSCol)

	f.run(getKeyDGSCol(dgsCol, t))
}

func TestDoNothingBecauseOfCoolDown(t *testing.T) {
	f := newPodAutoScalerFixture(t)

	f.namegenerator.SetName("test0")
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.PodAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerPodAutoScalerDetails{
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
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec, f.namegenerator)

	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DedicatedGameServerStateRunning
	dgs.Labels[shared.LabelDedicatedGameServerState] = string(dgsv1alpha1.DedicatedGameServerStateRunning)

	dgs.Status.PodState = corev1.PodRunning
	dgs.Labels[shared.LabelPodState] = string(corev1.PodRunning)

	dgs.Status.ActivePlayers = 9
	dgs.Labels[shared.LabelActivePlayers] = "9"

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.PodAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 2
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateCreating

	//expect nothing

	f.run(getKeyDGSCol(dgsCol, t))
}

func TestWithMalformedLastScaleTime(t *testing.T) {

	f := newPodAutoScalerFixture(t)

	f.namegenerator.SetName("test0")
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgsCol.Spec.PodAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerPodAutoScalerDetails{
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
	dgsCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateRunning
	dgsCol.Status.PodCollectionState = corev1.PodRunning

	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec, f.namegenerator)

	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DedicatedGameServerStateRunning
	dgs.Labels[shared.LabelDedicatedGameServerState] = string(dgsv1alpha1.DedicatedGameServerStateRunning)

	dgs.Status.PodState = corev1.PodRunning
	dgs.Labels[shared.LabelPodState] = string(corev1.PodRunning)

	dgs.Status.ActivePlayers = 9
	dgs.Labels[shared.LabelActivePlayers] = "9"

	f.dgsColLister = append(f.dgsColLister, dgsCol)
	f.dgsObjects = append(f.dgsObjects, dgsCol)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	expDGSCol := dgsCol.DeepCopy()
	expDGSCol.Spec.PodAutoScalerDetails.LastScaleOperationDateTime = f.clock.Now().String()
	expDGSCol.Spec.Replicas = 2
	expDGSCol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DedicatedGameServerCollectionStateCreating

	f.expectUpdateDGSColAction(expDGSCol)

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
