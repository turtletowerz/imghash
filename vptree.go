package imghash

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"
	"time"
)

// TODO: remove commented out code if tests succeed

/*

Some theory discussed in danbooru's discord
	- Because of how (relatively) quick it is to rebuild the tree from scratch, it probably isn't necessary to balance
	  it after every insert, but to instead do a full rebuild once every day

	- Adding should be relatively simple to implement, just traverse down the tree by comparing the added hash to each radius

*/

type node struct {
	Point  Hash
	Radius int
	Near   *node
	Far    *node
}

type Tree struct {
	work  []int
	Root  *node
	count int
}

type heapItem struct {
	Item *Hash
	Dist int
}

// Less compares with > because we want it to be sorted by smallest to greatest distances from the reference node.
type Queue []heapItem

func (q *Queue) Max() heapItem        { return (*q)[0] }
func (q *Queue) Len() int             { return len(*q) }
func (q *Queue) Less(i, j int) bool   { return (*q)[i].Item == nil || (*q)[i].Dist > (*q)[j].Dist }
func (q *Queue) Push(x interface{})   { *q = append(*q, x.(heapItem)) }
func (q *Queue) Pop() (i interface{}) { i, *q = (*q)[len(*q)-1], (*q)[:len(*q)-1]; return i }
func (q *Queue) Swap(i, j int)        { (*q)[i], (*q)[j] = (*q)[j], (*q)[i] }

// Constructs a new tree and returns it.
func NewTree(p []Hash) *Tree {
	rand.Seed(time.Now().Unix())

	t := new(Tree)
	t.count = len(p)
	t.work = make([]int, t.count)
	t.Root = t.build(p)
	return t
}

// Faster than sort.Slice, and allows for some flexibility in future optimizations
type byDist struct {
	dists  []int
	points []Hash
}

func (c byDist) Len() int           { return len(c.dists) }
func (c byDist) Less(i, j int) bool { return c.dists[i] < c.dists[j] }
func (c byDist) Swap(i, j int) {
	c.dists[i], c.dists[j] = c.dists[j], c.dists[i]
	c.points[i], c.points[j] = c.points[j], c.points[i]
}

func (t *Tree) build(p []Hash) *node {
	// Handle basic cases
	switch len(p) {
	case 0:
		return nil
	case 1:
		return &node{Point: p[0]}
	}

	n := node{Point: p[rand.Intn(len(p))]}

	// Construct working distances that we can then use to quickly sort the remaining points
	t.work = t.work[:len(p)]
	for i, p := range p {
		t.work[i] = n.Point.Distance(p)
	}

	// Sorting is slow without using a slice of the dists
	sort.Sort(byDist{dists: t.work, points: p})
	// sort.Slice(s, func(i, j int) bool { return n.Point.Distance(s[i]) < n.Point.Distance(s[j]) })

	half := len(p) / 2
	n.Radius = n.Point.Distance(p[half])
	n.Near = t.build(p[1:half])
	n.Far = t.build(p[half:])
	return &n
}

// Returns the number of elements in the tree.
func (t *Tree) Len() int {
	return t.count
}

//type comparable func(int) bool

// func (t *Tree) nearest(q *Queue, e *Hash, c comparable) {
// 	if t.Root == nil {
// 		return
// 	}

// 	t.Root.search(q, e, c)

// 	// Remove the MaxInt that is added by nearest searches
// 	removeInit := (q.Len() > 0 && q.Max().Item == nil)
// 	sort.Sort(sort.Reverse(q))

// 	if removeInit {
// 		q.Pop()
// 	}
// }

// Returns the nearest items to q, doing a length == cap validation if check is true
func (t *Tree) nearest(q *Queue, e *Hash, check bool) {
	if t.Root == nil {
		return
	}

	t.Root.search(q, e, check)

	// Remove the MaxInt that is added by nearest searches
	removeInit := (q.Len() > 0 && q.Max().Item == nil)
	sort.Sort(sort.Reverse(q))

	if removeInit {
		q.Pop()
	}
}

// Returns the closest entry to e
func (t *Tree) Nearest(e *Hash) (*Hash, int) {
	q := t.NearestN(e, 2)

	// If no matching entries were found, return empty
	if len(q) == 0 || q[0].Item == nil {
		return nil, -1
	}
	return q[0].Item, q[0].Dist
}

// Returns the closest N entries to entry e
func (t *Tree) NearestN(e *Hash, n int) (q Queue) {
	q = make(Queue, 1, n)
	q[0].Dist = math.MaxInt64

	t.nearest(&q, e, true)
	// t.nearest(&q, e, func(dist int) bool {
	// 	if dist <= q.Max().Dist {
	// 		if len(q) == cap(q) {
	// 			heap.Pop(&q)
	// 		}
	// 		return true
	// 		//heap.Push(&q, heapItem{h, dist})
	// 	}
	// 	return false
	// })
	return
}

// Returns all entries where e.Distance(entry) <= d
func (t *Tree) NearestDist(e *Hash, d int) (q Queue) {
	q = Queue{{Dist: d}}
	t.nearest(&q, e, false)

	// t.nearest(&q, e, func(dist int) bool {
	// 	/*
	// 		if dist <= q.Max().Dist {
	// 			heap.Push(&q, heapItem{h, dist})
	// 		}
	// 	*/
	// 	return dist <= q.Max().Dist
	// })
	return
}

//func (n *node) search(q *Queue, e *Hash, comp comparable) {
func (n *node) search(q *Queue, e *Hash, check bool) {
	// We've reached a leaf's child with nowhere to go
	if n == nil {
		return
	}

	// Gets the distance and comapres it to the max in the queue, popping if full and adding the new entry
	threshold := e.Distance(n.Point)
	if threshold <= q.Max().Dist {
		if check && len(*q) == cap(*q) {
			heap.Pop(q)
		}

		heap.Push(q, heapItem{&n.Point, threshold})
	}

	// Checks near or far node recursively
	if threshold < n.Radius {
		n.Near.search(q, e, check)
		if threshold+q.Max().Dist >= n.Radius {
			n.Far.search(q, e, check)
		}
	} else {
		n.Far.search(q, e, check)
		if threshold-q.Max().Dist <= n.Radius {
			n.Near.search(q, e, check)
		}
	}
}
