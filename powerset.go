package powerset

import (
	"fmt"
	"strings"
	"sync"

	"github.com/amoffat/linkedlist"
)

// represents a node in a pathway through the powerset tree.  going left through the tree means the index at which we've
// decided to go left will not be included, going right means it will be included
type PathNode struct {
	Index    int
	Included bool
}

type Path []*PathNode

type NodeCallback func(Path, bool, interface{}, chan<- interface{}) (bool, int, interface{})
type internalCallback func(*linkedlist.Node, bool, interface{}) (bool, int, interface{})

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

// a helper for validating that two Paths match.  useful in a callback
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

// generate the powerset but at each leaf node call the callback
func Callback(lenItems int, cb NodeCallback, state interface{}) <-chan interface{} {
	indices := linkedlist.New(nil)
	path := linkedlist.New(nil)
	out := make(chan interface{})
	wrappedCb := func(indices *linkedlist.Node, isLeaf bool, state interface{}) (bool, int, interface{}) {
		return cb(llToPath(indices), isLeaf, state, out)
	}
	go powerSetCallback(0, lenItems, indices, wrappedCb, path, state, out)
	return out
}

// convert a linked list to a fixed size array of booleans where the indices contained in the linkedlist are true in the
// fixed array, otherwise false
func llToIndicesFixed(lenItems int, indices *linkedlist.Node) []bool {
	unpackedIndices := make([]bool, lenItems)
	head := indices

	if head.Next == nil {
		return unpackedIndices
	} else {
		for head != nil && head.Data != nil {
			idx := head.Data.(int)
			unpackedIndices[idx] = true
			head = head.Next
		}
		return unpackedIndices
	}
}

// convert a linked list to a variable array of integer indices contained in the linked list
func llToIndicesVariable(indices *linkedlist.Node) []int {
	unpackedIndices := []int{}
	head := indices

	if head.Next == nil {
		return unpackedIndices
	} else {
		for head != nil && head.Data != nil {
			unpackedIndices = append(unpackedIndices, head.Data.(int))
			head = head.Next
		}
		return unpackedIndices
	}
}

func llToPath(indices *linkedlist.Node) []*PathNode {
	unpacked := []*PathNode{}
	head := indices

	if head.Next == nil {
		return unpacked
	} else {
		for head != nil && head.Data != nil {
			unpacked = append(unpacked, head.Data.(*PathNode))
			head = head.Next
		}
		return unpacked
	}
}

// generates a powerset of fixed size items.  each item returned on the output channel has a length of lenItems and each
// element is either true or false, indicating that the index is included in the combination
func FixedSize(lenItems int) (<-chan []bool, func()) {
	out := make(chan []bool)
	indicesOut := make(chan linkedlist.Node)
	stopIn := make(chan bool)
	indices := linkedlist.New(nil)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go powerSet(0, lenItems, indices, indicesOut, wg, stopIn)

	go func() {
		defer close(out)
		defer wg.Done()

		for indices := range indicesOut {
			unpackedIndices := llToIndicesFixed(lenItems, &indices)
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

// generates a variable size powerset.  each slice returned on the output channel is a variable size slice containing
// the index numbers othemselves of the items included in each combination
func VariableSize(lenItems int) (<-chan []int, func()) {
	out := make(chan []int)
	indicesOut := make(chan linkedlist.Node)
	stopIn := make(chan bool)
	indices := linkedlist.New(nil)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go powerSet(0, lenItems, indices, indicesOut, &wg, stopIn)

	go func() {
		defer close(out)
		defer wg.Done()

		for indices := range indicesOut {
			unpackedIndices := llToIndicesVariable(&indices)
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

// the internal mechanism for generating a powerset
func powerSet(n int, k int, indices *linkedlist.Node, out chan<- linkedlist.Node, wg *sync.WaitGroup, stopIn <-chan bool) bool {
	if n == 0 {
		defer close(out)
		defer wg.Done()
	}

	done := false

	if n == k {
		select {
		case <-stopIn:
			return true
		case out <- *indices:
		}
		return done
	}

	select {
	case <-stopIn:
		return true
	default:
		done = powerSet(n+1, k, indices, out, wg, stopIn)
		if !done {
			indices = indices.Push(n)
			done = powerSet(n+1, k, indices, out, wg, stopIn)
			indices, _ = indices.Pop()
		}
	}

	// if we've made it this far, we're not at a leaf node
	return done
}

// internal function that creates a powerset but calls a callback at each node, including the leaves.  if the callback
// returns true for "done", we stop
func powerSetCallback(n int, k int, indices *linkedlist.Node, cb internalCallback, path *linkedlist.Node,
	state interface{}, out chan<- interface{}) (bool, int) {

	stop := false
	stopNode := 0
	isRoot := n == 0
	isLeaf := n == k

	if isRoot {
		defer close(out)
	}

	stop, stopNode, state = cb(path, isLeaf, state)

	// our callback says to stop, but where do we stop?
	// if we're deeper than our stop node, we need to return early and tell callers to also stop
	if stop && n > stopNode {
		return true, stopNode
	}

	if isLeaf {
		return false, 0
	}

	path = path.Push(&PathNode{Index: n, Included: false})
	stop, stopNode = powerSetCallback(n+1, k, indices, cb, path, state, out)
	path, _ = path.Pop()

	// if our left branch told us to stop, let's figure out what we need to do
	if stop {
		if n > stopNode {
			// we're deeper in the tree than our stop node, which means we need to terminate going any deeper and
			// propagate the stop
			return stop, stopNode
		} else {
			// otherwise, we're at the stop node or earlier, which means we can discontinue stopping and continue as
			// normal
			stop = false
		}
	}

	indices = indices.Push(n)

	path = path.Push(&PathNode{Index: n, Included: true})
	stop, stopNode = powerSetCallback(n+1, k, indices, cb, path, state, out)

	path, _ = path.Pop()
	indices, _ = indices.Pop()

	return stop, stopNode
}
