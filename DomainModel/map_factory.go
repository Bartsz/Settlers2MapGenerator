package domainmodel

import (
	"fmt"
	"math"
)

// NewMapData constructs only a logical map, without any SWD header or blocks.
// The binary serializer will allocate native layers when ToSWD is requested.
func NewMapData(width, height, players int) (*MapData, error) {
	if width < 3 || height < 3 || width > 2048 || height > 2048 {
		return nil, fmt.Errorf("invalid map size %dx%d", width, height)
	}
	if players < 1 || players > 7 {
		return nil, fmt.Errorf("expected 1..7 players")
	}
	m := &MapData{Width: width, Height: height, Heights: grid[byte](width, height),
		Textures: grid[TextureType](width, height), TextureTriangles: grid[TextureTriangles](width, height),
		Resources: grid[ResourceType](width, height), ResourceAmounts: grid[byte](width, height),
		Animals: grid[AnimalType](width, height), Original: grid[OriginalCell](width, height),
		Starts: make([]Point, players), OriginalStarts: make([]Point, players)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			m.Heights[y][x] = 1
			m.Textures[y][x] = TextureMeadow
			m.TextureTriangles[y][x] = TextureTriangles{8, 9}
			m.Original[y][x] = OriginalCell{Texture: TextureMeadow, Resource: ResourceNone, Animal: AnimalNone}
		}
	}
	taken := map[Point]bool{}
	for p := 0; p < players; p++ {
		angle := 2 * math.Pi * float64(p) / float64(players)
		x := int(math.Round(float64(width-1)/2 + float64(width)/3*math.Cos(angle)))
		y := int(math.Round(float64(height-1)/2 + float64(height)/3*math.Sin(angle)))
		x = max(1, min(width-2, x))
		y = max(1, min(height-2, y))
		loc := Point{x, y}
		if taken[loc] {
			return nil, fmt.Errorf("map too small for distinct HQ positions")
		}
		taken[loc] = true
		m.Starts[p] = loc
		m.OriginalStarts[p] = Point{-1, -1}
	}
	return m, nil
}
