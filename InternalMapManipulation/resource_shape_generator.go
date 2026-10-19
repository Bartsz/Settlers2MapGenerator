package internalmapmanipulation

import (
	"container/heap"
	"fmt"
	"math/rand"
)

type pointQueueItem struct{ index, radius, tie int }
type pointQueue []pointQueueItem

func (q pointQueue) Len() int { return len(q) }
func (q pointQueue) Less(i, j int) bool {
	if q[i].radius != q[j].radius {
		return q[i].radius < q[j].radius
	}
	if q[i].tie != q[j].tie {
		return q[i].tie < q[j].tie
	}
	return q[i].index < q[j].index
}
func (q pointQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *pointQueue) Push(v any)   { *q = append(*q, v.(pointQueueItem)) }
func (q *pointQueue) Pop() any     { n := len(*q); v := (*q)[n-1]; *q = (*q)[:n-1]; return v }

// Shape is a square odd-size mask indexed [y][x], whose centre is [n/2][n/2].
type Shape = [][]bool

// GenerateShape is the only generator of compact, six-neighbour-connected shapes.
// Its output can be reused for water, ores, forests, animals and surface stones.
func GenerateShape(tiles int, seed int64) (Shape, error) {
	return generateShapeWithNeighbours(tiles, seed, hexNeighbors[:])
}

// GenerateShapeForPlayers retains one shape-generating algorithm. In square
// modes, 4-way connected masks remain connected after X/Y reflection; these
// are also connected on the game's six-neighbour hex lattice.
func GenerateShapeForPlayers(tiles int, seed int64, players int) (Shape, error) {
	if players == 2 || players == 4 {
		return generateShapeWithNeighbours(tiles, seed, hexNeighbors[:4])
	}
	return GenerateShape(tiles, seed)
}
func generateShapeWithNeighbours(tiles int, seed int64, neighbors [][2]int) (Shape, error) {
	if tiles < 1 {
		return nil, fmt.Errorf("shape requires at least one tile")
	}
	radius := 0
	for 1+3*radius*(radius+1) < tiles {
		radius++
	}
	if radius > 1024 {
		return nil, fmt.Errorf("shape too large")
	}
	n := 2*radius + 1
	mask := grid[bool](n, n)
	seen := grid[bool](n, n)
	rng := rand.New(rand.NewSource(seed))
	q := &pointQueue{}
	heap.Init(q)
	var enqueue func(int, int)
	enqueue = func(x, y int) {
		if x < 0 || y < 0 || x >= n || y >= n || seen[y][x] || hexDistance(x-radius, y-radius) > radius {
			return
		}
		seen[y][x] = true
		heap.Push(q, pointQueueItem{index: y*n + x, radius: hexDistance(x-radius, y-radius), tie: rng.Intn(1 << 24)})
	}
	enqueue(radius, radius)
	for placed := 0; placed < tiles; placed++ {
		if q.Len() == 0 {
			return nil, fmt.Errorf("cannot produce %d tiles", tiles)
		}
		p := heap.Pop(q).(pointQueueItem)
		x, y := p.index%n, p.index/n
		mask[y][x] = true
		for _, d := range neighbors {
			enqueue(x+d[0], y+d[1])
		}
	}
	return mask, nil
}

// ExpandShape grows a mask by hex-neighbour steps, for terrain/shore envelopes.
func ExpandShape(mask Shape, steps int) (Shape, error) {
	return expandShapeUsingNeighbors(mask, steps, hexNeighbors[:])
}
func ExpandShapeForPlayers(mask Shape, steps, players int) (Shape, error) {
	if players == 2 || players == 4 {
		return expandShapeUsingNeighbors(mask, steps, hexNeighbors[:4])
	}
	return ExpandShape(mask, steps)
}
func expandShapeUsingNeighbors(mask Shape, steps int, neighbors [][2]int) (Shape, error) {
	if err := validateShape(mask); err != nil {
		return nil, err
	}
	if steps < 0 || steps > 16 {
		return nil, fmt.Errorf("invalid expansion %d", steps)
	}
	n := len(mask)
	out := grid[bool](n+2*steps, n+2*steps)
	for y, row := range mask {
		for x, on := range row {
			out[y+steps][x+steps] = on
		}
	}
	for t := 0; t < steps; t++ {
		next := grid[bool](len(out), len(out))
		for y, row := range out {
			for x, on := range row {
				if !on {
					continue
				}
				next[y][x] = true
				for _, d := range neighbors {
					xx, yy := x+d[0], y+d[1]
					if xx >= 0 && yy >= 0 && xx < len(out) && yy < len(out) {
						next[yy][xx] = true
					}
				}
			}
		}
		out = next
	}
	return out, nil
}

func validateShape(mask Shape) error {
	n := len(mask)
	if n < 1 || n%2 == 0 {
		return fmt.Errorf("shape must be an odd square")
	}
	for _, r := range mask {
		if len(r) != n {
			return fmt.Errorf("ragged shape")
		}
	}
	return nil
}

// ConnectedPrefix keeps the same outline and centre, with fewer cells.
func ConnectedPrefix(mask Shape, tiles int) (Shape, error) {
	return connectedPrefixWithNeighbors(mask, tiles, hexNeighbors[:])
}
func ConnectedPrefixForPlayers(mask Shape, tiles, players int) (Shape, error) {
	if players == 2 || players == 4 {
		return connectedPrefixWithNeighbors(mask, tiles, hexNeighbors[:4])
	}
	return ConnectedPrefix(mask, tiles)
}
func connectedPrefixWithNeighbors(mask Shape, tiles int, neighbors [][2]int) (Shape, error) {
	if err := validateShape(mask); err != nil {
		return nil, err
	}
	n := len(mask)
	if tiles < 0 {
		return nil, fmt.Errorf("negative tile count")
	}
	out := grid[bool](n, n)
	if tiles == 0 {
		return out, nil
	}
	r := n / 2
	if !mask[r][r] {
		return nil, fmt.Errorf("shape has no centre")
	}
	queue := [][2]int{{r, r}}
	visited := grid[bool](n, n)
	visited[r][r] = true
	count := 0
	for len(queue) > 0 && count < tiles {
		a := queue[0]
		queue = queue[1:]
		x, y := a[0], a[1]
		out[y][x] = true
		count++
		for _, d := range neighbors {
			xx, yy := x+d[0], y+d[1]
			if xx < 0 || yy < 0 || xx >= n || yy >= n || visited[yy][xx] || !mask[yy][xx] {
				continue
			}
			visited[yy][xx] = true
			queue = append(queue, [2]int{xx, yy})
		}
	}
	if count != tiles {
		return nil, fmt.Errorf("shape capacity smaller than %d", tiles)
	}
	return out, nil
}
