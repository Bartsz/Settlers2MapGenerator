package domainmodel

import "fmt"

// Point uses the native map's Cartesian (x, y) coordinate space.
type Point struct{ X, Y int }

// TextureType classifies an entire cell for terrain design. SWD encodes two
// separate triangles per cell; TextureTriangles retains their exact values.
type TextureType uint8

const (
	TextureMeadow TextureType = iota
	TextureMountain
	TextureRocky // buildable road/desert texture for surface stones
	TextureWater
	TextureMixed
	TextureOther
)

type TextureTriangles struct{ Upper, Lower byte }

type ResourceType uint8

const (
	ResourceNone ResourceType = iota
	ResourceCoal
	ResourceIron
	ResourceGold
	ResourceGranite
	ResourceFish
	ResourceUndergroundWater
	ResourceSurfaceStone1
	ResourceSurfaceStone2
	ResourceSurfaceStone3
	ResourceSurfaceStone4
	ResourceSurfaceStone5
	ResourceSurfaceStone6
	ResourceTree1
	ResourceTree2
	ResourceTree3
	ResourceTree4
	ResourceTree5
	ResourceTree6
	ResourceTree7
	ResourceTree8
)

type AnimalType byte

const (
	AnimalNone          AnimalType = 0
	AnimalRabbit        AnimalType = 1
	AnimalFox           AnimalType = 2
	AnimalStag          AnimalType = 3
	AnimalDeer          AnimalType = 4
	AnimalDuck          AnimalType = 5
	AnimalSheep         AnimalType = 6
	AnimalDeerAlternate AnimalType = 7
	AnimalDuckAlternate AnimalType = 8
	AnimalPackDonkey    AnimalType = 9
)

type OriginalCell struct {
	Texture  TextureType
	Resource ResourceType
	Animal   AnimalType
	Strength byte
}

// MapData is the editable domain model. [y][x] is used consistently, and SWD
// byte codes are contained in swd_codec.go and the adapters below.
// Unmodified metadata, textures, objects and extra SWD layers are retained
// losslessly so editing one property cannot destroy unrelated map information.
type MapData struct {
	Width, Height    int
	Heights          [][]byte
	Textures         [][]TextureType
	TextureTriangles [][]TextureTriangles
	Resources        [][]ResourceType
	ResourceAmounts  [][]byte
	Animals          [][]AnimalType
	Starts           []Point

	RawHeader      []byte
	RawLayers      [14][]byte
	Original       [][]OriginalCell
	OriginalStarts []Point
}

func grid[T any](width, height int) [][]T {
	result := make([][]T, height)
	for y := range result {
		result[y] = make([]T, width)
	}
	return result
}

func (m *MapData) CheckCoordinates(x, y int) error {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return fmt.Errorf("coordinate (%d,%d) outside %dx%d map", x, y, m.Width, m.Height)
	}
	return nil
}

func (m *MapData) SetTexture(x, y int, t TextureType) error {
	if err := m.CheckCoordinates(x, y); err != nil {
		return err
	}
	if t == TextureMixed || t == TextureOther {
		return fmt.Errorf("texture %d requires explicit triangle values", t)
	}
	m.Textures[y][x] = t
	if t == TextureWater {
		m.Heights[y][x] = 0
	} else if m.Heights[y][x] == 0 {
		m.Heights[y][x] = 1
	}
	return nil
}

func (m *MapData) SetResource(x, y int, r ResourceType) error {
	if err := m.CheckCoordinates(x, y); err != nil {
		return err
	}
	if r > ResourceTree8 {
		return fmt.Errorf("unknown resource %d", r)
	}
	m.Resources[y][x] = r
	if r >= ResourceCoal && r <= ResourceGranite {
		m.ResourceAmounts[y][x] = 7
	}
	if r >= ResourceSurfaceStone1 && r <= ResourceSurfaceStone6 {
		m.ResourceAmounts[y][x] = byte(r - ResourceSurfaceStone1 + 1)
	}
	return nil
}

// MergeResource applies an odd-sized, centre-anchored shape atomically. Existing
// resources are protected from accidental replacement; false cells are ignored.
func (m *MapData) MergeResource(r ResourceType, mask [][]bool, origin Point) error {
	if r == ResourceNone || r > ResourceTree8 {
		return fmt.Errorf("invalid resource shape type %d", r)
	}
	if len(mask) == 0 || len(mask)%2 == 0 || len(mask[0]) != len(mask) {
		return fmt.Errorf("resource shape must be an odd square")
	}
	for _, row := range mask {
		if len(row) != len(mask) {
			return fmt.Errorf("ragged resource shape")
		}
	}
	radius := len(mask) / 2
	var positions []Point
	for y, row := range mask {
		for x, on := range row {
			if !on {
				continue
			}
			pos := Point{origin.X + x - radius, origin.Y + y - radius}
			if err := m.CheckCoordinates(pos.X, pos.Y); err != nil {
				return err
			}
			if m.Resources[pos.Y][pos.X] != ResourceNone && m.Resources[pos.Y][pos.X] != ResourceUndergroundWater {
				return fmt.Errorf("resource collision at (%d,%d)", pos.X, pos.Y)
			}
			isWater := m.Textures[pos.Y][pos.X] == TextureWater
			if (r == ResourceFish) != isWater {
				return fmt.Errorf("resource %d incompatible with water at (%d,%d)", r, pos.X, pos.Y)
			}
			if r >= ResourceCoal && r <= ResourceGranite && m.Textures[pos.Y][pos.X] != TextureMountain {
				return fmt.Errorf("mining resource needs mountain at (%d,%d)", pos.X, pos.Y)
			}
			i := pos.Y*m.Width + pos.X
			if m.Animals[pos.Y][pos.X] != AnimalNone || len(m.RawLayers[5]) > i && m.RawLayers[5][i] != 0 {
				return fmt.Errorf("occupied resource tile (%d,%d)", pos.X, pos.Y)
			}
			if r >= ResourceCoal && r <= ResourceGranite {
				for _, hq := range m.Starts {
					if hexDistance(pos.X-hq.X, pos.Y-hq.Y) < 4 {
						return fmt.Errorf("mine too close to headquarters (%d,%d)", pos.X, pos.Y)
					}
				}
			}
			positions = append(positions, pos)
		}
	}
	for _, p := range positions {
		if err := m.SetResource(p.X, p.Y, r); err != nil {
			return err
		}
	}
	return nil
}

func hexDistance(dx, dy int) int {
	ab := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}
	a, b, c := ab(dx), ab(dy), ab(dx+dy)
	if a < b {
		a = b
	}
	if a < c {
		a = c
	}
	return a
}
