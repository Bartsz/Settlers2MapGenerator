# Changelog

This file summarizes implemented changes. Dates and release labels of earlier iterations are retained for context; they are **not** a claim of independent Windows/editor validation.

## English UI and mandatory five-player mask rotation — 2026-10-19

- Removed the `Rotate resource masks` checkbox and eliminated the alternate five-player replication branch. For five players, placement centers and full masks always rotate by successive 72° increments; the `RotateMask` and destination-placement repair/validation logic remains active.
- Removed `RotateResourceMasks` from the validated `MapRequirements` and from new JSON UI payloads. The obsolete `rotateResourceMasks` field in saved projects and external requests is ignored for backward compatibility; true/false produce the same five-player map.
- Moved generator errors and warnings into a persistent red alert inside the center preview panel, horizontally centered about 10rem below the panel top. The alert is not dismissible and clears on successful generation.
- Translated the visible HTML, JavaScript-generated controls, reports, status messages and generator placement errors to English. `lang="en"` identifies the page language.
- Rewrote `README.md`, `ARCHITECTURE.md`, `CHANGELOG.md` and `requirements.md` in English and marked historical requirements that do not describe the current version.
- Added regression coverage ensuring legacy rotation settings cannot alter a five-player SWD.

## Final Rotation — 2026-10-17 (historical)

- Added a configurable 0–8-ring mining-terrain expansion, default 3.
- Introduced optional five-player whole-mask rotation, preserving tile count and connectivity. **Superseded above:** this option is now mandatory.
- Kept the new 3/6-player local hex-mask rotations and eliminated selection of earlier circular algorithms.

## Rotation Update — 2026-10-17 (historical)

- Reverted mineral cluster placement to randomized valid centers; eliminated scoring that caused most ore regions to overlap nearby mountain footprints.
- Added circular HQ distance at 10–90% of radius (default 55%), with angular deviation of ±10% of player-sector angle.
- Added deterministic retries (up to 36 attempts in the generator entry point) for difficult placements. Infeasible configurations return an error rather than reducing quantities.

## Height & Geometry Refinement — 2026-10-17

- Circular playable boundary computed in raw X/Y instead of a projection that distorted circles in the raw view.
- Outer mountain height 12–20 at base land height 1, with four-cell transition and seeded variation.
- Mining/empty-mountain height 6, two-cell transition; adjacent heights are smoothed.
- Expanded mining ground independently of the ore core and kept it stable across variants.
- Allowed square layouts to use their full playable half/quarter rather than circular placement limits.
- Preserved the chosen amount of surface stones in mirrored/rotated variants without changing their rocky ground.

## MapTuning — 2026-10-17

- Made actual SWD dimensions match the size slider, with even sizes for 2/4 players and odd sizes for 3/5/6.
- Added square-sector full-matrix mirroring, symmetry lines and safeguards against lakes joining at mirror seams.
- Reduced protected HQ surroundings to three hex steps.
- Added empty mining-mountain groups, rocky texture under surface stones, grouped forest and wildlife in forest cells, with dark green preview coloring.
- Added common tile-count/cluster-count UI controls for wildlife and empty mountains.

## Domain Model and Unified Shapes — 2026-10-17

- Split the application into `DomainModel`, `InternalMapManipulation`, `API`, `static` and `tests`.
- Replaced the embedded `interface.go` source string with HTML, CSS and JavaScript included through `embed` at build time.
- Added typed domain layers, a shared connected-mask generator, validation and embedding operations, and SWD conversion at the serialization boundary.
- Kept `/api/map`, `/api/download`, local project JSON with variants and the Save and Run endpoint.

## Remaining limitations

- Fivefold hex symmetry requires rasterization and connected-shape repair; it cannot exactly preserve every hex tile under 72° rotation.
- Certain dense configurations cannot fit and return a placement error.
- Successful SWD structural validation and Windows cross-compilation do **not** prove successful import or gameplay in The Settlers II: 10th Anniversary. Manual editor testing remains necessary.
