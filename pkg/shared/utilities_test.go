package shared

import "testing"

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
