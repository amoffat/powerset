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
