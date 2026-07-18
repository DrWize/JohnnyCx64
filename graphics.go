package main

import (
	"fmt"
	"image/color"
	"log"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	MaxBMPSlots      = 6
	MaxSpritesPerBMP = 120
	MaxTTMSlots      = 10
	MaxTTMThreads    = 10
)

const (
	MaxFadeOutRadius = 800
)

type imageScaling string

type crtFilter string

type crtCapabilities struct {
	fast   bool
	hdr    bool
	lottes bool
}

type layerClip struct {
	x      int32
	y      int32
	width  int32
	height int32
}

const (
	crtOff         crtFilter = "off"
	crtLightweight crtFilter = "lightweight"
	crtFast        crtFilter = "fast"
	crtHDR         crtFilter = "hdr"
	crtLottes      crtFilter = "lottes"
)

var detectedCRTCapabilities = crtCapabilities{fast: true, hdr: true, lottes: true}

func (capabilities crtCapabilities) supports(mode crtFilter) bool {
	switch mode {
	case crtFast:
		return capabilities.fast
	case crtHDR:
		return capabilities.hdr
	case crtLottes:
		return capabilities.lottes
	case crtOff, crtLightweight:
		return true
	default:
		return false
	}
}

func (capabilities crtCapabilities) modes() []crtFilter {
	modes := []crtFilter{crtOff, crtLightweight}
	if capabilities.fast {
		modes = append(modes, crtFast)
	}
	if capabilities.hdr {
		modes = append(modes, crtHDR)
	}
	if capabilities.lottes {
		modes = append(modes, crtLottes)
	}
	return modes
}

func (capabilities crtCapabilities) fallback(mode crtFilter) crtFilter {
	if capabilities.supports(mode) {
		return mode
	}
	return crtLightweight
}

func (capabilities crtCapabilities) next(mode crtFilter) crtFilter {
	modes := capabilities.modes()
	for index, available := range modes {
		if available == mode {
			return modes[(index+1)%len(modes)]
		}
	}
	return capabilities.fallback(mode)
}

func parseCRTFilter(value string, legacyEnabled bool) crtFilter {
	mode := crtFilter(value)
	switch mode {
	case crtOff, crtLightweight, crtFast, crtHDR, crtLottes:
		return mode
	}
	if legacyEnabled {
		return crtLightweight
	}
	return crtOff
}

func (mode crtFilter) label() string {
	switch mode {
	case crtLightweight:
		return "Lightweight"
	case crtFast:
		return "Fast"
	case crtHDR:
		return "HDR Pop"
	case crtLottes:
		return "Lottes"
	default:
		return "Off"
	}
}

func (mode crtFilter) usesNativeFrame() bool {
	return mode == crtFast || mode == crtHDR || mode == crtLottes
}

const (
	fastCRTSoft = iota
	fastCRTBalanced
	fastCRTSharp
)

func normalizeFastCRTSharpness(value int) int {
	if value < fastCRTSoft || value > fastCRTSharp {
		return fastCRTBalanced
	}
	return value
}

func fastCRTSharpnessLabel(value int) string {
	switch normalizeFastCRTSharpness(value) {
	case fastCRTSoft:
		return "Soft"
	case fastCRTSharp:
		return "Sharp"
	default:
		return "Balanced"
	}
}

func fastCRTSharpnessValue(value int) float32 {
	switch normalizeFastCRTSharpness(value) {
	case fastCRTSoft:
		return 0.25
	case fastCRTSharp:
		return 0.9
	default:
		return 0.6
	}
}

func decodeFastCRTSharpness(saved int) int {
	if saved < 1 || saved > 3 {
		return fastCRTBalanced
	}
	return saved - 1
}

func encodeFastCRTSharpness(value int) int {
	return normalizeFastCRTSharpness(value) + 1
}

const (
	scalingNearest  imageScaling = "nearest"
	scalingBilinear imageScaling = "bilinear"
	scalingScale2x  imageScaling = "scale2x"
)

func parseImageScalingMode(value string, legacySmoothing bool) imageScaling {
	mode := imageScaling(value)
	switch mode {
	case scalingNearest, scalingBilinear, scalingScale2x:
		return mode
	}
	if legacySmoothing {
		return scalingBilinear
	}
	return scalingNearest
}

func (mode imageScaling) label() string {
	switch mode {
	case scalingBilinear:
		return "Bilinear"
	case scalingScale2x:
		return "Scale2x"
	default:
		return "Nearest"
	}
}

const scale2xFragmentShader = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
uniform sampler2D texture0;
uniform vec2 sourceSize;
out vec4 finalColor;

bool sameColor(vec4 a, vec4 b) {
    return all(equal(a, b));
}

void main() {
    vec2 pixel = fragTexCoord * sourceSize;
    vec2 centerUV = (floor(pixel) + vec2(0.5)) / sourceSize;
    vec2 stepUV = vec2(1.0) / sourceSize;
    vec4 b = texture(texture0, centerUV - vec2(0.0, stepUV.y));
    vec4 d = texture(texture0, centerUV - vec2(stepUV.x, 0.0));
    vec4 e = texture(texture0, centerUV);
    vec4 f = texture(texture0, centerUV + vec2(stepUV.x, 0.0));
    vec4 h = texture(texture0, centerUV + vec2(0.0, stepUV.y));
    vec2 quadrant = fract(pixel);
    vec4 result = e;
    if (quadrant.x < 0.5 && quadrant.y < 0.5 && sameColor(d, b) && !sameColor(d, h) && !sameColor(b, f)) result = d;
    if (quadrant.x >= 0.5 && quadrant.y < 0.5 && sameColor(b, f) && !sameColor(b, d) && !sameColor(f, h)) result = f;
    if (quadrant.x < 0.5 && quadrant.y >= 0.5 && sameColor(d, h) && !sameColor(d, b) && !sameColor(h, f)) result = d;
    if (quadrant.x >= 0.5 && quadrant.y >= 0.5 && sameColor(h, f) && !sameColor(d, h) && !sameColor(b, f)) result = f;
    finalColor = result * fragColor;
}`

// Adapted to Raylib's GLSL 330 interface from Timothy Lottes' public-domain
// CRT shader. It remains a single pass, but includes beam shaping, bloom,
// an RGB shadow mask, gamma conversion, and subtle tube curvature.
const crtLottesFragmentShader = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
uniform sampler2D texture0;
uniform vec2 sourceSize;
out vec4 finalColor;

const float hardScan = -8.0;
const float hardPix = -3.0;
const float hardBloomPix = -1.5;
const float hardBloomScan = -2.0;
const float bloomAmount = 0.15;
const float warpX = 0.031;
const float warpY = 0.041;
const float maskDark = 0.5;
const float maskLight = 1.5;
const float shape = 2.0;

float toLinear1(float c) {
    return c <= 0.04045 ? c / 12.92 : pow((c + 0.055) / 1.055, 2.4);
}
vec3 toLinear(vec3 c) {
    return vec3(toLinear1(c.r), toLinear1(c.g), toLinear1(c.b));
}
float toSrgb1(float c) {
    return c < 0.0031308 ? c * 12.92 : 1.055 * pow(c, 1.0 / 2.4) - 0.055;
}
vec3 toSrgb(vec3 c) {
    c = max(c, vec3(0.0));
    return vec3(toSrgb1(c.r), toSrgb1(c.g), toSrgb1(c.b));
}
vec3 fetchPixel(vec2 pos, vec2 offset) {
    pos = (floor(pos * sourceSize + offset) + vec2(0.5)) / sourceSize;
    if (pos.x < 0.0 || pos.x > 1.0 || pos.y < 0.0 || pos.y > 1.0) return vec3(0.0);
    return toLinear(texture(texture0, pos).rgb);
}
vec2 pixelDistance(vec2 pos) {
    vec2 pixel = pos * sourceSize;
    return -((pixel - floor(pixel)) - vec2(0.5));
}
float gauss(float position, float scale) {
    return exp2(scale * pow(abs(position), shape));
}
vec3 horizontal3(vec2 pos, float offset) {
    vec3 b = fetchPixel(pos, vec2(-1.0, offset));
    vec3 c = fetchPixel(pos, vec2( 0.0, offset));
    vec3 d = fetchPixel(pos, vec2( 1.0, offset));
    float distance = pixelDistance(pos).x;
    float wb = gauss(distance - 1.0, hardPix);
    float wc = gauss(distance, hardPix);
    float wd = gauss(distance + 1.0, hardPix);
    return (b * wb + c * wc + d * wd) / (wb + wc + wd);
}
vec3 horizontal5(vec2 pos, float offset) {
    vec3 a = fetchPixel(pos, vec2(-2.0, offset));
    vec3 b = fetchPixel(pos, vec2(-1.0, offset));
    vec3 c = fetchPixel(pos, vec2( 0.0, offset));
    vec3 d = fetchPixel(pos, vec2( 1.0, offset));
    vec3 e = fetchPixel(pos, vec2( 2.0, offset));
    float distance = pixelDistance(pos).x;
    float wa = gauss(distance - 2.0, hardPix);
    float wb = gauss(distance - 1.0, hardPix);
    float wc = gauss(distance, hardPix);
    float wd = gauss(distance + 1.0, hardPix);
    float we = gauss(distance + 2.0, hardPix);
    return (a * wa + b * wb + c * wc + d * wd + e * we) / (wa + wb + wc + wd + we);
}
vec3 horizontal7(vec2 pos, float offset) {
    vec3 a = fetchPixel(pos, vec2(-3.0, offset));
    vec3 b = fetchPixel(pos, vec2(-2.0, offset));
    vec3 c = fetchPixel(pos, vec2(-1.0, offset));
    vec3 d = fetchPixel(pos, vec2( 0.0, offset));
    vec3 e = fetchPixel(pos, vec2( 1.0, offset));
    vec3 f = fetchPixel(pos, vec2( 2.0, offset));
    vec3 g = fetchPixel(pos, vec2( 3.0, offset));
    float distance = pixelDistance(pos).x;
    float wa = gauss(distance - 3.0, hardBloomPix);
    float wb = gauss(distance - 2.0, hardBloomPix);
    float wc = gauss(distance - 1.0, hardBloomPix);
    float wd = gauss(distance, hardBloomPix);
    float we = gauss(distance + 1.0, hardBloomPix);
    float wf = gauss(distance + 2.0, hardBloomPix);
    float wg = gauss(distance + 3.0, hardBloomPix);
    return (a*wa+b*wb+c*wc+d*wd+e*we+f*wf+g*wg)/(wa+wb+wc+wd+we+wf+wg);
}
float scan(vec2 pos, float offset) {
    return gauss(pixelDistance(pos).y + offset, hardScan);
}
float bloomScan(vec2 pos, float offset) {
    return gauss(pixelDistance(pos).y + offset, hardBloomScan);
}
vec3 scanlines(vec2 pos) {
    vec3 a = horizontal3(pos, -1.0);
    vec3 b = horizontal5(pos,  0.0);
    vec3 c = horizontal3(pos,  1.0);
    return a*scan(pos,-1.0) + b*scan(pos,0.0) + c*scan(pos,1.0);
}
vec3 bloom(vec2 pos) {
    vec3 a = horizontal5(pos,-2.0);
    vec3 b = horizontal7(pos,-1.0);
    vec3 c = horizontal7(pos, 0.0);
    vec3 d = horizontal7(pos, 1.0);
    vec3 e = horizontal5(pos, 2.0);
    return a*bloomScan(pos,-2.0) + b*bloomScan(pos,-1.0) + c*bloomScan(pos,0.0) +
           d*bloomScan(pos,1.0) + e*bloomScan(pos,2.0);
}
vec2 warp(vec2 pos) {
    pos = pos * 2.0 - 1.0;
    pos *= vec2(1.0 + pos.y*pos.y*warpX, 1.0 + pos.x*pos.x*warpY);
    return pos * 0.5 + 0.5;
}
vec3 shadowMask(vec2 pixel) {
    vec3 mask = vec3(maskDark);
    pixel.x += pixel.y * 3.0;
    pixel.x = fract(pixel.x / 6.0);
    if (pixel.x < 0.333) mask.r = maskLight;
    else if (pixel.x < 0.666) mask.g = maskLight;
    else mask.b = maskLight;
    return mask;
}
void main() {
    vec2 pos = warp(fragTexCoord);
    if (pos.x <= 0.0 || pos.x >= 1.0 || pos.y <= 0.0 || pos.y >= 1.0) {
        finalColor = vec4(0.0, 0.0, 0.0, 1.0);
        return;
    }
    vec3 color = scanlines(pos) + bloom(pos) * bloomAmount;
    color *= shadowMask(gl_FragCoord.xy);
    finalColor = vec4(toSrgb(color), 1.0) * fragColor;
}`

// Original low-cost CRT shader for this Raylib renderer. It deliberately uses
// a small, fixed sample count: manual sharp interpolation, brightness-shaped
// scanlines, gamma-aware processing, and a three-column aperture grille.
const crtFastFragmentShader = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
uniform sampler2D texture0;
uniform vec2 sourceSize;
uniform float sharpness;
out vec4 finalColor;

vec3 sampleLinear(vec2 pixel) {
    vec2 base = floor(pixel - vec2(0.5));
    vec2 fraction = fract(pixel - vec2(0.5));
    vec2 texel = vec2(1.0) / sourceSize;
    vec2 uv00 = (base + vec2(0.5, 0.5)) * texel;
    vec3 c00 = texture(texture0, uv00).rgb;
    vec3 c10 = texture(texture0, uv00 + vec2(texel.x, 0.0)).rgb;
    vec3 c01 = texture(texture0, uv00 + vec2(0.0, texel.y)).rgb;
    vec3 c11 = texture(texture0, uv00 + texel).rgb;
    return mix(mix(c00, c10, fraction.x), mix(c01, c11, fraction.x), fraction.y);
}

void main() {
    vec2 pixel = fragTexCoord * sourceSize;
    vec2 nearestUV = (floor(pixel) + vec2(0.5)) / sourceSize;
    vec3 nearestColor = texture(texture0, nearestUV).rgb;
    vec3 smoothColor = sampleLinear(pixel);
    vec3 color = mix(smoothColor, nearestColor, sharpness);

    color = pow(max(color, vec3(0.0)), vec3(2.2));
    float brightness = dot(color, vec3(0.2126, 0.7152, 0.0722));
    float lineDistance = abs(fract(pixel.y) - 0.5) * 2.0;
    float beamWidth = mix(1.7, 3.1, clamp(brightness, 0.0, 1.0));
    float scanline = 1.0 - 0.32 * pow(lineDistance, beamWidth);
    color *= scanline;

    float maskStrength = 0.18;
    int grille = int(mod(floor(gl_FragCoord.x), 3.0));
    vec3 mask = vec3(1.0 - maskStrength);
    if (grille == 0) mask.r = 1.0;
    else if (grille == 1) mask.g = 1.0;
    else mask.b = 1.0;
    color *= mask * (1.0 / (1.0 - maskStrength * 0.6666667));

    color = pow(max(color, vec3(0.0)), vec3(1.0 / 2.2));
    finalColor = vec4(color, 1.0) * fragColor;
}`

// HDR Pop is a restrained wide-screen enhancement pass for HDR-capable panels.
// Raylib's current Windows swapchain remains SDR, so this shader produces an
// HDR-style SDR signal for Windows and the display to tone-map rather than
// claiming HDR10 output metadata. It preserves black, adds local clarity and
// chroma separation, and gives bright colors a soft highlight shoulder.
const hdrPopFragmentShader = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
uniform sampler2D texture0;
uniform vec2 sourceSize;
out vec4 finalColor;

vec3 toLinear(vec3 color) {
    return pow(max(color, vec3(0.0)), vec3(2.2));
}

vec3 toDisplay(vec3 color) {
    return pow(clamp(color, vec3(0.0), vec3(1.0)), vec3(1.0 / 2.2));
}

vec3 sampleSource(vec2 uv) {
    vec2 halfTexel = vec2(0.5) / sourceSize;
    return toLinear(texture(texture0, clamp(uv, halfTexel, vec2(1.0) - halfTexel)).rgb);
}

void main() {
    vec2 pixel = fragTexCoord * sourceSize;
    vec2 uv = (floor(pixel) + vec2(0.5)) / sourceSize;
    vec2 texel = vec2(1.0) / sourceSize;

    vec3 center = sampleSource(uv);
    vec3 north = sampleSource(uv - vec2(0.0, texel.y));
    vec3 south = sampleSource(uv + vec2(0.0, texel.y));
    vec3 west = sampleSource(uv - vec2(texel.x, 0.0));
    vec3 east = sampleSource(uv + vec2(texel.x, 0.0));
    vec3 neighborhood = (north + south + west + east) * 0.25;

    // Preserve the pixel-art silhouette while improving clarity on very large
    // panels. Limit the detail term so palette transitions never halo heavily.
    vec3 detail = clamp(center - neighborhood, vec3(-0.18), vec3(0.18));
    vec3 color = max(center + detail * 0.24, vec3(0.0));

    // Expand color separation without shifting neutral grays or crushing black.
    float luminance = dot(color, vec3(0.2126, 0.7152, 0.0722));
    color = mix(vec3(luminance), color, 1.20);

    // A small, content-derived glow lifts bright palette entries. The shoulder
    // keeps the SDR output bounded for predictable Windows HDR tone mapping.
    vec3 highlight = max(neighborhood - vec3(0.52), vec3(0.0));
    color += highlight * 0.13;
    color = color * (vec3(1.0) + 0.16 * color) /
            (vec3(1.0) + 0.16 * color * color);

    finalColor = vec4(toDisplay(color), 1.0) * fragColor;
}`

const (
	ttmBufferBackground uint16 = iota
	ttmBufferStoredArea
	ttmBufferComposition
)

var (
	// added by r.c. to mimic screen saver behavior.
	screenSaverPos rl.Vector2 = rl.Vector2Zero()
)

var (
	ttmPalette = [16][4]uint8{}

	grDx = 0
	grDy = 0
	//int grWindowed    = 0

	isFadingOut   = false
	fadeOutRadius = 0

	grUpdateDelay     int = 0
	grBackgroundSur   *rl.RenderTexture2D
	grSavedZonesLayer *rl.RenderTexture2D
	grCompositionSur  *rl.RenderTexture2D
	grScale2xSur      *rl.RenderTexture2D
	grScale2xShader   rl.Shader
	grCRTFastShader   rl.Shader
	grCRTFastSharpLoc int32 = -1
	grHDRPopShader    rl.Shader
	grCRTLottesShader rl.Shader
	grLayerClips      = make(map[uint32]layerClip)
)

type TAdsScene struct {
	slot     uint16
	tag      uint16
	numPlays uint16
}

type TTtmSlot struct {
	name       string
	data       []byte
	dataSize   uint32
	tags       []TTtmTag
	numTags    int
	numSprites [MaxBMPSlots]int
	sprites    [MaxBMPSlots][MaxSpritesPerBMP]*rl.Texture2D
}

type TTtmTag struct { // TODO : rename, used for ADS too
	id     uint16
	offset uint32
}

type TTtmThread struct {
	ttmSlot         *TTtmSlot
	isRunning       int
	sceneSlot       uint16
	sceneTag        uint16
	sceneTimer      int16
	sceneIterations uint16
	ip              uint32
	delay           uint16
	timer           uint16
	nextGotoOffset  uint32
	selectedBmpSlot uint8
	fgColor         uint8
	bgColor         uint8
	ttmLayer        *rl.RenderTexture2D
}

func grReleaseScreen() {
	if grBackgroundSur == nil {
		return
	}
	if ttmBackgroundThread.ttmLayer == grBackgroundSur {
		ttmBackgroundThread.ttmLayer = nil
	}
	grFreeLayer(grBackgroundSur)
	grBackgroundSur = nil
}

func grReleaseSavedLayer() {
	if grSavedZonesLayer == nil {
		return
	}
	grFreeLayer(grSavedZonesLayer)
	grSavedZonesLayer = nil
}

func grPutPixel(sur *rl.RenderTexture2D, x, y uint16, c uint8) {
	clr := color.RGBA{
		R: ttmPalette[c][0],
		G: ttmPalette[c][1],
		B: ttmPalette[c][2],
		A: 0,
	}

	clipped := grBeginLayerDraw(sur)
	defer grEndLayerDraw(clipped)

	if x < 640 && y < 480 {
		rl.DrawPixel(int32(x), int32(y), clr)
	}
}

func grLoadPalette(palResource *TPALResource) {
	if palResource == nil {
		panic("nil palette")
	}

	for i := 0; i < 16; i++ {
		ttmPalette[i][0] = palResource.Colors[i].B << 2
		ttmPalette[i][1] = palResource.Colors[i].G << 2
		ttmPalette[i][2] = palResource.Colors[i].R << 2
		ttmPalette[i][3] = 0
	}
}

func graphicsInit() {
	grLoadPalette(&palResources[0])
	composition := rl.LoadRenderTexture(screenWidth, screenHeight)
	rl.SetTextureFilter(composition.Texture, rl.FilterPoint)
	grCompositionSur = &composition
	scaled := rl.LoadRenderTexture(screenWidth*2, screenHeight*2)
	rl.SetTextureFilter(scaled.Texture, rl.FilterPoint)
	grScale2xSur = &scaled
	grScale2xShader = rl.LoadShaderFromMemory("", scale2xFragmentShader)
	if rl.IsShaderValid(grScale2xShader) {
		sourceSizeLocation := rl.GetShaderLocation(grScale2xShader, "sourceSize")
		rl.SetShaderValue(grScale2xShader, sourceSizeLocation, []float32{screenWidth, screenHeight}, rl.ShaderUniformVec2)
	} else {
		log.Printf("Scale2x shader failed to load; falling back to nearest scaling")
		imageScalingMode = scalingNearest
	}
	grCRTFastShader = rl.LoadShaderFromMemory("", crtFastFragmentShader)
	detectedCRTCapabilities.fast = rl.IsShaderValid(grCRTFastShader)
	if detectedCRTCapabilities.fast {
		fastSourceLocation := rl.GetShaderLocation(grCRTFastShader, "sourceSize")
		rl.SetShaderValue(grCRTFastShader, fastSourceLocation, []float32{screenWidth, screenHeight}, rl.ShaderUniformVec2)
		grCRTFastSharpLoc = rl.GetShaderLocation(grCRTFastShader, "sharpness")
		applyFastCRTSharpness()
	} else {
		log.Printf("Fast CRT shader failed to load; falling back to the lightweight CRT filter")
	}
	grHDRPopShader = rl.LoadShaderFromMemory("", hdrPopFragmentShader)
	detectedCRTCapabilities.hdr = rl.IsShaderValid(grHDRPopShader)
	if detectedCRTCapabilities.hdr {
		hdrSourceLocation := rl.GetShaderLocation(grHDRPopShader, "sourceSize")
		rl.SetShaderValue(grHDRPopShader, hdrSourceLocation, []float32{screenWidth, screenHeight}, rl.ShaderUniformVec2)
	} else {
		log.Printf("HDR Pop shader failed to load; falling back to the lightweight filter")
	}
	grCRTLottesShader = rl.LoadShaderFromMemory("", crtLottesFragmentShader)
	detectedCRTCapabilities.lottes = rl.IsShaderValid(grCRTLottesShader)
	if detectedCRTCapabilities.lottes {
		lottesSourceLocation := rl.GetShaderLocation(grCRTLottesShader, "sourceSize")
		rl.SetShaderValue(grCRTLottesShader, lottesSourceLocation, []float32{screenWidth, screenHeight}, rl.ShaderUniformVec2)
	} else {
		log.Printf("CRT-Lottes shader failed to load; falling back to the lightweight CRT filter")
	}
	crtFilterMode = detectedCRTCapabilities.fallback(crtFilterMode)
	performanceBenchmarkModes = detectedCRTCapabilities.modes()

	// r.c. added by me, to mimic screen saver behavior, captures initial mouse position.
	screenSaverPos = rl.GetMousePosition()
	performanceInitialize()
}

func graphicsEnd() {
	graphicsReleaseContent()
	grFreeLayer(grScale2xSur)
	grScale2xSur = nil
	grFreeLayer(grCompositionSur)
	grCompositionSur = nil
	if grScale2xShader.ID != 0 {
		rl.UnloadShader(grScale2xShader)
		grScale2xShader.ID = 0
	}
	if grCRTFastShader.ID != 0 {
		rl.UnloadShader(grCRTFastShader)
		grCRTFastShader.ID = 0
		grCRTFastSharpLoc = -1
	}
	if grHDRPopShader.ID != 0 {
		rl.UnloadShader(grHDRPopShader)
		grHDRPopShader.ID = 0
	}
	if grCRTLottesShader.ID != 0 {
		rl.UnloadShader(grCRTLottesShader)
		grCRTLottesShader.ID = 0
	}
}

func applyFastCRTSharpness() {
	if grCRTFastShader.ID == 0 || grCRTFastSharpLoc < 0 {
		return
	}
	rl.SetShaderValue(grCRTFastShader, grCRTFastSharpLoc, []float32{fastCRTSharpnessValue(fastCRTSharpness)}, rl.ShaderUniformFloat)
}

func graphicsReleaseContent() {
	for i := range ttmThreads {
		if ttmThreads[i].ttmLayer != nil {
			grFreeLayer(ttmThreads[i].ttmLayer)
			ttmThreads[i].ttmLayer = nil
		}
	}
	if ttmHolidayThread.ttmLayer != nil {
		grFreeLayer(ttmHolidayThread.ttmLayer)
		ttmHolidayThread.ttmLayer = nil
	}
	if ttmCloudsThread.ttmLayer != nil {
		grFreeLayer(ttmCloudsThread.ttmLayer)
		ttmCloudsThread.ttmLayer = nil
	}
	ttmBackgroundThread.ttmLayer = nil

	for i := range ttmSlots {
		ttmResetSlot(&ttmSlots[i])
	}
	ttmResetSlot(&ttmBackgroundSlot)
	ttmResetSlot(&ttmHolidaySlot)
	ttmResetSlot(&ttmCloudsSlot)

	grReleaseSavedLayer()
	grReleaseScreen()
	if grCompositionSur != nil && grCompositionSur.ID != 0 {
		rl.BeginTextureMode(*grCompositionSur)
		rl.ClearBackground(rl.Black)
		rl.EndTextureMode()
	}
}

func grToggleFullscreen() {
	if appSettings.screenSaver || appSettings.previewParent != 0 {
		return
	}
	rl.ToggleBorderlessWindowed()
	appSettings.windowed = !appSettings.windowed
	if appSettings.windowed {
		rl.EnableCursor()
		rl.ShowCursor()
	} else if foregroundOverlayVisible() {
		rl.ShowCursor()
	} else {
		rl.DisableCursor()
		rl.HideCursor()
	}
	performanceModeChanged()
	persistDisplayPlaybackSettings()
}

func grUpdateDisplay(
	ttmBGThread *TTtmThread,
	ttmThreads []TTtmThread,
	ttmHolidayThread *TTtmThread,
	ttmCloudsThread *TTtmThread,
) {
	draw := func() {
		overlayWasVisible := foregroundOverlayVisible()
		mousePosition := rl.GetMousePosition()
		queuedKey := rl.GetKeyPressed()
		mouseButtonPressed := rl.IsMouseButtonPressed(rl.MouseButtonLeft) ||
			rl.IsMouseButtonPressed(rl.MouseButtonRight) ||
			rl.IsMouseButtonPressed(rl.MouseButtonMiddle)
		uiObserveActivity(rl.GetTime(), mousePosition, mouseButtonPressed, queuedKey != 0 && queuedKey != rl.KeyF12)
		if rl.IsKeyReleased(rl.KeyLeftShift) {
			debugEnabled = !debugEnabled
		}

		if rl.WindowShouldClose() {
			requestExit()
		}

		renderStarted := rl.GetTime()
		composeScene(ttmThreads, ttmHolidayThread, ttmCloudsThread)
		prepareScale2x()

		rl.BeginDrawing()
		defer func() {
			rl.EndDrawing()
			if screenshotRequested {
				screenshotRequested = false
				path, err := captureScreenshot()
				if err != nil {
					log.Printf("screenshot failed: %v", err)
					menuShowStatus("Screenshot failed: " + err.Error())
				} else {
					log.Printf("screenshot saved: %s", path)
					menuShowStatus("Screenshot saved: " + compactMiddle(path, 58))
				}
			}
			performanceRecord(rl.GetTime() - renderStarted)
		}()

		// Opaque black makes the very wide pillarboxes intentional in 32:9.
		rl.ClearBackground(rl.Black)
		drawComposedScene()

		if isFadingOut {
			xPos := float32(rl.GetScreenWidth()) / 2
			yPos := float32(rl.GetScreenHeight()) / 2
			rl.DrawCircle(int32(xPos), int32(yPos), float32(fadeOutRadius), rl.Black)
		}

		drawCRTFilter()
		performanceDraw()

		// Debug stuff added by me, r.c.
		if debugEnabled {
			fontSize := int32(35)
			yPos := int32(rl.GetScreenHeight()) - (fontSize * 2)
			offset := int32(3)
			rl.DrawText(fmt.Sprintf("Story: %d", storyCurrentDay), fontSize, yPos, fontSize, rl.Black)
			rl.DrawText(fmt.Sprintf("Story: %d", storyCurrentDay), fontSize-offset, yPos-offset, fontSize, rl.White)

			rl.DrawFPS(10, 10)
		}

		menuUpdateAndDraw(queuedKey)
		traceUpdateAndDraw()
		handleDoubleEscape()

		if appSettings.previewParent != 0 && !previewHostAvailable(appSettings.previewParent) {
			requestExit()
		}

		if appSettings.screenSaver && appSettings.previewParent == 0 {
			overlayVisible := foregroundOverlayVisible()
			controlPressed := screenSaverControlPressed(overlayWasVisible || overlayVisible)
			interactive := overlayWasVisible || overlayVisible || controlPressed
			otherKeyPressed := !controlPressed && queuedKey != 0
			if shouldExitScreenSaver(screenSaverPos != mousePosition, interactive, otherKeyPressed) {
				rl.SetMasterVolume(0)
				requestExit()
			} else if interactive {
				// Menu/log mouse movement and recognized hotkeys are interactive,
				// so make their current position the new screensaver baseline.
				screenSaverPos = mousePosition
			}
		}
	}

	// TODO: Wait for the tick ...
	// r.c. (this is not like original C code which uses SDL, Raylib still requires calls to Begin/End draw
	// in addition to checking for WindowClose
	start := rl.GetTime()
	for {
		draw()
		const fps = 30
		const frameDelayMS = 1000 / fps
		time.Sleep(time.Millisecond * time.Duration(frameDelayMS))
		if playbackPaused {
			// Keep rendering menus and input without returning to the engine's
			// timer-decrement path. Reset the delay origin so resume does not
			// immediately consume all time spent paused.
			start = rl.GetTime()
			continue
		}
		if (menuVisible || dataManagerVisible) && !appSettings.screenSaver {
			continue
		}

		// r.c. - This is my own logic, trying to KISS
		if isFadingOut {
			if fadeOutRadius > MaxFadeOutRadius {
				isFadingOut = false
				fadeOutRadius = 0
				break
			} else {
				fadeOutRadius += 20
				continue // <-- important to not advance the story while this is happening.
			}
		}

		end := rl.GetTime()
		if grUpdateDelay == 0 ||
			(end-start) >= (float64(grUpdateDelay)*0.01) { //*0.02) {
			break
		}
	}

	// Original C code is below.
	// eventsWaitTick(grUpdateDelay)

	// ... and refresh the display
	// SDL_UpdateWindowSurface(sdl_window)
}

func shouldExitScreenSaver(mouseMoved, interactiveMouse, otherKeyPressed bool) bool {
	return otherKeyPressed || (mouseMoved && !interactiveMouse)
}

func screenSaverControlPressed(overlayVisible bool) bool {
	baseKeys := []int32{
		rl.KeyF1, rl.KeyF2, rl.KeyF3, rl.KeyF4, rl.KeyF5, rl.KeyF7, rl.KeyF8, rl.KeyF9, rl.KeyF10, rl.KeyF12, rl.KeySpace,
		rl.KeyD, rl.KeyN, rl.KeyT, rl.KeyH, rl.KeyEscape,
	}
	for _, key := range baseKeys {
		if rl.IsKeyPressed(key) {
			return true
		}
	}
	if !overlayVisible {
		return false
	}
	overlayKeys := []int32{
		rl.KeyUp, rl.KeyDown, rl.KeyEnter,
		rl.KeyPageUp, rl.KeyPageDown, rl.KeyEnd, rl.KeyTab,
		rl.KeyF6, rl.KeyC, rl.KeyE, rl.KeyL, rl.KeyB, rl.KeyR, rl.KeyO,
	}
	for _, key := range overlayKeys {
		if rl.IsKeyPressed(key) {
			return true
		}
	}
	controlDown := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	if controlDown && rl.IsKeyPressed(rl.KeyF) {
		return true
	}
	return rl.IsMouseButtonPressed(rl.MouseButtonLeft) ||
		rl.IsMouseButtonPressed(rl.MouseButtonRight)
}

func drawLayerToComposition(rt *rl.RenderTexture2D) {
	if rt == nil || rt.Texture.ID == 0 {
		return
	}
	src := rl.NewRectangle(0, 0, float32(rt.Texture.Width), -float32(rt.Texture.Height))
	dst := rl.NewRectangle(0, 0, screenWidth, screenHeight)
	rl.DrawTexturePro(rt.Texture, src, dst, rl.Vector2Zero(), 0, rl.White)
}

func composeScene(ttmThreads []TTtmThread, holiday, clouds *TTtmThread) {
	if grCompositionSur == nil {
		return
	}
	rl.BeginTextureMode(*grCompositionSur)
	rl.ClearBackground(rl.Black)
	drawLayerToComposition(grBackgroundSur)
	if clouds != nil && clouds.isRunning != 0 {
		drawLayerToComposition(clouds.ttmLayer)
	}
	drawLayerToComposition(grSavedZonesLayer)
	for i := 0; i < len(ttmThreads) && i < MaxTTMThreads; i++ {
		if ttmThreads[i].isRunning != 0 {
			drawLayerToComposition(ttmThreads[i].ttmLayer)
		}
	}
	if holiday != nil && holiday.isRunning != 0 {
		drawLayerToComposition(holiday.ttmLayer)
	}
	rl.EndTextureMode()
}

func prepareScale2x() {
	if imageScalingMode != scalingScale2x || grCompositionSur == nil || grScale2xSur == nil {
		return
	}
	rl.BeginTextureMode(*grScale2xSur)
	rl.ClearBackground(rl.Black)
	rl.BeginShaderMode(grScale2xShader)
	src := rl.NewRectangle(0, 0, screenWidth, -screenHeight)
	dst := rl.NewRectangle(0, 0, screenWidth*2, screenHeight*2)
	rl.DrawTexturePro(grCompositionSur.Texture, src, dst, rl.Vector2Zero(), 0, rl.White)
	rl.EndShaderMode()
	rl.EndTextureMode()
}

type displayViewport struct {
	x, y, width, height float32
}

func calculateDisplayViewport(windowWidth, windowHeight float32, stretch bool) displayViewport {
	if windowWidth <= 0 || windowHeight <= 0 {
		return displayViewport{}
	}
	if stretch {
		return displayViewport{width: windowWidth, height: windowHeight}
	}
	scale := min(windowWidth/screenWidth, windowHeight/screenHeight)
	width := screenWidth * scale
	height := screenHeight * scale
	return displayViewport{
		x:      (windowWidth - width) / 2,
		y:      (windowHeight - height) / 2,
		width:  width,
		height: height,
	}
}

func drawComposedScene() {
	source := grCompositionSur
	if !crtFilterMode.usesNativeFrame() && imageScalingMode == scalingScale2x && grScale2xSur != nil {
		source = grScale2xSur
	}
	if source == nil {
		return
	}
	filter := rl.FilterPoint
	if !crtFilterMode.usesNativeFrame() && imageScalingMode == scalingBilinear {
		filter = rl.FilterBilinear
	}
	rl.SetTextureFilter(source.Texture, filter)
	viewport := calculateDisplayViewport(float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight()), appSettings.stretch)
	src := rl.NewRectangle(0, 0, float32(source.Texture.Width), -float32(source.Texture.Height))
	dst := rl.NewRectangle(viewport.x, viewport.y, viewport.width, viewport.height)
	if crtFilterMode == crtFast {
		rl.BeginShaderMode(grCRTFastShader)
		defer rl.EndShaderMode()
	} else if crtFilterMode == crtHDR {
		rl.BeginShaderMode(grHDRPopShader)
		defer rl.EndShaderMode()
	} else if crtFilterMode == crtLottes {
		rl.BeginShaderMode(grCRTLottesShader)
		defer rl.EndShaderMode()
	}
	rl.DrawTexturePro(source.Texture, src, dst, rl.Vector2Zero(), 0, rl.White)
}

func drawCRTFilter() {
	if crtFilterMode != crtLightweight {
		return
	}

	width := int32(rl.GetScreenWidth())
	height := int32(rl.GetScreenHeight())
	for y := int32(1); y < height; y += 4 {
		rl.DrawRectangle(0, y, width, 1, rl.NewColor(0, 0, 0, 38))
	}

	// A lightweight vignette avoids a shader dependency and works on older GPUs.
	layers := int32(12)
	for i := int32(0); i < layers; i++ {
		alpha := uint8(5 + (layers-i)*2)
		color := rl.NewColor(0, 0, 0, alpha)
		rl.DrawRectangleLines(i, i, width-i*2, height-i*2, color)
	}
}

func grNewLayer() *rl.RenderTexture2D {
	rt := rl.LoadRenderTexture(screenWidth, screenHeight)
	setRenderTextureFilter(&rt)
	return &rt
}

func setRenderTextureFilter(rt *rl.RenderTexture2D) {
	if rt == nil || rt.Texture.ID == 0 {
		return
	}
	// Scene layers are always composited one-to-one. Filtering is applied only
	// to the completed scene, and Scale2x requires exact neighboring colors.
	rl.SetTextureFilter(rt.Texture, rl.FilterPoint)
}

func cycleImageScalingMode() {
	performanceBenchmarkCancel()
	switch imageScalingMode {
	case scalingNearest:
		imageScalingMode = scalingBilinear
	case scalingBilinear:
		imageScalingMode = scalingScale2x
	default:
		imageScalingMode = scalingNearest
	}
	performanceModeChanged()
	persistDisplayPlaybackSettings()
}

func grFreeLayer(sur *rl.RenderTexture2D) {
	if sur == nil {
		return
	}
	delete(grLayerClips, sur.ID)
	if sur.ID == 0 {
		return
	}
	rl.UnloadRenderTexture(*sur)
	sur.ID = 0
	sur.Texture.ID = 0
	sur.Depth.ID = 0
}

func grSetClipZone(sur *rl.RenderTexture2D, x1, y1, x2, y2 int16) {
	if sur == nil || sur.ID == 0 {
		return
	}
	x1 += int16(grDx)
	y1 += int16(grDy)
	x2 += int16(grDx)
	y2 += int16(grDy)

	grLayerClips[sur.ID] = normalizeLayerClip(int32(x1), int32(y1), int32(x2), int32(y2))
}

func normalizeLayerClip(x1, y1, x2, y2 int32) layerClip {
	x1 = max(int32(0), min(int32(screenWidth), x1))
	y1 = max(int32(0), min(int32(screenHeight), y1))
	x2 = max(x1, min(int32(screenWidth), x2))
	y2 = max(y1, min(int32(screenHeight), y2))
	return layerClip{x: x1, y: y1, width: x2 - x1, height: y2 - y1}
}

func grBeginLayerDraw(sur *rl.RenderTexture2D) bool {
	rl.BeginTextureMode(*sur)
	clip, ok := grLayerClips[sur.ID]
	if !ok {
		return false
	}
	rl.BeginScissorMode(clip.x, clip.y, clip.width, clip.height)
	return true
}

func grEndLayerDraw(clipped bool) {
	if clipped {
		rl.EndScissorMode()
	}
	rl.EndTextureMode()
}

func grCopyZoneToBg(sur *rl.RenderTexture2D, x, y, width, height uint16) {
	x += uint16(grDx)
	y += uint16(grDy)

	if grSavedZonesLayer == nil {
		grSavedZonesLayer = grNewLayer()
		grClearScreen(grSavedZonesLayer)
	}

	// Without the +2 there is a two-pixel gap on the GJVIS6.TTM cargo
	// hull. The original renderer appears to round this boundary outward.
	grCopyRenderTextureRegion(sur, grSavedZonesLayer, x, y, width+2, height)

	// BELOW IS ORIGINAL C Code

	// r.c. NOTE: this block is just to document SDL2 which is the source vs dst surface.
	// int SDL_BlitSurface(SDL_Surface *src,
	//                    const SDL_Rect *srcrect,
	//                    SDL_Surface *dst,
	//                    SDL_Rect *dstrect);

	// original SDL code
	//SDL_BlitSurface(sfc, &rect, grSavedZonesLayer, &rect);

}

func grSaveImage1(sur *rl.RenderTexture2D, arg0, arg1, arg2, arg3 uint16) { // // TODO : rename ?
	// r.c. in the original C code, these are NOT implemented!

	//    ttmSetColors(4,4);
	//    ttmDrawRect(arg0,arg1,arg2,arg3);
	//    ttmSaveImage0(arg0,arg1,arg2,arg3);
	//    ttmUpdate();
}

func grSaveZone(sur *rl.RenderTexture2D, x, y, width, height uint16) {
	// r.c. in the original C code, these are NOT implemented!

	// Minimalistic implementation: we don't really save the zone,
	// and let grRestoreZone() simply erase the 'saved zones' layer
}

func grRestoreZone(sur *rl.RenderTexture2D, x, y, width, height uint16) {
	// In Johnny's TTMs, we never have RESTORE_ZONE called
	// while several zones are saved. So we simply free the
	// whole saved zones layer
	grReleaseSavedLayer()
}

func grDrawPixel(sur *rl.RenderTexture2D, x, y int16, clr uint8) {
	x += int16(grDx)
	y += int16(grDy)
	grPutPixel(sur, uint16(x), uint16(y), clr)
}

func grDrawLine(sur *rl.RenderTexture2D, x1, y1, x2, y2 int16, colorIdx uint8) {
	x1 += int16(grDx)
	y1 += int16(grDy)
	x2 += int16(grDx)
	y2 += int16(grDy)

	clr := ttmPalette[colorIdx&0x0f]
	c := color.RGBA{
		// Note color order -> this matches what's in the C implementation.
		R: clr[2],
		G: clr[1],
		B: clr[0],
		A: 0xff,
	}

	clipped := grBeginLayerDraw(sur)
	defer grEndLayerDraw(clipped)

	rl.DrawLine(int32(x1), int32(y1), int32(x2), int32(y2), c)
}

func grDrawHorizontalLine(sur *rl.RenderTexture2D, x1, x2, y int16, color uint8) {
	if y < 0 || y > 479 {
		return
	}

	if x1 < 0 {
		x1 = 0
	}
	if x2 > 639 {
		x2 = 639
	}

	for x := x1; x < x2; x++ {
		grPutPixel(sur, uint16(x), uint16(y), color)
	}
}

func grDrawRect(sur *rl.RenderTexture2D, x, y int16, width, height uint16, colorIdx uint8) {
	x += int16(grDx)
	y += int16(grDy)

	// r.c. testing this out, not ready yet.

	clr := ttmPalette[colorIdx&0x0f]
	c := color.RGBA{
		// Note color order -> this matches what's in the C implementation.
		R: clr[2],
		G: clr[1],
		B: clr[0],
		A: 0xff,
	}

	clipped := grBeginLayerDraw(sur)
	defer grEndLayerDraw(clipped)

	rl.DrawRectangle(int32(x), int32(y), int32(width), int32(height), c)
}

func grDrawCircle(sur *rl.RenderTexture2D, x1, y1 int16, width, height uint16, fgColor, bgColor uint8) {
	x1 += int16(grDx)
	y1 += int16(grDy)

	// We can only draw regular circles
	if width != height {
		fmt.Println("Warning : grDrawCircle() : unable to draw ellipse")
		return
	}

	// In original data, every width is even
	if width%2 != 0 {
		fmt.Println("Warning : grDrawCircle() : unable to process odd diameters")
		return
	}

	// Note: Original uses fully manual pixel drawing, we will just chat with Raylib's circle drawing facilities
	// Comments from the original C code below.
	// Bresenham's circle drawing algorithm
	// Note : the code below intends to be pixel-perfect
	clipped := grBeginLayerDraw(sur)
	defer grEndLayerDraw(clipped)

	grabColor := func(idx uint8) color.RGBA {
		clr := ttmPalette[idx&0x0f]
		return color.RGBA{
			// Note color order -> this matches what's in the C implementation.
			R: clr[2],
			G: clr[1],
			B: clr[0],
			A: 0xff,
		}
	}

	fgClr := grabColor(fgColor)
	rl.DrawCircle(int32(x1), int32(y1), float32(width), fgClr)

	if fgColor != bgColor {
		bgClr := grabColor(bgColor)
		rl.DrawCircle(int32(x1)+1, int32(y1)+1, float32(width), bgClr)
	}
}

func grDrawSprite(sur *rl.RenderTexture2D, ttmSlot *TTtmSlot, x, y int16, spriteNo, imageNo uint16) {
	if int(spriteNo) >= ttmSlot.numSprites[imageNo] {
		fmt.Printf("Warning : grDrawSprite(): less than %d sprites loaded in slot %d\n", imageNo, spriteNo)
		return
	}

	x += int16(grDx)
	y += int16(grDy)

	srcSurface := ttmSlot.sprites[imageNo][spriteNo]

	clipped := grBeginLayerDraw(sur)
	defer grEndLayerDraw(clipped)

	// NOTE: this clears the layer, and only the instruction-set should clear it when it deems necessary.
	//rl.ClearBackground(rl.Blank)

	// Use rl.Red for troubleshooting to render Red colored flipped sprites.
	xx := float32(x)
	yy := float32(y)
	w := float32(srcSurface.Width)
	h := float32(srcSurface.Height)

	// debugging bounding box.
	//if debugEnabled {
	//	rl.DrawRectangleLines(int32(xx), int32(yy), int32(w), int32(h), rl.Red)
	//}

	src := rl.NewRectangle(0, 0, w, h)
	dst := rl.NewRectangle(xx, yy, w, h)
	rl.DrawTexturePro(*srcSurface, src, dst, rl.Vector2Zero(), 0.0, rl.White)
}

func grDrawSpriteFlip(sur *rl.RenderTexture2D, ttmSlot *TTtmSlot, x, y int16, spriteNo, imageNo uint16) {
	if int(spriteNo) >= ttmSlot.numSprites[imageNo] {
		fmt.Printf("Warning : grDrawSprite(): less than %d sprites loaded in slot %d\n", imageNo, spriteNo)
		return
	}

	x += int16(grDx)
	y += int16(grDy)

	srcSurface := ttmSlot.sprites[imageNo][spriteNo]
	//x += int16(srcSurface.Width) - 1 // In original C, but NOT NEEDED, in Raylib.

	clipped := grBeginLayerDraw(sur)
	defer grEndLayerDraw(clipped)

	// NOTE: this clears the layer, and only the instruction-set should clear it when it deems necessary.
	//rl.ClearBackground(rl.Blank)

	// Use rl.Red for troubleshooting to render Red colored flipped sprites.
	xx := float32(x)
	yy := float32(y)
	w := float32(srcSurface.Width)
	h := float32(srcSurface.Height)

	// For debugging purposes.
	//if debugEnabled {
	//	rl.DrawRectangleLines(int32(xx), int32(yy), int32(w), int32(h), rl.Red)
	//}

	src := rl.NewRectangle(0, 0, -w, h)
	dst := rl.NewRectangle(xx, yy, w, h)
	rl.DrawTexturePro(*srcSurface, src, dst, rl.Vector2Zero(), 0.0, rl.White) //rl.Red)
}

func grClearScreen(sur *rl.RenderTexture2D) {
	// NOTE: original game colors the key color, but when it renders does it show up? I doubt it.
	//keyKnockoutColor := color.RGBA{R: 0xa8, G: 0x00, B: 0xa8, A: 0xff}
	//keyKnockoutColor := color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00}
	rl.BeginTextureMode(*sur)
	defer rl.EndTextureMode()

	rl.ClearBackground(rl.Blank)
}

// grCopyBuffer implements TTM opcode 0xB606. DGDS names the three buffers
// background (0), stored area (1), and composition (2). This port represents
// composition as the current TTM thread's layer and stored area as the shared
// saved-zones layer.
func grCopyBuffer(ttmThread *TTtmThread, x, y, width, height, sourceID, destinationID uint16) {
	if ttmThread == nil || width == 0 || height == 0 || x >= screenWidth || y >= screenHeight {
		return
	}
	if sourceID > ttmBufferComposition || destinationID > ttmBufferComposition {
		log.Printf("ignoring DRAW_SCREEN with invalid buffers %d -> %d", sourceID, destinationID)
		return
	}
	if sourceID == destinationID {
		return
	}

	if int(x)+int(width) > screenWidth {
		width = uint16(screenWidth - int(x))
	}
	if int(y)+int(height) > screenHeight {
		height = uint16(screenHeight - int(y))
	}

	source := grTTMBuffer(ttmThread, sourceID, false)
	destination := grTTMBuffer(ttmThread, destinationID, true)
	if source == nil || destination == nil || source.Texture.ID == 0 || destination.ID == 0 {
		log.Printf("ignoring DRAW_SCREEN with unavailable buffers %d -> %d", sourceID, destinationID)
		return
	}

	// Render textures are vertically inverted in OpenGL. Select the matching
	// logical rectangle upside-down so the copy remains upright when the
	// destination render texture is later displayed.
	grCopyRenderTextureRegion(source, destination, x, y, width, height)
}

func grCopyRenderTextureRegion(source, destination *rl.RenderTexture2D, x, y, width, height uint16) {
	if source == nil || destination == nil || width == 0 || height == 0 {
		return
	}
	sourceRect := grRenderTextureRect(x, y, width, height, source.Texture.Height)
	destinationRect := rl.NewRectangle(float32(x), float32(y), float32(width), float32(height))
	clipped := grBeginLayerDraw(destination)
	defer grEndLayerDraw(clipped)
	rl.DrawTexturePro(source.Texture, sourceRect, destinationRect, rl.Vector2Zero(), 0, rl.White)
}

func grTTMBuffer(ttmThread *TTtmThread, bufferID uint16, create bool) *rl.RenderTexture2D {
	switch bufferID {
	case ttmBufferBackground:
		if grBackgroundSur == nil && create {
			grBackgroundSur = grNewLayer()
			grClearScreen(grBackgroundSur)
		}
		return grBackgroundSur
	case ttmBufferStoredArea:
		if grSavedZonesLayer == nil && create {
			grSavedZonesLayer = grNewLayer()
			grClearScreen(grSavedZonesLayer)
		}
		return grSavedZonesLayer
	case ttmBufferComposition:
		if ttmThread.ttmLayer == nil && create {
			ttmThread.ttmLayer = grNewLayer()
			grClearScreen(ttmThread.ttmLayer)
		}
		return ttmThread.ttmLayer
	default:
		return nil
	}
}

func grRenderTextureRect(x, y, width, height uint16, textureHeight int32) rl.Rectangle {
	return rl.NewRectangle(
		float32(x),
		float32(textureHeight)-float32(y)-float32(height),
		float32(width),
		-float32(height),
	)
}

func grLoadScreen(screenName string) {
	if grBackgroundSur != nil {
		grReleaseScreen()
	}

	if grSavedZonesLayer != nil {
		grReleaseSavedLayer()
	}

	scrResource := findSCRResource(screenName)
	pixelData, width, height := screenPixelData(scrResource)

	spriteImg := rl.NewImage(pixelData, int32(width), int32(height), 1, rl.UncompressedR8g8b8a8)
	spriteTexture := rl.LoadTextureFromImage(spriteImg)
	defer rl.UnloadTexture(spriteTexture)

	// Original SCR files are not all a full 640x480. Preserve every source
	// pixel at its original position, but pad short backgrounds to the engine
	// canvas with the dominant color from their bottom row. This avoids a black
	// band without vertically stretching the artwork or misaligning sprites.
	fillColor := dominantBottomColor(pixelData, width, height)

	rt := rl.LoadRenderTexture(screenWidth, screenHeight)
	setRenderTextureFilter(&rt)
	grBackgroundSur = &rt

	rl.BeginTextureMode(rt)
	defer rl.EndTextureMode()

	rl.ClearBackground(fillColor)
	rl.DrawTexture(spriteTexture, 0, 0, rl.White)
}

func screenPixelData(scrResource *TSCRResource) ([]byte, int, int) {
	if scrResource == nil {
		panic("grLoadScreen(): nil screen resource")
	}

	if (scrResource.Width % 2) == 1 {
		panic("Warning: grLoadScreen(): can't manage odd widths")
	}

	if scrResource.Width > 640 || scrResource.Height > 480 {
		panic("grLoadScreen(): can't manage more than 640x480 resolutions")
	}

	width := int(scrResource.Width)
	height := int(scrResource.Height)
	bytesPerRow := int(width) / 2
	if len(scrResource.UncompressedData) < bytesPerRow*height {
		panic("grLoadScreen(): screen data is shorter than its dimensions")
	}

	data := scrResource.UncompressedData
	pixelData := make([]byte, 4*width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			byteIdx := y*bytesPerRow + (x / 2)

			// NOTE: This is a 4bit/per pixel color index
			var colorIdx int
			if x%2 == 0 {
				colorIdx = int((data[byteIdx] >> 4) & 0x0f)
			} else {
				colorIdx = int(data[byteIdx] & 0x0f)
			}
			clr := ttmPalette[colorIdx]
			c := color.RGBA{
				// Note color order -> this matches what's in the C implementation.
				R: clr[2],
				G: clr[1],
				B: clr[0],
				A: 0xff,
			}

			idx := (y*width + x) * 4
			pixelData[idx] = c.R
			pixelData[idx+1] = c.G
			pixelData[idx+2] = c.B
			pixelData[idx+3] = c.A
		}
	}

	return pixelData, width, height
}

func dominantBottomColor(pixelData []byte, width, height int) color.RGBA {
	fillColor := color.RGBA{A: 0xff}
	if width <= 0 || height <= 0 || len(pixelData) < width*height*4 {
		return fillColor
	}

	counts := make(map[uint32]int)
	bestCount := 0
	for x := 0; x < width; x++ {
		idx := ((height-1)*width + x) * 4
		key := uint32(pixelData[idx])<<24 |
			uint32(pixelData[idx+1])<<16 |
			uint32(pixelData[idx+2])<<8 |
			uint32(pixelData[idx+3])
		counts[key]++
		if counts[key] > bestCount {
			bestCount = counts[key]
			fillColor = color.RGBA{
				R: pixelData[idx],
				G: pixelData[idx+1],
				B: pixelData[idx+2],
				A: pixelData[idx+3],
			}
		}
	}
	return fillColor
}

func grInitEmptyBackground() {
	if grBackgroundSur != nil {
		grReleaseScreen()
	}

	if grSavedZonesLayer != nil {
		grReleaseSavedLayer()
	}

	rt := rl.LoadRenderTexture(screenWidth, screenHeight)
	setRenderTextureFilter(&rt)
	grBackgroundSur = &rt

	rl.BeginTextureMode(*grBackgroundSur)
	rl.ClearBackground(rl.Black)
	rl.EndTextureMode()
}

func grLoadBmp(ttmSlot *TTtmSlot, slotNo uint16, name string) {
	if ttmSlot.numSprites[slotNo] != 0 {
		grReleaseBmp(ttmSlot, slotNo)
	}

	bmpResource, found := lookupBMPResource(name)
	if !found {
		// FIRE.TTM in every verified original archive references these two
		// sprites without shipping them. Keep the slot empty so its draw
		// instructions become no-ops instead of terminating the application.
		if name == "FLAME.BMP" || name == "FLURRY.BMP" {
			ttmSlot.numSprites[slotNo] = 0
			log.Printf("skipping known missing original sprite %s", name)
			return
		}
		panic("bmp resource: " + name + " not found")
	}

	ttmSlot.numSprites[slotNo] = int(bmpResource.NumImages)

	data := bmpResource.UncompressedData
	dataOffset := 0 // dataOffset is where each bmp sprites data begins

	for img := 0; img < int(bmpResource.NumImages); img++ {
		if (bmpResource.Widths[img] % 2) == 1 {
			panic("grLoadBmp(): can't manage odd widths")
		}

		width := int(bmpResource.Widths[img])
		height := int(bmpResource.Heights[img])
		bytesPerRow := int(width) / 2

		pixelData := make([]byte, 4*width*height)

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				byteIdx := y*bytesPerRow + (x / 2)

				// NOTE: This is a 4bit/per pixel color index
				var colorIdx int
				if x%2 == 0 {
					colorIdx = int((data[byteIdx] >> 4) & 0x0f)
				} else {
					colorIdx = int(data[byteIdx] & 0x0f)
				}
				clr := ttmPalette[colorIdx]

				c := color.RGBA{
					// Note color order -> this matches what's in the C implementation.
					R: clr[2],
					G: clr[1],
					B: clr[0],
					A: 0xff,
				}

				// When Pink Key Color!!!! Knock it out!
				// if RGB => 0xa8, 0x00, 0xa8 it's the key color and must not be rendered,
				// hence alpha is set to 0x00.
				if clr[0] == 0xa8 && clr[1] == 0x00 && clr[2] == 0xa8 {
					c = color.RGBA{
						R: 0x0,
						G: 0x0,
						B: 0x0,
						A: 0x0,
					}
				}

				idx := (y*width + x) * 4
				pixelData[idx] = c.R
				pixelData[idx+1] = c.G
				pixelData[idx+2] = c.B
				pixelData[idx+3] = c.A

				dataOffset = byteIdx
			}
		}
		// segments the data to be the next cel of the sprite.
		data = data[dataOffset+1:]
		spriteImg := rl.NewImage(pixelData, int32(width), int32(height), 1, rl.UncompressedR8g8b8a8)
		spriteTexture := rl.LoadTextureFromImage(spriteImg)
		ttmSlot.sprites[slotNo][img] = &spriteTexture
	}
}

func grReleaseBmp(ttmSlot *TTtmSlot, bmpSlotNo uint16) {
	if ttmSlot == nil || bmpSlotNo >= MaxBMPSlots {
		return
	}
	for i := 0; i < ttmSlot.numSprites[bmpSlotNo]; i++ {
		spr := ttmSlot.sprites[bmpSlotNo][i]
		if spr != nil && spr.ID != 0 {
			rl.UnloadTexture(*spr)
			spr.ID = 0
		}
		ttmSlot.sprites[bmpSlotNo][i] = nil
	}

	ttmSlot.numSprites[bmpSlotNo] = 0
}

func grFadeOut() {
	// Does screen transitions like iris, rect sliding
	// Don't necessarily need this day 1
	// May be able to fake it with just simple assets
	isFadingOut = true
}
