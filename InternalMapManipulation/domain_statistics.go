package internalmapmanipulation

import (
	"fmt"
	"math"
	domain "settlers2-final-go/DomainModel"
)

func ownerOfSized(p Point, players, size int) int {
	if players == 2 {
		if p.X >= size/2 {
			return 0
		}
		return 1
	}
	if players == 4 {
		x, y := p.X >= size/2, p.Y >= size/2
		if x && y {
			return 0
		}
		if !x && y {
			return 1
		}
		if !x && !y {
			return 2
		}
		return 3
	}
	c := float64(size-1) / 2
	angle := math.Atan2(float64(p.Y)-c, float64(p.X)-c)
	wedge := 2 * math.Pi / float64(players)
	k := int(math.Floor((angle+wedge/2+2*math.Pi)/wedge)) % players
	return k
}

func collectDomainStats(m *domain.MapData, players, diameter, lakeCount, lakeSize int) (symmetryStats, error) {
	stats := symmetryStats{Players: players, TemplateSource: 0}
	makeArr := func() []int { return make([]int, players) }
	stats.Water = makeArr()
	stats.Coal = makeArr()
	stats.Iron = makeArr()
	stats.Gold = makeArr()
	stats.Granite = makeArr()
	stats.Rocks = makeArr()
	stats.Forests = makeArr()
	stats.Fish = makeArr()
	stats.Animals = makeArr()
	stats.Lakes = makeArr()
	sectorArea := makeArr()

	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if !playable(x, y, diameter, players) {
				continue
			}
			k := ownerOfSized(Point{X: x, Y: y}, players, m.Width)
			if k < 0 || k >= players {
				return stats, fmt.Errorf("bad sector for %d,%d", x, y)
			}
			sectorArea[k]++
			switch m.Resources[y][x] {
			case domain.ResourceCoal:
				stats.Coal[k]++
			case domain.ResourceIron:
				stats.Iron[k]++
			case domain.ResourceGold:
				stats.Gold[k]++
			case domain.ResourceGranite:
				stats.Granite[k]++
			case domain.ResourceFish:
				stats.Fish[k]++
			}
			if m.Resources[y][x] >= domain.ResourceTree1 && m.Resources[y][x] <= domain.ResourceTree8 {
				stats.Forests[k]++
			}
			if m.Resources[y][x] >= domain.ResourceSurfaceStone1 && m.Resources[y][x] <= domain.ResourceSurfaceStone6 {
				stats.Rocks[k]++
			}
			if m.Textures[y][x] == domain.TextureWater {
				stats.Water[k]++
			}
			if m.Animals[y][x] != domain.AnimalNone {
				stats.Animals[k]++
			}
		}
	}
	stats.PlayablePerPlayer = sectorArea[0]
	// Inspect actual connected water bodies, not just the requested lake count.
	// Prevent reporting a valid map when rasterization breaks a lake into parts.
	visited := make([]bool, m.Width*m.Height)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			at := y*m.Width + x
			if visited[at] || m.Textures[y][x] != domain.TextureWater {
				continue
			}
			if !playable(x, y, diameter, players) {
				return stats, fmt.Errorf("lake outside playable map")
			}
			owner := ownerOfSized(Point{X: x, Y: y}, players, m.Width)
			visited[at] = true
			queue := []Point{{X: x, Y: y}}
			for head := 0; head < len(queue); head++ {
				p := queue[head]
				for _, dir := range hexNeighbors {
					xx, yy := p.X+dir[0], p.Y+dir[1]
					if xx < 0 || yy < 0 || xx >= m.Width || yy >= m.Height {
						continue
					}
					idx := yy*m.Width + xx
					if visited[idx] || m.Textures[yy][xx] != domain.TextureWater {
						continue
					}
					if ownerOfSized(Point{X: xx, Y: yy}, players, m.Width) != owner {
						return stats, fmt.Errorf("lake crosses player sectors")
					}
					visited[idx] = true
					queue = append(queue, Point{X: xx, Y: yy})
				}
			}
			stats.Lakes[owner]++
			if len(queue) != lakeSize {
				return stats, fmt.Errorf("player %d: lake has %d tiles instead of %d", owner+1, len(queue), lakeSize)
			}
		}
	}
	for player, count := range stats.Lakes {
		if count != lakeCount {
			return stats, fmt.Errorf("player %d has %d lakes instead of %d", player+1, count, lakeCount)
		}
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.Resources[y][x] == domain.ResourceFish && m.Textures[y][x] != domain.TextureWater {
				return stats, fmt.Errorf("fish outside lake at %d,%d", x, y)
			}
		}
	}
	return stats, nil
}

func validateParity(stats symmetryStats) error {
	for _, series := range []struct {
		name   string
		values []int
	}{{"lake water", stats.Water}, {"coal", stats.Coal}, {"iron", stats.Iron}, {"gold", stats.Gold}, {"granite", stats.Granite}, {"fish", stats.Fish}, {"animals", stats.Animals}, {"surface stones", stats.Rocks}, {"forest", stats.Forests}, {"lakes", stats.Lakes}} {
		for k, v := range series.values {
			if v != series.values[0] {
				return fmt.Errorf("resource imbalance %s player %d: %d instead of %d", series.name, k+1, v, series.values[0])
			}
		}
	}
	return nil
}
