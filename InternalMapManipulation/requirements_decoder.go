package internalmapmanipulation

import (
	"encoding/json"
	"fmt"
	domain "settlers2-final-go/DomainModel"
)

func decodeMapRequirements(payload []byte) (MapRequirements, error) {
	// The legacy flat JSON contract is kept unchanged for the browser's
	// saved projects, REST endpoints and command-line mode.
	var wire domain.LegacyMapRequest
	if err := json.Unmarshal(payload, &wire); err != nil {
		return MapRequirements{}, fmt.Errorf("invalid map requirements: %w", err)
	}
	defaultSize := defaultDiameter
	if wire.Sym == 2 || wire.Sym == 4 {
		defaultSize = 108
	}
	req := MapRequirements{MapSize: defaultSize, PlayableDiameter: defaultSize, MapSymmetry: wire.Sym, Seed: wire.Seed, WoodsFillInRate: wire.Forest, MountainRate: wire.Mountain, LandLevel: 1, Variant: wire.Variant, VariantTypes: wire.VariantTypes}
	if wire.MapSize != nil {
		req.MapSize = *wire.MapSize
	}
	if wire.Diameter != nil {
		req.PlayableDiameter = *wire.Diameter
		req.MapSize = *wire.Diameter
	}
	// Rotation is always enabled for five players. Legacy rotateResourceMasks input is ignored.
	req.StartRadiusPercent = 55
	req.MiningTerrainExpansion = 3
	if wire.StartRadiusPercent != nil {
		req.StartRadiusPercent = *wire.StartRadiusPercent
	}
	if wire.MiningTerrainExpansion != nil {
		req.MiningTerrainExpansion = *wire.MiningTerrainExpansion
	}
	if wire.LandLevel != nil {
		req.LandLevel = *wire.LandLevel
	}
	totals := []int{150, 50, 20, 40, 30}
	groups := []int{3, 2, 1, 2, 2}
	if wire.Totals != nil {
		if len(wire.Totals) != 5 {
			return req, fmt.Errorf("totals must contain five entries")
		}
		copy(totals, wire.Totals)
	}
	if wire.Groups != nil {
		if len(wire.Groups) != 5 {
			return req, fmt.Errorf("groups must contain five entries")
		}
		copy(groups, wire.Groups)
	}
	if wire.SurfaceStoneGroups != nil {
		groups[4] = *wire.SurfaceStoneGroups
	}
	requirements := []*ResourceRequirements{&req.CoalRequirements, &req.IronRequirements, &req.GoldRequirements, &req.GraniteRequirements, &req.SurfaceStoneRequirements}
	for i, r := range requirements {
		*r = ResourceRequirements{TilesPerUser: totals[i], ResourceRegionsPerUser: groups[i], MinTilesPerRegion: 1}
	}
	lakes, lakeSize := 3, 25
	if wire.Lakes != nil {
		lakes = *wire.Lakes
	}
	if wire.LakeSize != nil {
		lakeSize = *wire.LakeSize
	}
	req.LakeRequirements = ResourceRequirements{TilesPerUser: lakes * lakeSize, ResourceRegionsPerUser: lakes, MinTilesPerRegion: lakeSize}
	forestTiles, forestGroups := 100, 4
	if wire.ForestTiles != nil {
		forestTiles = *wire.ForestTiles
	}
	if wire.ForestGroups != nil {
		forestGroups = *wire.ForestGroups
	}
	req.ForestRequirements = ResourceRequirements{TilesPerUser: forestTiles, ResourceRegionsPerUser: forestGroups, MinTilesPerRegion: 1}
	emptyTiles, emptyGroups := 0, 0
	if wire.EmptyMountainTiles != nil {
		emptyTiles = *wire.EmptyMountainTiles
	}
	if wire.EmptyMountainGroups != nil {
		emptyGroups = *wire.EmptyMountainGroups
	}
	req.EmptyMountainRequirements = ResourceRequirements{TilesPerUser: emptyTiles, ResourceRegionsPerUser: emptyGroups, MinTilesPerRegion: 1}
	req.AnimalRequirements = ResourceRequirements{TilesPerUser: wire.Animals, ResourceRegionsPerUser: wire.AnimalGroups, MinTilesPerRegion: 1}
	if err := req.Validate(); err != nil {
		return req, err
	}
	return req, nil
}
