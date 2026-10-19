package internalmapmanipulation

const defaultDiameter = 109

// Six axial neighbours of an SWD hexagonal grid.
var hexNeighbors = [6][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, 1}, {1, -1}}
