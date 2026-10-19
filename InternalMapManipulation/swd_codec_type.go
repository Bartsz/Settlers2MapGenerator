package internalmapmanipulation

import "encoding/binary"

// swd is the private native-layer record used only by the SWD codec/adapter.
// It is NOT the editable game domain model: DomainModel.MapData owns logical
// textures, heights, resources and animals. Only encodeSWD writes file bytes.
type swd struct {
	b             []byte // the 2352-byte native header, including HQ and area metadata
	layers        [14][]byte
	size          int
	width, height int
}

func (s swd) layer(i int) []byte { return s.layers[i] }

func (s swd) isWater(i int) bool { return s.layer(1)[i]&63 == 5 && s.layer(2)[i]&63 == 5 }

func (s swd) placeHQ(starts [][2]int) {
	for i, p := range starts {
		if i >= 7 {
			break
		}
		binary.LittleEndian.PutUint16(s.b[0x38+2*i:], uint16(p[0]))
		binary.LittleEndian.PutUint16(s.b[0x46+2*i:], uint16(p[1]))
	}
}
