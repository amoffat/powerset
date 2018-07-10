package powerset

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/amoffat/linkedlist"
)

func TestVarSize(t *testing.T) {
	out, _ := VariableSize(3)
	correct := [][]int{
		{},
		{2},
		{1},
		{2, 1},
		{0},
		{2, 0},
		{1, 0},
		{2, 1, 0},
	}

	allValues := [][]int{}
	for indices := range out {
		allValues = append(allValues, indices)
	}
	if !reflect.DeepEqual(correct, allValues) {
		t.Fatalf("%v doesn't match expected %v", allValues, correct)
	}
}

func TestFixedSize(t *testing.T) {
	out, _ := FixedSize(3)
	correct := [][]bool{
		{false, false, false},
		{false, false, true},
		{false, true, false},
		{false, true, true},
		{true, false, false},
		{true, false, true},
		{true, true, false},
		{true, true, true},
	}

	allValues := [][]bool{}
	for indices := range out {
		allValues = append(allValues, indices)
	}
	if !reflect.DeepEqual(correct, allValues) {
		t.Fatalf("\n%v\n\n!=\n\n%v", allValues, correct)
	}
}

func TestStopFixedSize(t *testing.T) {
	out, stop := FixedSize(3)

	allValues := [][]bool{}
	i := 0
	for indices := range out {
		if i == 3 {
			stop()
			break
		}
		allValues = append(allValues, indices)
		i++
	}

	correct := [][]bool{
		{false, false, false},
		{false, false, true},
		{false, true, false},
	}
	if !reflect.DeepEqual(correct, allValues) {
		t.Fatalf("\n%v\n\n!=\n\n%v", allValues, correct)
	}
}

func TestStopVarSize(t *testing.T) {
	out, stop := VariableSize(3)

	allValues := [][]int{}
	i := 0
	for indices := range out {
		if i == 4 {
			stop()
			break
		}
		allValues = append(allValues, indices)
		i++
	}

	correct := [][]int{
		{},
		{2},
		{1},
		{2, 1},
	}
	if !reflect.DeepEqual(correct, allValues) {
		t.Fatalf("\n%v\n\n!=\n\n%v", allValues, correct)
	}
}

func TestLinkedListToFixed(t *testing.T) {
	check := func(correct []bool, fixed []bool) {
		if !reflect.DeepEqual(correct, fixed) {
			t.Fatalf("\n%v\n\n!=\n\n%v", fixed, correct)
		}
	}

	indices := linkedlist.New(nil)
	fixed := llToIndicesFixed(3, indices)
	correct := []bool{false, false, false}
	check(correct, fixed)

	indices = indices.Push(1)
	fixed = llToIndicesFixed(3, indices)
	correct = []bool{false, true, false}
	check(correct, fixed)

	indices = indices.Push(2)
	indices = indices.Push(0)
	fixed = llToIndicesFixed(3, indices)
	correct = []bool{true, true, true}
	check(correct, fixed)
}

func TestLinkedListToVar(t *testing.T) {
	check := func(correct []int, variable []int) {
		if !reflect.DeepEqual(correct, variable) {
			t.Fatalf("\n%v\n\n!=\n\n%v", variable, correct)
		}
	}

	indices := linkedlist.New(nil)
	variable := llToIndicesVariable(indices)
	correct := []int{}
	check(correct, variable)

	indices = indices.Push(1)
	variable = llToIndicesVariable(indices)
	correct = []int{1}
	check(correct, variable)

	indices = indices.Push(2)
	indices = indices.Push(0)
	variable = llToIndicesVariable(indices)
	correct = []int{0, 2, 1}
	check(correct, variable)
}

type answer struct {
	path  Path
	state string
}

func stringState(lastState string, node *PathNode) string {
	prefix := "-"
	comma := ""
	if len(lastState) > 0 {
		comma = ","
	}
	if node.Included {
		prefix = "+"
	}
	state := prefix + strconv.Itoa(node.Index) + comma + lastState
	return state
}

func TestCallback(t *testing.T) {
	correct := []answer{
		{Path{}, ""},
		{Path{{0, false}}, "-0"},
		{Path{{1, false}, {0, false}}, "-1,-0"},
		{Path{{2, false}, {1, false}, {0, false}}, "-2,-1,-0"},
		{Path{{2, true}, {1, false}, {0, false}}, "+2,-1,-0"},
		{Path{{1, true}, {0, false}}, "+1,-0"},
		{Path{{2, false}, {1, true}, {0, false}}, "-2,+1,-0"},
		{Path{{2, true}, {1, true}, {0, false}}, "+2,+1,-0"},
		{Path{{0, true}}, "+0"},
		{Path{{1, false}, {0, true}}, "-1,+0"},
		{Path{{2, false}, {1, false}, {0, true}}, "-2,-1,+0"},
		{Path{{2, true}, {1, false}, {0, true}}, "+2,-1,+0"},
		{Path{{1, true}, {0, true}}, "+1,+0"},
		{Path{{2, false}, {1, true}, {0, true}}, "-2,+1,+0"},
		{Path{{2, true}, {1, true}, {0, true}}, "+2,+1,+0"},
	}

	i := 0
	visit := func(path Path, isLeaf bool, rawState interface{}, out chan<- interface{}) (bool, int, interface{}) {
		var state string
		if len(path) > 0 {
			node := path[0]
			state = stringState(rawState.(string), node)
		}
		curCorrect := correct[i]

		if state != curCorrect.state {
			t.Fatalf("state doesn't match")
		}

		out <- path
		i++
		return false, 0, state
	}

	out := Callback(3, visit, "")

	k := 0
	for pathRaw := range out {
		path := pathRaw.(Path)
		curCorrect := correct[k]

		for j, segment := range path {
			correctSeg := curCorrect.path[j]
			if *segment != *correctSeg {
				t.Fatalf("index k=%d j=%d\n%+v\n\ndoesn't match\n\n%+v", k, j, segment, correctSeg)
			}
		}
		k++
	}

	if i != len(correct) {
		t.Fatalf("callback not called enough times! %d", i)
	}
}

func TestCallbackTerminate(t *testing.T) {
	correct := []answer{
		{Path{}, ""},
		{Path{{0, false}}, "-0"},
		{Path{{1, false}, {0, false}}, "-1,-0"},
		{Path{{2, false}, {1, false}, {0, false}}, "-2,-1,-0"},
	}

	i := 0
	visit := func(path Path, isLeaf bool, rawState interface{}, out chan<- interface{}) (bool, int, interface{}) {
		var state string
		if len(path) > 0 {
			node := path[0]
			state = stringState(rawState.(string), node)
		}
		curCorrect := correct[i]

		if state != curCorrect.state {
			t.Fatalf("state doesn't match")
		}

		i++

		if i == 4 {
			return true, -1, state // terminate to *before* the root node (0)
		} else {
			return false, 0, state
		}
	}
	out := Callback(3, visit, nil)

	k := 0
	for pathRaw := range out {
		path := pathRaw.(Path)
		curCorrect := correct[k]

		if k >= len(correct) {
			t.Fatalf("too many results (didn't terminate correctly)")
		}

		for j, segment := range path {
			correctSeg := curCorrect.path[j]
			if *segment != *correctSeg {
				t.Fatalf("index k=%d j=%d\n%+v\n\ndoesn't match\n\n%+v", k, j, segment, correctSeg)
			}
		}
	}

	if i != len(correct) {
		t.Fatalf("callback not called enough times!")
	}
}

func TestCallbackPartialTerminate(t *testing.T) {
	correct := []answer{
		{Path{}, ""},
		{Path{{0, false}}, "-0"},
		{Path{{1, false}, {0, false}}, "-1,-0"},
		{Path{{2, false}, {1, false}, {0, false}}, "-2,-1,-0"},
		{Path{{2, true}, {1, false}, {0, false}}, "+2,-1,-0"},
		{Path{{1, true}, {0, false}}, "+1,-0"},

		// callback terminates, preventing two nodes from being evaluated:
		//{Path{{2, false}, {1, true}, {0, false}}, "-2,+1,-0"},
		//{Path{{2, true}, {1, true}, {0, false}}, "+2,+1,-0"},

		{Path{{0, true}}, "+0"},
		{Path{{1, false}, {0, true}}, "-1,+0"},
		{Path{{2, false}, {1, false}, {0, true}}, "-2,-1,+0"},
		{Path{{2, true}, {1, false}, {0, true}}, "+2,-1,+0"},
		{Path{{1, true}, {0, true}}, "+1,+0"},
		{Path{{2, false}, {1, true}, {0, true}}, "-2,+1,+0"},
		{Path{{2, true}, {1, true}, {0, true}}, "+2,+1,+0"},
	}

	i := 0
	visit := func(path Path, isLeaf bool, rawState interface{}, out chan<- interface{}) (bool, int, interface{}) {
		var state string
		if len(path) > 0 {
			node := path[0]
			state = stringState(rawState.(string), node)
		}
		curCorrect := correct[i]

		if state != curCorrect.state {
			t.Fatalf("state doesn't match")
		}

		i++

		if ValidatePath(path, Path{{1, true}, {0, false}}) {
			return true, 0, state
		}

		return false, 0, state
	}
	out := Callback(3, visit, "")

	k := 0
	for pathRaw := range out {
		path := pathRaw.(Path)
		curCorrect := correct[k]

		if k >= len(correct) {
			t.Fatalf("too many results (didn't terminate correctly)")
		}

		for j, segment := range path {
			correctSeg := curCorrect.path[j]
			if *segment != *correctSeg {
				t.Fatalf("index k=%d j=%d\n%+v\n\ndoesn't match\n\n%+v", k, j, segment, correctSeg)
			}
		}

		k++
	}

	if i != len(correct) {
		t.Fatalf("callback not called enough times!")
	}
}
