package internalmapmanipulation

import (
	"fmt"
	"math"

	domain "settlers2-final-go/DomainModel"
)

// ReplicateShapes copies one source sector. 2/4 players use literal matrix
// mirrors; 3/6 use exact local 60-degree hex rotations; 5 use polar centre
// rotation and always rotate each complete mask by n*72 degrees.
func ReplicateShapes(m *domain.MapData, placements []domain.ShapePlacement, scatter []Point, players, diameter int) error {
	if players == 2 || players == 4 {
		return replicateSquareMatrices(m, players)
	}
	return replicateCircularMapNew(m, placements, scatter, players, diameter)
}

func replicateCircularMapNew(m *domain.MapData, placements []domain.ShapePlacement, scatter []Point, players, diameter int) error {
	return replicateCircularPlacements(m, placements, scatter, players, diameter, players == 3 || players == 6)
}

func replicateCircularPlacements(m *domain.MapData, placements []domain.ShapePlacement, scatter []Point, players, diameter int, exactHex bool) error {
	for player := 1; player < players; player++ {
		for i, entry := range placements {
			rotated, err := RotatePointSized(entry.Center, player, players, m.Width)
			if err != nil {
				return err
			}
			sourceShape := Shape(entry.Shape)
			if entry.Kind == domain.PlaceMine {
				full := Shape(entry.TerrainShape)
				if len(full) == 0 {
					full = sourceShape
				}
				var err error
				halo, err := resolveRotatedMask(m, full, entry.Center, player, players, diameter, exactHex, func(x, y int) bool {
					return (m.Textures[y][x] == domain.TextureMeadow || m.Textures[y][x] == domain.TextureMountain) && m.Animals[y][x] == domain.AnimalNone && !nearAnyHQ(m, x, y, 3)
				})
				if err != nil {
					return fmt.Errorf("player %d mine %d terrain: %w", player+1, i+1, err)
				}
				mask, centre := maskFromPoints(halo, rotated)
				if err := EmbedTexture(m, mask, centre, domain.TextureMountain); err != nil {
					return err
				}
				if entry.Count == 0 {
					continue
				}
				// The mineral content is filled inside its capacity-independent mountain
				// envelope. The variant may change the number/type without terrain changes.
				core, err := resolveRotatedMask(m, sourceShape, entry.Center, player, players, diameter, exactHex, func(x, y int) bool {
					return m.Textures[y][x] == domain.TextureMountain && m.Resources[y][x] == domain.ResourceNone && !nearAnyHQ(m, x, y, 3)
				})
				if err != nil {
					return fmt.Errorf("player %d mine %d resource: %w", player+1, i+1, err)
				}
				prefix, err := connectedPointPrefixForPlayers(core, rotated, entry.Count, players)
				if err != nil {
					return err
				}
				mask, centre = maskFromPoints(prefix, rotated)
				if err := EmbedResource(m, mask, centre, entry.Resource); err != nil {
					return err
				}
				continue
			}
			allowed := func(x, y int) bool {
				if nearAnyHQ(m, x, y, 3) {
					return false
				}
				switch entry.Kind {
				case domain.PlaceLake:
					if m.Textures[y][x] != domain.TextureMeadow || m.Resources[y][x] != domain.ResourceNone || m.Animals[y][x] != domain.AnimalNone {
						return false
					}
					for _, offset := range hexNeighbors {
						xx, yy := x+offset[0], y+offset[1]
						if xx >= 0 && yy >= 0 && xx < m.Width && yy < m.Height && m.Textures[yy][xx] == domain.TextureWater {
							return false
						}
					}
					return true
				case domain.PlaceStone, domain.PlaceForest, domain.PlaceEmptyMountain:
					return m.Textures[y][x] == domain.TextureMeadow && m.Resources[y][x] == domain.ResourceNone && m.Animals[y][x] == domain.AnimalNone
				case domain.PlaceAnimals:
					r := m.Resources[y][x]
					return m.Textures[y][x] == domain.TextureMeadow && m.Animals[y][x] == domain.AnimalNone && (r >= domain.ResourceTree1 && r <= domain.ResourceTree8)
				}
				return false
			}
			points, err := resolveRotatedMask(m, sourceShape, entry.Center, player, players, diameter, exactHex, allowed)
			if err != nil {
				return fmt.Errorf("player %d shape %d: %w", player+1, i+1, err)
			}
			mask, centre := maskFromPoints(points, rotated)
			switch entry.Kind {
			case domain.PlaceLake:
				err = EmbedLake(m, mask, centre)
				if err == nil {
					for j, p := range points {
						if j%4 == 0 {
							err = m.SetResource(p.X, p.Y, domain.ResourceFish)
							if err != nil {
								break
							}
							m.ResourceAmounts[p.Y][p.X] = 7
						}
					}
				}
			case domain.PlaceStone:
				// The rock terrain is capacity-wide, but surface objects obey the
				// currently selected variant count just like ore clusters.
				err = EmbedTexture(m, mask, centre, domain.TextureRocky)
				if err == nil && entry.Count > 0 {
					active, e := connectedPointPrefixForPlayers(points, rotated, entry.Count, players)
					if e != nil {
						return e
					}
					activeMask, activeCentre := maskFromPoints(active, rotated)
					err = EmbedSurfaceStones(m, activeMask, activeCentre)
				}
			case domain.PlaceForest:
				err = EmbedForest(m, mask, centre)
				if err == nil && (players == 2 || players == 3 || players == 4 || players == 6) {
					// Exact mirrors and 60/120-degree rotations carry the
					// original species, not a fresh coordinate hash.
					source, _ := ShapePoints(sourceShape, entry.Center)
					for _, src := range source {
						dest, e := rotateShapeCell(src, entry.Center, player, players, m.Width, exactHex)
						if e == nil && dest.X >= 0 && dest.Y >= 0 && dest.X < m.Width && dest.Y < m.Height && m.Resources[dest.Y][dest.X] >= domain.ResourceTree1 && m.Resources[dest.Y][dest.X] <= domain.ResourceTree8 {
							m.Resources[dest.Y][dest.X] = m.Resources[src.Y][src.X]
						}
					}
				}
			case domain.PlaceAnimals:
				err = EmbedAnimals(m, mask, centre)
			case domain.PlaceEmptyMountain:
				err = EmbedTexture(m, mask, centre, domain.TextureMountain)
			default:
				err = fmt.Errorf("unsupported shape type %d", entry.Kind)
			}
			if err != nil {
				return fmt.Errorf("player %d shape %d: %w", player+1, i+1, err)
			}
		}
		// Additional forest filling is a set of single-tile source masks. Preserve
		// the tree species; relocate only when raster rounding would collide.
		for _, p := range scatter {
			dest, err := RotatePointSized(p, player, players, m.Width)
			if err != nil {
				return err
			}
			if !eligibleScatter(m, dest, player, players, diameter) {
				found := false
				for radius := 1; radius <= m.Width/2 && !found; radius++ {
					for dy := -radius; dy <= radius && !found; dy++ {
						for dx := -radius; dx <= radius; dx++ {
							if hexDistance(dx, dy) != radius {
								continue
							}
							at := Point{X: dest.X + dx, Y: dest.Y + dy}
							if eligibleScatter(m, at, player, players, diameter) {
								dest = at
								found = true
								break
							}
						}
					}
				}
				if !found {
					return fmt.Errorf("player %d: insufficient forest space after rasterization", player+1)
				}
			}
			m.Resources[dest.Y][dest.X] = m.Resources[p.Y][p.X]
			m.ResourceAmounts[dest.Y][dest.X] = 1
		}
	}
	return nil
}

func nearAnyHQ(m *domain.MapData, x, y, limit int) bool {
	for _, at := range m.Starts {
		if hexDistance(x-at.X, y-at.Y) <= limit {
			return true
		}
	}
	return false
}
func eligibleScatter(m *domain.MapData, p Point, player, players, diameter int) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < m.Width && p.Y < m.Height && playable(p.X, p.Y, diameter, players) && ownerOfSized(p, players, m.Width) == player &&
		m.Textures[p.Y][p.X] == domain.TextureMeadow && m.Resources[p.Y][p.X] == domain.ResourceNone && m.Animals[p.Y][p.X] == domain.AnimalNone && !nearAnyHQ(m, p.X, p.Y, 3)
}

func connectedPoints(points []Point) bool {
	if len(points) == 0 {
		return true
	}
	all := map[Point]bool{}
	for _, p := range points {
		all[p] = true
	}
	if len(all) != len(points) {
		return false
	}
	seen := map[Point]bool{points[0]: true}
	queue := []Point{points[0]}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		for _, o := range hexNeighbors {
			v := Point{X: p.X + o[0], Y: p.Y + o[1]}
			if all[v] && !seen[v] {
				seen[v] = true
				queue = append(queue, v)
			}
		}
	}
	return len(seen) == len(points)
}

func resolveRotatedMask(m *domain.MapData, shape Shape, from Point, k, players, diameter int, exactHex bool, allowed func(x, y int) bool) ([]Point, error) {
	source, err := ShapePoints(shape, from)
	if err != nil {
		return nil, err
	}
	rotatedCentre, err := RotatePointSized(from, k, players, m.Width)
	if err != nil {
		return nil, err
	}
	desired := make([]Point, 0, len(source))
	if players == 5 {
		// Rotate around the mask centre, then translate to the polar-rotated
		// placement centre. Counts/connectivity are repaired before embedding.
		rotatedMask, e := RotateMask(float64(k)*72, shape)
		if e != nil {
			return nil, e
		}
		desired, e = ShapePoints(rotatedMask, rotatedCentre)
		if e != nil {
			return nil, e
		}
	} else {
		for _, p := range source {
			d, e := rotateShapeCell(p, from, k, players, m.Width, exactHex)
			if e != nil {
				return nil, e
			}
			desired = append(desired, d)
		}
	}
	allValid := true
	used := map[Point]bool{}
	for _, d := range desired {
		if used[d] || !inReplicaArea(m, d, k, players, diameter) || !allowed(d.X, d.Y) {
			allValid = false
		}
		used[d] = true
	}
	if len(desired) != len(source) {
		return nil, fmt.Errorf("rotated mask cardinality changed: %d -> %d", len(source), len(desired))
	}
	if allValid && connectedPoints(desired) {
		return desired, nil
	}
	start := rotatedCentre
	valid := func(p Point) bool { return inReplicaArea(m, p, k, players, diameter) && allowed(p.X, p.Y) }
	if !valid(start) {
		best := math.MaxFloat64
		found := false
		for r := 1; r <= 5; r++ {
			for dy := -r; dy <= r; dy++ {
				for dx := -r; dx <= r; dx++ {
					p := Point{X: start.X + dx, Y: start.Y + dy}
					if !valid(p) {
						continue
					}
					sc := pointScore(p, desired)
					if sc < best {
						best = sc
						start = p
						found = true
					}
				}
			}
			if found {
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("rotated shape has no valid centre")
		}
	}
	result := []Point{start}
	placed := map[Point]bool{start: true}
	for len(result) < len(source) {
		best := math.MaxFloat64
		var winner Point
		found := false
		for _, base := range result {
			for _, o := range hexNeighbors {
				p := Point{X: base.X + o[0], Y: base.Y + o[1]}
				if placed[p] || !valid(p) {
					continue
				}
				score := pointScore(p, desired)
				if score < best || (score == best && (!found || p.Y < winner.Y || (p.Y == winner.Y && p.X < winner.X))) {
					winner = p
					best = score
					found = true
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("cannot retain %d-cell connected shape after rotation (%d placed)", len(source), len(result))
		}
		result = append(result, winner)
		placed[winner] = true
	}
	return result, nil
}
func pointScore(p Point, desired []Point) float64 {
	best := math.MaxFloat64
	for _, q := range desired {
		dx, dy := float64(p.X-q.X), float64(p.Y-q.Y)
		d := dx*dx + dy*dy
		if d < best {
			best = d
		}
	}
	return best
}
func inReplicaArea(m *domain.MapData, p Point, k, players, diameter int) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < m.Width && p.Y < m.Height && playable(p.X, p.Y, diameter, players) && ownerOfSized(p, players, m.Width) == k
}

func maskFromPoints(points []Point, at Point) (Shape, Point) {
	radius := 0
	for _, p := range points {
		radius = max(radius, max(absInt(p.X-at.X), absInt(p.Y-at.Y)))
	}
	mask := grid[bool](2*radius+1, 2*radius+1)
	for _, p := range points {
		mask[p.Y-at.Y+radius][p.X-at.X+radius] = true
	}
	return mask, at
}
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func connectedPointPrefixForPlayers(points []Point, at Point, count, players int) ([]Point, error) {
	dirs := hexNeighbors[:]
	if players == 2 || players == 4 {
		dirs = hexNeighbors[:4]
	}
	return connectedPointPrefixWithNeighbors(points, at, count, dirs)
}
func connectedPointPrefixWithNeighbors(points []Point, at Point, count int, dirs [][2]int) ([]Point, error) {
	if count < 0 || count > len(points) {
		return nil, fmt.Errorf("invalid prefix count")
	}
	if count == 0 {
		return nil, nil
	}
	all := map[Point]bool{}
	for _, p := range points {
		all[p] = true
	}
	seed := at
	if !all[seed] {
		seed = points[0]
		for _, p := range points {
			if hexDistance(p.X-at.X, p.Y-at.Y) < hexDistance(seed.X-at.X, seed.Y-at.Y) {
				seed = p
			}
		}
	}
	queue := []Point{seed}
	seen := map[Point]bool{seed: true}
	result := []Point{}
	for len(queue) > 0 && len(result) < count {
		p := queue[0]
		queue = queue[1:]
		result = append(result, p)
		for _, dir := range dirs {
			v := Point{X: p.X + dir[0], Y: p.Y + dir[1]}
			if all[v] && !seen[v] {
				seen[v] = true
				queue = append(queue, v)
			}
		}
	}
	if len(result) != count {
		return nil, fmt.Errorf("could not preserve connected prefix")
	}
	return result, nil
}

// replicateSquareMatrices performs literal matrix copies, not independent
// placement or shape repair. 2p: right half -> left half with both axes
// reversed. 4p: bottom-right quarter -> bottom-left -> entire top half.
func replicateSquareMatrices(m *domain.MapData, players int) error {
	if m.Width != m.Height || m.Width%2 != 0 {
		return fmt.Errorf("square reflection requires even square map")
	}
	n := m.Width
	half := n / 2
	copyCell := func(sx, sy, dx, dy int) {
		m.Heights[dy][dx] = m.Heights[sy][sx]
		m.Textures[dy][dx] = m.Textures[sy][sx]
		m.TextureTriangles[dy][dx] = m.TextureTriangles[sy][sx]
		m.Resources[dy][dx] = m.Resources[sy][sx]
		m.ResourceAmounts[dy][dx] = m.ResourceAmounts[sy][sx]
		m.Animals[dy][dx] = m.Animals[sy][sx]
	}
	if players == 2 {
		for y := 0; y < n; y++ {
			for x := half; x < n; x++ {
				copyCell(x, y, n-1-x, n-1-y)
			}
		}
		return nil
	}
	for y := half; y < n; y++ {
		for x := half; x < n; x++ {
			copyCell(x, y, n-1-x, y)
		}
	}
	for y := half; y < n; y++ {
		for x := 0; x < n; x++ {
			copyCell(x, y, x, n-1-y)
		}
	}
	return nil
}

// rotateShapeCell is an exact hex-grid automorphism relative to the shape's
// centre: (q,r) -> (-r,q+r) for each 60-degree step. The centre is placed
// by the existing Cartesian rotation; this matters because the playable
// boundary is a circle in raw X/Y coordinates, not the axial hex metric.
func rotateShapeCell(cell, sourceCentre Point, player, players, size int, exactHex bool) (Point, error) {
	if !exactHex || (players != 3 && players != 6) {
		return RotatePointSized(cell, player, players, size)
	}
	centre, err := RotatePointSized(sourceCentre, player, players, size)
	if err != nil {
		return Point{}, err
	}
	q, r := cell.X-sourceCentre.X, cell.Y-sourceCentre.Y
	steps := player
	if players == 3 {
		steps *= 2
	}
	for i := 0; i < steps; i++ {
		q, r = -r, q+r
	}
	return Point{X: centre.X + q, Y: centre.Y + r}, nil
}
