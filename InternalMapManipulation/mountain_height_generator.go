package internalmapmanipulation

import (
	"math"
	domain "settlers2-final-go/DomainModel"
)

// RaiseMiningAreas is applied after sector replication. Every connected
// mountain patch within the playable area has a two-cell inward transition:
// boundary=land+2, next=land+4, core=land+5 (level 6 for flat level 1).
// It applies equally to ore and deliberately empty mountain patches.
func RaiseMiningAreas(m *domain.MapData, diameter, players, landLevel int) {
	dirs := hexNeighbors[:]
	if players == 2 || players == 4 {
		dirs = hexNeighbors[:4]
	}
	distances := make([][]int, m.Height)
	queue := make([]Point, 0, m.Width*m.Height)
	for y := 0; y < m.Height; y++ {
		distances[y] = make([]int, m.Width)
		for x := 0; x < m.Width; x++ {
			if !playable(x, y, diameter, players) || m.Textures[y][x] != domain.TextureMountain {
				distances[y][x] = -1
				continue
			}
			// A land cell at the edge of mountain texture is transition level 1.
			boundary := false
			for _, d := range dirs {
				xx, yy := x+d[0], y+d[1]
				if xx < 0 || yy < 0 || xx >= m.Width || yy >= m.Height || m.Textures[yy][xx] != domain.TextureMountain || !playable(xx, yy, diameter, players) {
					boundary = true
					break
				}
			}
			if boundary {
				distances[y][x] = 0
				queue = append(queue, Point{X: x, Y: y})
			} else {
				distances[y][x] = -2
			}
		}
	}
	for head := 0; head < len(queue); head++ {
		p := queue[head]
		for _, d := range dirs {
			x, y := p.X+d[0], p.Y+d[1]
			if x < 0 || y < 0 || x >= m.Width || y >= m.Height || distances[y][x] != -2 {
				continue
			}
			distances[y][x] = distances[p.Y][p.X] + 1
			queue = append(queue, Point{X: x, Y: y})
		}
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if distances[y][x] < 0 {
				continue
			}
			level := landLevel + min(5, 2*(distances[y][x]+1))
			if level > 255 {
				level = 255
			}
			m.Heights[y][x] = byte(level)
		}
	}
}

// RaiseOuterMountains shapes ONLY the inaccessible ring for round maps.
// Four terrain cells form a smooth ramp towards levels 12..20 on the default
// level-1 landscape. Coarse bilinearly interpolated noise avoids height spikes
// and does not depend on iteration order or on the placement RNG.
func RaiseOuterMountains(m *domain.MapData, diameter, players, landLevel int, seed int64) {
	if players == 2 || players == 4 {
		return
	}
	centre := float64(diameter-1) / 2
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if playable(x, y, diameter, players) {
				continue
			}
			dist := math.Hypot(float64(x)-centre, float64(y)-centre) - centre
			// A height increment of at most ~3 per node across four transition rings.
			rise := math.Min(11, math.Max(0, (dist+1)*2.75))
			if dist >= 3 {
				noise := smoothMountainNoise(float64(x), float64(y), seed)
				rise += noise * math.Min(1, math.Max(0, (dist-3)/3))
			}
			level := landLevel + int(math.Round(rise))
			if level > 255 {
				level = 255
			}
			m.Heights[y][x] = byte(level)
		}
	}
	// The SWD editor tolerates a maximum of five height units between
	// neighbouring hex nodes. The 4-ring slope plus a diagonal step and
	// correlated noise can occasionally exceed that bound by one unit.
	// Lower only local high points, never raise water or the playable ground.
	// Legacy experimental levels 40/100 retain their original water handling.
	if landLevel == 1 {
		LimitNeighbourSlopes(m, 5)
	}
}

// LimitNeighbourSlopes removes accidental cliffs after all terrain passes.
// Heights only decrease, so the result is deterministic and terminates.
func LimitNeighbourSlopes(m *domain.MapData, maximum int) {
	if maximum < 1 {
		return
	}
	for pass := 0; pass < 32; pass++ {
		changed := false
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				h := int(m.Heights[y][x])
				for _, d := range hexNeighbors {
					nx, ny := x+d[0], y+d[1]
					if nx < 0 || ny < 0 || nx >= m.Width || ny >= m.Height {
						continue
					}
					if limit := int(m.Heights[ny][nx]) + maximum; h > limit {
						h = limit
					}
				}
				if h < int(m.Heights[y][x]) {
					m.Heights[y][x] = byte(h)
					changed = true
				}
			}
		}
		if !changed {
			return
		}
	}
}

func mountainHash(x, y int, seed int64) float64 {
	h := uint64(seed) ^ uint64(int64(x)+65536)*0x9e3779b185ebca87 ^ uint64(int64(y)+65536)*0xc2b2ae3d27d4eb4f
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	h *= 0x94d049bb133111eb
	h ^= h >> 31
	return float64(h%10001) / 10000 * 8
}
func smoothMountainNoise(x, y float64, seed int64) float64 {
	const cell = 9.0
	gx, gy := math.Floor(x/cell), math.Floor(y/cell)
	u, v := x/cell-gx, y/cell-gy
	u = u * u * (3 - 2*u)
	v = v * v * (3 - 2*v)
	a := mountainHash(int(gx), int(gy), seed)
	b := mountainHash(int(gx)+1, int(gy), seed)
	c := mountainHash(int(gx), int(gy)+1, seed)
	d := mountainHash(int(gx)+1, int(gy)+1, seed)
	return (a*(1-u)+b*u)*(1-v) + (c*(1-u)+d*u)*v
}
