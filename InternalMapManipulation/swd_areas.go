package internalmapmanipulation

import (
	"encoding/binary"
	"errors"
)

func (s swd) rebuildAreasSized() error {
	width, height := s.width, s.height
	areas := s.layer(13)
	for i := range areas {
		areas[i] = 0xff
	}
	clear(s.b[0x5c:0x926])
	areaID := 0
	for i := 0; i < s.size; i++ {
		if areas[i] != 0xff {
			continue
		}
		if areaID >= 250 {
			return errors.New("too many separate water bodies (>250)")
		}
		wt := s.isWater(i)
		q := []int{i}
		areas[i] = byte(areaID)
		for head := 0; head < len(q); head++ {
			at := q[head]
			x, y := at%width, at/width
			for _, d := range hexNeighbors {
				xx, yy := x+d[0], y+d[1]
				if xx < 0 || yy < 0 || xx >= width || yy >= height {
					continue
				}
				j := yy*width + xx
				if areas[j] == 0xff && s.isWater(j) == wt {
					areas[j] = byte(areaID)
					q = append(q, j)
				}
			}
		}
		o := 0x5c + areaID*9
		if wt {
			s.b[o] = 2
		} else {
			s.b[o] = 1
		}
		binary.LittleEndian.PutUint16(s.b[o+1:], uint16(i%width))
		binary.LittleEndian.PutUint16(s.b[o+3:], uint16(i/width))
		binary.LittleEndian.PutUint32(s.b[o+5:], uint32(len(q)))
		areaID++
	}
	return nil
}
