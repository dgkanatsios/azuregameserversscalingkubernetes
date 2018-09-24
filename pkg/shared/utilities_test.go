package shared

import (
	"fmt"
	"testing"
)

func TestAreMapsSame(t *testing.T) {
	map1 := map[string]string{
		"a": "av",
		"b": "bv",
	}
	//same as map1
	map2 := map[string]string{
		"a": "av",
		"b": "bv",
	}
	//same key, different value
	map3 := map[string]string{
		"a": "av",
		"b": "abv",
	}
	//+1 entry
	map4 := map[string]string{
		"a": "av",
		"b": "bv",
		"c": "cv",
	}
	//-1 entry
	map5 := map[string]string{
		"a": "av",
	}
	//diff key, same value
	map6 := map[string]string{
		"a":  "av",
		"bb": "bv",
	}

	check1 := AreMapsSame(map1, map2)
	if check1 == false {
		t.Error("Should be true")
	}

	check2 := AreMapsSame(map1, map3)
	if check2 == true {
		t.Error("Should be false")
	}

	check3 := AreMapsSame(map1, map4)
	if check3 == true {
		t.Error("Should be false")
	}

	check4 := AreMapsSame(map1, map5)
	if check4 == true {
		t.Error("Should be false")
	}

	check5 := AreMapsSame(map1, map6)
	if check5 == true {
		t.Error("Should be false")
	}
}

func TestRandString(t *testing.T) {
	s := randString(5)
	if len(s) != 5 {
		t.Error("Error in random string creation")
	}
}

func TestGetRandomInt(t *testing.T) {
	a := GetRandomInt(1, 1)
	if a != 0 {
		t.Error("Error in a")
	}
	b := GetRandomInt(5, 1)
	if b != 0 {
		t.Error("Error in b")
	}
	c := GetRandomInt(1, 2)
	if c != 1 && c != 2 {
		fmt.Print(c)
		t.Error("Error in c")
	}
}

func TestGetRandomIndexes(t *testing.T) {
	a := GetRandomIndexes(10, 5)
	if len(a) != 5 {
		t.Error("Length should be 5")
	}
}

func TestLogger(t *testing.T) {
	l := Logger()
	if l == nil {
		t.Error("Should not be nil")
	}
}
