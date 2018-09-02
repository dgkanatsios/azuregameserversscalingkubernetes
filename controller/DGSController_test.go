package controller

import (
	"testing"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeinformers "k8s.io/client-go/informers"
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

	dgsLister []*dgsv1alpha1.DedicatedGameServer
	podLister []*corev1.Pod
	// Actions expected to happen on the client.
	k8sActions []core.Action
	dgsActions []core.Action
	// Objects from here preloaded into NewSimpleFake.
	k8sObjects []runtime.Object
	dgsObjects []runtime.Object
}

func newDGSFixture(t *testing.T) *dgsFixture {

	f := &dgsFixture{}
	f.t = t

	f.dgsObjects = []runtime.Object{}
	f.k8sObjects = []runtime.Object{}

	return f
}

func (f *dgsFixture) newDedicatedGameServerController() (*DedicatedGameServerController,
	dgsinformers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {

	f.k8sClient = k8sfake.NewSimpleClientset(f.k8sObjects...)
	f.dgsClient = fake.NewSimpleClientset(f.dgsObjects...)

	k8sInformers := kubeinformers.NewSharedInformerFactory(f.k8sClient, noResyncPeriodFunc())
	dgsInformers := dgsinformers.NewSharedInformerFactory(f.dgsClient, noResyncPeriodFunc())

	testController := NewDedicatedGameServerController(f.k8sClient, f.dgsClient,
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers(), k8sInformers.Core().V1().Pods(), k8sInformers.Core().V1().Nodes())

	testController.dgsListerSynced = alwaysReady
	testController.podListerSynced = alwaysReady

	testController.recorder = &record.FakeRecorder{}

	for _, dgs := range f.dgsLister {
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers().Informer().GetIndexer().Add(dgs)
	}

	for _, pod := range f.podLister {
		k8sInformers.Core().V1().Pods().Informer().GetIndexer().Add(pod)
	}

	return testController, dgsInformers, k8sInformers
}

func (f *dgsFixture) run(dgsName string) {
	f.runController(dgsName, true, false)
}

func (f *dgsFixture) runExpectError(dgsName string) {
	f.runController(dgsName, true, true)
}

func (f *dgsFixture) runController(dgsName string, startInformers bool, expectError bool) {

	testController, dgsInformers, k8sInformers := f.newDedicatedGameServerController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		dgsInformers.Start(stopCh)
		k8sInformers.Start(stopCh)
	}

	err := testController.syncHandler(dgsName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing DGS: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing DGS, got nil")
	}

	actions := filterInformerActionsDGS(f.dgsClient.Actions())

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

	k8sActions := filterInformerActionsDGS(f.k8sClient.Actions())
	for i, action := range k8sActions {
		if len(f.k8sActions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.k8sActions), k8sActions[i:])
			break
		}

		expectedAction := f.k8sActions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.k8sActions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.k8sActions)-len(k8sActions), f.k8sActions[len(k8sActions):])
	}
}

func (f *dgsFixture) expectCreatePodAction(p *corev1.Pod) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "pods"}, p.Namespace, p)
	f.k8sActions = append(f.k8sActions, action)
}

func (f *dgsFixture) expectDeleteDGSAction(dgs *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewDeleteAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs.Name)
	f.dgsActions = append(f.dgsActions, action)
}

func (f *dgsFixture) expectUpdateDGSAction(dgs *dgsv1alpha1.DedicatedGameServer) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Resource: "dedicatedgameservers"}, dgs.Namespace, dgs)
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

func TestCreatesPod(t *testing.T) {
	f := newDGSFixture(t)

	nameSuffix = "0"
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	expPod := shared.NewPod(dgs, shared.APIDetails{"", ""})

	f.expectCreatePodAction(expPod)

	f.run(getKeyDGS(dgs, t))
}

func TestDeleteDGSWithZeroActivePlayers(t *testing.T) {
	f := newDGSFixture(t)

	nameSuffix = "0"
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)

	dgs.Spec.ActivePlayers = 0
	dgs.Labels[shared.LabelActivePlayers] = "0"
	dgs.Status.DedicatedGameServerState = dgsv1alpha1.DedicatedGameServerStateMarkedForDeletion
	dgs.Labels[shared.LabelDedicatedGameServerState] = string(dgsv1alpha1.DedicatedGameServerStateMarkedForDeletion)

	delPod := shared.NewPod(dgs, shared.APIDetails{"", ""})

	f.podLister = append(f.podLister, delPod)
	f.k8sObjects = append(f.k8sObjects, delPod)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.expectDeleteDGSAction(dgs)

	f.run(getKeyDGS(dgs, t))
}

func TestDGSStatusIsUpdated(t *testing.T) {
	f := newDGSFixture(t)

	nameSuffix = "0"
	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, podSpec)
	dgs := shared.NewDedicatedGameServer(dgsCol, podSpec)

	dgs.Spec.ActivePlayers = 0
	dgs.Labels[shared.LabelActivePlayers] = "0"
	dgs.Labels[shared.LabelPodState] = ""

	pod := shared.NewPod(dgs, shared.APIDetails{"", ""})

	f.podLister = append(f.podLister, pod)
	f.k8sObjects = append(f.k8sObjects, pod)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.expectUpdateDGSAction(dgs)

	f.run(getKeyDGS(dgs, t))
}

// filterInformerActionsDGS filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// noise level in our tests.
func filterInformerActionsDGS(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "pods") ||
				action.Matches("watch", "pods") ||
				action.Matches("list", "dedicatedgameservers") ||
				action.Matches("watch", "dedicatedgameservers") ||
				action.Matches("list", "nodes") ||
				action.Matches("watch", "nodes")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}
