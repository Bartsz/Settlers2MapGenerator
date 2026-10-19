package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	engine "settlers2-final-go/InternalMapManipulation"
	"testing"
)

func shapeCount(shape engine.Shape) int {
	count := 0
	for _, row := range shape {
		for _, v := range row {
			if v {
				count++
			}
		}
	}
	return count
}
func connectedShape(shape engine.Shape) bool {
	offsets := [6][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, 1}, {1, -1}}
	n := len(shape)
	startX, startY := -1, -1
	for y, row := range shape {
		for x, yes := range row {
			if yes {
				startX, startY = x, y
				break
			}
		}
		if startX >= 0 {
			break
		}
	}
	if startX < 0 {
		return false
	}
	seen := map[[2]int]bool{{startX, startY}: true}
	q := [][2]int{{startX, startY}}
	for i := 0; i < len(q); i++ {
		for _, o := range offsets {
			nx, ny := q[i][0]+o[0], q[i][1]+o[1]
			key := [2]int{nx, ny}
			if nx >= 0 && ny >= 0 && nx < n && ny < n && shape[ny][nx] && !seen[key] {
				seen[key] = true
				q = append(q, key)
			}
		}
	}
	return len(seen) == shapeCount(shape)
}

func TestRotateMaskCountConnectivityAndDeterminism(t *testing.T) {
	for _, n := range []int{1, 2, 3, 11, 29, 76, 130} {
		source, err := engine.GenerateShape(n, int64(n)*111)
		if err != nil {
			t.Fatal(err)
		}
		for _, degrees := range []float64{0, 72, 144, 216, 288, 360} {
			t.Run(fmt.Sprintf("cells%d-angle%.0f", n, degrees), func(t *testing.T) {
				a, err := engine.RotateMask(degrees, source)
				if err != nil {
					t.Fatal(err)
				}
				b, err := engine.RotateMask(degrees, source)
				if err != nil {
					t.Fatal(err)
				}
				if shapeCount(a) != n || len(a)%2 != 1 || !connectedShape(a) {
					t.Fatalf("rotated mask has %d cells, connected=%v", shapeCount(a), connectedShape(a))
				}
				if !bytes.Equal(encodeShape(a), encodeShape(b)) {
					t.Fatal("mask rotation is nondeterministic")
				}
			})
		}
	}
}
func encodeShape(a engine.Shape) []byte { out, _ := json.Marshal(a); return out }

func TestMiningExpansionExactFullRings(t *testing.T) {
	core, err := engine.GenerateShape(1, 42)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []int{2, 3, 4, 5, 6} {
		prev := 0
		for _, rings := range []int{0, 1, 2, 3, 5, 8} {
			mask, err := engine.MineralTerrainMask(core, rings, p)
			if err != nil {
				t.Fatal(err)
			}
			if got := shapeCount(mask); got <= prev {
				t.Fatalf("players=%d rings=%d count=%d prev=%d", p, rings, got, prev)
			} else {
				prev = got
			}
			if !connectedShape(mask) {
				t.Fatalf("disconnected mining terrain rings=%d", rings)
			}
		}
	}
}

func TestFivePlayerMaskRotationParityAndMiningExpansion(t *testing.T) {
	for _, seed := range []int64{23, 117, 73519} {
		for _, rings := range []int{0, 3} {
			data := buildRotationPayload(5, 109, 55, false)
			var p map[string]any
			if err := json.Unmarshal(data, &p); err != nil {
				t.Fatal(err)
			}
			p["seed"] = seed
			p["rotateResourceMasks"] = true
			p["miningTerrainExpansion"] = rings
			data, _ = json.Marshal(p)
			snap, err := engine.Generate(data)
			if err != nil {
				t.Fatalf("seed=%d rings=%d: %v", seed, rings, err)
			}
			if err := engine.ValidateSWD(snap.SWD); err != nil {
				t.Fatal(err)
			}
			repeated, err := engine.Generate(data)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(snap.SWD, repeated.SWD) {
				t.Fatal("nondeterministic SWD")
			}
		}
	}
}
func TestInvalidMiningExpansion(t *testing.T) {
	for _, n := range []int{-1, 9} {
		var p map[string]any
		_ = json.Unmarshal(buildRotationPayload(5, 109, 55, true), &p)
		p["miningTerrainExpansion"] = n
		data, _ := json.Marshal(p)
		if _, err := engine.Generate(data); err == nil {
			t.Fatalf("accepted expansion %d", n)
		}
	}
}
