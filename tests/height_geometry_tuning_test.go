package tests

import (
	"encoding/json"
	"fmt"
	"math"
	dm "settlers2-final-go/DomainModel"
	engine "settlers2-final-go/InternalMapManipulation"
	"testing"
)

func TestRoundRasterCircleAndMountainHeights(t *testing.T) {
	for _, players := range []int{3, 5, 6} {
		t.Run(fmt.Sprintf("players%d", players), func(t *testing.T) {
			got, err := engine.Generate(makePayload(players, 109, 123456, true))
			if err != nil {
				t.Fatal(err)
			}
			m, err := engine.MapDataFromSWD(got.SWD)
			if err != nil {
				t.Fatal(err)
			}
			c := float64(m.Width-1) / 2
			innerMine := 0
			for y := 0; y < m.Height; y++ {
				for x := 0; x < m.Width; x++ {
					r := math.Hypot(float64(x)-c, float64(y)-c)
					outside := r > c
					if outside {
						if m.Textures[y][x] != dm.TextureMountain {
							t.Fatalf("nonmountain outside raw circle %d,%d", x, y)
						}
						if r-c >= 6 && (m.Heights[y][x] < 12 || m.Heights[y][x] > 20) {
							t.Fatalf("outer height outside 12..20: %d at %d,%d", m.Heights[y][x], x, y)
						}
					} else if m.Textures[y][x] == dm.TextureMountain {
						if m.Heights[y][x] < 3 || m.Heights[y][x] > 6 {
							t.Fatalf("mine hill height %d at %d,%d", m.Heights[y][x], x, y)
						}
						if m.Heights[y][x] == 6 {
							innerMine++
						}
					}
					for _, d := range [][2]int{{1, 0}, {0, 1}, {-1, 1}} {
						xx, yy := x+d[0], y+d[1]
						if xx >= m.Width || yy >= m.Height || xx < 0 || yy < 0 {
							continue
						}
						delta := int(m.Heights[y][x]) - int(m.Heights[yy][xx])
						if delta < 0 {
							delta = -delta
						}
						if delta > 5 {
							t.Fatalf("height step %d at (%d,%d) to (%d,%d)", delta, x, y, xx, yy)
						}
					}
				}
			}
			if innerMine == 0 {
				t.Fatal("no mine plateaus at height 6")
			}
		})
	}
}

func TestSquareHeightsAndOriginsUseFullQuadrant(t *testing.T) {
	for _, players := range []int{2, 4} {
		farX, farY, nearY := false, false, false
		for seed := int64(1); seed <= 60; seed++ {
			minimal := map[string]any{"sym": players, "diameter": 108, "seed": seed * 981719, "totals": []int{0, 0, 0, 0, 0}, "groups": []int{0, 0, 0, 0, 0}, "forestTiles": 0, "forestGroups": 0, "forest": 0, "animals": 0, "animalGroups": 0, "emptyMountainTiles": 0, "emptyMountainGroups": 0, "lakes": 0, "lakeSize": 10, "landLevel": 1}
			request, _ := json.Marshal(minimal)
			got, err := engine.Generate(request)
			if err != nil {
				t.Fatal(err)
			}
			m, err := engine.MapDataFromSWD(got.SWD)
			if err != nil {
				t.Fatal(err)
			}
			hq := m.Starts[0]
			if hq.X > 90 {
				farX = true
			}
			if hq.Y > 90 {
				farY = true
			}
			if hq.Y < 20 && players == 2 {
				nearY = true
			}
		}
		if !farX || !farY || (players == 2 && !nearY) {
			t.Fatalf("HQ not distributed over source rectangle, players=%d: %v %v %v", players, farX, farY, nearY)
		}
	}
}

func TestOreTerrainIsWiderThanOre(t *testing.T) {
	for _, players := range []int{2, 3, 4, 5, 6} {
		size := 108
		if players == 3 || players == 5 || players == 6 {
			size = 109
		}
		got, err := engine.Generate(makePayload(players, size, 123456, true))
		if err != nil {
			t.Fatal(err)
		}
		m, err := engine.MapDataFromSWD(got.SWD)
		if err != nil {
			t.Fatal(err)
		}
		ore, terrain := 0, 0
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if m.Textures[y][x] == dm.TextureMountain {
					terrain++
				}
				if m.Resources[y][x] >= dm.ResourceCoal && m.Resources[y][x] <= dm.ResourceGranite {
					ore++
				}
			}
		}
		// Includes outer mountains in round maps, so it is intentionally a loose lower bound.
		if ore == 0 || terrain < ore*2 {
			t.Fatalf("players=%d mining envelope small: %d terrain vs %d minerals", players, terrain, ore)
		}
	}
}

// The amount of ore a variant draws must never move the prepared mine texture.
func TestVariantAmountPreservesTerrainForCrowdedMap(t *testing.T) {
	payload := makePayload(6, 109, 92119, true)
	var input map[string]any
	if err := json.Unmarshal(payload, &input); err != nil {
		t.Fatal(err)
	}
	baseline, err := engine.Generate(payload)
	if err != nil {
		t.Fatal(err)
	}
	var preview struct {
		Clusters []dm.CompactDescriptor `json:"clusters"`
	}
	if err := json.Unmarshal(baseline.Preview, &preview); err != nil {
		t.Fatal(err)
	}
	amounts, kinds := make([]int, len(preview.Clusters)), make([]int, len(preview.Clusters))
	for i, c := range preview.Clusters {
		amounts[i] = c.Count / 2
		kinds[i] = c.Type
	}
	input["variant"] = amounts
	input["variantTypes"] = kinds
	updatedPayload, _ := json.Marshal(input)
	changed, err := engine.Generate(updatedPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range []int{0, 1, 2, 3, 8, 9, 12, 13} {
		size := 109
		offset := 2352 + layer*(size*size+16) + 16
		for i := 0; i < size*size; i++ {
			if baseline.SWD[offset+i] != changed.SWD[offset+i] {
				t.Fatalf("terrain changed in layer %d, cell %d", layer, i)
			}
		}
	}
}
