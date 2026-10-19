package internalmapmanipulation

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// CLI mode is optional: the normal executable starts the local editor UI.
// It uses exactly the same snapshot generator as the UI and HTTP exporter.
func ExportFromCLI() error {
	out := flag.String("out", "", "Sciezka do zapisu mapy SWD (wymagana w trybie CLI)")
	players := flag.Int("players", 6, "Liczba graczy: 2-6")
	diameter := flag.Int("diameter", 109, "Rozmiar SWD: parzysty dla 2/4 graczy, nieparzysty dla 3/5/6")
	seed := flag.Int64("seed", 123456, "Ziarno generatora")
	lakes := flag.Int("lakes", 3, "Liczba jezior na gracza: 0-10")
	lakeSize := flag.Int("lake-size", 25, "Rozmiar jeziora (kafelki): 1-120")
	landLevel := flag.Int("land-level", 1, "Wysokosc ladu: 1,40,100 (zalecane 1)")
	coal := flag.Int("coal", 150, "Kafelki wegla na gracza")
	iron := flag.Int("iron", 50, "Kafelki zelaza na gracza")
	gold := flag.Int("gold", 20, "Kafelki zlota na gracza")
	granite := flag.Int("granite", 40, "Kafelki granitu kopalnianego na gracza")
	stone := flag.Int("stone", 30, "Kafelki kamienia powierzchniowego na gracza")
	coalGroups := flag.Int("coal-groups", 3, "Skupiska wegla")
	ironGroups := flag.Int("iron-groups", 2, "Skupiska zelaza")
	goldGroups := flag.Int("gold-groups", 1, "Skupiska zlota")
	graniteGroups := flag.Int("granite-groups", 2, "Skupiska granitu")
	stoneGroups := flag.Int("stone-groups", 2, "Skupiska kamieni powierzchniowych")
	animals := flag.Int("animals", 40, "Zwierzetna na gracza")
	emptyTiles := flag.Int("empty-mountain-tiles", 0, "Puste gory: kafelki na gracza")
	emptyGroups := flag.Int("empty-mountain-groups", 0, "Puste gory: skupiska")
	animalGroups := flag.Int("animal-groups", 3, "Skupiska zwierzyny")
	forest := flag.Int("forest", 27, "Dodatkowe wypelnienie wolnych pol drzewami (0-85%)")
	forestTiles := flag.Int("forest-tiles", 100, "Kafelki zwartych lasow na gracza (0-1000)")
	forestGroups := flag.Int("forest-groups", 4, "Skupiska lasow na gracza (0-40)")
	flag.Parse()
	if *out == "" {
		return fmt.Errorf("podaj -out nazwa.swd lub uruchom program bez argumentow, aby otworzyc edytor")
	}
	payload, err := json.Marshal(map[string]any{
		"sym": *players, "diameter": *diameter, "seed": *seed, "lakes": *lakes, "lakeSize": *lakeSize, "landLevel": *landLevel,
		"totals":             []int{*coal, *iron, *gold, *granite, *stone},
		"groups":             []int{*coalGroups, *ironGroups, *goldGroups, *graniteGroups, *stoneGroups},
		"surfaceStoneGroups": *stoneGroups, "animals": *animals, "animalGroups": *animalGroups, "forest": *forest,
		"forestTiles": *forestTiles, "forestGroups": *forestGroups, "emptyMountainTiles": *emptyTiles, "emptyMountainGroups": *emptyGroups,
	})
	if err != nil {
		return err
	}
	snap, err := Generate(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, snap.SWD, 0644); err != nil {
		return err
	}
	fmt.Printf("SWD zapisano: %s (%d bajtow, %d graczy, %dx%d)\n", *out, len(snap.SWD), *players, *diameter, *diameter)
	return nil
}
