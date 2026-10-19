package internalmapmanipulation

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"

	domain "settlers2-final-go/DomainModel"
)

// UnifiedGenerate is a linear, domain-first pipeline. No SWD binary data is
// written until geometry, all resource masks and symmetry are finalized.
// UnifiedGenerate retries crowded but feasible configurations with a stable
// seed-derived sequence. All attempts use the same requested capacities: no
// resources are silently removed and a given seed always yields identical SWD.
func UnifiedGenerate(req MapRequirements) (Snapshot, error) {
	if err := req.Validate(); err != nil {
		return Snapshot{}, err
	}
	const attempts = 36
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		candidate := req
		if attempt > 0 {
			v := rand.New(rand.NewSource(req.Seed))
			for j := 0; j < attempt; j++ {
				candidate.Seed = v.Int63()
			}
		}
		result, err := unifiedGenerateOnce(candidate)
		if err == nil {
			return result, nil
		}
		last = err
	}
	return Snapshot{}, fmt.Errorf("could not place map features after %d attempts: %w", attempts, last)
}

func unifiedGenerateOnce(req MapRequirements) (Snapshot, error) {
	if err := req.Validate(); err != nil {
		return Snapshot{}, err
	}
	m, err := NewMapData(req.MapSize, req.MapSize, req.MapSymmetry)
	if err != nil {
		return Snapshot{}, err
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			m.Heights[y][x] = byte(req.LandLevel)
			if !playable(x, y, req.PlayableDiameter, req.MapSymmetry) {
				m.Textures[y][x] = domain.TextureMountain
			}
		}
	}
	rng := rand.New(rand.NewSource(req.Seed))
	origin := chooseOrigin(req, rng)
	m.Starts = make([]Point, req.MapSymmetry)
	for k := 0; k < req.MapSymmetry; k++ {
		p, e := RotatePointSized(origin, k, req.MapSymmetry, req.MapSize)
		if e != nil {
			return Snapshot{}, e
		}
		m.Starts[k] = p
	}
	// Capacity reservations are independent of the currently selected mineral
	// variant. Switching ore type or count must not move future terrain shapes.
	reserved := make([][]bool, m.Height)
	for y := range reserved {
		reserved[y] = make([]bool, m.Width)
	}
	// A neutral low-land clear zone prevents forests, lakes and minerals near HQ.
	clearZone := func(x, y int) bool { return hexDistance(x-origin.X, y-origin.Y) <= 3 }
	available := func(x, y int, preferTrees bool) bool {
		if !playable(x, y, req.PlayableDiameter, req.MapSymmetry) || !inMasterSectorSized(Point{X: x, Y: y}, req.MapSymmetry, req.MapSize) || clearZone(x, y) {
			return false
		}
		if m.Textures[y][x] != domain.TextureMeadow || m.Animals[y][x] != domain.AnimalNone {
			return false
		}
		r := m.Resources[y][x]
		if reserved[y][x] {
			return false
		}
		if preferTrees {
			return r >= domain.ResourceTree1 && r <= domain.ResourceTree8
		}
		return r == domain.ResourceNone
	}
	placements := make([]domain.ShapePlacement, 0)
	scatteredTrees := make([]Point, 0)
	var descriptors []compactDescriptor
	// 1. Lakes first: shoreline is a deterministic texture envelope, fish belong to water.
	lakeN, lakeSize := req.LakeRequirements.ResourceRegionsPerUser, req.LakeRequirements.MinTilesPerRegion
	for j := 0; j < lakeN; j++ {
		shape, e := GenerateShapeForPlayers(lakeSize, rng.Int63(), req.MapSymmetry)
		if e != nil {
			return Snapshot{}, e
		}
		at, e := choosePlacement(req, rng, shape, func(x, y int) bool {
			// Protect a one-cell land buffer along square reflection axes.
			// Otherwise a mirrored lake can merge with itself across the seam.
			if (req.MapSymmetry == 2 && x <= req.MapSize/2) || (req.MapSymmetry == 4 && (x <= req.MapSize/2 || y <= req.MapSize/2)) {
				return false
			}
			if !available(x, y, false) {
				return false
			}
			// Square mirrors invert hex diagonal adjacency. Protect all eight
			// neighbours so distinct lakes stay distinct after reflection.
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if req.MapSymmetry != 2 && req.MapSymmetry != 4 && dx*dy == 1 {
						continue
					}
					xx, yy := x+dx, y+dy
					if xx >= 0 && yy >= 0 && xx < m.Width && yy < m.Height && m.Textures[yy][xx] == domain.TextureWater {
						return false
					}
				}
			}
			return true
		})
		if e != nil {
			return Snapshot{}, fmt.Errorf("jezioro %d: %w", j+1, e)
		}
		if err := EmbedLake(m, shape, at); err != nil {
			return Snapshot{}, err
		}
		placements = append(placements, domain.ShapePlacement{Shape: shape, Center: at, Kind: domain.PlaceLake, Count: lakeSize})
		// The shore is left as buildable meadow; water and its edge remain flat.
		// Fish appear only inside lake masks, at reproducible positions.
		pts, _ := ShapePoints(shape, at)
		for idx, p := range pts {
			if idx%4 == 0 {
				if e := m.SetResource(p.X, p.Y, domain.ResourceFish); e != nil {
					return Snapshot{}, e
				}
				m.ResourceAmounts[p.Y][p.X] = 7
			}
		}
	}
	// 2. Four mineral families share ONE compact shape generator.
	types := []domain.ResourceType{domain.ResourceCoal, domain.ResourceIron, domain.ResourceGold, domain.ResourceGranite}
	requirements := []ResourceRequirements{req.CoalRequirements, req.IronRequirements, req.GoldRequirements, req.GraniteRequirements, req.SurfaceStoneRequirements}
	for kind, rule := range requirements {
		if rule.TilesPerUser == 0 {
			continue
		}
		for j := 0; j < rule.ResourceRegionsPerUser; j++ {
			count := rule.TilesPerUser / rule.ResourceRegionsPerUser
			if j < rule.TilesPerUser%rule.ResourceRegionsPerUser {
				count++
			}
			shape, e := GenerateShapeForPlayers(count, rng.Int63(), req.MapSymmetry)
			if e != nil {
				return Snapshot{}, e
			}
			footprint := shape
			if kind != 4 {
				footprint, e = MineralTerrainMask(shape, req.MiningTerrainExpansion, req.MapSymmetry)
				if e != nil {
					return Snapshot{}, e
				}
			}
			var at Point
			if kind != 4 {
				at, e = choosePlacement(req, rng, footprint, func(x, y int) bool {
					return playable(x, y, req.PlayableDiameter, req.MapSymmetry) &&
						inMasterSectorSized(Point{X: x, Y: y}, req.MapSymmetry, req.MapSize) &&
						!clearZone(x, y) && (m.Textures[y][x] == domain.TextureMeadow || m.Textures[y][x] == domain.TextureMountain) &&
						m.Animals[y][x] == domain.AnimalNone &&
						hexDistance(x-origin.X, y-origin.Y) >= 4
				}, placementChoice{accept: func(at Point) bool {
					pts, _ := ShapePoints(shape, at)
					for _, p := range pts {
						if m.Resources[p.Y][p.X] != domain.ResourceNone || reserved[p.Y][p.X] {
							return false
						}
					}
					return true
				}})
			} else {
				at, e = choosePlacement(req, rng, footprint, func(x, y int) bool { return available(x, y, false) && hexDistance(x-origin.X, y-origin.Y) >= 4 })
			}
			if e != nil {
				return Snapshot{}, fmt.Errorf("surowiec %d, skupisko %d: %w", kind+1, j+1, e)
			}
			desc := compactDescriptor{Type: kind, Capacity: count, Count: count, Center: []int{at.X, at.Y}}
			variantIndex := len(descriptors)
			if len(req.Variant) > 0 {
				if variantIndex >= len(req.Variant) {
					return Snapshot{}, fmt.Errorf("variant has %d regions, expected at least %d", len(req.Variant), variantIndex+1)
				}
				desc.Count = req.Variant[variantIndex]
				desc.Type = req.VariantTypes[variantIndex]
				if desc.Count > desc.Capacity {
					return Snapshot{}, fmt.Errorf("wariant %d exceeds capacity", variantIndex+1)
				}
				if (kind == 4) != (desc.Type == 4) {
					return Snapshot{}, fmt.Errorf("surface stones cannot become underground minerals")
				}
			}
			if kind != 4 {
				if e := EmbedTexture(m, footprint, at, domain.TextureMountain); e != nil {
					return Snapshot{}, e
				}
			} else {
				if e := EmbedTexture(m, footprint, at, domain.TextureRocky); e != nil {
					return Snapshot{}, e
				}
			}
			if desc.Count > 0 {
				active, e := ConnectedPrefixForPlayers(shape, desc.Count, req.MapSymmetry)
				if e != nil {
					return Snapshot{}, e
				}
				if kind == 4 {
					e = EmbedSurfaceStones(m, active, at)
				} else {
					e = EmbedResource(m, active, at, types[desc.Type])
				}
				if e != nil {
					return Snapshot{}, fmt.Errorf("deposit %d: %w", variantIndex+1, e)
				}
			}
			placementKind := domain.PlaceMine
			resource := types[0]
			if kind == 4 {
				placementKind = domain.PlaceStone
				resource = domain.ResourceSurfaceStone6
			} else {
				resource = types[desc.Type]
			}
			// Reserve the FULL capacity shape even if this variant uses fewer cells.
			allCapacityPoints, _ := ShapePoints(shape, at)
			for _, p := range allCapacityPoints {
				reserved[p.Y][p.X] = true
			}
			placements = append(placements, domain.ShapePlacement{Shape: shape, TerrainShape: footprint, Center: at, Kind: placementKind, Resource: resource, Count: desc.Count})
			descriptors = append(descriptors, desc)
		}
	}
	if len(req.Variant) > 0 && len(req.Variant) != len(descriptors) {
		return Snapshot{}, fmt.Errorf("variant length does not match clusters (%d vs %d)", len(req.Variant), len(descriptors))
	}
	// Empty mining hills are separate connected terrain clusters without ore.
	if rule := req.EmptyMountainRequirements; rule.TilesPerUser > 0 {
		for j := 0; j < rule.ResourceRegionsPerUser; j++ {
			count := rule.TilesPerUser / rule.ResourceRegionsPerUser
			if j < rule.TilesPerUser%rule.ResourceRegionsPerUser {
				count++
			}
			shape, e := GenerateShapeForPlayers(count, rng.Int63(), req.MapSymmetry)
			if e != nil {
				return Snapshot{}, e
			}
			at, e := choosePlacement(req, rng, shape, func(x, y int) bool { return available(x, y, false) })
			if e != nil {
				return Snapshot{}, fmt.Errorf("empty mountain cluster %d: %w", j+1, e)
			}
			if e := EmbedTexture(m, shape, at, domain.TextureMountain); e != nil {
				return Snapshot{}, e
			}
			placements = append(placements, domain.ShapePlacement{Shape: shape, Center: at, Kind: domain.PlaceEmptyMountain})
		}
	}
	// 3. Grouped forests, then the adjustable scatter fill on unoccupied meadow.
	if rule := req.ForestRequirements; rule.TilesPerUser > 0 {
		for j := 0; j < rule.ResourceRegionsPerUser; j++ {
			count := rule.TilesPerUser / rule.ResourceRegionsPerUser
			if j < rule.TilesPerUser%rule.ResourceRegionsPerUser {
				count++
			}
			shape, e := GenerateShapeForPlayers(count, rng.Int63(), req.MapSymmetry)
			if e != nil {
				return Snapshot{}, e
			}
			at, e := choosePlacement(req, rng, shape, func(x, y int) bool { return available(x, y, false) })
			if e != nil {
				return Snapshot{}, fmt.Errorf("las %d: %w", j+1, e)
			}
			placements = append(placements, domain.ShapePlacement{Shape: shape, Center: at, Kind: domain.PlaceForest, Count: count})
			if e := EmbedForest(m, shape, at); e != nil {
				return Snapshot{}, e
			}
		}
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if available(x, y, false) && rng.Intn(100) < req.WoodsFillInRate {
				scatteredTrees = append(scatteredTrees, Point{X: x, Y: y})
				m.Resources[y][x] = domain.ResourceTree1 + domain.ResourceType(rng.Intn(8))
				m.ResourceAmounts[y][x] = 1
			}
		}
	}
	// 4. Animals are separate from objects but require contiguous forest clusters.
	if rule := req.AnimalRequirements; rule.TilesPerUser > 0 {
		for j := 0; j < rule.ResourceRegionsPerUser; j++ {
			count := rule.TilesPerUser / rule.ResourceRegionsPerUser
			if j < rule.TilesPerUser%rule.ResourceRegionsPerUser {
				count++
			}
			shape, e := GenerateShapeForPlayers(count, rng.Int63(), req.MapSymmetry)
			if e != nil {
				return Snapshot{}, e
			}
			at, e := chooseAnimalPlacement(req, rng, shape, m, placements)
			if e != nil {
				return Snapshot{}, fmt.Errorf("zwierzyna %d: %w", j+1, e)
			}
			placements = append(placements, domain.ShapePlacement{Shape: shape, Center: at, Kind: domain.PlaceAnimals, Count: count})
			if e := EmbedAnimals(m, shape, at); e != nil {
				return Snapshot{}, e
			}
		}
	}
	// 5. ALL source-sector matrices are reflected/rotated together in this pass.
	if err := ReplicateShapes(m, placements, scatteredTrees, req.MapSymmetry, req.PlayableDiameter); err != nil {
		return Snapshot{}, err
	}
	// 6. Heights are derived from the final, replicated terrain, NOT from
	// source sector masks, so square mirrors and round sectors remain coherent.
	RaiseMiningAreas(m, req.PlayableDiameter, req.MapSymmetry, req.LandLevel)
	RaiseOuterMountains(m, req.PlayableDiameter, req.MapSymmetry, req.LandLevel, req.Seed)
	// 7. Validate quantifiable resources in the final domain, before SWD serialization.
	stats, e := collectDomainStats(m, req.MapSymmetry, req.PlayableDiameter, lakeN, lakeSize)
	if e != nil {
		return Snapshot{}, e
	}
	if e := validateParity(stats); e != nil {
		return Snapshot{}, e
	}
	for _, p := range m.Starts {
		if p.X < 0 || p.Y < 0 || p.X >= m.Width || p.Y >= m.Height || m.Textures[p.Y][p.X] != domain.TextureMeadow || m.Resources[p.Y][p.X] != domain.ResourceNone || m.Animals[p.Y][p.X] != domain.AnimalNone {
			return Snapshot{}, fmt.Errorf("HQ at %d,%d obstructed", p.X, p.Y)
		}
	}
	native, err := ToSWD(m)
	if err != nil {
		return Snapshot{}, fmt.Errorf("export SWD: %w", err)
	}
	if err := validateSWDDimensions(native, req.MapSize); err != nil {
		return Snapshot{}, err
	}
	final, e := MapDataFromSWD(native)
	if e != nil {
		return Snapshot{}, e
	}
	actual, e := collectDomainStats(final, req.MapSymmetry, req.PlayableDiameter, lakeN, lakeSize)
	if e != nil {
		return Snapshot{}, e
	}
	if e := validateParity(actual); e != nil {
		return Snapshot{}, e
	}
	if e := validateStartingPositionsFromBytes(native, req.MapSymmetry); e != nil {
		return Snapshot{}, e
	}
	// Preview uses the SAVED model after an SWD roundtrip, never a second simulation.
	preview := makeDomainPreview(final, actual, descriptors, req)
	raw, e := json.Marshal(preview)
	if e != nil {
		return Snapshot{}, e
	}
	return Snapshot{Preview: raw, SWD: native}, nil
}

// chooseOrigin uses user-selected radial position (10%-90%) for round maps.
// A landing is accepted only when it and all its rotated copies have enough
// flat land for a three-cell HQ clearance; impossible near-edge results retry.
// Square maps are sampled across the ENTIRE source half/quarter instead.
func chooseOrigin(req MapRequirements, rng *rand.Rand) Point {
	n := req.MapSize
	centre := float64(n-1) / 2
	round := req.MapSymmetry != 2 && req.MapSymmetry != 4
	for attempt := 0; attempt < 5000; attempt++ {
		var p Point
		if round {
			radius := float64(n-1) / 2 * float64(req.StartRadiusPercent) / 100
			angle := (rng.Float64()*2 - 1) * .20 * math.Pi / float64(req.MapSymmetry)
			p = Point{X: int(math.Round(centre + radius*math.Cos(angle))), Y: int(math.Round(centre + radius*math.Sin(angle)))}
		} else {
			// Uniform in the source sector, not in an inscribed polar circle.
			margin := 5
			minX, minY := n/2+margin, margin
			if req.MapSymmetry == 4 {
				minY = n/2 + margin
			}
			if n-margin <= minX || n-margin <= minY {
				continue
			}
			p = Point{X: minX + rng.Intn(n-margin-minX), Y: minY + rng.Intn(n-margin-minY)}
		}
		valid := true
		for k := 0; k < req.MapSymmetry; k++ {
			q, _ := RotatePointSized(p, k, req.MapSymmetry, n)
			if q.X < 4 || q.Y < 4 || q.X >= n-4 || q.Y >= n-4 || !playable(q.X, q.Y, req.PlayableDiameter, req.MapSymmetry) || ownerOfSized(q, req.MapSymmetry, n) != k {
				valid = false
				break
			}
		}
		if valid {
			return p
		}
	}
	// A safe deterministic fallback when a very small map cannot provide an
	// HQ at the requested radius. This branch is not used in normal maps.
	if req.MapSymmetry == 4 {
		return Point{X: n * 3 / 4, Y: n * 3 / 4}
	}
	return Point{X: n * 3 / 4, Y: n / 2}
}

// choosePlacement samples sectors in different geometries. In square mode
// this is a uniform rectangle/quarter; it does NOT inherit the restrictive
// radial placement used by round maps.
type placementChoice struct {
	accept func(Point) bool
	score  func(Point) int
}

func choosePlacement(req MapRequirements, rng *rand.Rand, shape Shape, allowed func(x, y int) bool, option ...placementChoice) (Point, error) {
	n := req.MapSize
	best, bestScore, haveBest := Point{}, -1, false
	for trial := 0; trial < 300+n*n; trial++ {
		var p Point
		if req.MapSymmetry == 2 || req.MapSymmetry == 4 {
			startX, startY := n/2, 0
			if req.MapSymmetry == 4 {
				startY = n / 2
			}
			p = Point{X: startX + rng.Intn(n-startX), Y: startY + rng.Intn(n-startY)}
		} else {
			radius := float64(n-1) / 2 * (.10 + rng.Float64()*.82)
			angle := (rng.Float64()*2 - 1) * .90 * math.Pi / float64(req.MapSymmetry)
			c := float64(n-1) / 2
			p = Point{X: int(math.Round(c + radius*math.Cos(angle))), Y: int(math.Round(c + radius*math.Sin(angle)))}
		}
		if trial == 300 && haveBest {
			return best, nil
		}
		if trial >= 300 {
			i := trial - 300
			p = Point{X: i % n, Y: i / n}
		}
		pts, _ := ShapePoints(shape, p)
		valid := true
		for _, pos := range pts {
			if pos.X < 0 || pos.Y < 0 || pos.X >= n || pos.Y >= n || !allowed(pos.X, pos.Y) {
				valid = false
				break
			}
			for player := 1; player < req.MapSymmetry; player++ {
				at, e := RotatePointSized(pos, player, req.MapSymmetry, n)
				if e != nil || at.X < 0 || at.Y < 0 || at.X >= n || at.Y >= n || !playable(at.X, at.Y, req.PlayableDiameter, req.MapSymmetry) || ownerOfSized(at, req.MapSymmetry, n) != player {
					valid = false
					break
				}
			}
			if !valid {
				break
			}
		}
		if valid && (len(option) == 0 || option[0].accept == nil || option[0].accept(p)) {
			if len(option) == 0 || option[0].score == nil {
				return p, nil
			}
			score := option[0].score(p)
			if !haveBest || score > bestScore {
				best, bestScore, haveBest = p, score, true
			}
		}
	}
	if haveBest {
		return best, nil
	}
	return Point{}, fmt.Errorf("no room for a connected %d-tile shape; reduce the number of clusters or resources", countShape(shape))
}

func countShape(mask Shape) int {
	n := 0
	for _, row := range mask {
		for _, b := range row {
			if b {
				n++
			}
		}
	}
	return n
}

func validateStartingPositionsFromBytes(data []byte, players int) error {
	s, err := parseSwd(data)
	if err != nil {
		return err
	}
	return validateStartingPositions(s, players)
}

func makeDomainPreview(m *domain.MapData, stats symmetryStats, clusters []compactDescriptor, req MapRequirements) map[string]any {
	tiles := make([]int, m.Width*m.Height)
	deposits := make([]int, len(tiles))
	hq := make([][]int, len(m.Starts))
	for k, p := range m.Starts {
		hq[k] = []int{p.X, p.Y}
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			r := m.Resources[y][x]
			switch m.Textures[y][x] {
			case domain.TextureWater:
				tiles[i] = 5
			case domain.TextureMountain:
				tiles[i] = 4
			case domain.TextureRocky:
				tiles[i] = 6
			default:
				if r >= domain.ResourceTree1 && r <= domain.ResourceTree8 {
					tiles[i] = 2
					if m.Animals[y][x] != domain.AnimalNone {
						tiles[i] = 3
					}
				}
			}
			switch r {
			case domain.ResourceCoal:
				deposits[i] = 1
			case domain.ResourceIron:
				deposits[i] = 2
			case domain.ResourceGold:
				deposits[i] = 3
			case domain.ResourceGranite:
				deposits[i] = 4
			case domain.ResourceFish:
				deposits[i] = 6
			default:
				if r >= domain.ResourceSurfaceStone1 && r <= domain.ResourceSurfaceStone6 {
					deposits[i] = 5
				} else if m.Animals[y][x] != domain.AnimalNone {
					deposits[i] = 7
				}
			}
		}
	}
	report := map[string]any{"perPlayer": req.LakeRequirements.ResourceRegionsPerUser, "tilesPerLake": req.LakeRequirements.MinTilesPerRegion, "totalWaterPerPlayer": req.LakeRequirements.TilesPerUser, "fishPerPlayer": stats.Fish[0], "method": "unified-mask-and-rotation"}
	return map[string]any{"width": m.Width, "height": m.Height, "tiles": tiles, "deposits": deposits, "hq": hq, "harbors": [][]int{}, "clusters": clusters, "symmetryReport": stats, "lakesReport": report, "circleRatio": 1.0, "issues": []string{}, "architecture": "domain-masks-rotation-swd"}
}

// chooseAnimalPlacement picks an already-forested contiguous group. The same
// GenerateShape mask is reused, but may only be embedded over actual trees.
func chooseAnimalPlacement(req MapRequirements, rng *rand.Rand, shape Shape, m *domain.MapData, placements []domain.ShapePlacement) (Point, error) {
	centres := make([]Point, 0)
	forestCells := map[Point]bool{}
	for _, entry := range placements {
		if entry.Kind != domain.PlaceForest {
			continue
		}
		points, _ := ShapePoints(entry.Shape, entry.Center)
		for _, p := range points {
			forestCells[p] = true
		}
	}
	if len(forestCells) == 0 {
		return Point{}, fmt.Errorf("zwierzyna wymaga co najmniej jednego wygenerowanego skupiska lasu")
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if forestCells[Point{X: x, Y: y}] && inMasterSectorSized(Point{X: x, Y: y}, req.MapSymmetry, req.MapSize) && m.Resources[y][x] >= domain.ResourceTree1 && m.Resources[y][x] <= domain.ResourceTree8 {
				centres = append(centres, Point{X: x, Y: y})
			}
		}
	}
	rng.Shuffle(len(centres), func(i, j int) { centres[i], centres[j] = centres[j], centres[i] })
	for _, at := range centres {
		pts, _ := ShapePoints(shape, at)
		valid := true
		for _, p := range pts {
			if p.X < 0 || p.Y < 0 || p.X >= m.Width || p.Y >= m.Height || !forestCells[p] || !playable(p.X, p.Y, req.PlayableDiameter, req.MapSymmetry) || !inMasterSectorSized(p, req.MapSymmetry, req.MapSize) || m.Animals[p.Y][p.X] != domain.AnimalNone || m.Resources[p.Y][p.X] < domain.ResourceTree1 || m.Resources[p.Y][p.X] > domain.ResourceTree8 || nearAnyHQ(m, p.X, p.Y, 3) {
				valid = false
				break
			}
		}
		if valid {
			return at, nil
		}
	}
	return Point{}, fmt.Errorf("insufficient connected forest area for a %d-animal cluster; increase forest coverage or reduce wildlife", countShape(shape))
}
