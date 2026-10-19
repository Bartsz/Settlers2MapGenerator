package internalmapmanipulation

import (
	"fmt"
	domain "settlers2-final-go/DomainModel"
)

// embedMask is the sole iteration/copy primitive for writing shape cells.
// It validates the full footprint before making changes (atomic collision semantics).
func embedMask(m *domain.MapData, mask Shape, origin Point, allowed func(int, int) bool, write func(int, int) error) error {
	if err := ValidatePlacement(m, mask, origin, allowed); err != nil {
		return err
	}
	pts, _ := ShapePoints(mask, origin)
	for _, p := range pts {
		if err := write(p.X, p.Y); err != nil {
			return err
		}
	}
	return nil
}

func EmbedTexture(m *domain.MapData, shape Shape, origin Point, t domain.TextureType) error {
	return embedMask(m, shape, origin, nil, func(x, y int) error { return m.SetTexture(x, y, t) })
}
func EmbedResource(m *domain.MapData, shape Shape, origin Point, t domain.ResourceType) error {
	return m.MergeResource(t, shape, origin)
}
func EmbedLake(m *domain.MapData, shape Shape, origin Point) error {
	return embedMask(m, shape, origin, func(x, y int) bool {
		return m.Textures[y][x] == domain.TextureMeadow && m.Resources[y][x] == domain.ResourceNone && m.Animals[y][x] == domain.AnimalNone
	}, func(x, y int) error { return m.SetTexture(x, y, domain.TextureWater) })
}
func EmbedForest(m *domain.MapData, shape Shape, origin Point) error {
	return embedMask(m, shape, origin, func(x, y int) bool {
		return m.Textures[y][x] == domain.TextureMeadow && m.Resources[y][x] == domain.ResourceNone
	}, func(x, y int) error {
		return m.SetResource(x, y, domain.ResourceTree1+domain.ResourceType((x*3+y*11)%8))
	})
}
func EmbedAnimals(m *domain.MapData, shape Shape, origin Point) error {
	return embedMask(m, shape, origin, func(x, y int) bool {
		return m.Textures[y][x] == domain.TextureMeadow && m.Animals[y][x] == domain.AnimalNone && m.Resources[y][x] >= domain.ResourceTree1 && m.Resources[y][x] <= domain.ResourceTree8
	}, func(x, y int) error { m.Animals[y][x] = domain.AnimalRabbit; return nil })
}
func EmbedSurfaceStones(m *domain.MapData, shape Shape, origin Point) error {
	return embedMask(m, shape, origin, func(x, y int) bool {
		return m.Textures[y][x] == domain.TextureRocky && m.Resources[y][x] == domain.ResourceNone && m.Animals[y][x] == domain.AnimalNone
	}, func(x, y int) error { return m.SetResource(x, y, domain.ResourceSurfaceStone6) })
}
func EnsureMineTerrain(m *domain.MapData, shape Shape, origin Point) error {
	expanded, err := ExpandShape(shape, 1)
	if err != nil {
		return err
	}
	if err := ValidatePlacement(m, expanded, origin, func(x, y int) bool {
		return m.Textures[y][x] == domain.TextureMeadow && m.Resources[y][x] == domain.ResourceNone && m.Animals[y][x] == domain.AnimalNone
	}); err != nil {
		return fmt.Errorf("mine terrain: %w", err)
	}
	return EmbedTexture(m, expanded, origin, domain.TextureMountain)
}
