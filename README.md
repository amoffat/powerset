# Powerset.go

A flexible library for generating powersets incrementally to solve problems that require considering the set of all
possible subsets, often used with [backtracking](https://en.wikipedia.org/wiki/Backtracking).  See it in action:
[N-Queens Problem](#example-n-queens)

# Usage

There are two simple functions, `FixedSize`, and `VariableSize` to generate a powerset and return the results over a
channel.  There is also a very advanced function `Callback` to also generate a powerset over a channel, but utilizes a
callback as it traverses the nodes of the powerset tree.  All 3 functions support early termination.

## Fixed-size method

```go
out, stop := powerset.FixedSize(3)
for indices := range out {
    fmt.Println(indices)
}
```

Output:

```
[false false false]
[false false true]
[false true false]
[false true true]
[true false false]
[true false true]
[true true false]
[true true true]
```

`out` is the output channel that will yield indices of type `[size]bool`, where size is the integer passed into
`FixedSize`.  Each array of indices contains a true if that index is included, or a false if it isn't.  For example,
`[false, false, false]` corresponds to the null set, while `[true, false, true]` is the set `{0,2}`.

`stop` is a function with signature `func()`.  When called, it terminates the goroutine generating the powerset.  An
example of stopping:

```go
out, stop := powerset.FixedSize(3)
for indices := range out {
    if indices[1] {
        stop()
        break
    }
    fmt.Println(indices)
}
```

## Variable-size method

```go
out, stop := powerset.VariableSize(3)
for indices := range out {
    fmt.Println(indices)
}
```

Output:

```
[]
[2]
[1]
[2 1]
[0]
[2 0]
[1 0]
[2 1 0]
```

`out` is the output channel that will yield indices of type `[]int`.  Each indices element contains only the indices
included.  For example, `[]` is the null set, while `[0,2]` is the set `{0,2}`.

## Callback method

The callback version is the most advanced version of powerset generation.  This version allows you to provide a callback
that is called at each intermediary node of the powerset tree, and gives you the option to terminate generation up to an
arbitrary parent if you want to discontinue a subtree.  This allows you to choose to not evaluate specific branches of
the powerset based on user logic in the callback.

```go
cb := func(path powerset.Path, isLeaf bool, state interface{}, out chan<- interface{}) (bool, int, interface{}) {
    out <- path
    return false, 0, nil
}

out := powerset.Callback(3, cb, nil)
for path := range out {
    fmt.Println(path)
}
```

Output:

```
{}
{0 false}
{1 false} {0 false}
{2 false} {1 false} {0 false}
{2 true} {1 false} {0 false}
{1 true} {0 false}
{2 false} {1 true} {0 false}
{2 true} {1 true} {0 false}
{0 true}
{1 false} {0 true}
{2 false} {1 false} {0 true}
{2 true} {1 false} {0 true}
{1 true} {0 true}
{2 false} {1 true} {0 true}
{2 true} {1 true} {0 true}
```

The callback `cb` is evaluated at each node of the powerset tree.  The path to the current node is represented with the
first argument to the callback, of value `powerset.Path`.  Here, we're writing path to our output channel, to visualize
it.  You can see the each path corresponds to each node of the powerset tree:

![https://i.imgur.com/NUvTFxP.jpg](https://i.imgur.com/NUvTFxP.jpg)

Each leaf in the tree is a specific set in the powerset, starting from the null set (far left), to the set `{0,1,2}`
(far right).  The intermediary nodes are paths through the tree specified by whether or not indices were included or
excluded.

The callback's first argument is receiving a `Path`, which is a type alias for `[]*PathNode`.  Each `PathNode` struct
contains an index and whether or not the index is explicitly included at the current powerset node.  Below is an example
pathway through the tree.  The green dotted line indicates the pathway `{+1,-0}`.  So the `Path` that the callback would
receive at this node would be `[]*PathNodes{{1,true}, {0,false}}`:

![https://i.imgur.com/5UmMQ0c.jpg](https://i.imgur.com/5UmMQ0c.jpg)

### Arguments

The first argument is our `powerset.Path` which we covered above.

The second argument, `isLeaf bool` is a simple flag to let your callback know if we're on a leaf or intermediary node.

The third argument, `state interface{}` gives us the bulk of the power.  It represents some arbitrary state that we can
mutate, whose value will propagate only to child nodes.  For example, in the [n-queens problem](#example-n-queens) at
the end of this README, the state represents the board with the queens positions.  When new branches of the powerset
tree are explored, this state is updated to add new queens, and when we backtrack to previous nodes, we automatically
re-use old board states.  State can be anything you want, hence `interface{}`, but it should be noted, *deep copies must
be made manually at each node.*  If your state is a struct containing a map, for example, although the struct itself is
passed by value to the callback, the underlying mapping will share its heap-allocated data with the previous state,
causing problems in your algorithm.  If you experience weird behavior, it's likely that you didn't fully copy your state
between nodes.

The last argument is self explanatory and is a channel for communicating your solutions to the caller.  In the n-queens
example, we use this to write out our valid board positions.

### Return value

The return value of the callback is a tuple `bool, int, interface{}`.

The first and second argument go together.  The `bool` is whether or not we should terminate generation of child nodes,
and the `int` is which parent we should continue the algorithm.  In most cases, when you terminate, you will usually
want to continue at the parent, which would corrspond to `len(path) - 1`.  You can see in this image, from the blue
lines to the right, that the root node is zero-indexed, which would make our current node `len(path)`, because a node's
height index will receive that many path segments:

![https://i.imgur.com/5UmMQ0c.jpg](https://i.imgur.com/5UmMQ0c.jpg)

If we terminate to the parent from a left branch, processing will continue at the right sibling.  If we terminate to the
parent from the *right* branch, processing will continue with the grandparent (since the parent has no more children to
explore).  Using `true, 0` will terminate to the root, while `true, -1` will terminate to *before* the root, meaning the
algorithm terminates completely.

This termination logic is critical in exploring large state space trees for solutions, since we can backtrack early and
skip potentially quintillions (not a typo, see the n-queens output!) of nodes.

# Example: N-Queens 

The n-queens problem is about finding all possible arrangements of n queens on an n-by-n sized chess board, such that no
queens can attack eachother.  We can solve this using [backtracking](https://en.wikipedia.org/wiki/Backtracking), which
fits in nicely with `powerset.Callback`.

The basic idea is that we'll represent our n-by-n board grid cells as integers and examine the powerset of the cells.
If a cell is included in a set, it has a queen on it, otherwise it doesn't.  A brute force search would have to visit
(2^(n\*n-1))-1 nodes just to search the solution space for all possible arrangements.  Fortunately, by backtracking when
we immediately find an invalid solution, we can skip out on the vast majority of nodes.  On an 8x8 sized board, the
number of nodes we actually examine is only 1,849,097, while the number of nodes we skip is 18,446,744,073,707,702,518.

We'll choose to backtrack up to the parent node whenever `valid()` is false.  We'll also yield board results on the
output channel when the number of queens on the board matches n.

```go
package main

import (
    "fmt"
    "math"
    "os"
    "strconv"

    "github.com/amoffat/powerset"
)

// a 2d slice representing our chess board.  true means a queen occupies that location
type Board [][]bool

// a state which we will pass through the powerset tree.  it contains optimization fields for quickly determining if
// there's a nearby queen which can attack
type boardState struct {
    numQueens      int
    board          Board
    occupiedRows   map[int]bool
    occupiedCols   map[int]bool
    occupiedLDiags map[int]bool
    occupiedRDiags map[int]bool
}

type Position struct {
    x int
    y int
}

// outputs the board in a friendly format:
//
// 0 0 0 0 0 1 0 0
// 1 0 0 0 0 0 0 0
// 0 0 0 0 1 0 0 0
// 0 1 0 0 0 0 0 0
// 0 0 0 0 0 0 0 1
// 0 0 1 0 0 0 0 0
// 0 0 0 0 0 0 1 0
// 0 0 0 1 0 0 0 0
func PrintBoard(board Board) {
    size := len(board)
    for y := 0; y < size; y++ {
        for x := 0; x < size; x++ {
            hasQueen := board[x][y]
            val := 0
            if hasQueen {
                val = 1
            }
            fmt.Printf("%v ", val)
        }
        fmt.Println("")
    }
}

// for an index from [0-n*n), return the x,y position in the board n units wide
func idxToPos(i int, width int) *Position {
    pos := Position{x: i % width, y: i / width}
    return &pos
}

// for a given board state, use its optimization fields to determine if a candidate position is valid or not, meaning it
// can't be attacked from the row, column, or the diagonals of any other queen position
func valid(state boardState, candidate *Position) bool {
    if _, ok := state.occupiedCols[candidate.x]; ok {
        return false
    }

    if _, ok := state.occupiedRows[candidate.y]; ok {
        return false
    }

    ldiag := candidate.y + candidate.x
    if _, ok := state.occupiedLDiags[ldiag]; ok {
        return false
    }

    rdiag := candidate.y - candidate.x
    if _, ok := state.occupiedRDiags[rdiag]; ok {
        return false
    }

    return true
}

func newBoard(size int) Board {
    board := make(Board, size)
    for i := 0; i < size; i++ {
        board[i] = make([]bool, size)
    }
    return board
}

func newBoardState(size int) boardState {
    state := boardState{
        numQueens:      0,
        board:          newBoard(size),
        occupiedRows:   make(map[int]bool),
        occupiedCols:   make(map[int]bool),
        occupiedLDiags: make(map[int]bool),
        occupiedRDiags: make(map[int]bool),
    }
    return state
}

func copyBoard(board Board) Board {
    size := len(board)
    newBoard := make(Board, size)
    for i := 0; i < size; i++ {
        newBoard[i] = make([]bool, size)
        copy(newBoard[i], board[i])
    }
    return newBoard
}

func copyMap(src map[int]bool) map[int]bool {
    dst := make(map[int]bool)
    for k, v := range src {
        dst[k] = v
    }
    return dst
}

// deep-copies a board state.  required when propagating node state to children
func copyState(state boardState) boardState {
    newState := state
    newState.board = copyBoard(state.board)

    newState.occupiedRows = copyMap(state.occupiedRows)
    newState.occupiedCols = copyMap(state.occupiedCols)
    newState.occupiedLDiags = copyMap(state.occupiedLDiags)
    newState.occupiedRDiags = copyMap(state.occupiedRDiags)

    return newState
}

func main() {
    boardSize, _ := strconv.Atoi(os.Args[1])

    powersetSize := boardSize * boardSize
    state := newBoardState(boardSize)

    // we'll keep track of all the nodes we visited vs all the nodes we skipped, for logging
    var visited, skipped uint64

    // our callback to the powerset.Callback function.  it is in charge of determining if a queen position is valid, and
    // if it isn't, to backtrack
    cb := func(path powerset.Path, isLeaf bool, rawState interface{}, out chan<- interface{}) (bool, int, interface{}) {
        visited++

        // root node won't have any items in the path
        isRoot := len(path) == 0

        if isRoot {
            return false, 0, rawState
        }

        // we have to copy the state because the state contains maps, which have shared internal data
        state := copyState(rawState.(boardState))
        board := state.board
        node := path[0]

        // here we'll count a node being included as a queen being placed on that space
        if node.Included {

            // convert the index yielded from our powerset to a board position
            pos := idxToPos(node.Index, boardSize)

            // determine if our candidate position is feasible
            feasible := valid(state, pos)

            // if we're not feasible, we need to backtrack up to the parent node, and let the other branch (if there is
            // one) be explored
            if !feasible {
                // the following few lines are for book keeping to see how many nodes we skipped by backtracking
                remainingHeight := powersetSize - len(path)
                // +1 is for counting total nodes in a binary tree, which is 2^(height+1)-1, and the -2 comes from
                // skipping the current node, -1, since we visited it, and combining it with the -1
                skipped = skipped + uint64(math.Pow(2.0, float64(remainingHeight+1))) - 2

                // backtrack up to our parent
                parent := len(path) - 1
                return true, parent, nil
            }

            // if we get this far, our solution is feasible, so let's place the queen and update our state so child
            // nodes will see the queen on the board.  we'll also update some optimization structures to help us quickly
            // determine if there's a queen collision without iterating over cells
            board[pos.x][pos.y] = true
            state.occupiedRows[pos.y] = true
            state.occupiedCols[pos.x] = true
            state.occupiedRDiags[pos.y-pos.x] = true
            state.occupiedLDiags[pos.y+pos.x] = true
            state.numQueens++
        }

        // we've hit a fully evaluated leaf node.  if there are n queens, yield a copy of the board
        if isLeaf && state.numQueens == boardSize {
            out <- copyBoard(board)
        }

        // otherwise, continue evaluating nodes
        return false, 0, state
    }

    // start generating the powerset
    out := powerset.Callback(powersetSize, cb, state)

    solutions := 0
    for board := range out {
        solutions++
        PrintBoard(board.(Board))
        fmt.Println("")
    }

    fmt.Printf("solutions = %v, visited = %+v, skipped = %+v\n", solutions, visited, skipped)
}
```
