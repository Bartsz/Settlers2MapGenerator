package internalmapmanipulation

// MineralTerrainMask expands the ore capacity mask by exactly the requested
// number of hex-neighbour rings. Zero gives no additional mining ground.
// The terrain uses full cluster capacity, independently of the active variant.
func MineralTerrainMask(resource Shape, rings, players int) (Shape, error) {
	return ExpandShapeForPlayers(resource, rings, players)
}
