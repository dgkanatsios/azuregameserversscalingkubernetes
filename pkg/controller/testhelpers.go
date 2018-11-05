package controller

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/testing"
)

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected extendedAction, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateAction:
		// e, _ := expected.Action.(core.CreateAction)
		// expObject := e.GetObject()
		object := a.GetObject()
		if expected.assertions != nil {
			expected.assertions(object)
		}
	case core.UpdateAction:
		// e, _ := expected.Action.(core.UpdateAction)
		// expObject := e.GetObject()
		object := a.GetObject()
		if expected.assertions != nil {
			expected.assertions(object)
		}
	}

}

type extendedAction struct {
	core.Action
	assertions func(runtime.Object)
}
