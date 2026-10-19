package internalmapmanipulation

import (
	"encoding/binary"
	"fmt"
)

const headquartersType byte = 0x80

func isMountainTexture(v byte) bool { return v == 1 || v == 11 || v == 12 || v == 13 || v == 34 }
func fishByte(v byte) bool          { return (v >= 0x80 && v <= 0x87) || v == 0xf7 }
func hexDistance(dx, dy int) int {
	z := dx + dy
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if z < 0 {
		z = -z
	}
	return max(dx, max(dy, z))
}

func validateStartingPositions(s swd, players int) error {
	if int(s.b[0x23]) != players {
		return fmt.Errorf("incorrect SWD player count")
	}
	used := map[int]bool{}
	for k := 0; k < 7; k++ {
		x, y := int(binary.LittleEndian.Uint16(s.b[0x38+2*k:])), int(binary.LittleEndian.Uint16(s.b[0x46+2*k:]))
		if k >= players {
			if x != 65535 || y != 65535 {
				return fmt.Errorf("unexpected HQ %d", k+1)
			}
			continue
		}
		if x < 0 || y < 0 || x >= s.width || y >= s.height {
			return fmt.Errorf("HQ %d outside map", k+1)
		}
		at := y*s.width + x
		if used[at] || s.layer(5)[at] != headquartersType || s.layer(4)[at] != byte(k) || s.isWater(at) || isMountainTexture(s.layer(1)[at]&63) || isMountainTexture(s.layer(2)[at]&63) || s.layer(11)[at] != 0 || s.layer(8)[at]&7 != 4 {
			return fmt.Errorf("invalid HQ %d", k+1)
		}
		used[at] = true
	}
	n := 0
	for i, b := range s.layer(5) {
		if b == headquartersType {
			n++
			if !used[i] {
				return fmt.Errorf("orphan HQ at %d", i)
			}
		}
	}
	if n != players {
		return fmt.Errorf("missing HQ markers: %d/%d", n, players)
	}
	return nil
}
