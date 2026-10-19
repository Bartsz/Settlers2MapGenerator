package tests

import (
	"encoding/json"
	"fmt"
	dm "settlers2-final-go/DomainModel"
	engine "settlers2-final-go/InternalMapManipulation"
	"testing"
)

func makePayload(players, size int, seed int64, empty bool) []byte {
	spec := map[string]any{"sym": players, "diameter": size, "seed": seed, "totals": []int{150, 50, 20, 40, 30}, "groups": []int{3, 2, 1, 2, 2},
		"lakes": 3, "lakeSize": 25, "forest": 27, "forestTiles": 100, "forestGroups": 4, "animals": 40, "animalGroups": 3, "landLevel": 1}
	if empty {
		spec["emptyMountainTiles"] = 45
		spec["emptyMountainGroups"] = 2
	}
	b, _ := json.Marshal(spec)
	return b
}
func TestDynamicSizeAndTerrainModes(t *testing.T) {
	for _, players := range []int{2, 3, 4, 5, 6} {
		size := 108
		if players == 3 || players == 5 || players == 6 {
			size = 109
		}
		t.Run(fmt.Sprintf("players-%d", players), func(t *testing.T) {
			snap, e := engine.Generate(makePayload(players, size, 123456, false))
			if e != nil {
				t.Fatal(e)
			}
			m, e := engine.MapDataFromSWD(snap.SWD)
			if e != nil {
				t.Fatal(e)
			}
			if m.Width != size || m.Height != size {
				t.Fatalf("wrong dimensions %dx%d", m.Width, m.Height)
			}
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					if (players == 2 || players == 4) && m.Textures[y][x] == dm.TextureMountain {
						// Mining mountains are allowed; the map's corners must not be mountain belt.
						if x == 0 && y == 0 {
							t.Fatal("unwanted outer mountain belt")
						}
					}
					r := m.Resources[y][x]
					if r >= dm.ResourceSurfaceStone1 && r <= dm.ResourceSurfaceStone6 && m.Textures[y][x] != dm.TextureRocky {
						t.Fatalf("stone not on rocky terrain at %d,%d", x, y)
					}
					if m.Animals[y][x] != dm.AnimalNone && (r < dm.ResourceTree1 || r > dm.ResourceTree8) {
						t.Fatalf("animal outside forest at %d,%d", x, y)
					}
				}
			}
			if players == 3 || players == 5 || players == 6 {
				if m.Textures[0][0] != dm.TextureMountain || int(m.Heights[0][0]) <= 1 {
					t.Fatal("missing raised outer mountains")
				}
			}
			source := m.Starts[0]
			for k := 1; k < players; k++ {
				dest, e := engine.RotatePointSized(source, k, players, size)
				if e != nil {
					t.Fatal(e)
				}
				if dest != m.Starts[k] {
					t.Fatalf("HQ symmetry %d: %v != %v", k, dest, m.Starts[k])
				}
			}
		})
	}
}
func TestEmptyMineRegionsAndVariableMapSizes(t *testing.T) {
	for _, players := range []int{2, 3, 4, 5, 6} {
		size := 126
		if players == 3 || players == 5 || players == 6 {
			size = 127
		}
		for _, seed := range []int64{13, 321, 8719} {
			t.Run(fmt.Sprintf("%dp-%d", players, seed), func(t *testing.T) {
				snap, err := engine.Generate(makePayload(players, size, seed, true))
				if err != nil {
					t.Fatal(err)
				}
				if err = engine.ValidateSWD(snap.SWD); err != nil {
					t.Fatal(err)
				}
				var preview struct {
					Width    int
					Height   int
					Symmetry dm.SymmetryStats `json:"symmetryReport"`
				}
				if e := json.Unmarshal(snap.Preview, &preview); e != nil {
					t.Fatal(e)
				}
				if preview.Width != size || preview.Height != size {
					t.Fatal("wrong preview dimensions")
				}
				for _, arr := range [][]int{preview.Symmetry.Coal, preview.Symmetry.Iron, preview.Symmetry.Gold, preview.Symmetry.Granite, preview.Symmetry.Rocks, preview.Symmetry.Animals, preview.Symmetry.Forests, preview.Symmetry.Water} {
					for k := 1; k < players; k++ {
						if arr[k] != arr[0] {
							t.Fatalf("imbalanced %+v", arr)
						}
					}
				}
			})
		}
	}
}

// The requested square modes are TRUE matrix copies, not similar sectors.
func TestSquareMirrorsCopyFullCells(t *testing.T) {
	for _, players := range []int{2, 4} {
		size := 108
		snap, err := engine.Generate(makePayload(players, size, 123456, false))
		if err != nil {
			t.Fatal(err)
		}
		m, err := engine.MapDataFromSWD(snap.SWD)
		if err != nil {
			t.Fatal(err)
		}
		yStart := size / 2
		if players == 2 {
			yStart = 0
		}
		for y := yStart; y < size; y++ {
			for x := size / 2; x < size; x++ {
				for k := 1; k < players; k++ {
					dest, e := engine.RotatePointSized(dm.Point{X: x, Y: y}, k, players, size)
					if e != nil {
						t.Fatal(e)
					}
					if m.Textures[y][x] != m.Textures[dest.Y][dest.X] || m.Resources[y][x] != m.Resources[dest.Y][dest.X] || m.Animals[y][x] != m.Animals[dest.Y][dest.X] || m.Heights[y][x] != m.Heights[dest.Y][dest.X] {
						t.Fatalf("%dp mismatched copied cell %d,%d vs %d,%d textures %d/%d resource %d/%d animal %d/%d height %d/%d", players, x, y, dest.X, dest.Y, m.Textures[y][x], m.Textures[dest.Y][dest.X], m.Resources[y][x], m.Resources[dest.Y][dest.X], m.Animals[y][x], m.Animals[dest.Y][dest.X], m.Heights[y][x], m.Heights[dest.Y][dest.X])
					}
				}
			}
		}
	}
}

func TestExtremeSizesWithLightConfiguration(t *testing.T) {
	for _, players := range []int{2, 3, 4, 5, 6} {
		sizes := []int{64, 140}
		if players == 3 || players == 5 || players == 6 {
			sizes = []int{65, 139}
		}
		for _, size := range sizes {
			t.Run(fmt.Sprintf("%dp-%d", players, size), func(t *testing.T) {
				var req map[string]any
				_ = json.Unmarshal(makePayload(players, size, 573, false), &req)
				// Minimal maps still have to work when the requested amount
				// of terrain is realistic for their limited playable area.
				req["totals"] = []int{6, 4, 3, 3, 2}
				req["groups"] = []int{1, 1, 1, 1, 1}
				req["lakes"] = 1
				req["lakeSize"] = 4
				req["forestTiles"] = 12
				req["forestGroups"] = 1
				req["forest"] = 5
				req["animals"] = 2
				req["animalGroups"] = 1
				b, _ := json.Marshal(req)
				got, e := engine.Generate(b)
				if e != nil {
					t.Fatal(e)
				}
				m, e := engine.MapDataFromSWD(got.SWD)
				if e != nil {
					t.Fatal(e)
				}
				if m.Width != size || m.Height != size {
					t.Fatal("requested size not encoded")
				}
			})
		}
	}
}

func TestEmptyMountainsAddExactlyRequestedTexture(t *testing.T) {
	countMountain := func(raw []byte) int {
		m, e := engine.MapDataFromSWD(raw)
		if e != nil {
			t.Fatal(e)
		}
		total := 0
		for _, row := range m.Textures {
			for _, v := range row {
				if v == dm.TextureMountain {
					total++
				}
			}
		}
		return total
	}
	for _, players := range []int{2, 4, 5, 6} {
		size := 126
		if players == 5 || players == 6 {
			size = 127
		}
		base, e := engine.Generate(makePayload(players, size, 8, false))
		if e != nil {
			t.Fatal(e)
		}
		ext, e := engine.Generate(makePayload(players, size, 8, true))
		if e != nil {
			t.Fatal(e)
		}
		delta := countMountain(ext.SWD) - countMountain(base.SWD)
		if delta != 45*players {
			t.Fatalf("%d players: added %d empty mountain tiles, expected %d", players, delta, 45*players)
		}
	}
}
