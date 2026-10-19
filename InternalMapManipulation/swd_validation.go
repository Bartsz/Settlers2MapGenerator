package internalmapmanipulation

import (
	"encoding/binary"
	"fmt"
)

func validateSWDDimensions(b []byte, expected int) error {
	s, e := parseSwd(b)
	if e != nil {
		return e
	}
	if s.width != expected || s.height != expected {
		return fmt.Errorf("SWD has %dx%d, wanted %dx%d", s.width, s.height, expected, expected)
	}
	for _, o := range []int{0x1e, 0x20, 0x92c, 0x92e} {
		if int(binary.LittleEndian.Uint16(b[o:])) != expected {
			return fmt.Errorf("incorrect dimensions in SWD header at 0x%x", o)
		}
	}
	return nil
}
