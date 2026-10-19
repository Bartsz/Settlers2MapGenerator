package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	dm "settlers2-final-go/DomainModel"
	engine "settlers2-final-go/InternalMapManipulation"
	"testing"
)

func buildRotationPayload(players, size, radius int, minimal bool) []byte {
	var p map[string]any
	_ = json.Unmarshal(makePayload(players, size, 73519, false), &p)
	p["startRadiusPercent"] = radius
	if minimal {
		p["totals"] = []int{0, 0, 0, 0, 0}
		p["groups"] = []int{0, 0, 0, 0, 0}
		p["lakes"] = 0
		p["forest"] = 0
		p["forestTiles"] = 0
		p["forestGroups"] = 0
		p["animals"] = 0
		p["animalGroups"] = 0
	}
	b, _ := json.Marshal(p)
	return b
}

func TestSelectedStartRadiusAndAngularSpread(t *testing.T) {
	for _, players := range []int{3, 5, 6} {
		for _, percent := range []int{10, 25, 55, 90} {
			t.Run(fmt.Sprintf("%dp-%d", players, percent), func(t *testing.T) {
				snap, err := engine.Generate(buildRotationPayload(players, 109, percent, true))
				if err != nil {
					t.Fatal(err)
				}
				m, err := engine.MapDataFromSWD(snap.SWD)
				if err != nil {
					t.Fatal(err)
				}
				p := m.Starts[0]
				c := 54.0
				r := math.Hypot(float64(p.X)-c, float64(p.Y)-c)
				target := 54 * float64(percent) / 100
				if math.Abs(r-target) > 1.2 {
					t.Fatalf("radius %.2f expected %.2f, HQ %v", r, target, p)
				}
				a := math.Abs(math.Atan2(float64(p.Y)-c, float64(p.X)-c))
				if a > 0.2*math.Pi/float64(players)+0.125 {
					t.Fatalf("HQ outside ±10%% sector plus rounding: %v angle %.3f", p, a)
				}
			})
		}
	}
}

func TestCircularRotationModesAndBalance(t *testing.T) {
	for _, players := range []int{3, 5, 6} {
		t.Run(fmt.Sprintf("%dp-mandatory-rotation", players), func(t *testing.T) {
			payload := buildRotationPayload(players, 109, 55, false)
			got, err := engine.Generate(payload)
			if err != nil {
				t.Fatal(err)
			}
			if err := engine.ValidateSWD(got.SWD); err != nil {
				t.Fatal(err)
			}
			gotAgain, err := engine.Generate(payload)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.SWD, gotAgain.SWD) {
				t.Fatal("not deterministic")
			}
			var preview struct {
				Symmetry dm.SymmetryStats `json:"symmetryReport"`
			}
			if err := json.Unmarshal(got.Preview, &preview); err != nil {
				t.Fatal(err)
			}
			for _, counts := range [][]int{preview.Symmetry.Coal, preview.Symmetry.Iron, preview.Symmetry.Gold, preview.Symmetry.Granite, preview.Symmetry.Lakes, preview.Symmetry.Forests, preview.Symmetry.Water, preview.Symmetry.Fish, preview.Symmetry.Animals, preview.Symmetry.Rocks} {
				for k := 1; k < players; k++ {
					if counts[k] != counts[0] {
						t.Fatalf("unequal counts %v", counts)
					}
				}
			}
		})
	}
}

func TestInvalidStartRadius(t *testing.T) {
	for _, percent := range []int{9, 91} {
		_, err := engine.Generate(buildRotationPayload(6, 109, percent, true))
		if err == nil {
			t.Fatalf("accepted radius %d", percent)
		}
	}
}
