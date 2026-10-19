package internalmapmanipulation

import domain "settlers2-final-go/DomainModel"

type Point = domain.Point
type TextureType = domain.TextureType
type TextureTriangles = domain.TextureTriangles
type ResourceType = domain.ResourceType
type AnimalType = domain.AnimalType
type MapData = domain.MapData
type MapRequirements = domain.MapRequirements
type ResourceRequirements = domain.ResourceRequirements

const (
	TextureMeadow            = domain.TextureMeadow
	TextureMountain          = domain.TextureMountain
	TextureRocky             = domain.TextureRocky
	TextureWater             = domain.TextureWater
	TextureMixed             = domain.TextureMixed
	TextureOther             = domain.TextureOther
	ResourceNone             = domain.ResourceNone
	ResourceCoal             = domain.ResourceCoal
	ResourceIron             = domain.ResourceIron
	ResourceGold             = domain.ResourceGold
	ResourceGranite          = domain.ResourceGranite
	ResourceFish             = domain.ResourceFish
	ResourceUndergroundWater = domain.ResourceUndergroundWater
	ResourceSurfaceStone1    = domain.ResourceSurfaceStone1
	ResourceSurfaceStone2    = domain.ResourceSurfaceStone2
	ResourceSurfaceStone3    = domain.ResourceSurfaceStone3
	ResourceSurfaceStone4    = domain.ResourceSurfaceStone4
	ResourceSurfaceStone5    = domain.ResourceSurfaceStone5
	ResourceSurfaceStone6    = domain.ResourceSurfaceStone6
	ResourceTree1            = domain.ResourceTree1
	ResourceTree2            = domain.ResourceTree2
	ResourceTree3            = domain.ResourceTree3
	ResourceTree4            = domain.ResourceTree4
	ResourceTree5            = domain.ResourceTree5
	ResourceTree6            = domain.ResourceTree6
	ResourceTree7            = domain.ResourceTree7
	ResourceTree8            = domain.ResourceTree8
	AnimalNone               = domain.AnimalNone
	AnimalRabbit             = domain.AnimalRabbit
	AnimalFox                = domain.AnimalFox
	AnimalStag               = domain.AnimalStag
	AnimalDeer               = domain.AnimalDeer
	AnimalDuck               = domain.AnimalDuck
	AnimalSheep              = domain.AnimalSheep
	AnimalDeerAlternate      = domain.AnimalDeerAlternate
	AnimalDuckAlternate      = domain.AnimalDuckAlternate
	AnimalPackDonkey         = domain.AnimalPackDonkey
)

func grid[T any](width, height int) [][]T {
	out := make([][]T, height)
	for i := range out {
		out[i] = make([]T, width)
	}
	return out
}

type compactDescriptor = domain.CompactDescriptor
type symmetryStats = domain.SymmetryStats
type Snapshot = domain.MapSnapshot
