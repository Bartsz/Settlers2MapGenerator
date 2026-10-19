package internalmapmanipulation

import (
	"fmt"
	"math"
)

// RotateMask rotates a complete odd-size hex-neighbour mask in raw map X/Y
// coordinates about its centre. The 72-degree rotation used for five players
// does not preserve the discrete hex lattice, so raster collisions and gaps
// are corrected deterministically while preserving the original cell count
// and six-neighbour connectivity. All output shapes remain odd squares.
func RotateMask(degrees float64, original Shape) (Shape, error) {
	if err := validateShape(original); err != nil {
		return nil, err
	}
	c := len(original) / 2
	angle := degrees * math.Pi / 180
	sn, cs := math.Sin(angle), math.Cos(angle)
	type floatPoint struct{ x, y float64 }
	ideal := make([]floatPoint, 0)
	unique := map[Point]bool{}
	rounded := make([]Point, 0)
	for y, row := range original {
		for x, on := range row {
			if !on {
				continue
			}
			dx, dy := float64(x-c), float64(y-c)
			xx, yy := dx*cs-dy*sn, dx*sn+dy*cs
			ideal = append(ideal, floatPoint{xx, yy})
			p := Point{X: int(math.Round(xx)), Y: int(math.Round(yy))}
			if !unique[p] {
				unique[p] = true
				rounded = append(rounded, p)
			}
		}
	}
	if len(ideal) == 0 {
		return nil, fmt.Errorf("cannot rotate empty shape")
	}
	// No repair needed when rounding preserved both cardinality and connectivity.
	if len(rounded) == len(ideal) && connectedPoints(rounded) {
		out, _ := maskFromPoints(rounded, Point{})
		return out, nil
	}
	// Repair near the geometrical centre. Extending only the connected
	// frontier prevents holes or detached one-cell islands. Candidate cells
	// close to an ideally rotated true cell have priority; all ties are stable.
	var seed Point
	closest := math.MaxFloat64
	for _, p := range rounded {
		d := float64(p.X*p.X + p.Y*p.Y)
		if d < closest || (d == closest && (p.Y < seed.Y || (p.Y == seed.Y && p.X < seed.X))) {
			closest, seed = d, p
		}
	}
	selected := []Point{seed}
	used := map[Point]bool{seed: true}
	scores := make(map[Point]float64)
	score := func(p Point) float64 {
		if v, ok := scores[p]; ok {
			return v
		}
		best := math.MaxFloat64
		for _, q := range ideal {
			dx, dy := float64(p.X)-q.x, float64(p.Y)-q.y
			d := dx*dx + dy*dy
			if d < best {
				best = d
			}
		}
		scores[p] = best
		return best
	}
	for len(selected) < len(ideal) {
		best := math.MaxFloat64
		var winner Point
		found := false
		for _, base := range selected {
			for _, o := range hexNeighbors {
				p := Point{X: base.X + o[0], Y: base.Y + o[1]}
				if used[p] {
					continue
				}
				v := score(p)
				if !found || v < best || (v == best && (p.X*p.X+p.Y*p.Y < winner.X*winner.X+winner.Y*winner.Y || (p.X*p.X+p.Y*p.Y == winner.X*winner.X+winner.Y*winner.Y && (p.Y < winner.Y || (p.Y == winner.Y && p.X < winner.X))))) {
					best, winner, found = v, p, true
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("could not repair rotated mask")
		}
		used[winner] = true
		selected = append(selected, winner)
	}
	if len(selected) != len(ideal) || !connectedPoints(selected) {
		return nil, fmt.Errorf("rotation mask count/connectivity mismatch")
	}
	out, _ := maskFromPoints(selected, Point{})
	return out, nil
}
