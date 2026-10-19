package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	dm "settlers2-final-go/DomainModel"
	manip "settlers2-final-go/InternalMapManipulation"
)

func parameters(players int) []byte {
	v, _ := json.Marshal(map[string]any{"seed": 123456, "sym": players, "diameter": func() int {
		if players == 2 || players == 4 {
			return 108
		}
		return 109
	}(),
		"lakes": 3, "lakeSize": 25, "landLevel": 1, "totals": []int{150, 50, 20, 40, 30},
		"groups": []int{3, 2, 1, 2, 2}, "forest": 27, "forestTiles": 100, "forestGroups": 4,
		"animals": 40, "animalGroups": 3})
	return v
}
func snapshot(t *testing.T, players int) dm.MapSnapshot {
	t.Helper()
	s, err := manip.Generate(parameters(players))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestGeneratorSymmetryAndSWD(t *testing.T) {
	for _, players := range []int{2, 3, 4, 5, 6} {
		t.Run(fmt.Sprintf("%dp", players), func(t *testing.T) {
			s := snapshot(t, players)
			size := 108
			if players == 3 || players == 5 || players == 6 {
				size = 109
			}
			if len(s.SWD) != 2352+14*(16+size*size)+1 {
				t.Fatalf("invalid SWD file size %d", len(s.SWD))
			}
			if err := manip.ValidateSWD(s.SWD); err != nil {
				t.Fatal(err)
			}
			model, err := manip.MapDataFromSWD(s.SWD)
			if err != nil {
				t.Fatal(err)
			}
			if model.Width != size || model.Height != size || len(model.Starts) != players {
				t.Fatalf("invalid domain dimensions or HQ")
			}
			var p struct {
				Width, Height int
				Symmetry      dm.SymmetryStats       `json:"symmetryReport"`
				Clusters      []dm.CompactDescriptor `json:"clusters"`
				Deposits      []int                  `json:"deposits"`
			}
			if err = json.Unmarshal(s.Preview, &p); err != nil {
				t.Fatal(err)
			}
			if p.Width != size || p.Height != size || len(p.Clusters) != 10 {
				t.Fatalf("invalid preview dimensions/clusters: %+v", p)
			}
			if len(p.Deposits) != size*size {
				t.Fatal("missing preview")
			}
			for k := 0; k < players; k++ {
				if p.Symmetry.Water[k] != 75 || p.Symmetry.Coal[k] != 150 || p.Symmetry.Iron[k] != 50 || p.Symmetry.Gold[k] != 20 || p.Symmetry.Granite[k] != 40 || p.Symmetry.Rocks[k] != 30 || p.Symmetry.Animals[k] != 40 || p.Symmetry.Lakes[k] != 3 || p.Symmetry.Forests[k] <= 100 || p.Symmetry.Forests[k] != p.Symmetry.Forests[0] || p.Symmetry.PlayablePerPlayer <= 0 {
					t.Fatalf("unbalanced player %d: %+v", k, p.Symmetry)
				}
			}
			if !bytes.Equal(s.SWD, snapshot(t, players).SWD) {
				t.Fatal("not reproducible")
			}
			again, err := manip.ToSWD(model)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(again, s.SWD) {
				t.Fatal("SWD not lossless after decode")
			}
		})
	}
}

func TestVariationsDoNotChangeTerrain(t *testing.T) {
	base := snapshot(t, 6)
	var preview struct {
		Clusters []dm.CompactDescriptor `json:"clusters"`
	}
	if err := json.Unmarshal(base.Preview, &preview); err != nil {
		t.Fatal(err)
	}
	counts, kinds := make([]int, len(preview.Clusters)), make([]int, len(preview.Clusters))
	for k, c := range preview.Clusters {
		counts[k] = c.Count
		kinds[k] = c.Type
	}
	kinds[0] = 1 // coal to iron; mountain footprint must be identical
	var obj map[string]any
	if err := json.Unmarshal(parameters(6), &obj); err != nil {
		t.Fatal(err)
	}
	obj["variant"] = counts
	obj["variantTypes"] = kinds
	payload, _ := json.Marshal(obj)
	modified, err := manip.Generate(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range []int{0, 1, 2, 3, 8, 9, 12, 13} {
		size := 109
		offset := 2352 + layer*(size*size+16) + 16
		if !bytes.Equal(base.SWD[offset:offset+size*size], modified.SWD[offset:offset+size*size]) {
			t.Fatalf("changed terrain layer %d", layer)
		}
	}
	if bytes.Equal(base.SWD, modified.SWD) {
		t.Fatal("variant had no effect")
	}
}

func TestInvalidConfigurationRejected(t *testing.T) {
	for _, obj := range []map[string]any{
		{"seed": 1, "sym": 7}, {"seed": 1, "sym": 1},
		{"sym": 6, "diameter": 108},
		{"sym": 6, "groups": []int{0, 1, 1, 1, 1}},
		{"sym": 6, "forestTiles": 25, "forestGroups": 0},
	} {
		data, _ := json.Marshal(obj)
		if _, err := manip.Generate(data); err == nil {
			t.Fatalf("expected rejection: %v", obj)
		}
	}
}
