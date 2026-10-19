package tests

import (
	"bytes"
	"testing"

	dm "settlers2-final-go/DomainModel"
	manip "settlers2-final-go/InternalMapManipulation"
)

func TestPureDomainConstructionAndEditing(t *testing.T) {
	m, err := dm.NewMapData(31, 31, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.RawHeader) != 0 || len(m.RawLayers[0]) != 0 {
		t.Fatal("domain incorrectly created native SWD layer")
	}
	shape, err := manip.GenerateShape(19, 1)
	if err != nil {
		t.Fatal(err)
	}
	enlarged, err := manip.ExpandShape(shape, 1)
	if err != nil {
		t.Fatal(err)
	}
	centre := dm.Point{X: 15, Y: 15}
	if err := manip.EmbedTexture(m, enlarged, centre, dm.TextureMountain); err != nil {
		t.Fatal(err)
	}
	if err := manip.EmbedResource(m, shape, centre, dm.ResourceGold); err != nil {
		t.Fatal(err)
	}
	swd, err := manip.ToSWD(m)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := manip.MapDataFromSWD(swd)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for y := 0; y < 31; y++ {
		for x := 0; x < 31; x++ {
			if decoded.Resources[y][x] == dm.ResourceGold {
				n++
				if decoded.ResourceAmounts[y][x] != 7 {
					t.Fatal("missing ore strength")
				}
			}
		}
	}
	if n != 19 {
		t.Fatalf("got %d gold cells, want 19", n)
	}
	if untouched, err := manip.ToSWD(decoded); err != nil || !bytes.Equal(swd, untouched) {
		t.Fatal("non-lossless domain roundtrip")
	}
}

func TestShapeGeneratorIsAlwaysConnected(t *testing.T) {
	for _, n := range []int{1, 2, 5, 13, 25, 40, 120, 1000} {
		mask, err := manip.GenerateShape(n, 9876)
		if err != nil {
			t.Fatal(err)
		}
		if len(mask)%2 == 0 || len(mask) != len(mask[0]) {
			t.Fatal("bad shape dimensions")
		}
		found := 0
		for _, row := range mask {
			for _, v := range row {
				if v {
					found++
				}
			}
		}
		if found != n {
			t.Fatalf("wanted %d got %d", n, found)
		}
		for _, cut := range []int{1, n / 2, n} {
			if cut < 1 {
				continue
			}
			part, err := manip.ConnectedPrefix(mask, cut)
			if err != nil {
				t.Fatal(err)
			}
			cnt := 0
			for _, row := range part {
				for _, v := range row {
					if v {
						cnt++
					}
				}
			}
			if cnt != cut {
				t.Fatal("shape prefix lost cells")
			}
		}
	}
}

func TestMergeAtomicAndForestAnimalOverlap(t *testing.T) {
	m, err := dm.NewMapData(25, 25, 2)
	if err != nil {
		t.Fatal(err)
	}
	shape, _ := manip.GenerateShape(7, 4)
	centre := dm.Point{X: 13, Y: 13}
	if err := manip.EmbedForest(m, shape, centre); err != nil {
		t.Fatal(err)
	}
	if err := manip.EmbedAnimals(m, shape, centre); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, p := range []dm.Point{{X: 13, Y: 13}, {X: 12, Y: 13}, {X: 14, Y: 13}} {
		if m.Animals[p.Y][p.X] != dm.AnimalNone && m.Resources[p.Y][p.X] >= dm.ResourceTree1 {
			n++
		}
	}
	if n == 0 {
		t.Fatal("animals not allowed inside forest")
	}
	if err := manip.EmbedSurfaceStones(m, shape, centre); err == nil {
		t.Fatal("overlap should be rejected")
	}
}

func TestExactHexRotations(t *testing.T) {
	for _, players := range []int{2, 3, 4, 6} {
		for x := 80; x < 100; x += 4 {
			point := dm.Point{X: x, Y: 75}
			seen := map[dm.Point]bool{}
			for k := 0; k < players; k++ {
				out, err := manip.RotatePoint(point, k, players)
				if err != nil {
					t.Fatal(err)
				}
				if seen[out] {
					t.Fatal("rotation collapsed two sectors")
				}
				seen[out] = true
			}
		}
	}
}
