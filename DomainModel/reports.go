package domainmodel

// CompactDescriptor identifies a persistent mineral footprint. The capacity
// and centre remain stable when variants replace only the resource content.
type CompactDescriptor struct {
	Type     int   `json:"type"`
	Capacity int   `json:"capacity"`
	Count    int   `json:"count"`
	Center   []int `json:"center"`
}

// SymmetryStats are measured from the resulting map, not a proposed layout.
type SymmetryStats struct {
	Players           int   `json:"players"`
	PlayablePerPlayer int   `json:"playablePerPlayer"`
	Lakes             []int `json:"lakes"`
	Water             []int `json:"water"`
	Coal              []int `json:"coal"`
	Iron              []int `json:"iron"`
	Gold              []int `json:"gold"`
	Granite           []int `json:"granite"`
	Rocks             []int `json:"rocks"`
	Forests           []int `json:"forests"`
	Fish              []int `json:"fish"`
	Animals           []int `json:"animals"`
	TemplateSource    int   `json:"templateSourceSector"`
}

// MapSnapshot contains the preview and final export generated from one MapData.
type MapSnapshot struct {
	Preview []byte
	SWD     []byte
}

// PlacementKind identifies which logical layer receives a rotated footprint.
type PlacementKind uint8

const (
	PlaceLake PlacementKind = iota
	PlaceMine
	PlaceStone
	PlaceForest
	PlaceAnimals
	PlaceEmptyMountain
)

// ShapePlacement is a stored template footprint, used for sector replication.
type ShapePlacement struct {
	Shape        [][]bool
	TerrainShape [][]bool // distinct resource-free halo for mineral footprints
	Center       Point
	Kind         PlacementKind
	Resource     ResourceType
	Count        int
}
