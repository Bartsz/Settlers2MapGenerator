# Architecture — Domain Model, Shape Generation, Symmetry and SWD

## Dependency boundaries

```text
 static/index.html + static/styles.css + static/code.js
                        │
                 localhost HTTP
                        │
                    API/
                        │
            InternalMapManipulation/
             ├─ shape operations
             ├─ map generation/symmetry
             └─ SWD serialization
                        │
                  DomainModel/
            (MapRequirements, MapData)
```

`DomainModel` does not depend on the HTTP API or binary SWD codec. Its `[y][x]` layers contain heights, logical textures, resource types and amounts, wildlife, start points and auxiliary information needed for fidelity. `TextureTriangles` preserves the two triangle texture layers in the native format; representing them as one tile enum alone would lose information. Raw headers/layers are retained when importing existing SWD files to support lossless details not represented in a logical enum. Creating a fresh `MapData` does **not** generate SWD bytes.

The executable embeds `static/` files with Go's `//go:embed` mechanism, and `API/server.go` exposes them on loopback HTTP alongside the map endpoints.

## Generation pipeline

1. **Decode and validate:** Convert the flat JSON request to `MapRequirements`. Validate player count, physical size, parity, resource counts, lake settings, seed-dependent parameters, height mode and mining terrain expansion.
2. **Create the map:** Allocate `MapData` at the requested size rather than using an implicit 150×150 backing array. Use even dimensions for 2/4 players, odd dimensions for 3/5/6; square and circular playable regions are generated differently.
3. **Place HQs:** Choose a starting point for the source player. For circular maps, use a user-selected 10–90% radius and angular offset within ±10% of the 360°/players sector. Validate usable land and reserve three hex steps of immediate HQ clearance.
4. **Generate masks:** `GenerateShape` / `GenerateShapeForPlayers` produces connected, odd-sided `[][]bool` footprints. Lake, mineral, grouped forest, stone, wildlife and empty-mountain placement share this shape-generation approach. Square layouts use a four-direction connected shape where necessary to preserve connectivity after reflection.
5. **Validate and embed:** `ValidatePlacement` and the dedicated `Embed*` operations place masks onto the appropriate logical layers. Mineral cluster cores remain reserved separately from their terrain halo. `MineralTerrainMask` expands mountain texture by 0–8 full hex-neighbor rings (default 3) around the entire cluster capacity; variants do not change this texture. Empty mountains create resource-free mining terrain. Surface stones receive rocky ground. Wildlife is only placed where forest clusters provide usable cells.
6. **Fill vacant meadow:** Apply the additional tree-fill rate, avoiding protected or ineligible cells.
7. **Replicate the source sector:** `ReplicateShapes` uses the mirror algorithm for 2/4 and circular replication for 3/5/6. All other sectors derive from the same source placements; they are not independently randomized.
8. **Adjust heights:** `RaiseMiningAreas` raises mining and empty-mountain footprints to height 6 with a two-cell inward transition (for base land height 1). `RaiseOuterMountains` raises land outside the circular playable area to heights 12–20 using a four-cell transition and seeded noise. Height smoothing constrains adjacent hex differences.
9. **Validate balance:** Compute per-player resource counts on the completed map; detect lake count, cell-count, connectedness or placement violations and fail if the map cannot satisfy requirements.
10. **Serialize:** `ToSWD` writes the `WORLD_V1.0` header, 14 native data layers and HQ object markers, then checks the resulting SWD structure. `MapDataFromSWD` decodes the result for preview and final balance statistics.

## Replication modes

| Players | Active strategy | Constraints |
|---|---|---|
| 2 | Copy the right source half to the opposite half with both coordinate axes reflected | Literal matrix copying; square playable area |
| 4 | Reflect a source quarter to a half, then the resulting half to the full map | Literal matrix copying; square playable area and horizontal/vertical sector boundaries |
| 3 | Rotate centers in raw X/Y, and rotate local source hex-cell offsets in 60° increments | Exact **local** hex transformation; not a globally exact X/Y rotation |
| 5 | Rotate placement centers by `n × 72°` in X/Y **and the entire mask by `n × 72°`**, then rasterize, repair and validate | Whole-mask rotation is mandatory; no old path or UI selector |
| 6 | Rotate centers in raw X/Y, and local hex offsets by multiples of 60° | Exact local hex transform combined with approximate global positioning |

The five-player method calls `RotateMask`, which preserves the source mask's number of active cells and repairs rasterization artifacts to maintain hex-connectedness. `resolveRotatedMask` additionally tests the resulting placement against the destination sector, neighboring objects and available space. For all circular layouts, position transforms take place around the actual center `(size-1)/2`. A perfect 72° automorphism of the regular hex lattice does not exist, so **resource parity and connectedness**, not exact cell-for-cell global rotation, are the acceptance conditions.

The legacy API field `rotateResourceMasks` is deliberately ignored, regardless of whether it is `true` or `false`. The removed `useOldCircularMethod` setting is also ignored. There is one active algorithm per player-count mode. These fields are omitted from newly saved UI projects but older JSON can still be loaded.

## Geometry, previews and terrain

- The circular playable region is a circle in **raw X/Y coordinates**. The optional `Hexagonal Projection` view (`q+r/2`, `r×sqrt(3)/2`) is visualization only and can make that circle look elliptical. Generated map geometry and SWD bytes remain unchanged.
- Rectangular source sampling for 2/4 players uses the available square map area, rather than constraining it to a circle. Lakes are kept separated across mirrored sector seams.
- Minerals cannot overlap, although resource-free outer mountain texture can overlap between nearby mining regions. New mining sites are sampled without the old preference to maximize overlap with existing mountain ground.
- The default land height is 1, water height is 0; levels 40 and 100 remain selectable for older maps. The height-100 setting is experimental.
- The preview canvas offers Terrain, Terrain + Deposits and Deposits Only. Forest cells containing wildlife use a dark green preview color. The preview and comparison table are derived from the exported SWD data, not from a separate generator.
- A generation error is displayed in a non-dismissible, high-contrast red `<div>` at the top-middle of the center preview panel (~10rem from its top) and clears on the next successful generation.

## SWD format and APIs

The codec writes a **2352-byte header**, followed by 14 blocks each containing a 16-byte descriptor and `width × height` payload bytes, and a final byte. Both the header and blocks encode the selected dimensions. Player HQ positions and relevant object markers are serialized together.

Relevant implementation files: `swd_codec.go`, `swd_codec_type.go`, `swd_validation.go`, `swd_areas.go`, `model_adapter.go`. Reference: https://settlers2.net/documentation/world-map-file-format-wldswd/ . External compatibility with The Settlers II: 10th Anniversary editor has not been verified in this environment.

`API/server.go` exposes `POST /api/map`, `POST /api/download` and `POST /api/save-and-run`. `Save and Run` writes an atomic replacement under the host machine's Downloads directory, then attempts OS-associated opening of `.swd`; a launch failure is reported to the client. The browser cannot choose an arbitrary server filesystem path.

## Testing, reproducibility and non-goals

Tests in `tests/` exercise mask rotations, shape connectivity, request validation, resource equality for 2–6 players, dimensions, SWD parsing, variant invariants and HTTP behavior. The Go commands are `go test ./...` and `go vet ./...`; the browser script can be syntax-checked with `node --check static/code.js`.

A fixed seed is expected to produce repeatable output **within this version** for identical settings. It is not intended to be byte-identical to archived algorithms. Infeasible placement produces HTTP 422 and a visible UI error rather than a truncated or partially exported map. Native S2M export and a verified in-game integration test are outside the currently demonstrated implementation.
