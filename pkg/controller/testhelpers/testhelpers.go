package testhelpers

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/testing"
)

var (
	AlwaysReady        = func() bool { return true }
	NoResyncPeriodFunc = func() time.Duration { return 0 }
)

var FixedTime = time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)

var PodSpec = corev1.PodSpec{
	Containers: []corev1.Container{
		{
			Ports: []corev1.ContainerPort{}, // if we create any ports here we need to initialize port registry before (during??) running the test
		},
	},
}

// CheckAction verifies that expected and actual actions are equal and both have
// same attached resources
func CheckAction(expected ExtendedAction, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateAction:
		// e, _ := expected.Action.(core.CreateAction)
		// expObject := e.GetObject()
		object := a.GetObject()
		if expected.Assertions != nil {
			expected.Assertions(object)
		}
	case core.UpdateAction:
		// e, _ := expected.Action.(core.UpdateAction)
		// expObject := e.GetObject()
		object := a.GetObject()
		if expected.Assertions != nil {
			expected.Assertions(object)
		}
	}

}

type ExtendedAction struct {
	core.Action
	Assertions func(runtime.Object)
}
