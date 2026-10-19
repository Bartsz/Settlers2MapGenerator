package internalmapmanipulation

import (
	"encoding/binary"
	"fmt"
	domain "settlers2-final-go/DomainModel"
)

func textureAt(a, b byte) TextureType {
	a, b = a&63, b&63
	if (a == 4 || a == 7) && (b == 4 || b == 7) {
		return TextureRocky
	}
	if a == 5 && b == 5 {
		return TextureWater
	}
	if isMountainTexture(a) && isMountainTexture(b) {
		return TextureMountain
	}
	isGreen := func(v byte) bool { return v == 0 || v == 8 || v == 9 || v == 10 || v == 14 || v == 15 }
	if isGreen(a) && isGreen(b) {
		return TextureMeadow
	}
	if a != b {
		return TextureMixed
	}
	return TextureOther
}

func resourceAt(ore, objectIndex, objectType byte) (ResourceType, byte) {
	switch {
	case ore >= 0x40 && ore <= 0x47:
		return ResourceCoal, ore & 7
	case ore >= 0x48 && ore <= 0x4f:
		return ResourceIron, ore & 7
	case ore >= 0x50 && ore <= 0x57:
		return ResourceGold, ore & 7
	case ore >= 0x58 && ore <= 0x5f:
		return ResourceGranite, ore & 7
	case fishByte(ore):
		return ResourceFish, ore & 7
	case objectType == 0xcc || objectType == 0xc8 || objectType == 0xc9 || objectType == 0xcd:
		n := int(objectIndex)
		if n < 1 || n > 6 {
			n = 6
		}
		return ResourceType(int(ResourceSurfaceStone1) + n - 1), byte(n)
	case objectType == 0xc4 || objectType == 0xc1 || objectType == 0xc2:
		return ResourceType(int(ResourceTree1) + int(objectIndex&7)), 1
	case ore == 0x21:
		return ResourceUndergroundWater, 0
	default:
		return ResourceNone, 0
	}
}

// MapDataFromSWD reads a native file and converts it into editable typed grids.
func MapDataFromSWD(data []byte) (*MapData, error) {
	s, err := parseSwd(data)
	if err != nil {
		return nil, err
	}
	return mapDataFromLayers(s)
}

func mapDataFromLayers(s swd) (*MapData, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	m := &MapData{
		Width: s.width, Height: s.height,
		Heights: grid[byte](s.width, s.height), Textures: grid[TextureType](s.width, s.height),
		TextureTriangles: grid[TextureTriangles](s.width, s.height),
		Resources:        grid[ResourceType](s.width, s.height), ResourceAmounts: grid[byte](s.width, s.height),
		Animals: grid[AnimalType](s.width, s.height), Original: grid[domain.OriginalCell](s.width, s.height),
	}
	m.RawHeader = append([]byte(nil), s.b...)
	for i := range s.layers {
		m.RawLayers[i] = append([]byte(nil), s.layers[i]...)
	}
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			i := y*s.width + x
			m.Heights[y][x] = s.layer(0)[i]
			up, down := s.layer(1)[i], s.layer(2)[i]
			tex := textureAt(up, down)
			m.Textures[y][x] = tex
			m.TextureTriangles[y][x] = TextureTriangles{Upper: up, Lower: down}
			rt, strength := resourceAt(s.layer(11)[i], s.layer(4)[i], s.layer(5)[i])
			m.Resources[y][x] = rt
			m.ResourceAmounts[y][x] = strength
			m.Animals[y][x] = AnimalType(s.layer(6)[i])
			m.Original[y][x] = domain.OriginalCell{Texture: tex, Resource: rt, Animal: m.Animals[y][x], Strength: strength}
		}
	}
	for player := 0; player < int(s.b[0x23]); player++ {
		if player >= 7 {
			break
		}
		pos := Point{X: int(binary.LittleEndian.Uint16(s.b[0x38+2*player:])), Y: int(binary.LittleEndian.Uint16(s.b[0x46+2*player:]))}
		m.Starts = append(m.Starts, pos)
	}
	m.OriginalStarts = append([]Point(nil), m.Starts...)
	return m, nil
}

func copyLayers(s swd) swd {
	result := newEmptySWD(s.width, s.height)
	copy(result.b, s.b)
	for i := range s.layers {
		copy(result.layers[i], s.layers[i])
	}
	return result
}

// ToLayers applies typed edits to a copy of native layers. Unedited bytes are
// reproduced exactly. This bridge will be removed as legacy algorithms migrate.
func ToLayers(m *MapData) (swd, error) {
	if m == nil || m.Width < 3 || m.Height < 3 {
		return swd{}, fmt.Errorf("invalid model dimensions")
	}
	if len(m.RawHeader) != 0 && (len(m.RawHeader) < swdHeaderSize || len(m.RawLayers[0]) != m.Width*m.Height) {
		return swd{}, fmt.Errorf("invalid native source dimensions")
	}
	if len(m.Heights) != m.Height || len(m.Textures) != m.Height || len(m.Resources) != m.Height || len(m.Animals) != m.Height || len(m.TextureTriangles) != m.Height || len(m.ResourceAmounts) != m.Height {
		return swd{}, fmt.Errorf("invalid grid height")
	}
	var s swd
	if len(m.RawHeader) == 0 {
		s = newEmptySWD(m.Width, m.Height)
		initializeBlankSWD(s, len(m.Starts))
	} else {
		s = copyLayers(swd{b: m.RawHeader, layers: m.RawLayers, size: m.Width * m.Height, width: m.Width, height: m.Height})
	}
	terrainChanged := false
	heightChanged := len(m.RawHeader) == 0
	for y := 0; y < m.Height; y++ {
		if len(m.Heights[y]) != m.Width || len(m.Textures[y]) != m.Width || len(m.Resources[y]) != m.Width || len(m.Animals[y]) != m.Width || len(m.TextureTriangles[y]) != m.Width || len(m.ResourceAmounts[y]) != m.Width {
			return swd{}, fmt.Errorf("invalid grid width at row %d", y)
		}
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			if s.layer(0)[i] != m.Heights[y][x] {
				heightChanged = true
			}
			s.layer(0)[i] = m.Heights[y][x]
			t := m.Textures[y][x]
			pair := m.TextureTriangles[y][x]
			switch {
			case t != m.Original[y][x].Texture:
				var upper, lower byte
				switch t {
				case TextureMeadow:
					upper, lower = 8, 9
				case TextureMountain:
					upper, lower = 1, 11
				case TextureRocky:
					upper, lower = 4, 7
				case TextureWater:
					upper, lower = 5, 5
				default:
					return swd{}, fmt.Errorf("unsupported edited texture at %d,%d", x, y)
				}
				pair = TextureTriangles{Upper: upper, Lower: lower}
			}
			if pair.Upper != s.layer(1)[i] || pair.Lower != s.layer(2)[i] {
				s.layer(1)[i], s.layer(2)[i] = pair.Upper, pair.Lower
				terrainChanged = true
				switch t {
				case TextureWater:
					s.layer(8)[i] = 0x78
				case TextureMeadow:
					s.layer(8)[i] = 4
				case TextureMountain:
					s.layer(8)[i] = 5
				case TextureRocky:
					s.layer(8)[i] = 4
				}
			}
			s.layer(6)[i] = byte(m.Animals[y][x])
			kind := m.Resources[y][x]
			if kind == m.Original[y][x].Resource && m.ResourceAmounts[y][x] == m.Original[y][x].Strength {
				continue
			}
			// Validate new/modified resources before serializing native encodings.
			actualTexture := textureAt(s.layer(1)[i], s.layer(2)[i])
			if kind >= ResourceCoal && kind <= ResourceGranite && actualTexture != TextureMountain {
				return swd{}, fmt.Errorf("mine at %d,%d requires mountain terrain", x, y)
			}
			if kind == ResourceFish && actualTexture != TextureWater {
				return swd{}, fmt.Errorf("fish at %d,%d requires water terrain", x, y)
			}
			if kind >= ResourceSurfaceStone1 && kind <= ResourceTree8 && actualTexture == TextureWater {
				return swd{}, fmt.Errorf("surface resource at %d,%d cannot be on water", x, y)
			}
			if kind != ResourceNone && s.layer(5)[i] == headquartersType {
				return swd{}, fmt.Errorf("resource at %d,%d overlaps headquarters", x, y)
			}
			// Clear only previously typed resource elements, not unrelated objects.
			old := m.Original[y][x].Resource
			if old >= ResourceCoal && old <= ResourceUndergroundWater {
				s.layer(11)[i] = 0
			}
			if old >= ResourceSurfaceStone1 && old <= ResourceTree8 {
				s.layer(4)[i], s.layer(5)[i] = 0, 0
			}
			switch {
			case kind == ResourceNone: // leave unrelated raw layer data intact
			case kind == ResourceCoal:
				s.layer(11)[i] = 0x40 | (m.ResourceAmounts[y][x] & 7)
			case kind == ResourceIron:
				s.layer(11)[i] = 0x48 | (m.ResourceAmounts[y][x] & 7)
			case kind == ResourceGold:
				s.layer(11)[i] = 0x50 | (m.ResourceAmounts[y][x] & 7)
			case kind == ResourceGranite:
				s.layer(11)[i] = 0x58 | (m.ResourceAmounts[y][x] & 7)
			case kind == ResourceFish:
				s.layer(11)[i] = 0x80 | (m.ResourceAmounts[y][x] & 7)
			case kind == ResourceUndergroundWater:
				s.layer(11)[i] = 0x21
			case kind >= ResourceSurfaceStone1 && kind <= ResourceSurfaceStone6:
				s.layer(4)[i], s.layer(5)[i] = byte(kind-ResourceSurfaceStone1+1), 0xcc
			case kind >= ResourceTree1 && kind <= ResourceTree8:
				s.layer(4)[i], s.layer(5)[i] = 0x30+byte(kind-ResourceTree1), 0xc4
			default:
				return swd{}, fmt.Errorf("unsupported resource at %d,%d", x, y)
			}
		}
	}
	if len(m.Starts) != len(m.OriginalStarts) {
		return swd{}, fmt.Errorf("wrong number of headquarters")
	}
	changedStart := false
	for i, p := range m.Starts {
		if p != m.OriginalStarts[i] {
			changedStart = true
		}
		if err := m.CheckCoordinates(p.X, p.Y); err != nil {
			return swd{}, err
		}
	}
	if changedStart {
		// Object markers and header positions move together.
		for i := range s.layer(5) {
			if s.layer(5)[i] == headquartersType {
				s.layer(4)[i], s.layer(5)[i] = 0, 0
			}
		}
		for p, at := range m.Starts {
			i := at.Y*m.Width + at.X
			if s.isWater(i) || isMountainTexture(s.layer(1)[i]&63) || isMountainTexture(s.layer(2)[i]&63) {
				return swd{}, fmt.Errorf("start %d on unsuitable terrain", p+1)
			}
			s.layer(4)[i], s.layer(5)[i], s.layer(8)[i] = byte(p), headquartersType, 4
			binary.LittleEndian.PutUint16(s.b[0x38+2*p:], uint16(at.X))
			binary.LittleEndian.PutUint16(s.b[0x46+2*p:], uint16(at.Y))
		}
	}
	if heightChanged {
		// Gouraud layer records illumination. Without this an elevated ring
		// appears flat despite its varied height values.
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				at := y*m.Width + x
				h := func(xx, yy int) int {
					if xx < 0 || yy < 0 || xx >= m.Width || yy >= m.Height {
						return int(m.Heights[y][x])
					}
					return int(m.Heights[yy][xx])
				}
				shade := 64 + 4*(h(x-1, y)-h(x+1, y)) + 3*(h(x, y-1)-h(x, y+1))
				if shade < 0 {
					shade = 0
				}
				if shade > 128 {
					shade = 128
				}
				s.layer(12)[at] = byte(shade)
			}
		}
	}
	if terrainChanged {
		if err := s.rebuildAreasSized(); err != nil {
			return swd{}, err
		}
	}
	if terrainChanged || changedStart {
		if err := validateStartingPositions(s, len(m.Starts)); err != nil {
			return swd{}, err
		}
	}
	if err := s.validate(); err != nil {
		return swd{}, err
	}
	return s, nil
}

// ToSWD produces an SWD only on demand, directly from the editable model.
func ToSWD(m *MapData) ([]byte, error) {
	layers, err := ToLayers(m)
	if err != nil {
		return nil, err
	}
	return encodeSWD(layers)
}

// initializeBlankSWD creates native metadata only at export time.
func initializeBlankSWD(s swd, players int) {
	copy(s.b[:10], "WORLD_V1.0")
	copy(s.b[0x0a:0x1e], "Settlers Go Map")
	binary.LittleEndian.PutUint16(s.b[0x1e:], uint16(s.width))
	binary.LittleEndian.PutUint16(s.b[0x20:], uint16(s.height))
	s.b[0x22] = 0
	s.b[0x23] = byte(players)
	copy(s.b[0x24:0x38], "Unified Map Model")
	binary.LittleEndian.PutUint16(s.b[0x926:], 0x2711)
	binary.LittleEndian.PutUint16(s.b[0x92c:], uint16(s.width))
	binary.LittleEndian.PutUint16(s.b[0x92e:], uint16(s.height))
	for k := 0; k < 7; k++ {
		binary.LittleEndian.PutUint16(s.b[0x38+2*k:], 0xffff)
		binary.LittleEndian.PutUint16(s.b[0x46+2*k:], 0xffff)
		if k < players {
			s.b[0x55+k] = byte(k)
		}
	}
	for i := 0; i < s.size; i++ {
		s.layer(0)[i] = 1
		s.layer(1)[i] = 8
		s.layer(2)[i] = 9
		s.layer(8)[i] = 4
		s.layer(9)[i] = 7
		s.layer(12)[i] = 64
	}
}
