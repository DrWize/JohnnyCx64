# Johnny Castaway 2026 — Release and fidelity worklist

Updated: 2026-07-17

## Active work — do next

1. [x] **Auto-hide informational UI after 10 seconds of inactivity.**
   Fade nonessential on-screen information away after 10 seconds without mouse
   or keyboard input. Show it again immediately when the mouse moves, a mouse
   button is used, or a keyboard key is pressed, then restart the idle timer.
   Keep active dialogs, error messages, and controls that currently have focus
   visible and usable. Add timing and input-wake regression coverage.

   Completed on 2026-07-17: the shortcut/footer and performance information
   remain fully visible for eight seconds, fade for two, and wake immediately
   on mouse movement, mouse buttons, or keyboard input. Dialogs and error/status
   messages remain independent of the informational idle timer.

2. [ ] **Finish the remaining fidelity comparison.**
   Compare long-running scene order, transitions, timing, cloud speed and
   wrapping, waves, tides, holidays, and day/night behavior. The 11-hour video
   samples at the beginning and around 1, 5, 10, and 11 hours confirm the
   640x480 composition and horizontal cloud drift, but all sampled sections are
   daytime. Find a separate trustworthy original night reference before making
   further night-palette or transition changes.

3. [ ] **Run the physical CRT performance matrix.**
   Use `F9` at 1920x1080, 3840x2160, 5120x1440, and 7680x2160, including at
   least one integrated or lower-powered GPU. Record Off, Lightweight, Fast,
   and Lottes results. Confirm that shader work is limited to the centered 4:3
   viewport rather than the 32:9 pillarboxes.

4. [ ] **Set automatic shader defaults only after physical measurements.**
   Add conservative inadequate-performance thresholds with a user override.
   Keep the existing shader compile/capability fallback regardless of measured
   defaults.

## Release path — after the local review

1. [x] **Create one focused branch and commit for the current delta.**
   Include clip-zone enforcement, corrected and moving clouds, day/night
   preview, CRT capability handling, data-folder management, the `scrantic`
   default, interface improvements, tests, and documentation.

   Completed on 2026-07-17 on `agent/fidelity-ui-rc2`, including the
   informational-UI inactivity behavior and its regression coverage.

2. [ ] **Open a pull request and require both Windows CI jobs to pass.**
   Review the final PR diff and artifact contents before merging into `main`.
   Do not include generated executables or original game data.

3. [ ] **Create `v2026.1.0-rc.2` from the merged fidelity commit.**
   RC1 points to `a1f395e` and does not include the current fidelity,
   configuration, interface, or regression changes. Do not silently promote
   the older RC1 artifacts.

4. [ ] **Test RC2 on a separate Windows account or machine.**
   Verify clean first launch with the default `scrantic` convention, explicit
   `--data-dir`, saved Data Files selection, x64 startup, x86 `/c`, `/s`, and
   Windows Screen Saver Settings preview/install behavior. Also verify missing
   data errors, saved settings, unsigned-binary warnings, and normal input exit.

5. [ ] **Promote the verified candidate to `v2026.1.0` stable.**
   Publish final SHA-256 values and release notes, retain the source-only/data
   policy, and confirm both downloadable artifacts were built from the stable
   tag.

6. [ ] **Publish the prepared `DrWize/home` portfolio update after stable.**
   Update the `agent/all-project-portfolio` branch from RC1 to the stable
   release. Keep the x64/x86 summary, CI badges, and data-policy note. Omit the
   screenshot unless a clearly publishable image can be produced without
   redistributing Sierra/Dynamix artwork.

## Completed in the current change set

- [x] Added `Space` playback pause/resume outside overlays. Pause holds the
  engine timer boundary so scenes, walking, waves, clouds, and sounds resume
  without consuming idle time; a persistent pause badge and responsive dock
  label show the state. Runtime Log keeps Space for trace-capture pause.
- [x] Stabilized Johnny's walking and palm-tree occlusion. Walking now resets
  its island-relative draw offset every frame, uses bounded indexes instead of
  unsafe pointer arithmetic, preserves contiguous source-frame order, and
  always draws Johnny before the fixed trunk and leaves while he passes behind
  the tree. All 41 TTMs passed the 2,460-advance stability gate afterward.
- [x] Completed the final local review of renderer, resource/configuration,
  Windows integration, screensaver input, settings interface, tests, QA scripts,
  and documentation. Generated binaries, history, logs, local profiles,
  screenshots, `scrantic`, and Sierra/Dynamix data remain excluded. The final
  gates passed with zero skipped tests, `go vet`, race-enabled tests, x64/x86
  release builds, x86 `/c`/`/s`/preview QA, and a clean 41-TTM/Full Story sweep.
- [x] Added the `F10` Data Files manager with native Windows folder browsing,
  archive-hash verification, persisted selection, Explorer access, first-run
  recovery, keyboard controls, and mouse controls.
- [x] Made `scrantic` the verified default data-folder convention. Explicit
  `--data-dir` and a saved Data Files selection take priority. Development EXE
  and SCR builds find `E:\ai\Johnny\scrantic` automatically.
- [x] Restored moving clouds. Cloud sprites load once per island, move 1–2
  native pixels per animation tick, and wrap beyond the 640-pixel canvas.
- [x] Corrected cloud placement to match the C reference and added movement,
  wrapping, and placement tests.
- [x] Added the Settings `Sky` button to cycle Day, Night, and Automatic without
  changing the Windows clock; no `F11` shortcut is assigned.
- [x] Added `D` to force the Full Story background to Day. Kept `N` assigned to
  Next TTM; Night and Automatic remain available from the Settings `Sky` button.
- [x] Fixed automatic day/night state so daytime resets a previous night state.
  Automatic mode uses day from 06:00 through 17:59 and night otherwise.
- [x] Added clip-zone enforcement for drawing operations and regression tests.
- [x] Added capability-aware CRT shader fallback, Fast CRT sharpness presets,
  the `F8` performance display, and the `F9` comparison benchmark.
- [x] Added persistent scene order, scaling, CRT, window, audio, aspect, monitor,
  and data-directory settings with explicit command-line precedence.
- [x] Added screensaver-safe Settings and Runtime Log interaction, the fading
  shortcut footer, and ordinary-input exit behavior.
- [x] Added `F` to toggle the normal application between a centered resizable
  window and borderless fullscreen while preserving Runtime Log `Ctrl+F`
  search and screensaver/preview behavior.
- [x] Kept one Raylib window and graphics context while switching between Full
  Story and individual TTMs.
- [x] Completed the 41-TTM stability sweep: 2,460 scene advances, all 41 TTMs
  switched and wrapped, and a healthy Full Story run.
- [x] Added automated resource, parser, renderer, configuration, story,
  navigation, and Windows command-line regression coverage.
- [x] Built and tested current Windows x64 EXE and x86 SCR outputs.
- [x] Decided not to create replacement `FLAME.BMP` or `FLURRY.BMP` artwork for
  the stable release. Their effects remain safely omitted and documented; any
  future optional replacement must have documented authorship and licensing
  and must not be represented as recovered Sierra/Dynamix data.

## Known limitations and deferred scope

- Night fidelity remains observationally unverified because the sampled
  11-hour recording sections are all daytime.
- `FLAME.BMP` and `FLURRY.BMP` remain unavailable; their effects are safely
  omitted.
- Automatic CRT performance defaults remain blocked on physical GPU and
  ultrawide measurements.
- macOS, Linux, WebAssembly, and simultaneous multi-monitor playback remain
  deferred until they have dedicated implementation and QA plans.

## Reference status

- The direct Git upstream and timing baseline is
  [deckarep/Johnny-Castaway-2026-Public](https://github.com/deckarep/Johnny-Castaway-2026-Public).
- The C implementation at
  [jno6809/jc_reborn](https://github.com/jno6809/jc_reborn) remains the formula
  and engine-behavior reference.
- [tallPete/JohnnyCastaway](https://github.com/tallPete/JohnnyCastaway) was
  reviewed for slot limits, clipping, walking, cloud placement, stability,
  persistence, and rendering regressions.
- [castaway.xesf.net](https://castaway.xesf.net/) is useful for palette and
  composition checks but is not a full-story behavior reference.
- [alexbevi/xbak](https://github.com/alexbevi/xbak) is retained only as a
  historical resource/parser reference; no source or assets were copied.
- Continue observation against the
  [11-hour recording](https://www.youtube.com/watch?v=l8D6qppreiI), without
  downloading or bundling it.

Implementation and comparison details are recorded in
[`docs/FIDELITY_AUDIT.md`](docs/FIDELITY_AUDIT.md), while longer-term platform
and display decisions remain in [`ROADMAP.md`](ROADMAP.md).
