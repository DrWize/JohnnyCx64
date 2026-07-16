# Roadmap evaluations: CRT, fidelity, and Linux

Updated: 2026-07-16

## Phase 5 — optional CRT modes

### CRT-Easymode

The upstream shader describes itself as a flat CRT shader intended for 1080p or
higher output. Its beam, mask, gamma, brightness, and sharpness controls fit the
intended high-resolution role. However, its source header states only
`License: GPL` without identifying a GPL version or an `or later` option.

Decision: do not copy, port, or distribute CRT-Easymode until its license version
is clarified by an authoritative source. Technical suitability does not resolve
ambiguous redistribution terms.

Primary source:
[libretro CRT-Easymode shader](https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-easymode.glsl).

### CRT-Geom

CRT-Geom explicitly permits redistribution under GPL version 2 or later, which
is compatible with this GPLv3-or-later project. It provides curvature, rounded
corners, overscan, tilt, dot-mask, gamma, saturation, and optional interlace
simulation. The feature set is materially larger than the current single-pass
flat modes and overlaps the already available Lottes option.

Decision: technically and legally viable, but do not retain a port yet. A curved
mode is optional rather than a release requirement, and the incomplete physical
performance matrix cannot establish its cost or default parameters. If it is
prototyped later, interlace must default off and the port must preserve the
upstream copyright and GPL-2.0-or-later notice.

Primary source:
[libretro CRT-Geom shader](https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-geom.glsl).

### Phase 5 outcome

No additional shader is retained. Off, Lightweight, Fast, and Lottes remain the
supported set. This avoids adding an ambiguously licensed mode or an unmeasured
curved mode merely to increase the option count.

## Phase 6 — missing effects and fidelity

`FLAME.BMP` and `FLURRY.BMP` are absent from both verified original archives and
the reviewed public implementations. The engine already treats them as optional
so `FIRE.TTM` and Full Story continue safely.

Decision: keep the effects omitted and documented for the stable release. Do not
present generated or newly drawn images as recovered original assets. A future
replacement is acceptable only when it is original work, has documented
authorship and redistribution terms, is clearly labeled as a replacement, and
can be disabled for original-data fidelity comparisons.

Remaining fidelity work is observational: representative-scene screenshots,
long-run order and timing, day/night changes, holidays, and known engine edge
cases must be recorded without downloading or redistributing copyrighted video
or game assets.

## Phase 7 — Linux compile proof

Linux is the first additional platform because Raylib documents its native Linux
CGO dependencies and GitHub provides a reproducible Ubuntu runner. The proof adds
only the platform boundary needed to compile:

- terminal diagnostics replace Windows message boxes;
- Unix `flock` provides process-level single-instance behavior;
- Windows screensaver arguments are not interpreted on Linux;
- Windows preview embedding remains explicitly unsupported;
- Ubuntu CI runs tests and builds an amd64 binary with Raylib's documented
  development packages.

Primary source:
[raylib-go build requirements](https://github.com/gen2brain/raylib-go#requirements).

This is a compile proof, not a Linux release. Runtime X11/Wayland behavior,
audio, packaging, data-folder UI, idle detection, and XScreenSaver integration
remain unverified. The proof must not weaken or complicate the Windows builds.
