# Johnny Castaway 2026 — TODO for 2026-07-16

## Current priorities — execute top to bottom

1. [x] **Make the public source/data boundary truthful.** The source-only
   snapshot excludes and ignores Sierra/Dynamix archives and WAVs, loads them
   from a persisted `--data-dir`, treats sound as optional, and verifies both
   canonical archive MD5 values. A graphical folder picker remains future UI.
2. [x] **Use a compatible distribution license.** The source snapshot uses GNU
   GPL version 3 or later, preserves deckarep/jc_reborn attribution, includes
   `LICENSE` and `NOTICE.md`, and explicitly excludes Sierra/Dynamix data from
   the source license.
3. [x] Make Escape predictable everywhere: the first press closes an open
   overlay and arms quit; a second press within 1.5 seconds exits the desktop,
   fullscreen, or screensaver application. Keep the complete shortcut guide in
   the in-app footer/settings instead of duplicating every key in the README.
4. [x] Create the new public GitHub repository `DrWize/JohnnyCx86` with a clean
   source-only history containing no Sierra/Dynamix data or generated executable
   history. Preserve upstream attribution in README/NOTICE instead of importing
   the old asset-bearing Git history.
5. [x] Add a true Windows x86 `.scr` target to JohnnyCx86: build `GOARCH=386`,
   provide screensaver `/s`, preview `/p HWND`, and configure `/c` modes, add a
   32-bit Raylib/CGO toolchain and CI job, and test on supported 64-bit Windows
   through WOW64. `build\build_x86.bat` now produces a PE32
   `JohnnyCastaway-x86.scr`; `/s`, `/p HWND`, `/p:HWND`, `/c`, and `/c:HWND`
   are parsed and covered by regression tests, preview mode reparents the Raylib
   HWND into the Windows preview host, and the x86 CI job verifies `windows/386`
   metadata. Local QA on 64-bit Windows verifies all three modes and real child
   window parenting without executing the downloaded unsigned reference.
   Treat the downloaded unsigned `Screen Antics.scr` only as a behavior/PE
   reference; never execute or redistribute it.
6. [ ] Update [DrWize/home](https://github.com/DrWize/home) after JohnnyCx86 is
   public. Add the active projects, repository links, screenshots/build status,
   data-file policy, and a short description of the Windows x64/x86 work.
7. [x] Persist session settings: CRT mode, scaling mode, story order, normal
   window/fullscreen mode, audio mute, stretch/4:3 fit, and selected monitor.
   Explicit command-line choices override saved values; screensaver/test launches
   do not overwrite normal window preferences.
8. [x] Switch between Full Story and individual TTMs inside the existing Raylib
   window. Retain fullscreen placement, HWND, audio, shaders, compositor, menu,
   and runtime history while releasing only content-owned layers and sprites.
9. [x] Add the original low-cost Fast CRT shader mode with brightness-aware
   scanlines, adjustable sharpness, and a lightweight aperture mask.
10. [ ] Add a shader benchmark/capability view and test 1920x1080, 3840x2160,
   5120x1440, and 7680x2160 before selecting automatic quality defaults.
   The persisted F8 HUD is complete: it shows actual FPS, CPU frame-submission
   time and budget, active CRT/scaling modes, resolution, measured CPU impact,
   and expected GPU cost. F9 now performs an automatic four-mode comparison at
   the current output resolution, restores the original mode, and presents the
   results for twenty seconds. Physical/native-resolution runs remain pending.
11. [ ] Evaluate CRT-Easymode as the flat, sharp high-resolution alternative.
12. [ ] Evaluate CRT-Geom as an optional curved-tube mode for faster GPUs.
13. [ ] Decide whether to create original replacement artwork for the unavailable
   `FLAME.BMP` and `FLURRY.BMP` effects.

## Tomorrow's priorities

1. **Run a stability sweep across every embedded TTM and Full Story — completed 2026-07-15.**
   `build\stability_sweep.ps1` now inventories the real resource map and checks
   startup, repeated `T` scene navigation, `N` switching/wrapping, settings,
   the runtime log, and Full Story. All 41 TTMs wrapped through every selectable
   scene (up to 48 unique scenes in one TTM). The latest clean pass logged
   2,460 advances, 41 content switches, and all 41 TTMs wrapping, with no fatal
   resource or slice-bounds error and a healthy 25-second Full Story run.
2. **Add automated tests — completed 2026-07-15.**
   `main_test.go` covers command-line parsing, TTM scene navigation,
   next-TTM wrapping, string-bearing opcodes (`0xF` argument marker), missing
   fire sprites, and short-screen padding. `build\test.bat` runs the suite with
   the supported Windows CGO toolchain.
3. **Fix graphics resource cleanup — completed 2026-07-15.**
   Render textures, temporary screen textures, scene layers, and sprite slots
   now unload before ownership is cleared. The 41-TTM stress sweep passed 2,460
   scene advances and 41 teardown/reload switches after the change.
4. **Improve application logging — completed 2026-07-15.**
   Each session now starts with a separator, product version, and Git build
   identifier. `JohnnyCastaway.log` rotates to a single backup at 1 MiB.
5. **Validate every short `.SCR` background — completed 2026-07-15.**
   `JOFFICE.SCR`, `ISLAND2.SCR`, `ISLETEMP.SCR`, and `THEEND.SCR` were decoded,
   padded, and visually inspected. Black continues the office/end-card framing
   and blue continues the island water without visible seams, so no per-screen
   overrides are needed. The regression suite now checks every embedded short
   screen for valid dimensions, data bounds, and opaque padding.

## Confirmed remaining problems

- [ ] `FLAME.BMP` and `FLURRY.BMP` are absent from the verified original
  archives. The matching Apple Silicon app and known public implementations
  contain no alternate bitmap data. `FIRE.TTM` continues safely, but those two
  visual effects are omitted; a replacement would be new artwork rather than
  a recovered original resource.

## Reference comparisons

- [x] Establish [deckarep/Johnny-Castaway-2026-Public](https://github.com/deckarep/Johnny-Castaway-2026-Public)
  as the direct Git upstream. The local history already contains its merged PR
  and final timing fix through `ba5ae61`; the corrected `grUpdateDelay * 0.01`
  timing and 800-pixel fade radius remain in place. Retain its canonical hash
  and resource-format documentation, but do not copy or republish its tracked
  WAV files and do not infer redistribution rights from its missing license.
  Keep the [upstream pull-request index](https://github.com/deckarep/Johnny-Castaway-2026-Public/pulls)
  as a development-history reference; its merged PR #1 is already in the local
  ancestry.
- [ ] Revisit deckarep's macOS, Linux, and WebAssembly targets after the Windows
  x64/x86 release gates pass. Compare its menu-bar idle-timeout fallback with a
  native screensaver, and plan platform-specific CGO/Raylib builds rather than
  assuming the current Windows target cross-compiles unchanged.
- [ ] Compare long-run scene order, timing, day/night transitions, and visual
  output against the [11-hour recording](https://www.youtube.com/watch?v=l8D6qppreiI).
  Use it only as an observation point, never as a bundled or downloaded asset.
- [ ] Compare the Linux video-wrapper tradeoffs in
  [vasartori/jhonny-castaway-screensaver](https://github.com/vasartori/jhonny-castaway-screensaver):
  random starting offsets, day/night ranges, December selection, XScreenSaver
  embedding, and the difference between video playback and live emulation.
- [ ] Compare appearance and observable behavior with
  [castaway.xesf.net](https://castaway.xesf.net/) across representative scenes.
- [ ] Review [tallPete/JohnnyCastaway](https://github.com/tallPete/JohnnyCastaway)
  regressions against this engine: 10 TTM slots, multi-hour CPU spin/freeze,
  clip-zone enforcement, walk to Spot.A, scene-start flicker, cloud Y placement,
  intro failure handling, settings persistence, and one-monitor audio. Compare
  its resource/engine/renderer/screensaver package split and current 104 engine
  plus 11 renderer tests with gaps in this Go test suite.
- [ ] Use [alexbevi/xbak](https://github.com/alexbevi/xbak) only as a historical
  GPL-3.0 archive/parser reference; compare resource bounds and decompression
  assumptions without copying assets or incompatible code blindly.

## Future display work

### CRT shader priorities

1. [x] Add a single-pass, selectable CRT-Lottes mode as the first high-quality
   shader. Port the public-domain GLSL to Raylib, keep it independent from the
   scaling setting, persist the selection, and retain both Off and Lightweight.
2. [x] Keep the settings and runtime-log overlays usable in screensaver mode.
   `F1` should reveal the available controls while animation continues behind
   the overlay; recognized control keys and menu mouse input must not terminate
   the screensaver, while ordinary input should retain normal exit behavior.
   A compact footer now shows all primary shortcuts and live display settings
   for ten seconds, fading during the last two seconds; a control key reveals
   it again without pausing playback.
3. [x] Add a very-low-cost Fast CRT mode using an original Raylib-native shader
   with the zfast feature set: brightness-aware scanlines, adjustable sharpness,
   and a lightweight aperture mask. Do not copy GPL source unless the project
   first adopts a compatible distribution license.
4. [ ] Consider CRT-Easymode after Lottes for a flat, sharp 1080p/4K preset with
   configurable beam width, gamma, brightness, and RGB mask.
5. [ ] Consider CRT-Geom as an optional curved-tube preset, with curvature,
   rounded corners, overscan, and interlace disabled by default on slower GPUs.
6. [ ] Add shader capability fallback and a benchmark overlay, then measure all
   modes at 1920x1080, 3840x2160, 5120x1440, and 7680x2160 before choosing final
   defaults. Render only the centered 4:3 viewport, not the 32:9 pillarboxes.

- [x] Evaluated [CRT-Royale](https://emulation.gametechwiki.com/index.php/CRT-Royale)
  as a future experimental shader. The reference preset requires 12 passes and
  six LUT textures, while CRT-Royale Fast still requires eight passes and three
  LUTs. Neither libretro `.slangp` pipeline is directly loadable as a Raylib
  GLSL shader. Retain the lightweight overlay as the default; a future prototype
  should port CRT-Royale Fast, capability-gate it, and benchmark 4K/dual-4K.
- [x] Add a selectable [Scale2x](https://www.scale2x.it/algorithm) pixel-art
  filter. Produce a sharp edge-aware 1280x960 intermediate image from the
  640x480 canvas before final monitor scaling, while retaining nearest-neighbor
  and bilinear modes for comparison and lower-powered systems.
- [x] Add explicit 32:9 fullscreen and screensaver support through 7680x2160,
  covering Samsung Odyssey Neo G9-class displays. Preserve the original 4:3
  artwork by default, center it correctly across monitor origins, handle the
  large side regions intentionally, and verify borderless/topmost placement,
  cursor hiding, input-to-exit behavior, and multi-monitor selection at both
  5120x1440 and 7680x2160.

## Solved

- [x] Cleaned the project to the files required for the Windows x64 build.
- [x] Added a repeatable x64 build script and automatic temporary `.syso`
  cleanup.
- [x] Built a self-contained PE32+ AMD64 executable with embedded data, audio,
  icon, manifest, and version metadata.
- [x] Added Windows command-line options, error dialogs, logging, monitor
  selection, and single-instance behavior.
- [x] Added the settings panel and made an outside left-click or the first
  `Esc` close it; a second `Esc` within 1.5 seconds quits the application.
- [x] Added `N` for the next TTM and `T` for the next scene/tag inside a TTM.
- [x] Added `H` to preview Halloween, St. Patrick, Christmas, and New Year
  overlays without changing the system clock.
- [x] Added CRT, scene-order, and smoothing controls.
- [x] Added an `F5` foreground runtime log with active TTM, scene/tag, byte
  offset, decoded opcode, bounded history, scrolling, and live highlighting.
- [x] Fixed the runtime tracer crash on string opcodes using the `0xF` argument
  marker.
- [x] Prevented the missing `FLAME.BMP` and `FLURRY.BMP` resources from
  terminating Full Story or direct `FIRE.TTM` playback.
- [x] Removed black lower bands from short `.SCR` backgrounds by padding to
  640x480 with the dominant bottom-edge color without stretching artwork.
- [x] Added an automated regression suite and repeatable Windows test runner.
- [x] Audited GPU resource ownership and added guarded texture/layer cleanup.
- [x] Added versioned session logging with a 1 MiB rotating backup.
- [x] Validated padding for every short `.SCR`; no overrides were required.
- [x] Locked runtime data to the verified embedded archives, enforced both
  hashes during tests/builds, decompressed every resource in regression tests,
  and added narrow RLE/LZW bounds guards for clear corruption failures.
- [x] Added full ADS decoding, unified ADS/TTM history, and a complete ADS
  script view with the executing condition/instruction highlighted.
- [x] Added runtime search, source filters, capture pause, clipboard copy,
  timestamped export, and history clearing.
- [x] Persisted CRT, smoothing, and scene-order settings between runs.
- [x] Added the product/build identifier to settings and error dialogs.
- [x] Updated the README with build, controls, runtime-log, missing-resource,
  and background-padding behavior.
- [x] Implemented `DRAW_SCREEN` (`0xB606`) as the DGDS buffer copy operation,
  including rectangle clipping, render-texture orientation, and regression
  coverage for all six embedded uses.
- [x] Replaced provisional ADS block flags with tested AND/OR evaluation,
  `IF_NOT_PLAYED` tracking, asynchronous `IF_FINISHED` handling, and the
  `WHILE_RUNNING` continuation used by `ACTIVITY.ADS`.
- [x] Added a linted Windows x64 GitHub Actions workflow that verifies the
  embedded archives, runs tests, builds and checks the amd64 executable, and
  uploads it as a workflow artifact.
- [x] Made the stability sweep reject concurrent test instances so its
  single-instance and content-switch results cannot be contaminated.
- [x] Added persisted Off, Lightweight, and public-domain CRT-Lottes modes with
  shader validation and an automatic lightweight fallback.
- [x] Made F1/settings and F5/runtime-log controls interactive during screensaver
  playback without pausing the animation; unlisted input still exits normally.
- [x] Added a responsive screensaver footer that shows all primary shortcuts and
  current CRT/order/scaling modes for ten seconds before smoothly fading away.
- [x] Persisted normal window mode, audio, fit/stretch, and monitor selection in
  addition to CRT, scaling, and story order, with explicit CLI overrides.
- [x] Removed Raylib window destruction/recreation from TTM and Full Story
  switching; only content-owned GPU/audio state is reset between selections.
- [x] Added an original single-pass Fast CRT shader with gamma-aware,
  brightness-shaped scanlines, a lightweight aperture grille, persisted
  Soft/Balanced/Sharp presets on F7, and dedicated all-content sweep coverage.
- [x] Added a persisted F8 performance HUD and settings-panel comparison guide
  for actual FPS, CPU submission cost, frame budget, resolution, and expected
  GPU impact; removed the duplicate 30 FPS limiter so playback reaches its
  intended target and filter comparisons are meaningful.
- [x] Added an F9 current-display benchmark that measures Off, Lightweight,
  Fast, and Lottes for four seconds each while playback continues, displays a
  comparison table, and restores the user's original CRT selection.
- [x] Kept a stable latest executable while archiving every successful build as
  a millisecond-resolution timestamped copy under `build\history` for side-by-side
  testing without overwriting earlier builds.
- [x] Inspected `C:\Users\joaki\Downloads\johnnycastaway` without executing its
  binaries. `Screen Antics.scr` identifies itself as Sierra Online Screen Antics
  2.0.1.0, is unsigned, is 1,350,656 bytes, and has MD5
  `7b3ef4626a5e1937285f9a9e3cc947e8`; no standalone canonical `RESOURCE.MAP` or
  `RESOURCE.001` was present in the extracted folder.
