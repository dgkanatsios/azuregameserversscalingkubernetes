package dgs

import (
	"testing"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/fake"
	dgsinformers "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/informers/externalversions"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/controller/testhelpers"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	k8sActions []testhelpers.ExtendedAction
	dgsActions []testhelpers.ExtendedAction
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

func (f *dgsFixture) newDedicatedGameServerController() (*Controller,
	dgsinformers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {

	f.k8sClient = k8sfake.NewSimpleClientset(f.k8sObjects...)
	f.dgsClient = fake.NewSimpleClientset(f.dgsObjects...)

	k8sInformers := kubeinformers.NewSharedInformerFactory(f.k8sClient, testhelpers.NoResyncPeriodFunc())
	dgsInformers := dgsinformers.NewSharedInformerFactory(f.dgsClient, testhelpers.NoResyncPeriodFunc())

	testController := NewDedicatedGameServerController(f.k8sClient,
		f.dgsClient,
		dgsInformers.Azuregaming().V1alpha1().DedicatedGameServers(),
		k8sInformers.Core().V1().Pods(),
		k8sInformers.Core().V1().Nodes(), nil)

	testController.dgsListerSynced = testhelpers.AlwaysReady
	testController.podListerSynced = testhelpers.AlwaysReady

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
		testhelpers.CheckAction(expectedAction, action, f.t)
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
		testhelpers.CheckAction(expectedAction, action, f.t)
	}

	if len(f.k8sActions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.k8sActions)-len(k8sActions), f.k8sActions[len(k8sActions):])
	}
}

func (f *dgsFixture) expectCreatePodAction(p *corev1.Pod, assertions func(runtime.Object)) {
	action := core.NewCreateAction(schema.GroupVersionResource{Resource: "pods"}, p.Namespace, p)
	extAction := testhelpers.ExtendedAction{Action: action, Assertions: assertions}
	f.k8sActions = append(f.k8sActions, extAction)
}

func (f *dgsFixture) expectDeleteDGSAction(dgs *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewDeleteAction(schema.GroupVersionResource{Group: "azuregaming.com", Resource: "dedicatedgameservers", Version: "v1alpha1"}, dgs.Namespace, dgs.Name)
	extAction := testhelpers.ExtendedAction{Action: action, Assertions: assertions}
	f.dgsActions = append(f.dgsActions, extAction)
}

func (f *dgsFixture) expectUpdateDGSAction(dgs *dgsv1alpha1.DedicatedGameServer, assertions func(runtime.Object)) {
	action := core.NewUpdateAction(schema.GroupVersionResource{Group: "azuregaming.com", Resource: "dedicatedgameservers", Version: "v1alpha1"}, dgs.Namespace, dgs)
	extAction := testhelpers.ExtendedAction{Action: action, Assertions: assertions}
	f.dgsActions = append(f.dgsActions, extAction)
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

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, testhelpers.PodSpec)
	dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.k8sObjects = append(f.k8sObjects, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apiaccesscode",
			Namespace: shared.GameNamespace,
		},
		StringData: map[string]string{"code": ""},
	})

	expPod := shared.NewPod(dgs, shared.APIDetails{APIServerURL: "", Code: ""})

	f.expectCreatePodAction(expPod, nil)

	f.run(getKeyDGS(dgs, t))
}

func TestDeleteDGSWithZeroActivePlayers(t *testing.T) {
	f := newDGSFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, testhelpers.PodSpec)
	dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)

	dgs.Status.ActivePlayers = 0
	dgs.Status.Health = dgsv1alpha1.DGSHealthy
	dgs.Status.MarkedForDeletion = true

	delPod := shared.NewPod(dgs, shared.APIDetails{APIServerURL: "", Code: ""})

	f.podLister = append(f.podLister, delPod)
	f.k8sObjects = append(f.k8sObjects, delPod)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.expectDeleteDGSAction(dgs, nil)

	f.run(getKeyDGS(dgs, t))
}

func TestDGSStatusIsUpdated(t *testing.T) {
	f := newDGSFixture(t)

	dgsCol := shared.NewDedicatedGameServerCollection("test", shared.GameNamespace, 1, testhelpers.PodSpec)
	dgs := shared.NewDedicatedGameServer(dgsCol, testhelpers.PodSpec)

	//dgs.Status.ActivePlayers = 0

	pod := shared.NewPod(dgs, shared.APIDetails{APIServerURL: "", Code: ""})

	f.podLister = append(f.podLister, pod)
	f.k8sObjects = append(f.k8sObjects, pod)

	f.dgsLister = append(f.dgsLister, dgs)
	f.dgsObjects = append(f.dgsObjects, dgs)

	f.expectUpdateDGSAction(dgs, nil)

	f.run(getKeyDGS(dgs, t))
}

// filterInformerActionsDGS filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// noise level in our tests.
func filterInformerActionsDGS(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if action.Matches("list", "pods") ||
			action.Matches("watch", "pods") ||
			action.Matches("list", "dedicatedgameservers") ||
			action.Matches("watch", "dedicatedgameservers") ||
			action.Matches("list", "nodes") ||
			action.Matches("watch", "nodes") ||
			action.Matches("list", "secrets") ||
			action.Matches("watch", "secrets") ||
			action.Matches("get", "secrets") {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}
