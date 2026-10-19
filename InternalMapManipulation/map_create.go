package internalmapmanipulation

import domain "settlers2-final-go/DomainModel"

// NewMapData is an alias for the domain-only constructor.
func NewMapData(width, height, players int) (*MapData, error) {
	return domain.NewMapData(width, height, players)
}
