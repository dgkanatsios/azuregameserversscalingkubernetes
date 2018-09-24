package shared

import (
	"testing"
)

func TestGenerateName(t *testing.T) {
	r := NewRealRandomNameGenerator()
	n := r.GenerateName("prefix")
	if len(n) != len("prefix-")+RandStringSize {
		t.Error("Wrong string length")
	}
}

func TestSetName(t *testing.T) {
	r := NewFakeRandomNameGenerator()
	r.SetName("testname")

	if r.(*fakeRandomNameGenerator).name != "testname" {
		t.Error("Wrong name")
	}
}

func TestFakeGenerateName(t *testing.T) {
	r := NewFakeRandomNameGenerator()
	r.SetName("testname")

	if r.GenerateName("randomprefix") != "testname" {
		t.Error("Wrong name")
	}
}

func TestNewFakeRandomNameGenerator(t *testing.T) {
	r := NewFakeRandomNameGenerator()

	_, ok := r.(*fakeRandomNameGenerator)
	if !ok {
		t.Error("Wrong type")
	}
}

func TestNewRealRandomNameGenerator(t *testing.T) {
	r := NewRealRandomNameGenerator()

	_, ok := r.(*realRandomNameGenerator)
	if !ok {
		t.Error("Wrong type")
	}
}
