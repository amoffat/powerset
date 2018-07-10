package powerset

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
)

// PathNode represents a node in a pathway through the powerset tree.  going left through the tree means the index at
// which we've decided to go left will not be included, going right means it will be included
type PathNode struct {
	Index    int
	Included bool
}

// Path is an alias for a slice of PathNodes
type Path []*PathNode

// NodeCallback represents a callback to Callback
type NodeCallback func(Path, bool, interface{}, chan<- interface{}) (bool, int, interface{})
type internalCallback func(*list.List, bool, interface{}) (bool, int, interface{})

func (path Path) String() string {
	buf := []string{}
	for _, seg := range path {
		buf = append(buf, fmt.Sprintf("%v", *seg))
	}

	if len(path) == 0 {
		buf = append(buf, "{}")
	}
	return strings.Join(buf, " ")
}

// ValidatePath is a helper for validating that two Paths match.  useful in a callback
func ValidatePath(path Path, check Path) bool {
	if len(path) != len(check) {
		return false
	}

	for i, seg := range path {
		if *seg != *check[i] {
			return false
		}
	}

	return true
}

// Callback generates the powerset but at each leaf node call the callback
func Callback(lenItems int, cb NodeCallback, state interface{}) <-chan interface{} {
	indices := list.New()
	path := list.New()
	out := make(chan interface{})
	wrappedCb := func(indices *list.List, isLeaf bool, state interface{}) (bool, int, interface{}) {
		return cb(llToPath(indices), isLeaf, state, out)
	}
	go powerSetCallback(0, lenItems, indices, wrappedCb, path, state, out)
	return out
}

// convert a linked list to a fixed size array of booleans where the indices contained in the linkedlist are true in the
// fixed array, otherwise false
func llToIndicesFixed(lenItems int, indices *list.List) []bool {
	unpackedIndices := make([]bool, lenItems)
	head := indices.Front()

	for head != nil {
		idx := head.Value.(int)
		unpackedIndices[idx] = true
		head = head.Next()
	}
	return unpackedIndices
}

// convert a linked list to a variable array of integer indices contained in the linked list
func llToIndicesVariable(indices *list.List) []int {
	unpackedIndices := []int{}
	head := indices.Front()

	for head != nil {
		idx := head.Value.(int)
		unpackedIndices = append(unpackedIndices, idx)
		head = head.Next()
	}
	return unpackedIndices
}

func llToPath(indices *list.List) []*PathNode {
	unpacked := []*PathNode{}
	head := indices.Front()

	for head != nil {
		unpacked = append(unpacked, head.Value.(*PathNode))
		head = head.Next()
	}
	return unpacked
}

// FixedSize generates a powerset of fixed size items.  each item returned on the output channel has a length of
// lenItems and each element is either true or false, indicating that the index is included in the combination
func FixedSize(lenItems int) (<-chan []bool, func()) {
	out := make(chan []bool)
	indicesOut := make(chan *list.List)
	stopIn := make(chan bool)
	indices := list.New()

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go powerSet(0, lenItems, indices, indicesOut, wg, stopIn)

	go func() {
		defer close(out)
		defer wg.Done()

		for indices := range indicesOut {
			unpackedIndices := llToIndicesFixed(lenItems, indices)
			select {
			case <-stopIn:
				break
			case out <- unpackedIndices:
			}
		}
	}()

	stop := makeStopper(stopIn, wg)

	return out, stop
}

// VariableSize generates a variable size powerset.  each slice returned on the output channel is a variable size slice
// containing the index numbers othemselves of the items included in each combination
func VariableSize(lenItems int) (<-chan []int, func()) {
	out := make(chan []int)
	indicesOut := make(chan *list.List)
	stopIn := make(chan bool)
	indices := list.New()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go powerSet(0, lenItems, indices, indicesOut, &wg, stopIn)

	go func() {
		defer close(out)
		defer wg.Done()

		for indices := range indicesOut {
			unpackedIndices := llToIndicesVariable(indices)
			select {
			case <-stopIn:
				break
			case out <- unpackedIndices:
			}
		}
	}()

	stop := makeStopper(stopIn, &wg)

	return out, stop
}

// returns a closure that stops and waits for a goroutine to finish
func makeStopper(in chan<- bool, wg *sync.WaitGroup) func() {
	stop := func() {
		close(in)
		wg.Wait()
	}
	return stop
}

func copyLL(l *list.List) *list.List {
	newList := list.New()
	head := l.Front()
	for head != nil {
		newList.PushBack(head.Value)
		head = head.Next()
	}
	return newList
}

// the internal mechanism for generating a powerset
func powerSet(n int, k int, indices *list.List, out chan<- *list.List, wg *sync.WaitGroup, stopIn <-chan bool) bool {
	if n == 0 {
		defer close(out)
		defer wg.Done()
	}

	done := false

	if n == k {
		select {
		case <-stopIn:
			return true
		case out <- copyLL(indices):
		}
		return done
	}

	select {
	case <-stopIn:
		return true
	default:
		done = powerSet(n+1, k, indices, out, wg, stopIn)
		if !done {
			rightPushed := indices.PushFront(n)
			done = powerSet(n+1, k, indices, out, wg, stopIn)
			indices.Remove(rightPushed)
		}
	}

	// if we've made it this far, we're not at a leaf node
	return done
}

// internal function that creates a powerset but calls a callback at each node, including the leaves.  if the callback
// returns true for "done", we stop
func powerSetCallback(n int, k int, indices *list.List, cb internalCallback, path *list.List,
	state interface{}, out chan<- interface{}) (bool, int) {

	stopNode := 0
	isRoot := n == 0
	isLeaf := n == k

	if isRoot {
		defer close(out)
	}

	var stop bool
	stop, stopNode, state = cb(path, isLeaf, state)

	// our callback says to stop, but where do we stop?
	// if we're deeper than our stop node, we need to return early and tell callers to also stop
	if stop && n > stopNode {
		return true, stopNode
	}

	if isLeaf {
		return false, 0
	}

	leftPathPushed := path.PushFront(&PathNode{Index: n, Included: false})
	stop, stopNode = powerSetCallback(n+1, k, indices, cb, path, state, out)
	path.Remove(leftPathPushed)

	// if our left branch told us to stop, let's figure out what we need to do
	if stop && n > stopNode {
		// we're deeper in the tree than our stop node, which means we need to terminate going any deeper and
		// propagate the stop
		return stop, stopNode
	}

	rightIndexPushed := indices.PushFront(n)
	rightPathPushed := path.PushFront(&PathNode{Index: n, Included: true})
	stop, stopNode = powerSetCallback(n+1, k, indices, cb, path, state, out)

	path.Remove(rightPathPushed)
	indices.Remove(rightIndexPushed)

	return stop, stopNode
}
