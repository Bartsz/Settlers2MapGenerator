# Product Requirements Specification (PRS)

**Product:** Settlers II Map Generator — Go edition  
**Document version:** 1.1, reviewed 2026-10-19  
**Target platform:** Windows 10/11 x64; Go application with a localhost browser UI  
**Export:** `.swd` maps for use with the The Settlers II map-editing ecosystem

## 1. Product overview

The generator is intended to create fair, playable and reproducible multiplayer maps with symmetric player sectors. Creating such maps manually is labor-intensive, and independent random generation often results in unequal deposits, disconnected clusters, unusable building terrain, lakes or headquarters placed unevenly and difficulty changing only the mineral configuration.

**Core principle: Generate once, replicate for all players.** Generate one source sector and derive all others from it. Equal resource quantities and connectedness are mandatory; exact global hex-cell rotation is not always mathematically possible (notably at 72° for five players).

### What is historical, and what is current?

The original specification dated 2026-10-17 described a **fixed 150×150** SWD file, **3–6 players**, a few UI-specific defaults, and optional rotation modes introduced during development. These are historical baselines and **are superseded by the implementation described here**: physically variable 64–140-square maps, 2–6 players, mandatory rotated masks for five players, and an English-language UI. This document distinguishes **implemented / automatically tested**, **design acceptance requirements**, and **requires manual in-game verification**.

## 2. Current product parameters

| Property | Current behavior |
|---|---|
| Physical SWD size | Exactly the selected `size × size`, rather than fixed 150×150 |
| 2/4-player maps | Even width and height, 64–140; fully usable square; mirrored sections |
| 3/5/6-player maps | Odd width and height, 65–139; circular playable region in raw X/Y, mountains outside |
| Playable-sector replication | A single source sector copied to remaining players |
| Circular HQ radial position | 10–90% of radius, default 55%; angular deviation ±10% of sector angle |
| General land height | 1 (legacy 40; experimental 100) |
| Water height | 0 |
| Mine-ground height | 6 at land base 1; 2-cell terrain-height transition |
| Outer mountain height | 12–20 at land base 1; 4-cell transition with deterministic noise |
| Coal, iron, gold, mined granite | Each independently configurable, 0–1000 tiles and 0–40 clusters per player |
| Surface stones | Separate surface objects, 0–1000 tiles, 0–40 clusters per player |
| Empty mining terrain | 0–1000 tiles and 0–40 clusters per player; no ore |
| Mining terrain expansion | 0–8 full hex rings around each mineral cluster, default 3 |
| Lakes | 0–10 per player, each 1–120 tiles |
| Grouped forest | 0–1000 tiles and 0–40 clusters per player |
| Wildlife | 0–1000 objects, 0–40 clusters per player, located in generated forests |
| Additional forest fill | 0–85% of eligible remaining meadow tiles |
| Mineral values | Value 7 per occupied mineral cell |
| Project persistence | JSON, browser projects v2/v3 readable, UI saves v3 |
| Map export | SWD with 14 blocks, not native S2M |
| Five-player replication | Always rotates whole shapes and their centers by `n × 72°`, with raster repair |
| Failure reporting | HTTP 422 for impossible generation; English red error overlay in the preview |

## 3. Functional requirements and acceptance criteria

### US-01 — Standalone Windows application

**User need:** Generate maps without installing an external engine or a language runtime.

- AC-01.1: Build the program for Windows 10/11 x64.
- AC-01.2: A compiled EXE does not require a separate `engine.exe`, Python or a Go installation.
- AC-01.3: Serve the embedded HTML/CSS/JavaScript on a local HTTP listener and open the default browser.
- AC-01.4: `go build` and `go test ./...` should succeed from the complete source tree.

**Status:** Source and cross-compilation can be automatically verified. Real Windows/browser execution still requires manual confirmation.

### US-02 — Configurable map geometry

**User need:** Select a real physical map dimension, not merely the diameter of an area inside a fixed SWD.

- AC-02.1: SWD header and each of its 14 blocks must use the selected square dimensions.
- AC-02.2: For 2/4 players use even sizes 64–140 and square playable coverage.
- AC-02.3: For 3/5/6 players use odd sizes 65–139, with a circular playable boundary in raw X/Y and mountain terrain outside.
- AC-02.4: The raw-grid preview must correspond to actual map geometry; hex projection is a distinct, optional visualization, not a generation transform.
- AC-02.5: The same dimensions must be reflected in preview, downloadable SWD and Save and Run output.

**Status:** Implemented, with structural checks. Historical fixed-size constraints do not apply.

### US-03 — Symmetry and equivalent player resources

**User need:** A symmetric and quantitatively fair multiplayer environment.

- AC-03.1: Support 2, 3, 4, 5 and 6 players.
- AC-03.2: Create resource clusters in a single template sector.
- AC-03.3: For two players, reflect the source half horizontally and vertically into the opposite half.
- AC-03.4: For four players, reflect a source quarter twice to fill the entire map.
- AC-03.5: For three and six players, rotate sector centers and local hex shapes with 60°-step hex transformations.
- AC-03.6: **For five players, always rotate complete masks and their centers by `n × 72°`;** preserve active tile count and connectivity after rasterization.
- AC-03.7: Validate coal, iron, gold, granite, lake counts and sizes, surface stones, fish and animals for all sectors on final map data / serialized SWD.
- AC-03.8: If requested parity cannot be achieved, return an error, not a successful partial map.

**Status:** Current code implements these methods. Exact global hexahedral rotation at 72° is not possible; connectedness and equal quantities are tested instead.

### US-04 — Headquarters / starting positions

**User need:** A usable starting location within the assigned sector.

- AC-04.1: Each player has exactly one HQ marker and a matching start coordinate in the SWD header.
- AC-04.2: For circular maps, place the source HQ at user-selected 10–90% of radius, allowing ±10% of the player sector's total angular width.
- AC-04.3: HQ positions must not overlap water, mines or other objects; maintain a 3-hex-cell protected neighborhood.
- AC-04.4: Provide buildable terrain near HQ; relocate unsafe candidates within the sector.
- AC-04.5: Reject output if any required HQ is missing.

**Status:** Validated programmatically. Actual building usability needs game-editor verification.

### US-05 — Buildable ground and mine terrain

**User need:** Productive terrain instead of large, inaccessible and resource-free mountains.

- AC-05.1: Use meadow/buildable textures in the available interior where not explicitly occupied by mountains, water or objects.
- AC-05.2: Place mineral resources on mining terrain; use a terrain halo beyond the ore core.
- AC-05.3: Expose mining ground expansion as 0–8 full hex rings (default 3).
- AC-05.4: Allow explicitly requested **empty mountain clusters** without mineral content to create uncertainty for players.
- AC-05.5: Apply rocky terrain below surface stones.
- AC-05.6: Modifying a mineral variant must not change terrain footprints.

**Status:** Implemented, but physical mine-building accessibility needs game testing.

### US-06 — Terrain heights and smooth transitions

**User need:** Predictable, relatively flat building land and visually distinct mountains.

- AC-06.1: Water defaults to height 0 and regular land to 1.
- AC-06.2: Mine and empty-mountain terrain reaches height 6 with a two-cell transition, relative to base land 1.
- AC-06.3: Circular layouts' outside mountains reach 12–20 with a four-cell transition and seeded height noise.
- AC-06.4: Adjacent cells should avoid unintended abrupt height gaps.
- AC-06.5: Legacy land levels 40 and 100 remain selectable; 100 is experimental.
- AC-06.6: Changing land level must not move resources, lakes or HQs.

**Status:** Height generation/smoothing exists and can be tested numerically; the editor's visual rendering needs manual review.

### US-07 — Compact, connected mineral clusters

**User need:** Natural, buildable ore concentrations with independently configurable quantities.

- AC-07.1: Coal, iron, gold and mined granite accept 0–1000 occupied cells per player and 0–40 clusters per type.
- AC-07.2: Clusters begin as one cell and expand through connected neighbor cells.
- AC-07.3: A single cluster contains one mineral type at a time; each active mineral tile has value 7.
- AC-07.4: Do not overlap ore cores or silently mix separate types.
- AC-07.5: Choose centers across the viable sector without a scoring preference that causes ore deposits to bunch together.
- AC-07.6: Do not silently reduce requested counts when a placement cannot fit.

**Status:** Implemented. Unusually dense requests may fail after deterministic retries with a user-visible explanation.

### US-08 — Surface stones

**User need:** Distinguish surface stone gathering from mined granite.

- AC-08.1: Treat surface stones as a separate map object/resource type.
- AC-08.2: Provide its own count and cluster controls; 0 clusters disables new stone groups.
- AC-08.3: Use connected masks on suitable land with rocky ground, avoiding HQs, water and mining cores.
- AC-08.4: Copy equal quantities to every player sector.
- AC-08.5: Preserve terrain when changing active stone counts in variants.

**Status:** Implemented and covered by automated tests.

### US-09 — Lakes and fish

**User need:** Predictable lake counts and sizes, with valid fish placement.

- AC-09.1: Configure 0–10 lakes per player, each 1–120 tiles.
- AC-09.2: Each generated lake uses a connected hex-neighbor mask.
- AC-09.3: Prevent unrequested merging of lakes, including at square mirror seams.
- AC-09.4: Only place fish in generated lakes.
- AC-09.5: Report a placement error when no valid location remains.

**Status:** Implemented, with count and connectedness checks. Actual fishing behavior needs in-game verification.

### US-10 — Grouped forests and wildlife

**User need:** Control access to timber and hunting without blocking essential terrain.

- AC-10.1: Forests use configurable tile and cluster counts and a shared mask generator.
- AC-10.2: A separate 0–85% slider fills otherwise empty eligible meadow cells with trees.
- AC-10.3: Wildlife uses configurable tile and cluster counts and may be placed only in available forest groups.
- AC-10.4: Do not place animals on water, mined regions, surface-stone objects or HQ tiles.
- AC-10.5: Display forest cells with wildlife in dark green in the relevant preview modes.
- AC-10.6: Maintain per-player wildlife counts after replication.

**Status:** Implemented, with placement and balance validation.

### US-11 — Deposit variants without regenerating geometry

**User need:** Change ore assignment after obtaining a suitable map layout.

- AC-11.1: Store each generated cluster's identity and available capacity.
- AC-11.2: Present a right-side variants drawer, with one column per cluster and one row per variant.
- AC-11.3: Add a variant by copying the selected one.
- AC-11.4: Permit supported mineral type and quantity changes up to the cluster capacity.
- AC-11.5: Keep surface stone separate from underground mineral types; do not permit invalid direct swaps.
- AC-11.6: Preserve heights, lakes, HQs, mountain terrain, and other geometry across variants.
- AC-11.7: Update the preview and export to reflect the active variant.

**Status:** Implemented. The map preview is generated again from the selected configuration while the terrain geometry is kept invariant.

### US-12 — Map preview

**User need:** Inspect terrain and resource availability before export.

- AC-12.1: Provide Terrain, Terrain + Deposits, and Deposits Only modes.
- AC-12.2: Show distinct resource colors and a legend, HQ markers, sector boundaries and the playable boundary.
- AC-12.3: Provide Raw Map Grid and optional Hexagonal Projection, with projection acting only on visualization.
- AC-12.4: Derive preview data from the final SWD rather than a separate random map.
- AC-12.5: Show a red, accessible, non-dismissible error inside the center preview panel at approximately 10rem from the top upon generation failure; clear it on successful generation.
- AC-12.6: Present all UI text, errors, controls and generated statistics in English.

**Status:** Implemented. The local browser experience should be checked manually on Windows.

### US-13 — Player resource balance table

**User need:** See whether every player has equivalent quantities.

- AC-13.1: Include one row per player and separate columns for lakes, water, coal, iron, gold, granite, surface stones, fish and wildlife.
- AC-13.2: Calculate the results from generated map data and the final SWD.
- AC-13.3: Report unavailable values rather than outputting JavaScript `undefined`.
- AC-13.4: Recalculate when settings and active variants change.
- AC-13.5: Indicate when any counted resource differs across players.

**Status:** Implemented and covered by parity tests.

### US-14 — Save and reload project JSON

**User need:** Resume editing a map and its variants later.

- AC-14.1: Save seed, map settings, cluster information, variants and selected variant as JSON.
- AC-14.2: Restore those values and preview after load.
- AC-14.3: Reject malformed or unsupported project files with an understandable message.
- AC-14.4: Support existing UI project versions 2 and 3; emit version 3.
- AC-14.5: Ignore the old `rotateResourceMasks` value on import: five-player full-mask rotation is always active.
- AC-14.6: Ignore retired `useOldCircularMethod` settings.

**Status:** UI save/load logic exists. Older projects may produce different map bytes due to generation algorithm changes.

### US-15 — SWD export and game compatibility

**User need:** Produce a map the target editor can open and use.

- AC-15.1: Export `.swd`, not an unsupported native `.s2m` file.
- AC-15.2: Write a valid native header, selected dimensions and 14 data blocks.
- AC-15.3: Encode HQ markers, textures, heights, resources, lakes and other objects through the SWD adapter.
- AC-15.4: Export must match the currently displayed configuration, without silently choosing a new seed.
- AC-15.5: Structural SWD validation must pass before delivery.
- AC-15.6: Save and Run writes to the application's host Downloads directory and requests associated-editor opening.
- AC-15.7: A real Windows editor must successfully open the SWD and the game must handle the generated terrain and resources.

**Status:** The source implements structural validation, endpoint export and Save and Run. **AC-15.7 remains unverified without manual Windows game/editor tests.**

### US-16 — Reproducibility, validation and errors

**User need:** Repeat known layouts and see why requested layouts fail.

- AC-16.1: Identical seed and settings produce reproducible SWD bytes within the same release.
- AC-16.2: Offer a New Map Layout action that changes the seed.
- AC-16.3: Reject out-of-range quantities, dimensions, player counts and unsupported height values.
- AC-16.4: Retry difficult layouts deterministically but do not reduce requested resources without consent.
- AC-16.5: Return HTTP 422 on impossible generation; show the English error in the central red preview alert.
- AC-16.6: Do not offer a partially generated SWD as though successful.
- AC-16.7: Retain CLI generation as an alternative to the browser.

**Status:** Implemented. Stress testing can improve confidence in extreme input combinations but cannot guarantee feasibility of every valid range combination.

## 4. Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| NFR-01 | Standalone Go application, no external binary map engine | Build/source inspection |
| NFR-02 | All generation code maintained as Go source | Source review |
| NFR-03 | Deterministic result for fixed seed/settings in one version | Automated regression tests |
| NFR-04 | Windows x64 cross-compilation | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build` |
| NFR-05 | Basic generation accessible through the local English-language UI | Browser/manual acceptance |
| NFR-06 | Consistency across preview, statistics, variants and SWD | Integration/regression tests |
| NFR-07 | Test coverage for 2–6 players, connectivity, dimensions, exports and validation | `go test ./...` |
| NFR-08 | No cloud dependency; HTTP bound to local host | Source review and local checks |
| NFR-09 | Clear failure instead of corrupted map or missing requested resources | 422/validation tests |
| NFR-10 | Imports remain tolerant of retired JSON rotation settings | Backward-compatibility tests |

## 5. Reference default settings

These values represent defaults exposed by the current UI; they are **not** a claim that every resource combination fits every smaller map.

| Property | Default |
|---|---|
| Players / symmetry | 6 |
| Physical SWD size | 109 × 109 |
| Seed | 123456 |
| Height, land/water | 1 / 0 |
| Circular HQ start radius | 55% |
| Mining terrain expansion | 3 rings |
| Lakes | 3/player × 25 tiles |
| Coal | 150 tiles / 3 clusters per player |
| Iron | 50 tiles / 2 clusters per player |
| Gold | 20 tiles / 1 cluster per player |
| Mined granite | 40 tiles / 2 clusters per player |
| Surface stone | 30 tiles / 2 clusters per player |
| Forest clusters | 100 tiles / 4 clusters per player |
| Wildlife | 40 / 3 clusters per player |
| Empty mountain terrain | 0 tiles / 0 clusters |
| Extra fill with trees | 27% |

## 6. Definition of Done

The program can be accepted for production use once the following have been demonstrated:

- [x] Source builds in Go and does not depend on an external map engine.
- [x] Domain-model generation and structural SWD export exist.
- [x] Map dimensions follow the requested size.
- [x] Repeated sectors are generated from one source and measurable parity is validated.
- [x] SWD headquarters and start-coordinate layers are written.
- [x] Mask-based, connected clusters, lake generation and surface stone/forest/wildlife logic exist.
- [x] Mining terrain expansion, height bands, variants and JSON project support exist.
- [x] English UI and visible central generation errors are implemented.
- [ ] Verify the packaged EXE and full UI workflow on Windows 10/11.
- [ ] Confirm SWD opens in the actual The Settlers II: 10th Anniversary editor.
- [ ] Confirm buildings, mines, fishing and roads work correctly during gameplay.

Checked entries mean **implementation exists and/or is covered by structural tests**, not that every edge case has been proved.

## 7. Known constraints and historical supersessions

- The 2026-10-17 original PRS used fixed 150×150 maps and only 3–6 players. Current code exports the requested physical size and supports 2–6 players.
- Earlier 3/6-player old/new replication selectors have been removed; only the local hex-mask method remains.
- Earlier five-player `Rotate resource masks` toggle has been removed; complete-mask rotation is always active. Old JSON keys are tolerated but ignored.
- Hexagon-cell-perfect global fivefold symmetry is not a valid mathematical requirement for a 72° rotation of a hex lattice. The contract requires equal counts, connectivity and valid placement after raster repair.
- A 109×109 circular map represents an actual round playable area in raw X/Y surrounded by mountain terrain, rather than requiring a permanent 150×150 border.
- Structural SWD compliance does not establish end-to-end compatibility with the target game. Native S2M output is not implemented.
- Project/seed compatibility means reloading and reproducing **within the current generator version**, not byte-for-byte equality with earlier versions.

## 8. Development priorities

| Priority | Area |
|---|---|
| P0 | Editor opens SWD; valid HQs; sector parity; correct mine and road/building terrain |
| P1 | Cluster connectivity, lakes, heights, robust preview and error messages |
| P2 | Variant editing, JSON persistence, editor ergonomics and backward compatibility |
| P3 | Optional S2M export, new biomes and further terrain controls |

This specification describes a reproducible, balanced multiplayer map generator, not a general-purpose random-terrain generator. The most important measurable invariant is fair resource allocation across players, subject to valid clusters, usable land and structurally correct SWD export.
