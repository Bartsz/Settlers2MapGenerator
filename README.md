# Settlers II Map Generator (Go)

A standalone, local-browser map generator for The Settlers II, with an editable domain model and SWD export. The application is written in Go; it does not require an external `engine.exe`, Python, or a hosted service.

## Run and build

**Windows:** launch the compiled `Settlers2_Generator.exe`. It listens on a random port bound to `127.0.0.1` and opens the local editor in the default browser.

**Build from source:** Go 1.23 or later is recommended (the source module declares its required Go version in `go.mod`).

```sh
go run .
go test ./...
go vet ./...
go build -o Settlers2_Generator .
```
 
The application also supports command-line operation: `go run . -help`. CLI handling lives in `InternalMapManipulation/cli.go`.

## Source organization

```text
DomainModel/
  model.go, map_factory.go       Editable MapData and layer types
  requirements.go               Validated requirements and legacy JSON request DTO
  reports.go                    Shape placements, clusters, reports
InternalMapManipulation/
  map_generator.go               Single-sector generation pipeline
  resource_shape_generator.go    Shared connected-shape generator
  resource_shape_validator.go    Shape/placement checks
  resource_shape_embedder.go     MapData layer operations
  resource_mask_rotation.go      Whole-mask 72° rasterization and repair
  shape_replication.go           Matrix mirrors and circular sector copies
  mineral_terrain_generator.go   Configurable mining terrain halo
  mountain_height_generator.go   Terrain height and transition bands
  map_symmetry.go                Coordinate transformations
  requirements_decoder.go       Request-to-domain adaptation
  swd_codec*.go, swd_*.go        Binary SWD serialization and validation
  domain_statistics.go          Per-player resource checks
  cli.go, public.go, ...
API/server.go                    Local HTTP routes and Save and Run
static/index.html                Embedded UI markup
static/styles.css                Embedded UI styles
static/code.js                   Embedded UI logic
main.go                          HTTP bootstrap and go:embed
 tests/                          Integration and unit tests
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the data flow and geometry constraints.

## Editor behavior

The browser UI uses three columns: **resources/cluster counts**, **map preview and balance**, and **map configuration/actions**. All visible labels and dynamic feedback are in English.

- **2 or 4 players:** even map size (64–140), square playable area, matrix mirroring.
- **3, 5 or 6 players:** odd size (65–139), circular playable area in raw map X/Y, outer mountain ring, polar placement of player headquarters.
- **5 players:** every replica rotates both the placement center by `n × 72°` and the entire resource/terrain/forest/water/wildlife mask by `n × 72°`. This is mandatory; there is no longer a rotation checkbox or alternate five-player algorithm.
- **3 or 6 players:** local hex-mask rotations by multiples of 60° combined with rotated sector centers. These are not mathematically exact global rotations of the raw X/Y map.
- **Mining Terrain Expansion:** 0–8 complete hex rings beyond the mineral mask, default 3; changing this modifies the terrain footprint, not the mineral quantity.
- **Starting Position:** circular maps have a 10–90% radius slider, default 55%, with angular variation of ±10% of one sector's angle.
- Forests, lakes, mineral deposits, surface stones, empty mountains and wildlife use connected masks. Wildlife clusters require available forest areas. Trees can fill otherwise empty meadow terrain.
- The middle preview has raw and optional projected-hex display. A failed generation shows a **red, non-dismissible message** centered horizontally about 10rem from the top of the preview panel. The error clears after the next successful generation.
- Deposit variants change material assignments/counts within existing cluster capacities without regenerating the map geometry.

The preview, resource table, and exported SWD derive from the same final map. Save and Run saves the generated SWD to the **Downloads folder of the machine running the Go application**, then asks the operating system to open it using the `.swd` file association. Saving can succeed even if no editor is associated with that extension. A legacy download endpoint remains available to API clients.

## API

All API endpoints accept the browser's JSON map requirements; the schema is defined in `DomainModel/requirements.go` and normalized by `requirements_decoder.go`.

| Route | Response |
|---|---|
| `GET /` | Embedded HTML editor |
| `GET /static/code.js`, `GET /static/styles.css` | Embedded assets |
| `POST /api/map` | JSON preview, clusters and balance information |
| `POST /api/download` | SWD file bytes |
| `POST /api/save-and-run` | Saves SWD and requests OS file-association launch; JSON contains saved path on success |

New requests do not need `rotateResourceMasks`. The obsolete field, if present in an older JSON project or API request, is ignored; five-player full-mask rotation is always enabled. The older `useOldCircularMethod` field is likewise not part of the active configuration. Projects saved by the UI use version 3; the loader accepts versions 2 and 3 and retains variants and selected row.

## Current defaults and limits

| Setting | Default / permitted values |
|---|---|
| Players | 6; 2–6 supported |
| Map dimensions | 109×109 for circular default; physical SWD size follows slider |
| Seed | 123456 in UI |
| Land height | 1; 40 legacy and 100 experimental |
| Start radius | 55%; 10–90% for circular layouts |
| Mining terrain halo | 3; 0–8 full rings |
| Coal, iron, gold, mined granite | 150/50/20/40 tiles and 3/2/1/2 clusters per player |
| Surface stone | 30 tiles, 2 clusters per player |
| Lakes | 3 per player, 25 tiles each |
| Grouped forest | 100 tiles, 4 clusters per player |
| Wildlife | 40 objects, 3 clusters per player |
| Empty mining terrain | 0 tiles, 0 clusters by default |
| Additional tree fill | 27% of eligible remaining grassland |

The generator rejects infeasible configurations rather than silently reducing requested mineral/lake quantities. Deterministic placement retries are available; a reported placement error is not silently ignored.

## Compatibility and validation

SWD output stores the selected actual map dimensions in its header and 14 data blocks; this is **not** a fixed 150×150 exporter and does **not** export native S2M files. Terrain heights of 6 in mining regions and 12–20 in the outer mountain ring apply relative to land-level 1, with transition bands of two and four cells respectively. Legacy base land levels 40 and 100 use corresponding height offsets.

The source includes automated tests of masks, cardinality, deterministic output, SWD structure, symmetry balancing, variant handling and HTTP routes. An optional offline Chromium/Playwright UI smoke test lives at `tests/ui_smoke.py` (`python tests/ui_smoke.py` with Playwright and Chromium installed). Tests **do not** prove that a map opens or plays correctly in The Settlers II: 10th Anniversary. Test it in the target editor on Windows before treating game compatibility as verified. A fivefold rotation cannot exactly preserve every hex-grid vertex, so shape repair and quantity verification are required.

The original project requirements and the distinction between historical and implemented behavior are documented in [requirements.md](requirements.md). Changes from previous releases are summarized in [CHANGELOG.md](CHANGELOG.md).
