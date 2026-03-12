package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WaveformGenerator handles the creation of waveform icons
type WaveformGenerator struct {
	Width     int
	Height    int
	Color     string
	BgColor   string
	BarCount  int
	MinHeight float64
	MaxHeight float64
}

// NewWaveformGenerator creates a new generator with default settings
func NewWaveformGenerator() *WaveformGenerator {
	return &WaveformGenerator{
		Width:     48,
		Height:    48,
		Color:     "#2563eb", // Blue color
		BgColor:   "transparent",
		BarCount:  12,
		MinHeight: 0.1,
		MaxHeight: 0.9,
	}
}

// GenerateSVG creates an SVG representation of a waveform
func (w *WaveformGenerator) GenerateSVG() string {
	rand.Seed(time.Now().UnixNano())
	heights := make([]float64, w.BarCount)
	for i := range heights {
		baseHeight := 0.3 + 0.4*math.Sin(float64(i)*math.Pi/float64(w.BarCount-1))
		variation := (rand.Float64() - 0.5) * 0.4
		heights[i] = math.Max(w.MinHeight, math.Min(w.MaxHeight, baseHeight+variation))
	}

	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		w.Width, w.Height, w.Width, w.Height))

	if w.BgColor != "transparent" {
		svg.WriteString(fmt.Sprintf(`<rect width="100%%" height="100%%" fill="%s"/>`, w.BgColor))
	}

	totalPadding := float64(w.Width) * 0.2
	barAreaWidth := float64(w.Width) - totalPadding
	barWidth := barAreaWidth / float64(w.BarCount)
	padding := barWidth * 0.2
	actualBarWidth := barWidth - padding

	startX := totalPadding / 2

	for i, height := range heights {
		barHeight := height * float64(w.Height) * 0.8
		x := startX + float64(i)*barWidth
		y := (float64(w.Height) - barHeight) / 2

		svg.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" rx="%.1f"/>`,
			x, y, actualBarWidth, barHeight, w.Color, actualBarWidth/4,
		))
	}

	svg.WriteString("</svg>")
	return svg.String()
}

// GenerateSplashSVG creates an SVG splash screen: centered waveform on a background
func (w *WaveformGenerator) GenerateSplashSVG(canvasWidth, canvasHeight int) string {
	rand.Seed(time.Now().UnixNano())
	heights := make([]float64, w.BarCount)
	for i := range heights {
		baseHeight := 0.3 + 0.4*math.Sin(float64(i)*math.Pi/float64(w.BarCount-1))
		variation := (rand.Float64() - 0.5) * 0.4
		heights[i] = math.Max(w.MinHeight, math.Min(w.MaxHeight, baseHeight+variation))
	}

	// The waveform graphic is drawn at a fixed logical size, centered on the canvas
	iconSize := 200 // logical size of the waveform area
	offsetX := (canvasWidth - iconSize) / 2
	offsetY := (canvasHeight - iconSize) / 2

	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		canvasWidth, canvasHeight, canvasWidth, canvasHeight))

	// Background
	svg.WriteString(fmt.Sprintf(`<rect width="100%%" height="100%%" fill="%s"/>`, w.BgColor))

	totalPadding := float64(iconSize) * 0.2
	barAreaWidth := float64(iconSize) - totalPadding
	barWidth := barAreaWidth / float64(w.BarCount)
	padding := barWidth * 0.2
	actualBarWidth := barWidth - padding
	startX := totalPadding / 2

	for i, height := range heights {
		barHeight := height * float64(iconSize) * 0.8
		x := float64(offsetX) + startX + float64(i)*barWidth
		y := float64(offsetY) + (float64(iconSize)-barHeight)/2

		svg.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" rx="%.1f"/>`,
			x, y, actualBarWidth, barHeight, w.Color, actualBarWidth/4,
		))
	}

	svg.WriteString("</svg>")
	return svg.String()
}

// SaveSVG saves the SVG to a file
func (w *WaveformGenerator) SaveSVG(filename string) error {
	svg := w.GenerateSVG()
	return os.WriteFile(filename, []byte(svg), 0644)
}

func saveSVGString(filename, svg string) error {
	return os.WriteFile(filename, []byte(svg), 0644)
}

// checkFFmpeg verifies that ffmpeg is installed and available
func checkFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg not found. Please install ffmpeg to convert SVG to other formats")
	}
	return nil
}

// convertSVGToPNG converts SVG to PNG using ffmpeg
func convertSVGToPNG(svgPath, pngPath string, width, height int) error {
	cmd := exec.Command("ffmpeg", "-i", svgPath, "-vf", fmt.Sprintf("scale=%d:%d", width, height), "-y", pngPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to convert SVG to PNG (%dx%d): %v\nOutput: %s", width, height, err, string(output))
	}
	return nil
}

// convertSVGToICO converts SVG to ICO using ffmpeg
func convertSVGToICO(svgPath, icoPath string) error {
	cmd := exec.Command("ffmpeg", "-i", svgPath, "-vf", "scale=32:32", "-y", icoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to convert SVG to ICO: %v\nOutput: %s", err, string(output))
	}
	return nil
}

type iconOutput struct {
	path   string
	width  int
	height int
}

// GenerateAllFormats creates all icon and splash variants
func (w *WaveformGenerator) GenerateAllFormats(outputDir, name string) error {
	basePath := filepath.Join(outputDir, name)
	svgPath := basePath + ".svg"

	// ── Generate master SVG ──────────────────────────────────────────────

	fmt.Println("Generating SVG...")
	err := w.SaveSVG(svgPath)
	if err != nil {
		return fmt.Errorf("failed to save SVG: %v", err)
	}

	if err := checkFFmpeg(); err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Only SVG files were generated.")
		return nil
	}

	// ── Standard web icons ───────────────────────────────────────────────

	webIcons := []iconOutput{
		{basePath + "_16x16.png", 16, 16},
		{basePath + "_24x24.png", 24, 24},
		{basePath + "_48x48.png", 48, 48},
	}

	// ICO
	fmt.Println("Converting to ICO...")
	if err := convertSVGToICO(svgPath, basePath+".ico"); err != nil {
		return err
	}

	for _, ic := range webIcons {
		fmt.Printf("Generating %dx%d PNG...\n", ic.width, ic.height)
		if err := convertSVGToPNG(svgPath, ic.path, ic.width, ic.height); err != nil {
			return err
		}
	}

	// ── PWA icons ────────────────────────────────────────────────────────

	pwaDir := filepath.Join(outputDir, "pwa")
	os.MkdirAll(pwaDir, 0755)

	pwaIcons := []iconOutput{
		{filepath.Join(pwaDir, "icon-192.png"), 192, 192},
		{filepath.Join(pwaDir, "icon-512.png"), 512, 512},
	}

	for _, ic := range pwaIcons {
		fmt.Printf("Generating PWA icon %dx%d...\n", ic.width, ic.height)
		if err := convertSVGToPNG(svgPath, ic.path, ic.width, ic.height); err != nil {
			return err
		}
	}

	// PWA maskable icons (same waveform but with solid background and safe-zone padding)
	maskGen := *w
	maskGen.BgColor = "#1a1a1a"
	maskGen.Width = 512
	maskGen.Height = 512

	maskSVGPath := filepath.Join(pwaDir, "maskable-master.svg")
	maskSVG := maskGen.GenerateMaskableSVG()
	if err := saveSVGString(maskSVGPath, maskSVG); err != nil {
		return fmt.Errorf("failed to save maskable SVG: %v", err)
	}

	maskableIcons := []iconOutput{
		{filepath.Join(pwaDir, "icon-192-maskable.png"), 192, 192},
		{filepath.Join(pwaDir, "icon-512-maskable.png"), 512, 512},
	}

	for _, ic := range maskableIcons {
		fmt.Printf("Generating PWA maskable icon %dx%d...\n", ic.width, ic.height)
		if err := convertSVGToPNG(maskSVGPath, ic.path, ic.width, ic.height); err != nil {
			return err
		}
	}

	// Apple touch icon (180x180 with background)
	fmt.Println("Generating apple-touch-icon...")
	if err := convertSVGToPNG(maskSVGPath, filepath.Join(pwaDir, "apple-touch-icon.png"), 180, 180); err != nil {
		return err
	}

	// ── iOS Capacitor icon ───────────────────────────────────────────────

	iosDir := filepath.Join(outputDir, "ios")
	os.MkdirAll(iosDir, 0755)

	// iOS requires a 1024x1024 icon (no transparency, solid background)
	iosGen := *w
	iosGen.BgColor = "#1a1a1a"
	iosGen.Width = 1024
	iosGen.Height = 1024
	iosSVGPath := filepath.Join(iosDir, "ios-master.svg")
	iosSVG := iosGen.GenerateSVG()
	if err := saveSVGString(iosSVGPath, iosSVG); err != nil {
		return fmt.Errorf("failed to save iOS SVG: %v", err)
	}

	fmt.Println("Generating iOS app icon 1024x1024...")
	if err := convertSVGToPNG(iosSVGPath, filepath.Join(iosDir, "AppIcon-512@2x.png"), 1024, 1024); err != nil {
		return err
	}

	// ── Android Capacitor icons ──────────────────────────────────────────

	androidDir := filepath.Join(outputDir, "android")
	os.MkdirAll(androidDir, 0755)

	// Launcher icons (full icon with background)
	androidGen := *w
	androidGen.BgColor = "#1a1a1a"
	androidGen.Width = 512
	androidGen.Height = 512
	androidSVGPath := filepath.Join(androidDir, "android-master.svg")
	androidSVG := androidGen.GenerateSVG()
	if err := saveSVGString(androidSVGPath, androidSVG); err != nil {
		return fmt.Errorf("failed to save Android SVG: %v", err)
	}

	// Standard launcher + round icons per density
	type androidDensity struct {
		name string
		size int
	}
	densities := []androidDensity{
		{"mdpi", 48},
		{"hdpi", 72},
		{"xhdpi", 96},
		{"xxhdpi", 144},
		{"xxxhdpi", 192},
	}

	for _, d := range densities {
		dDir := filepath.Join(androidDir, "mipmap-"+d.name)
		os.MkdirAll(dDir, 0755)

		fmt.Printf("Generating Android %s launcher icon %dx%d...\n", d.name, d.size, d.size)
		if err := convertSVGToPNG(androidSVGPath, filepath.Join(dDir, "ic_launcher.png"), d.size, d.size); err != nil {
			return err
		}
		// Round variant is the same image
		if err := convertSVGToPNG(androidSVGPath, filepath.Join(dDir, "ic_launcher_round.png"), d.size, d.size); err != nil {
			return err
		}
	}

	// Adaptive icon foreground (waveform on transparent, larger canvas with safe zone)
	// Android adaptive foreground is 108dp per density where the icon occupies the inner 72dp
	adaptiveGen := *w
	adaptiveGen.BgColor = "transparent"
	adaptiveGen.Width = 108
	adaptiveGen.Height = 108
	// Waveform drawn at smaller scale within the 108x108 canvas to stay in the 66dp safe zone
	adaptiveSVG := adaptiveGen.GenerateAdaptiveForegroundSVG()
	adaptiveSVGPath := filepath.Join(androidDir, "adaptive-foreground.svg")
	if err := saveSVGString(adaptiveSVGPath, adaptiveSVG); err != nil {
		return fmt.Errorf("failed to save adaptive foreground SVG: %v", err)
	}

	type adaptiveDensity struct {
		name string
		size int // foreground size
	}
	adaptiveDensities := []adaptiveDensity{
		{"mdpi", 108},
		{"hdpi", 162},
		{"xhdpi", 216},
		{"xxhdpi", 324},
		{"xxxhdpi", 432},
	}

	for _, d := range adaptiveDensities {
		dDir := filepath.Join(androidDir, "mipmap-"+d.name)
		os.MkdirAll(dDir, 0755)

		fmt.Printf("Generating Android %s adaptive foreground %dx%d...\n", d.name, d.size, d.size)
		if err := convertSVGToPNG(adaptiveSVGPath, filepath.Join(dDir, "ic_launcher_foreground.png"), d.size, d.size); err != nil {
			return err
		}
	}

	// ── Splash screens ───────────────────────────────────────────────────

	splashDir := filepath.Join(outputDir, "splash")
	os.MkdirAll(splashDir, 0755)

	// Generate a high-res splash SVG (square, for iOS)
	splashSVG := w.GenerateSplashSVG(2732, 2732)
	splashSVGSquarePath := filepath.Join(splashDir, "splash-square.svg")
	if err := saveSVGString(splashSVGSquarePath, splashSVG); err != nil {
		return fmt.Errorf("failed to save splash SVG: %v", err)
	}

	// iOS splash: 3 copies at 2732x2732 (for 1x, 2x, 3x scales — same source image)
	fmt.Println("Generating iOS splash screens (2732x2732)...")
	iosSplashDir := filepath.Join(splashDir, "ios")
	os.MkdirAll(iosSplashDir, 0755)
	for _, suffix := range []string{"splash-2732x2732.png", "splash-2732x2732-1.png", "splash-2732x2732-2.png"} {
		if err := convertSVGToPNG(splashSVGSquarePath, filepath.Join(iosSplashDir, suffix), 2732, 2732); err != nil {
			return err
		}
	}

	// Android splash screens — portrait and landscape per density
	type splashSize struct {
		dir    string
		width  int
		height int
	}
	androidSplashes := []splashSize{
		// Portrait
		{"drawable-port-mdpi", 320, 480},
		{"drawable-port-hdpi", 480, 800},
		{"drawable-port-xhdpi", 720, 1280},
		{"drawable-port-xxhdpi", 960, 1600},
		{"drawable-port-xxxhdpi", 1280, 1920},
		// Landscape
		{"drawable-land-mdpi", 480, 320},
		{"drawable-land-hdpi", 800, 480},
		{"drawable-land-xhdpi", 1280, 720},
		{"drawable-land-xxhdpi", 1600, 960},
		{"drawable-land-xxxhdpi", 1920, 1280},
		// Default (landscape)
		{"drawable", 480, 320},
	}

	for _, s := range androidSplashes {
		sDir := filepath.Join(splashDir, "android", s.dir)
		os.MkdirAll(sDir, 0755)

		// Generate a splash SVG at the correct aspect ratio
		splSVG := w.GenerateSplashSVG(s.width, s.height)
		splSVGPath := filepath.Join(sDir, "splash-src.svg")
		if err := saveSVGString(splSVGPath, splSVG); err != nil {
			return err
		}

		fmt.Printf("Generating Android splash %s (%dx%d)...\n", s.dir, s.width, s.height)
		if err := convertSVGToPNG(splSVGPath, filepath.Join(sDir, "splash.png"), s.width, s.height); err != nil {
			return err
		}
		// Clean up intermediate SVG
		os.Remove(splSVGPath)
	}

	return nil
}

// GenerateMaskableSVG creates an SVG with the waveform drawn inside the maskable safe zone
// (inner 80% circle). The icon has a solid background and extra padding.
func (w *WaveformGenerator) GenerateMaskableSVG() string {
	rand.Seed(time.Now().UnixNano())
	heights := make([]float64, w.BarCount)
	for i := range heights {
		baseHeight := 0.3 + 0.4*math.Sin(float64(i)*math.Pi/float64(w.BarCount-1))
		variation := (rand.Float64() - 0.5) * 0.4
		heights[i] = math.Max(w.MinHeight, math.Min(w.MaxHeight, baseHeight+variation))
	}

	size := w.Width // assume square

	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		size, size, size, size))

	// Solid background
	svg.WriteString(fmt.Sprintf(`<rect width="100%%" height="100%%" fill="%s"/>`, w.BgColor))

	// Maskable safe zone is the inner 80% — draw waveform within ~60% to keep clear of edges
	innerSize := float64(size) * 0.6
	offset := (float64(size) - innerSize) / 2

	totalPadding := innerSize * 0.2
	barAreaWidth := innerSize - totalPadding
	barWidth := barAreaWidth / float64(w.BarCount)
	barPadding := barWidth * 0.2
	actualBarWidth := barWidth - barPadding
	startX := totalPadding / 2

	for i, height := range heights {
		barHeight := height * innerSize * 0.8
		x := offset + startX + float64(i)*barWidth
		y := offset + (innerSize-barHeight)/2

		svg.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" rx="%.1f"/>`,
			x, y, actualBarWidth, barHeight, w.Color, actualBarWidth/4,
		))
	}

	svg.WriteString("</svg>")
	return svg.String()
}

// GenerateAdaptiveForegroundSVG creates an SVG for Android adaptive icon foreground.
// The canvas is 108dp but the visible icon area is 72dp centered (the outer 18dp on each side
// may be clipped by different device masks).
func (w *WaveformGenerator) GenerateAdaptiveForegroundSVG() string {
	rand.Seed(time.Now().UnixNano())
	heights := make([]float64, w.BarCount)
	for i := range heights {
		baseHeight := 0.3 + 0.4*math.Sin(float64(i)*math.Pi/float64(w.BarCount-1))
		variation := (rand.Float64() - 0.5) * 0.4
		heights[i] = math.Max(w.MinHeight, math.Min(w.MaxHeight, baseHeight+variation))
	}

	canvasSize := 108
	safeZone := 66 // inner area that is always visible
	safeOffset := (canvasSize - safeZone) / 2

	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		canvasSize, canvasSize, canvasSize, canvasSize))

	innerSize := float64(safeZone)
	offset := float64(safeOffset)

	totalPadding := innerSize * 0.2
	barAreaWidth := innerSize - totalPadding
	barWidth := barAreaWidth / float64(w.BarCount)
	barPadding := barWidth * 0.2
	actualBarWidth := barWidth - barPadding
	startX := totalPadding / 2

	for i, height := range heights {
		barHeight := height * innerSize * 0.8
		x := offset + startX + float64(i)*barWidth
		y := offset + (innerSize-barHeight)/2

		svg.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" rx="%.1f"/>`,
			x, y, actualBarWidth, barHeight, w.Color, actualBarWidth/4,
		))
	}

	svg.WriteString("</svg>")
	return svg.String()
}

var outputDir = ""
var name = "waveform"

func main() {
	flag.StringVar(&outputDir, "o", "output", "Output directory for generated files")
	flag.StringVar(&name, "name", "waveform", "Base name for generated files (without extension)")
	flag.Parse()

	if outputDir == "" {
		log.Fatal("Output directory must be specified with -o flag")
	}

	generator := NewWaveformGenerator()

	os.MkdirAll(outputDir, 0755)

	err := generator.GenerateAllFormats(outputDir, name)
	if err != nil {
		log.Fatalf("Error generating files: %v", err)
	}

	fmt.Println("\nGeneration completed successfully!")
	fmt.Println("\nGenerated output structure:")
	fmt.Printf("  %s/\n", outputDir)
	fmt.Printf("  ├── %s.svg             (master SVG)\n", name)
	fmt.Printf("  ├── %s.ico             (favicon)\n", name)
	fmt.Printf("  ├── %s_16x16.png       (web)\n", name)
	fmt.Printf("  ├── %s_24x24.png       (web)\n", name)
	fmt.Printf("  ├── %s_48x48.png       (web)\n", name)
	fmt.Printf("  ├── pwa/\n")
	fmt.Printf("  │   ├── icon-192.png\n")
	fmt.Printf("  │   ├── icon-512.png\n")
	fmt.Printf("  │   ├── icon-192-maskable.png\n")
	fmt.Printf("  │   ├── icon-512-maskable.png\n")
	fmt.Printf("  │   └── apple-touch-icon.png\n")
	fmt.Printf("  ├── ios/\n")
	fmt.Printf("  │   └── AppIcon-512@2x.png  (1024x1024)\n")
	fmt.Printf("  ├── android/\n")
	fmt.Printf("  │   ├── mipmap-mdpi/        (48, 108)\n")
	fmt.Printf("  │   ├── mipmap-hdpi/        (72, 162)\n")
	fmt.Printf("  │   ├── mipmap-xhdpi/       (96, 216)\n")
	fmt.Printf("  │   ├── mipmap-xxhdpi/      (144, 324)\n")
	fmt.Printf("  │   └── mipmap-xxxhdpi/     (192, 432)\n")
	fmt.Printf("  └── splash/\n")
	fmt.Printf("      ├── ios/                (2732x2732)\n")
	fmt.Printf("      └── android/            (all densities, portrait + landscape)\n")
}
