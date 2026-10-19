package domainmodel

import "fmt"

const (
	DefaultDiameter = 109
	MinDiameter     = 64
	MaxDiameter     = 140
)

// ResourceRequirements describe quantities and compact groups PER player,
// never quantities for the entire map.
type ResourceRequirements struct {
	TilesPerUser           int
	ResourceRegionsPerUser int
	MinTilesPerRegion      int
}

type MapRequirements struct {
	MapSize                   int // real physical SWD width and height; matches PlayableDiameter
	PlayableDiameter          int
	MapSymmetry               int // number of players, supported: 2..6
	StartRadiusPercent        int // player HQ radius relative to map radius (10..90)
	MiningTerrainExpansion    int // full hex rings of mountain texture beyond ore, 0..8
	Seed                      int64
	CoalRequirements          ResourceRequirements
	IronRequirements          ResourceRequirements
	GoldRequirements          ResourceRequirements
	GraniteRequirements       ResourceRequirements
	SurfaceStoneRequirements  ResourceRequirements
	EmptyMountainRequirements ResourceRequirements
	LakeRequirements          ResourceRequirements // TilesPerUser = number*lakeSize; regions = lake count
	ForestRequirements        ResourceRequirements
	AnimalRequirements        ResourceRequirements
	WoodsFillInRate           int
	LandLevel                 int
	MountainRate              int
	Variant                   []int
	VariantTypes              []int
}

func (req MapRequirements) Validate() error {
	if req.MapSize != req.PlayableDiameter {
		return fmt.Errorf("physical SWD map size must equal selected map diameter")
	}
	if req.MapSymmetry < 2 || req.MapSymmetry > 6 {
		return fmt.Errorf("number of players must be 2 to 6")
	}
	if req.PlayableDiameter < MinDiameter || req.PlayableDiameter > MaxDiameter {
		return fmt.Errorf("map size must be %d..%d", MinDiameter, MaxDiameter)
	}
	if req.MapSymmetry <= 4 && req.MapSymmetry != 3 && req.PlayableDiameter%2 != 0 {
		return fmt.Errorf("two/four-player maps must have even dimensions")
	}
	if (req.MapSymmetry == 3 || req.MapSymmetry == 5 || req.MapSymmetry == 6) && req.PlayableDiameter%2 != 1 {
		return fmt.Errorf("three/five/six-player maps must have odd dimensions")
	}
	if req.StartRadiusPercent < 10 || req.StartRadiusPercent > 90 {
		return fmt.Errorf("start radius must be 10..90 percent")
	}
	if req.MiningTerrainExpansion < 0 || req.MiningTerrainExpansion > 8 {
		return fmt.Errorf("mining terrain expansion must be 0..8")
	}
	if req.LandLevel != 1 && req.LandLevel != 40 && req.LandLevel != 100 {
		return fmt.Errorf("land level must be 1, 40, or 100")
	}
	if req.WoodsFillInRate < 0 || req.WoodsFillInRate > 85 {
		return fmt.Errorf("forest fill rate must be 0..85")
	}
	rs := []struct {
		name string
		r    ResourceRequirements
	}{
		{"coal", req.CoalRequirements}, {"iron", req.IronRequirements}, {"gold", req.GoldRequirements}, {"granite", req.GraniteRequirements}, {"surface stone", req.SurfaceStoneRequirements}, {"empty mountains", req.EmptyMountainRequirements}, {"forest", req.ForestRequirements}, {"animals", req.AnimalRequirements},
	}
	for _, entry := range rs {
		r := entry.r
		if r.TilesPerUser < 0 || r.TilesPerUser > 1000 {
			return fmt.Errorf("%s: tiles per player must be 0..1000", entry.name)
		}
		if r.ResourceRegionsPerUser < 0 || r.ResourceRegionsPerUser > 40 {
			return fmt.Errorf("%s: regions per player must be 0..40", entry.name)
		}
	}
	if req.ForestRequirements.TilesPerUser > 0 && req.ForestRequirements.ResourceRegionsPerUser == 0 {
		return fmt.Errorf("forest tiles require at least one cluster")
	}
	if req.LakeRequirements.ResourceRegionsPerUser < 0 || req.LakeRequirements.ResourceRegionsPerUser > 10 || req.LakeRequirements.MinTilesPerRegion < 1 || req.LakeRequirements.MinTilesPerRegion > 120 {
		return fmt.Errorf("lakes must have 0..10 regions of 1..120 tiles")
	}
	for _, r := range rs {
		if r.r.TilesPerUser > 0 && r.r.ResourceRegionsPerUser == 0 {
			return fmt.Errorf("%s tiles require at least one cluster", r.name)
		}
		if r.r.TilesPerUser > 0 && r.r.TilesPerUser < r.r.ResourceRegionsPerUser {
			return fmt.Errorf("%s has more clusters than tiles", r.name)
		}
	}
	if req.LakeRequirements.ResourceRegionsPerUser > 0 && req.LakeRequirements.TilesPerUser != req.LakeRequirements.ResourceRegionsPerUser*req.LakeRequirements.MinTilesPerRegion {
		return fmt.Errorf("inconsistent lake tile count")
	}
	if req.Variant != nil && len(req.VariantTypes) != len(req.Variant) {
		return fmt.Errorf("variant and variantTypes sizes differ")
	}
	for _, kind := range req.VariantTypes {
		if kind < 0 || kind > 4 {
			return fmt.Errorf("unknown variant mineral type %d", kind)
		}
	}
	for _, qty := range req.Variant {
		if qty < 0 || qty > 1000 {
			return fmt.Errorf("variant count outside 0..1000")
		}
	}
	return nil
}

func (req MapRequirements) Totals() []int {
	return []int{req.CoalRequirements.TilesPerUser, req.IronRequirements.TilesPerUser, req.GoldRequirements.TilesPerUser, req.GraniteRequirements.TilesPerUser, req.SurfaceStoneRequirements.TilesPerUser}
}

// LegacyMapRequest is the public JSON DTO for existing projects and UI clients.
// The REST contract is kept independent of the validated MapRequirements.
type LegacyMapRequest struct {
	Seed                   int64 `json:"seed"`
	StartRadiusPercent     *int  `json:"startRadiusPercent"`
	MiningTerrainExpansion *int  `json:"miningTerrainExpansion"`
	Sym                    int   `json:"sym"`
	Diameter               *int  `json:"diameter"`
	MapSize                *int  `json:"mapSize"`
	Totals                 []int `json:"totals"`
	Groups                 []int `json:"groups"`
	SurfaceStoneGroups     *int  `json:"surfaceStoneGroups"`
	EmptyMountainTiles     *int  `json:"emptyMountainTiles"`
	EmptyMountainGroups    *int  `json:"emptyMountainGroups"`
	Lakes                  *int  `json:"lakes"`
	LakeSize               *int  `json:"lakeSize"`
	Animals                int   `json:"animals"`
	AnimalGroups           int   `json:"animalGroups"`
	Forest                 int   `json:"forest"`
	ForestTiles            *int  `json:"forestTiles"`
	ForestGroups           *int  `json:"forestGroups"`
	Mountain               int   `json:"mountain"`
	LandLevel              *int  `json:"landLevel"`
	Variant                []int `json:"variant"`
	VariantTypes           []int `json:"variantTypes"`
}

// SaveAndRunRequest supplies filename-safe metadata, not a target path.
type SaveAndRunRequest struct {
	Seed int64 `json:"seed"`
	Sym  int   `json:"sym"`
}
