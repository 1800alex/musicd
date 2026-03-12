<script setup lang="ts">
import { onMounted, ref, nextTick, defineProps, watch } from "vue";

export interface IColorPalette {
	background: string;
	text: string;
	whiteContrast: string;
	blackContrast: string;
	colors: string[];
}

// --- Props ---
const props = defineProps<{
	imageUrl: string | null;
}>();

// Color analysis variables
const isAnalyzingColor = ref(false);
const currentAnalysisUrl = ref<string | null>(null);
let debounceTimeout: ReturnType<typeof setTimeout> | null = null;

const palette = ref<IColorPalette>({
	background: "#000000",
	text: "#ffffff",
	whiteContrast: "4.5",
	blackContrast: "4.5",
	colors: ["#000000", "#ffffff"]
});

// Color analysis utilities
const rgbToHex = (r: number, g: number, b: number) => {
	return "#" + ((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1);
};

const hexToRgb = (hex: string) => {
	const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
	return result
		? {
				r: parseInt(result[1], 16),
				g: parseInt(result[2], 16),
				b: parseInt(result[3], 16)
			}
		: null;
};

const getLuminance = (hex: string) => {
	const rgb = hexToRgb(hex);
	if (!rgb) {
		return 0;
	}

	// Convert to relative luminance
	const rsRGB = rgb.r / 255;
	const gsRGB = rgb.g / 255;
	const bsRGB = rgb.b / 255;

	const r = rsRGB <= 0.03928 ? rsRGB / 12.92 : Math.pow((rsRGB + 0.055) / 1.055, 2.4);
	const g = gsRGB <= 0.03928 ? gsRGB / 12.92 : Math.pow((gsRGB + 0.055) / 1.055, 2.4);
	const b = bsRGB <= 0.03928 ? bsRGB / 12.92 : Math.pow((bsRGB + 0.055) / 1.055, 2.4);

	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};

const getContrastRatio = (color1: string, color2: string) => {
	const lum1 = getLuminance(color1);
	const lum2 = getLuminance(color2);
	const brightest = Math.max(lum1, lum2);
	const darkest = Math.min(lum1, lum2);
	return (brightest + 0.05) / (darkest + 0.05);
};

const analyzeImageColors = (imageUrl: string): Promise<string[]> => {
	return new Promise((resolve, reject) => {
		const img = new Image();
		img.crossOrigin = "anonymous";

		img.onload = () => {
			// Ensure image is fully loaded
			if (img.complete && img.naturalWidth > 0 && img.naturalHeight > 0) {
				const canvas = document.createElement("canvas");
				const ctx = canvas.getContext("2d");
				if (!ctx) {
					reject(new Error("Could not get canvas context"));
					return;
				}

				canvas.width = img.width;
				canvas.height = img.height;
				ctx.drawImage(img, 0, 0);

				const colorCounts = new Map<string, number>();

				// Sample colors from different areas of the image
				const samplePoints = [
					// Four corners
					{ x: 0, y: 0 },
					{ x: img.width - 1, y: 0 },
					{ x: 0, y: img.height - 1 },
					{ x: img.width - 1, y: img.height - 1 },
					// Center points
					{ x: Math.floor(img.width / 2), y: Math.floor(img.height / 2) },
					// Edge centers
					{ x: Math.floor(img.width / 2), y: 0 },
					{ x: Math.floor(img.width / 2), y: img.height - 1 },
					{ x: 0, y: Math.floor(img.height / 2) },
					{ x: img.width - 1, y: Math.floor(img.height / 2) },
					// Additional strategic points
					{ x: Math.floor(img.width * 0.25), y: Math.floor(img.height * 0.25) },
					{ x: Math.floor(img.width * 0.75), y: Math.floor(img.height * 0.75) }
				];

				// Sample more points for better color analysis
				for (let i = 0; i < 50; i++) {
					const x = Math.floor(Math.random() * img.width);
					const y = Math.floor(Math.random() * img.height);
					samplePoints.push({ x, y });
				}

				samplePoints.forEach((point) => {
					try {
						const imageData = ctx.getImageData(point.x, point.y, 1, 1);
						const [r, g, b] = imageData.data;
						const hex = rgbToHex(r, g, b);

						const count = colorCounts.get(hex) || 0;
						colorCounts.set(hex, count + 1);
					} catch (e) {
						// Skip if point is out of bounds
					}
				});

				// Get most frequent colors
				const sortedColors = Array.from(colorCounts.entries())
					.sort((a, b) => b[1] - a[1])
					.map(([color]) => color);

				resolve(sortedColors.slice(0, 5));
			} else {
				reject(new Error("Image not fully loaded or has invalid dimensions"));
			}
		};

		img.onerror = () => {
			reject(new Error("Failed to load image"));
		};

		img.src = imageUrl;
	});
};

const updateHeroColors = async (imageUrl: string) => {
	if (!imageUrl) {
		return;
	}

	// Avoid analyzing the same image multiple times
	if (currentAnalysisUrl.value === imageUrl) {
		return;
	}

	// Debounce rapid image URL changes
	if (debounceTimeout) {
		clearTimeout(debounceTimeout);
	}

	isAnalyzingColor.value = true;
	currentAnalysisUrl.value = imageUrl;

	try {
		const colors = await analyzeImageColors(imageUrl);

		if (0 === colors.length) {
			throw new Error("No colors extracted from image");
		}

		// Find the best background color (avoid pure black/white, prefer vibrant colors)
		let bestColor = colors[0];

		for (const color of colors) {
			const rgb = hexToRgb(color);
			if (!rgb) {
				continue;
			}

			// Skip colors that are too dark or too light
			const luminance = getLuminance(color);
			if (luminance > 0.05 && luminance < 0.8) {
				bestColor = color;
				break;
			}
		}

		// Ensure we have a valid color
		if (!bestColor || "#000000" === bestColor) {
			const fallbackColor =
				getComputedStyle(document.documentElement).getPropertyValue("--clr-primary").trim() || "#96aae0";
			console.warn("No suitable background color found, defaulting to", fallbackColor);
			bestColor = colors.find((color) => color !== "#000000") || fallbackColor;
		}

		// Determine text color based on background luminance
		const backgroundLuminance = getLuminance(bestColor);
		const whiteContrast = getContrastRatio(bestColor, "#ffffff");
		const blackContrast = getContrastRatio(bestColor, "#000000");

		palette.value.background = bestColor;
		// Use luminance threshold: if background is light (>0.5), use black text; otherwise use white text
		// Also ensure we meet WCAG contrast requirements (4.5:1 minimum)
		if (backgroundLuminance > 0.5) {
			// Light background - use black text if contrast is good enough, otherwise use white
			palette.value.text = blackContrast >= 4.5 ? "#000000" : "#ffffff";
		} else {
			// Dark background - use white text if contrast is good enough, otherwise use black
			palette.value.text = whiteContrast >= 4.5 ? "#ffffff" : "#000000";
		}
		palette.value.whiteContrast = whiteContrast.toFixed(2);
		palette.value.blackContrast = blackContrast.toFixed(2);
		palette.value.colors = colors.slice(0, 3);

		// emit color changes
		emit("colors", palette.value);
	} catch (error) {
		console.error("Error analyzing image colors:", error);
		// Fall back to theme colors (read from CSS custom properties)
		const getThemeColor = (varName: string) => {
			return getComputedStyle(document.documentElement).getPropertyValue(varName).trim();
		};

		const fallbackColors = [
			getThemeColor("--clr-primary") || "#96aae0",
			getThemeColor("--clr-success") || "#22946e",
			getThemeColor("--clr-warning") || "#a87a2a",
			getThemeColor("--clr-info") || "#21498A",
			getThemeColor("--clr-danger") || "#9c2121",
			getThemeColor("--clr-surface-higher") || "#575757"
		].filter((color) => color && color !== "");

		const randomColor = fallbackColors[Math.floor(Math.random() * fallbackColors.length)];
		palette.value.background = randomColor;
		palette.value.text = "#ffffff";

		console.log("Using fallback color:", randomColor);
	} finally {
		isAnalyzingColor.value = false;
	}
};

const emit = defineEmits<{
	colors: [palette: IColorPalette];
}>();

// Watch for image url changes with debouncing
watch(
	() => props.imageUrl,
	(newUrl) => {
		if (newUrl) {
			// Clear any existing debounce timeout
			if (debounceTimeout) {
				clearTimeout(debounceTimeout);
			}

			// Debounce the color analysis to avoid rapid consecutive calls
			debounceTimeout = setTimeout(() => {
				updateHeroColors(newUrl);
			}, 300); // 300ms debounce
		} else {
			// Reset to default colors when no image
			if (debounceTimeout) {
				clearTimeout(debounceTimeout);
			}
			isAnalyzingColor.value = false;
			currentAnalysisUrl.value = null;
			palette.value = {
				background: "#000000",
				text: "#ffffff",
				whiteContrast: "4.5",
				blackContrast: "4.5",
				colors: ["#000000", "#ffffff"]
			};
		}
	}
);

onMounted(() => {
	if (props.imageUrl) {
		// Use a small delay to ensure DOM is ready and image has a chance to load
		setTimeout(() => {
			updateHeroColors(props.imageUrl!);
		}, 100);
	}
});
</script>

<template>
	<section
		class="hero"
		:class="{ 'is-analyzing-colors': isAnalyzingColor }"
		:style="{
			backgroundColor: palette.background,
			transition: 'background-color 0.3s ease-in-out'
		}"
	>
		<div class="hero-body" :style="{ color: palette.text }">
			<slot />
		</div>
	</section>
</template>
