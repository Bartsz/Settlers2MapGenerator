package internalmapmanipulation

import (
	"fmt"
	"math"
	domain "settlers2-final-go/DomainModel"
)

func ShapePoints(shape Shape, centre Point) ([]Point, error) {
	if err := validateShape(shape); err != nil {
		return nil, err
	}
	r := len(shape) / 2
	out := make([]Point, 0)
	for y, row := range shape {
		for x, on := range row {
			if on {
				out = append(out, Point{X: centre.X + x - r, Y: centre.Y + y - r})
			}
		}
	}
	return out, nil
}

func ValidatePlacement(m *domain.MapData, mask Shape, at Point, predicate func(x, y int) bool) error {
	pts, err := ShapePoints(mask, at)
	if err != nil {
		return err
	}
	for _, p := range pts {
		if p.X < 0 || p.Y < 0 || p.X >= m.Width || p.Y >= m.Height {
			return fmt.Errorf("shape outside map at %v", p)
		}
		if predicate != nil && !predicate(p.X, p.Y) {
			return fmt.Errorf("shape collision at (%d,%d)", p.X, p.Y)
		}
	}
	return nil
}

// Circular maps use the native X/Y grid with no axial skew.
func axialRadius(x, y int, centre float64) float64 {
	q, r := float64(x)-centre, float64(y)-centre
	return math.Hypot(q+r*.5, r*math.Sqrt(3)/2)
}

func playable(x, y, diameter, players int) bool {
	if x < 0 || y < 0 || x >= diameter || y >= diameter {
		return false
	}
	// Square map modes have no inaccessible outer mountain belt.
	if players == 2 || players == 4 {
		return true
	}
	return math.Hypot(float64(x)-float64(diameter-1)/2, float64(y)-float64(diameter-1)/2) <= float64(diameter-1)/2
}
