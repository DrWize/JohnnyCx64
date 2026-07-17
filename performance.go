package main

import (
	"fmt"
	"log"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	performanceFrameBudgetMS          = 1000.0 / 30.0
	performanceRevealSeconds          = 10.0
	performanceBenchmarkModeSeconds   = 4.0
	performanceBenchmarkResultSeconds = 20.0
)

var performanceBenchmarkModes = []crtFilter{crtOff, crtLightweight, crtFast, crtHDR, crtLottes}

type performanceBenchmarkResult struct {
	mode     crtFilter
	fps      int32
	cpuMS    float64
	expected string
	width    int
	height   int
}

var (
	performanceAverageMS             float64
	performanceSampleCount           int
	performanceVisibleUntil          float64
	performancePinned                bool
	performanceBenchmarkActive       bool
	performanceBenchmarkIndex        int
	performanceBenchmarkStarted      float64
	performanceBenchmarkOriginal     crtFilter
	performanceBenchmarkResults      []performanceBenchmarkResult
	performanceBenchmarkResultsUntil float64
)

func performanceInitialize() {
	performanceAverageMS = 0
	performanceSampleCount = 0
	performanceReveal()
}

func performanceReveal() {
	performanceVisibleUntil = rl.GetTime() + performanceRevealSeconds
	menuRevealFooter()
}

func performanceModeChanged() {
	performanceAverageMS = 0
	performanceSampleCount = 0
	performanceReveal()
}

func performanceToggle() {
	performancePinned = !performancePinned
	performanceReveal()
	persistDisplayPlaybackSettings()
}

func performanceBenchmarkToggle() {
	if performanceBenchmarkActive {
		performanceBenchmarkCancel()
		return
	}
	performanceBenchmarkOriginal = crtFilterMode
	performanceBenchmarkResults = nil
	performanceBenchmarkIndex = 0
	performanceBenchmarkActive = true
	performanceBenchmarkApplyMode()
}

func performanceBenchmarkApplyMode() {
	crtFilterMode = performanceBenchmarkModes[performanceBenchmarkIndex]
	performanceBenchmarkStarted = rl.GetTime()
	performanceModeChanged()
}

func performanceBenchmarkCancel() {
	if !performanceBenchmarkActive {
		return
	}
	performanceBenchmarkActive = false
	crtFilterMode = performanceBenchmarkOriginal
	performanceModeChanged()
}

func performanceBenchmarkUpdate() {
	if !performanceBenchmarkActive || rl.GetTime()-performanceBenchmarkStarted < performanceBenchmarkModeSeconds {
		return
	}
	performanceBenchmarkResults = append(performanceBenchmarkResults, performanceBenchmarkResult{
		mode: crtFilterMode, fps: rl.GetFPS(), cpuMS: performanceAverageMS,
		expected: performanceExpectedImpact(), width: rl.GetScreenWidth(), height: rl.GetScreenHeight(),
	})
	performanceBenchmarkIndex++
	if performanceBenchmarkIndex < len(performanceBenchmarkModes) {
		performanceBenchmarkApplyMode()
		return
	}
	performanceBenchmarkActive = false
	performanceBenchmarkResultsUntil = rl.GetTime() + performanceBenchmarkResultSeconds
	crtFilterMode = performanceBenchmarkOriginal
	performanceModeChanged()
	performanceVisibleUntil = performanceBenchmarkResultsUntil
}

func performanceRecord(renderSeconds float64) {
	if renderSeconds < 0 {
		return
	}
	valueMS := renderSeconds * 1000
	if performanceSampleCount == 0 {
		performanceAverageMS = valueMS
	} else {
		// A short exponential window reacts quickly when the user changes modes.
		performanceAverageMS = performanceAverageMS*0.9 + valueMS*0.1
	}
	performanceSampleCount++
	if performanceSampleCount == 90 {
		log.Printf("display performance: fps=%d cpu_submit_ms=%.2f cpu_budget_percent=%.0f cpu_impact=%s expected_gpu_cost=%s mode=%s resolution=%dx%d",
			rl.GetFPS(), performanceAverageMS, performanceBudgetPercent(performanceAverageMS),
			performanceImpactLabel(performanceAverageMS), performanceExpectedImpact(), performanceModeLabel(),
			rl.GetScreenWidth(), rl.GetScreenHeight())
	}
}

func performanceBudgetPercent(renderMS float64) float64 {
	if renderMS <= 0 {
		return 0
	}
	return renderMS / performanceFrameBudgetMS * 100
}

func performanceImpactLabel(renderMS float64) string {
	percent := performanceBudgetPercent(renderMS)
	switch {
	case percent < 15:
		return "Very low"
	case percent < 35:
		return "Low"
	case percent < 60:
		return "Moderate"
	case percent < 90:
		return "High"
	default:
		return "Critical"
	}
}

func performanceExpectedImpact() string {
	switch crtFilterMode {
	case crtLightweight:
		if imageScalingMode == scalingScale2x {
			return "Low"
		}
		return "Very low"
	case crtFast:
		return "Low"
	case crtHDR:
		return "Moderate"
	case crtLottes:
		return "High"
	default:
		if imageScalingMode == scalingScale2x {
			return "Low"
		}
		return "Minimal"
	}
}

func performanceModeLabel() string {
	crt := crtFilterMode.label()
	if crtFilterMode == crtFast {
		crt += "/" + fastCRTSharpnessLabel(fastCRTSharpness)
	}
	scale := imageScalingMode.label()
	if crtFilterMode.usesNativeFrame() {
		scale += " (filter resampling)"
	}
	return fmt.Sprintf("Filter %s | Scale %s", crt, scale)
}

func performanceFooterText() string {
	if performanceBenchmarkActive {
		return fmt.Sprintf("F9 benchmark %d/%d: %s | %s", performanceBenchmarkIndex+1,
			len(performanceBenchmarkModes), crtFilterMode.label(), performanceFooterMetrics())
	}
	return performanceFooterMetrics()
}

func performanceFooterMetrics() string {
	if performanceSampleCount == 0 {
		return "Measuring display performance..."
	}
	return fmt.Sprintf("%d FPS | %.2f ms CPU submit | %d%% CPU budget | GPU cost %s",
		rl.GetFPS(), performanceAverageMS, int(math.Round(performanceBudgetPercent(performanceAverageMS))),
		performanceExpectedImpact())
}

func performanceDraw() {
	performanceBenchmarkUpdate()
	now := rl.GetTime()
	showResults := len(performanceBenchmarkResults) != 0 && now < performanceBenchmarkResultsUntil
	opacity := informationalUIOpacity(now)
	if opacity <= 0 || menuVisible || traceVisible || (!performancePinned && !performanceBenchmarkActive && !showResults && now >= performanceVisibleUntil) {
		return
	}

	actualFPS := rl.GetFPS()
	budget := int(math.Round(performanceBudgetPercent(performanceAverageMS)))
	line1 := fmt.Sprintf("%d FPS (30 target)  |  %.2f ms CPU frame submission  |  %d%% CPU frame budget",
		actualFPS, performanceAverageMS, budget)
	line2 := fmt.Sprintf("%s  |  CPU impact: %s  |  Expected GPU cost: %s  |  %dx%d",
		performanceModeLabel(), performanceImpactLabel(performanceAverageMS), performanceExpectedImpact(),
		rl.GetScreenWidth(), rl.GetScreenHeight())

	lines := []string{line1, line2}
	if performanceBenchmarkActive {
		lines = append(lines, fmt.Sprintf("F9 benchmark running: mode %d/%d — %s — %.1f seconds remaining",
			performanceBenchmarkIndex+1, len(performanceBenchmarkModes), crtFilterMode.label(),
			max(0.0, performanceBenchmarkModeSeconds-(rl.GetTime()-performanceBenchmarkStarted))))
	} else if showResults {
		lines = append(lines, fmt.Sprintf("F9 benchmark results at %dx%d (actual FPS / CPU submit / expected GPU cost)",
			performanceBenchmarkResults[0].width, performanceBenchmarkResults[0].height))
		for _, result := range performanceBenchmarkResults {
			lines = append(lines, fmt.Sprintf("%-12s  %2d FPS   %.2f ms CPU   %s GPU cost",
				result.mode.label(), result.fps, result.cpuMS, result.expected))
		}
	}

	fontSize := int32(16)
	availableWidth := int32(rl.GetScreenWidth()) - 56
	maxLineWidth := func(size int32) int32 {
		width := int32(0)
		for _, line := range lines {
			width = max(width, rl.MeasureText(line, size))
		}
		return width
	}
	for fontSize > 10 && maxLineWidth(fontSize) > availableWidth {
		fontSize--
	}
	maxWidth := maxLineWidth(fontSize)
	boxWidth := maxWidth + 28
	boxHeight := int32(18 + len(lines)*24)
	x := int32(14)
	y := int32(14)
	rl.DrawRectangle(x, y, boxWidth, boxHeight, rl.Fade(rl.NewColor(12, 16, 24, 225), opacity))
	rl.DrawRectangleLines(x, y, boxWidth, boxHeight, rl.Fade(rl.NewColor(145, 165, 195, 255), opacity))
	for index, line := range lines {
		color := rl.RayWhite
		if index == 0 || (showResults && index >= 3) {
			color = rl.Gold
		}
		rl.DrawText(line, x+14, y+9+int32(index)*24, fontSize, rl.Fade(color, opacity))
	}
}
