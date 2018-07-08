# Powerset.go

A flexible library for generating powersets incrementally.

# Usage

There are 2 public functions for generating powersets using a channel, and 1 public function for generating a powerset
using a callback.  All 3 functions support early termination.

## Fixed-size

```go
out, stop := powerset.FixedSize(3)
for indices := range out {
    fmt.Println(indices)
}
```

Outputs:

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

## Variable-size

```go
out, stop := powerset.VariableSize(3)
for indices := range out {
    fmt.Println(indices)
}
```

Outputs:

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

## Callback

The callback version is the most advanced version of powerset generation.  Instead of just providing you with the leaf
nodes, as the fixed and variable-size versions do, the callback version provides you with every intermediary node.  This
allows you to choose to not evaluate specific branches of the powerset based on user logic in the callback.

```go
cb := func(path Path, isLeaf bool) (bool, int) {
    return false, 0
}

powerset.Callback(3, cb)
```

The callback `cb` is evaluated at each node of the powerset tree.  You can visualize the powerset tree of `{0,1,2}` as
the following:

![https://i.imgur.com/NUvTFxP.jpg](https://i.imgur.com/NUvTFxP.jpg)

Each leaf in the tree is a specific set in the powerset, starting from the null set (far left), to the set `{0,1,2}`
(far right).  The intermediary nodes are paths through the tree specified by whether or not indices were included or
excluded.

The callback passed to `powerset.Callback` should expect a `Path`, which is a type alias for `[]*PathNode`.  Each
`PathNode` struct contains an index and whether or not the index is explicitly included at the current powerset node.
Below is an example pathway through the tree.  The green dotted line indicates the pathway `{+1,-0}`.  So the `Path`
that the callback would receive at this node would be `[]*PathNodes{{1,true}, {0,false}}`:

![https://i.imgur.com/5UmMQ0c.jpg](https://i.imgur.com/5UmMQ0c.jpg)

The return value of the callback is a tuple `bool, int`.  The bool is whether or not the generation should terminate.
The int is the node level we should terminate up to.  In the above image, the blue lines to the right indicate the
levels, with root being 0-indexed.  If you return `true, 0` from somewhere in the left tree, the powerset will abandon
its generation up until the root node, then continue down the righthand side.  If you return `true, -1`, the termination
will occur *before* the root node, meaning the right hand side will not be generated.

In effect, terminating up until a node says "this subtree does not yield a feasible solution, abandon it."

# Applications

One use is for solving problems that could be solved with backtracking.  We can generate the entire solution space
progressively, while checking if our current path through the space state tree is feasible, and if not, terminate our
current branch.

Example solving the N-queens problem:
