package internalmapmanipulation

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// SWD physical layout. None of the generation passes require a file buffer.
const (
	swdHeaderSize      = 2352
	swdLayerCount      = 14
	swdBlockHeaderSize = 16
)

func newEmptySWD(width, height int) swd {
	s := swd{b: make([]byte, swdHeaderSize), width: width, height: height, size: width * height}
	for i := range s.layers {
		s.layers[i] = make([]byte, s.size)
	}
	return s
}

func (s swd) validate() error {
	if s.width < 1 || s.height < 1 || s.width > 65535 || s.height > 65535 || s.width*s.height != s.size {
		return fmt.Errorf("incorrect map dimensions %dx%d", s.width, s.height)
	}
	if len(s.b) < swdHeaderSize || string(s.b[:10]) != "WORLD_V1.0" {
		return errors.New("invalid WORLD_V1.0 map header")
	}
	if int(binary.LittleEndian.Uint16(s.b[0x92c:])) != s.width || int(binary.LittleEndian.Uint16(s.b[0x92e:])) != s.height {
		return errors.New("map header dimensions differ from map data")
	}
	for i, layer := range s.layers {
		if len(layer) != s.size {
			return fmt.Errorf("layer %d: expected %d cells, got %d", i, s.size, len(layer))
		}
	}
	return nil
}

// encodeSWD is the ONLY file-format writer. The map exists as separate mutable
// in-memory layers until this step; no algorithm edits a serialized SWD buffer.
func encodeSWD(s swd) ([]byte, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	n := swdHeaderSize + swdLayerCount*(swdBlockHeaderSize+s.size) + 1
	buf := make([]byte, n)
	copy(buf[:swdHeaderSize], s.b[:swdHeaderSize])
	offset := swdHeaderSize
	for _, data := range s.layers {
		binary.LittleEndian.PutUint16(buf[offset:], 0x2710)
		binary.LittleEndian.PutUint16(buf[offset+6:], uint16(s.width))
		binary.LittleEndian.PutUint16(buf[offset+8:], uint16(s.height))
		binary.LittleEndian.PutUint16(buf[offset+10:], 1)
		binary.LittleEndian.PutUint32(buf[offset+12:], uint32(s.size))
		copy(buf[offset+swdBlockHeaderSize:], data)
		offset += swdBlockHeaderSize + s.size
	}
	buf[offset] = 0xff
	return buf, nil
}

// parseSwd decodes external SWD files into the same in-memory representation.
// Dimensions are read from the format header, not hard-coded to 120 or 150.
func parseSwd(buf []byte) (swd, error) {
	if len(buf) < swdHeaderSize+1 || string(buf[:10]) != "WORLD_V1.0" {
		return swd{}, errors.New("invalid SWD header")
	}
	w := int(binary.LittleEndian.Uint16(buf[0x92c:]))
	h := int(binary.LittleEndian.Uint16(buf[0x92e:]))
	if w < 1 || h < 1 || w > 2048 || h > 2048 {
		return swd{}, fmt.Errorf("unsupported SWD dimensions %dx%d", w, h)
	}
	s := newEmptySWD(w, h)
	copy(s.b, buf[:swdHeaderSize])
	expected := swdHeaderSize + swdLayerCount*(swdBlockHeaderSize+s.size) + 1
	if len(buf) != expected {
		return swd{}, fmt.Errorf("invalid SWD file length %d, expected %d", len(buf), expected)
	}
	offset := swdHeaderSize
	for i := 0; i < swdLayerCount; i++ {
		block := buf[offset:]
		if binary.LittleEndian.Uint16(block) != 0x2710 || binary.LittleEndian.Uint16(block[6:]) != uint16(w) || binary.LittleEndian.Uint16(block[8:]) != uint16(h) || binary.LittleEndian.Uint16(block[10:]) != 1 || int(binary.LittleEndian.Uint32(block[12:])) != s.size {
			return swd{}, fmt.Errorf("invalid SWD block %d", i)
		}
		copy(s.layers[i], block[swdBlockHeaderSize:swdBlockHeaderSize+s.size])
		offset += swdBlockHeaderSize + s.size
	}
	if buf[offset] != 0xff {
		return swd{}, errors.New("missing SWD end marker")
	}
	if err := s.validate(); err != nil {
		return swd{}, err
	}
	return s, nil
}
