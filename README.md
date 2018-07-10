# Powerset.go

[![Build Status](https://travis-ci.org/amoffat/powerset.svg?branch=master)](https://travis-ci.org/amoffat/powerset)
[![Go Report
Card](https://goreportcard.com/badge/github.com/amoffat/powerset)](https://goreportcard.com/report/github.com/amoffat/powerset)
[![Coverage
Status](https://coveralls.io/repos/github/amoffat/powerset/badge.svg?branch=master)](https://coveralls.io/github/amoffat/powerset?branch=master)

A flexible library for generating powersets incrementally to solve problems that require intelligently considering the
set of all possible subsets, often used with [backtracking](https://en.wikipedia.org/wiki/Backtracking).  See it in
action: [N-Queens Problem](#example-n-queens)

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

The basic idea is that we'll represent our n-by-n board grid cells as integers and examine the powerset of the integers.
If a cell is included in a set of the powerset, it has a queen on it, otherwise it doesn't.  A brute force search would
have to visit (2^(n\*n-1))-1 nodes just to search the solution space for all possible arrangements.  Fortunately, by
backtracking when we immediately find an invalid solution, we can skip out on the vast majority of nodes.  On an 8x8
sized board, the number of nodes we actually examine is only 1,849,097, while the number of nodes we skip is
18,446,744,073,707,702,518.

We'll choose to backtrack up to the parent node whenever `valid()` is false.  We'll also yield board results on the
output channel when the number of queens on the board matches n:

[examples/nqueens.go](https://github.com/amoffat/powerset/blob/master/examples/nqueens.go)
