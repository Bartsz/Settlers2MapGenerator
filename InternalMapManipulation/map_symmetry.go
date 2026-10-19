package internalmapmanipulation

import (
	"fmt"
	"math"
)

type rotationTrig struct{ sin, cos float64 }

var symmetryTrig = func() [7][6]rotationTrig {
	var out [7][6]rotationTrig
	for players := 3; players <= 6; players++ {
		for k := 0; k < players; k++ {
			angle := 2 * math.Pi * float64(k) / float64(players)
			out[players][k] = rotationTrig{sin: math.Sin(angle), cos: math.Cos(angle)}
		}
	}
	return out
}()

// RotatePoint is kept as a 150-cell compatibility API for existing callers.
func RotatePoint(p Point, player, players int) (Point, error) {
	return RotatePointSized(p, player, players, 150)
}

// RotatePointSized reflects two-player halves across BOTH axes, four-player
// quarters sequentially across vertical and horizontal axes, and rotates the
// actual axial hex grid for 3/6 players; five players use rounded polar rotation.
func RotatePointSized(p Point, player, players, size int) (Point, error) {
	if players < 2 || players > 6 || player < 0 || player >= players || size < 3 {
		return Point{}, fmt.Errorf("invalid player symmetry or size")
	}
	if players == 2 {
		if player == 0 {
			return p, nil
		}
		return Point{X: size - 1 - p.X, Y: size - 1 - p.Y}, nil
	}
	if players == 4 {
		switch player {
		case 0:
			return p, nil
		case 1:
			return Point{X: size - 1 - p.X, Y: p.Y}, nil
		case 2:
			return Point{X: size - 1 - p.X, Y: size - 1 - p.Y}, nil
		default:
			return Point{X: p.X, Y: size - 1 - p.Y}, nil
		}
	}
	// Circular maps are generated and displayed in the native square X/Y
	// raster. Rotate in that same Cartesian space; the connected-mask
	// rasterizer fixes rounding collisions when the angle is not 90 degrees.
	c := float64(size-1) / 2
	angle := symmetryTrig[players][player]
	dx, dy := float64(p.X)-c, float64(p.Y)-c
	return Point{
		X: int(math.Round(c + dx*angle.cos - dy*angle.sin)),
		Y: int(math.Round(c + dx*angle.sin + dy*angle.cos)),
	}, nil
}
func cubeRoundAxial(q, r float64) (int, int) {
	x, z := math.Round(q), math.Round(r)
	y := math.Round(-q - r)
	dx, dy, dz := math.Abs(x-q), math.Abs(y-(-q-r)), math.Abs(z-r)
	if dx > dy && dx > dz {
		x = -y - z
	} else if dy > dz {
		y = -x - z
	} else {
		z = -x - y
	}
	return int(x), int(z)
}
func inMasterSectorSized(p Point, players, size int) bool {
	half := size / 2
	if players == 2 {
		return p.X >= half
	}
	if players == 4 {
		return p.X >= half && p.Y >= half
	}
	c := float64(size-1) / 2
	angle := math.Atan2(float64(p.Y)-c, float64(p.X)-c)
	return math.Abs(angle) < math.Pi/float64(players)*.999
}
func inMasterSector(p Point, players int) bool { return inMasterSectorSized(p, players, 150) }
