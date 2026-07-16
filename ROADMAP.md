# JohnnyCx86 Roadmap

Updated: 2026-07-16

## Objective

Ship a trustworthy source-only Windows release of JohnnyCx86, then improve
visibility, performance tuning, visual fidelity, and platform reach in that
order.

## Phase 0 — Preserve and integrate today's x86 work (P0)

Target: the next development session.

- [x] Preserve the 12-file x86 delta before changing branches:
  `.gitignore`, `README.md`, `TODO.md`, `graphics.go`, `main.go`,
  `main_test.go`, `platform_windows.go`, `.github/workflows/windows-x86.yml`,
  `build/build_x86.bat`, `build/screensaver_qa.ps1`,
  `build/windows/JohnnyCastaway-x86.manifest`, and
  `build/windows/JohnnyCastaway-x86.rc`.
- [x] Apply that delta to a new branch based on `johnnycx86/main` (`3e7eb02`),
  not to the older asset-bearing `windows-support` history.
- [x] Confirm generated binaries, build history, Sierra/Dynamix archives, WAVs,
  icons derived from original artwork, and local logs are excluded.
- [x] Review the final diff as a source-only change and make one focused commit
  for Windows x86 screensaver support.

Exit criteria:

- The work is safely based on the public source-only history.
- The commit contains only source, tests, documentation, CI, and build metadata.
- No proprietary game data or generated executables are present.

## Phase 1 — Release candidate verification (P0)

Target: immediately after integration.

- [x] Run the full Go regression suite with the supported Windows CGO
  toolchains.
- [x] Build and inspect both targets:
  `JohnnyCastaway-x64.exe` must be PE32+/amd64 and
  `JohnnyCastaway-x86.scr` must be PE32/386.
- [x] Run `build\screensaver_qa.ps1` and smoke-test `/s`, `/p HWND`,
  `/p:HWND`, `/c`, and `/c:HWND` on 64-bit Windows through WOW64.
- [x] Confirm preview mode creates a real child window inside the Windows Screen
  Saver preview host and exits cleanly.
- [x] Re-run the all-TTM stability sweep and check for crashes, resource leaks,
  missing-resource failures, and content-switch regressions.
- [x] Run both GitHub Actions workflows and retain successful x64 and x86
  artifacts for release testing.
- [x] Perform a final source/data-boundary scan and verify the documented archive
  hashes and clean-clone build instructions.

Exit criteria:

- Local tests, stability sweep, x64 CI, and x86 CI are green.
- All three Windows screensaver modes behave correctly.
- A clean clone can build with user-supplied data and no undocumented files.

Local validation on 2026-07-16: amd64 and 386 tests/builds passed; the artifacts
reported `GOARCH=amd64` and `GOARCH=386` respectively; `/c`, `/s`, `/p HWND`, and
preview host-close behavior passed; canonical archive hashes matched; all 41
TTMs passed 2,460 forced scene advances and 41 content switches; and Full Story
remained healthy for the final 25-second run. CI remains the final Phase 1 gate.

## Phase 2 — Publish the first release (P0)

Target: after all Phase 1 gates pass.

- [x] Merge the verified x86 branch into `johnnycx86/main`.
- [x] Tag the first release candidate and prepare concise release notes covering
  x64 application support, x86 screensaver support, required user-supplied
  data, known missing effects, controls, and unsigned-binary expectations.
- [ ] Test the downloadable artifacts on a clean Windows account or machine.
- [ ] Promote the candidate to the first stable release if the clean-machine
  smoke test passes.

Exit criteria:

- The public repository has a reproducible, source-only stable release.
- Installation, data setup, screensaver configuration, and known limitations
  are clear to a new user.

## Phase 3 — DrWize/Home portfolio index (P1)

Target: directly after the stable release.

- [x] Inventory all DrWize projects and give each one a consistent entry
  in `DrWize/Home`, including status, purpose, technology, repository link, and
  a representative image where useful.
- [x] Treat `DrWize/Home` as the portfolio and discovery hub only. Keep every
  project's source, releases, issues, detailed documentation, and development
  history in that project's own repository.
- [x] Add JohnnyCx86 as one portfolio entry linking to the dedicated
  `DrWize/JohnnyCx86` repository and its releases; do not copy or mirror the
  JohnnyCx86 source tree into `DrWize/Home`.
- [ ] Give the JohnnyCx86 entry a concise x64/x86 summary, CI/release status,
  screenshot, and source/data-policy note. Keep build instructions and
  troubleshooting in the dedicated JohnnyCx86 repository.
- [x] Organize the portfolio so completed, active, experimental, and archived
  projects are easy to distinguish without implying that JohnnyCx86 is the
  whole purpose of `DrWize/Home`.

Exit criteria:

- Visitors can discover all DrWize projects and quickly understand their status.
- JohnnyCx86 remains an independent project with one canonical repository,
  release location, issue tracker, and documentation set.

## Phase 4 — Real-hardware performance matrix (P1)

Target: after release stability is established.

- [ ] Run F9 benchmarks at 1920x1080, 3840x2160, 5120x1440, and 7680x2160.
- [ ] Record Off, Lightweight, Fast, and Lottes results with GPU, monitor,
  scaling mode, fit/stretch mode, average FPS, and CPU submission time.
- [ ] Verify that only the centered 4:3 viewport incurs shader cost on 32:9
  displays.
- [ ] Add capability fallback for shader compilation or inadequate performance.
- [ ] Choose conservative automatic defaults from measured results; always keep
  a user override.

Exit criteria:

- Default display choices are based on physical-resolution measurements.
- Unsupported or slow shader paths fail safely to a usable mode.

Progress on 2026-07-16: all four production CRT paths held 30 FPS at a physical
3840x1080 output on an RTX 4080, with 0.15–0.19 ms CPU submission time. Results
and limitations are recorded in `docs/PERFORMANCE.md`. The four required target
resolutions and a lower-powered GPU remain physical-hardware gates, so automatic
defaults and Phases 5–7 are intentionally not advanced yet.

## Phase 5 — Optional CRT modes (P2)

Target: only after the performance matrix is complete.

- [x] Evaluate CRT-Easymode as a flat, sharp 1080p/4K option. Do not import it
  while its upstream header says only `GPL` without a license version.
- [x] Decide whether CRT-Easymode should enter the benchmark: no distributable
  prototype is retained until authoritative license clarification exists.
- [x] Evaluate CRT-Geom as an optional curved-tube mode. Its GPL-2.0-or-later
  terms are compatible, but defer a port until the physical matrix can measure
  it with curvature, overscan, rounded corners, and interlace off by default.
- [x] Keep the current four modes rather than adding an ambiguously licensed or
  unmeasured option. The complete evaluation is in
  `docs/ROADMAP_EVALUATIONS.md`.

Exit criteria:

- Every retained shader has a clear purpose, acceptable license, measured cost,
  persisted settings, and safe fallback behavior.

## Phase 6 — Fidelity and missing artwork decision (P2)

- [ ] Compare representative scenes, long-run order, timing, day/night changes,
  and holiday behavior with the documented observation references.
- [ ] Review known implementations for behavioral regressions and test gaps
  without copying incompatible code or assets.
- [x] Decide whether omitted `FLAME.BMP` and `FLURRY.BMP` effects should remain a
  documented limitation or receive clearly identified original replacement art.
- [x] Do not create replacement art for the stable release. If that decision is
  revisited, require documented authorship and licensing, label it as a
  replacement, and keep it optional for original-data comparisons.

Exit criteria:

- Remaining fidelity differences are either fixed, tested, or explicitly
  documented.
- The missing-effect policy is final and legally unambiguous.

## Phase 7 — Cross-platform exploration (P3)

- [x] Re-evaluate macOS, Linux/XScreenSaver, and WebAssembly. Select Linux for
  the first bounded compile proof because Raylib documents its Ubuntu CGO
  dependencies and CI can reproduce the build.
  Windows release and display paths are stable.
- [x] Design the first platform boundary for diagnostics, single-instance
  locking, Windows-only argument handling, and preview behavior. Runtime input,
  audio, idle-timeout, data-path UI, and packaging remain Linux release work.
  and packaging layers rather than assuming the Windows CGO target will port
  unchanged.
- [x] Add an Ubuntu test/build job as the Linux compile proof before committing
  to a multi-platform release plan.

Exit criteria:

- A documented proof of concept demonstrates that one additional platform is
  maintainable without weakening the Windows release.

Progress on 2026-07-16: Linux has a platform abstraction and Ubuntu compile CI.
This demonstrates build maintenance only; it is not evidence of runtime or
XScreenSaver support. Windows x64 and x86 CI remain mandatory release gates.

## Recommended immediate sequence

1. Preserve the 12-file delta.
2. Rebase the work conceptually onto `johnnycx86/main` using a clean branch.
3. Run local x64/x86 tests and screensaver QA.
4. Push the focused branch and let both CI workflows pass.
5. Merge and publish the first release candidate.
6. Update the all-project `DrWize/Home` portfolio only after the JohnnyCx86
   release URL and artifacts are stable, linking to its dedicated repository.
