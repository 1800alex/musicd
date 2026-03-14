<script setup lang="ts">
/**
 * WaveformBackground.vue
 * ----------------------
 * A drop‑in Vue 3 + TypeScript component that draws a glowing, animated waveform
 * background driven by an existing HTMLAudioElement passed via the `audioEl` prop.
 *
 * ✅ Keeps animating while audio is PAUSED (trail/sparkle effect continues with flat waveform)
 * ✅ Persists colors when the source changes rapidly (no hue reset on src changes)
 * ✅ Avoids NotSupportedError by creating MediaElementAudioSourceNode only once,
 *    and falling back to audioEl.captureStream() if needed.
 * ✅ Visualization‑only: does not mutate or control the provided audio element.
 */

import { onMounted, onBeforeUnmount, ref, watch, nextTick, defineProps, defineExpose } from "vue";
import type { Track } from "~/types";

// --- Props ---
const props = defineProps<{
	/** Existing HTMLAudioElement that is your source player (required). */
	audioEl: HTMLAudioElement;
	/** Whether to show the visualizer. Default: true */
	visible: boolean;
	/** Application state (for displaying current track info) */
	currentTrack: Track | null;
	/** If true, canvas is positioned fixed and fills the viewport behind content. */
	fixed?: boolean;
	/** Visual intensity multiplier (0.5..2). Default: 1 */
	intensity?: number;
	/** Global line scale (0.5..2). Default: 1 */
	lineScale?: number;
	/** Whether to draw a soft vignette. Default: true */
	vignette?: boolean;
	/** CPU performance mode: "low" | "medium" | "high" | "ultra". Default: "medium" */
	performance?: "low" | "medium" | "high" | "ultra";
}>();

const fixed = props.fixed ?? true;
const intensity = props.intensity ?? 1;
const lineScale = props.lineScale ?? 1;
const useVignette = props.vignette ?? true;
const performance = props.performance ?? "medium";

// Performance configuration
// "low" - Best for older devices/browsers (30fps, minimal effects)
// "medium" - Balanced performance (45fps, good visual quality)
// "high" - Good performance devices (60fps, enhanced effects)
// "ultra" - High-end devices only (60fps, maximum effects)
const getPerformanceConfig = () => {
	switch (performance) {
		case "low":
			return {
				fftSize: 2048, // Lower FFT size
				smoothing: 0, // More smoothing
				targetFPS: 10, // Lower frame rate
				pointDensity: 100, // Fewer points
				shadowBlur: 0, // Less blur
				particleSkip: 12, // Skip more particles
				devicePixelRatio: 1 // Force low DPR
			};
		case "medium":
			return {
				fftSize: 2048,
				smoothing: 0.85,
				targetFPS: 45,
				pointDensity: 200,
				shadowBlur: 40,
				particleSkip: 8,
				devicePixelRatio: Math.min(2, window.devicePixelRatio || 1)
			};
		case "high":
			return {
				fftSize: 2048,
				smoothing: 0.8,
				targetFPS: 60,
				pointDensity: 300,
				shadowBlur: 60,
				particleSkip: 6,
				devicePixelRatio: Math.min(2.5, window.devicePixelRatio || 1)
			};
		case "ultra":
			return {
				fftSize: 4096, // Higher FFT size
				smoothing: 0.75, // Less smoothing
				targetFPS: 60, // Max frame rate
				pointDensity: 500, // More points
				shadowBlur: 80, // More blur
				particleSkip: 3, // Fewer particle skips
				devicePixelRatio: window.devicePixelRatio || 1
			};
		default:
			return getPerformanceConfig(); // Default to medium
	}
};

const perfConfig = getPerformanceConfig();

const emit = defineEmits<{
	close: [val: boolean];
}>();

// --- Refs & state ---
const canvas = ref<HTMLCanvasElement | null>(null);
let ctx: CanvasRenderingContext2D | null = null;

const ac = ref<AudioContext | null>(null);
let analyserWave: AnalyserNode | null = null;
let analyserFreq: AnalyserNode | null = null;
let elementSource: MediaElementAudioSourceNode | null = null;
let streamSource: MediaStreamAudioSourceNode | null = null;
let activeSource: (MediaElementAudioSourceNode | MediaStreamAudioSourceNode | GainNode) | null = null;

let rafId: number | null = null;
let unsubResize: (() => void) | null = null;

const waveArray = new Uint8Array(2048);
const freqArray = new Uint8Array(512);
let hue = 220; // persisted across source changes
let glow = 0.7;
let time = 0;

const err = ref("");

// WeakMap to remember which elements already have MediaElementAudioSourceNode created by US
const elementToSource = new WeakMap<HTMLMediaElement, MediaElementAudioSourceNode>();

function close() {
	// emit
	emit("close", true);
}

function setError(message: string) {
	err.value = message;
}
function clearError() {
	err.value = "";
}

function ensureAudio(): AudioContext {
	if (!ac.value) {
		ac.value = new (window.AudioContext || (window as any).webkitAudioContext)();
	}
	return ac.value;
}

function ensureAnalysers() {
	const ctx = ensureAudio();
	if (!analyserWave) {
		analyserWave = ctx.createAnalyser();
		analyserWave.fftSize = perfConfig.fftSize;
		analyserWave.smoothingTimeConstant = perfConfig.smoothing;
	}
	if (!analyserFreq) {
		analyserFreq = ctx.createAnalyser();
		analyserFreq.fftSize = Math.max(512, perfConfig.fftSize / 2);
		analyserFreq.smoothingTimeConstant = perfConfig.smoothing;
	}
}

function disconnectAnalysers() {
	try {
		if (activeSource && analyserWave) {
			activeSource.disconnect(analyserWave);
		}
	} catch {}
	try {
		if (activeSource && analyserFreq) {
			activeSource.disconnect(analyserFreq);
		}
	} catch {}
}

function connectActive(node: MediaElementAudioSourceNode | MediaStreamAudioSourceNode | GainNode) {
	if (!node) {
		return;
	}
	disconnectAnalysers();
	activeSource = node;

	// Always connect to analyzers for visualization
	analyserWave && node.connect(analyserWave);
	analyserFreq && node.connect(analyserFreq);

	// For MediaElementAudioSourceNode, connect to destination only if not already connected
	// We use a try-catch approach since multiple connections to destination will throw an error
	const ctx = ensureAudio();
	try {
		node.connect(ctx.destination);
	} catch (e: any) {
		// This is expected if already connected - Web Audio API throws error for duplicate connections
		if ("InvalidAccessError" === e.name && e.message.includes("already connected")) {
			console.log("Node already connected to destination (expected)");
		} else {
			console.log("Unexpected connection error:", e.message);
		}
	}
}

function disconnectActive() {
	disconnectAnalysers();
	// Don't remove or disconnect the MediaElementAudioSourceNode
	// It should persist across component instances to avoid audio duplication
	activeSource = null;
}

/**
 * Attach analysers to the provided audio element.
 * 1) Try MediaElementAudioSource (created once per element)
 * 2) If NotSupportedError (already attached elsewhere), fallback to captureStream()
 */
function attachToAudioEl(el: HTMLAudioElement) {
	clearError();
	ensureAnalysers();

	// Reuse existing node if we previously created it for this element
	const existing = elementToSource.get(el);
	if (existing) {
		elementSource = existing;
		connectActive(existing);
		return;
	}

	// Try MediaElementAudioSource first
	try {
		elementSource = ensureAudio().createMediaElementSource(el);
		elementToSource.set(el, elementSource);
		connectActive(elementSource);
		return;
	} catch (e: any) {
		console.log("MediaElementAudioSource creation failed, trying captureStream");
		// Fallback to captureStream (or mozCaptureStream)
		try {
			const stream: MediaStream | null = (el as any).captureStream?.() || (el as any).mozCaptureStream?.();
			if (stream) {
				streamSource = ensureAudio().createMediaStreamSource(stream);
				connectActive(streamSource);
				return;
			}
			setError("Unable to access audio stream from the provided element. Start playback once, then retry.");
		} catch (e2: any) {
			setError("Audio attach failed: " + (e2?.message || e2));
		}
	}
}

function fitCanvas() {
	if (!canvas.value || !ctx) {
		return;
	}
	const dpr = perfConfig.devicePixelRatio;
	const w = window.innerWidth;
	const h = window.innerHeight;
	canvas.value.width = Math.floor(w * dpr);
	canvas.value.height = Math.floor(h * dpr);
	canvas.value.style.width = w + "px";
	canvas.value.style.height = h + "px";
	ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
}

function startRenderLoop() {
	if (!ctx || !analyserWave || !analyserFreq) {
		return;
	}
	if (rafId != null) {
		return;
	}

	let lastFrameTime = 0;
	const frameInterval = 1000 / perfConfig.targetFPS;

	const tick = (currentTime: number) => {
		if (!props.visible) {
			return;
		}

		// Frame rate limiting
		if (currentTime - lastFrameTime < frameInterval) {
			rafId = requestAnimationFrame(tick);
			return;
		}
		lastFrameTime = currentTime;

		rafId = requestAnimationFrame(tick);
		// Keep animating even when paused: analysers will output a flat line (128),
		// and our trail/sparkle/vignette still render.
		analyserWave!.getByteTimeDomainData(waveArray);
		analyserFreq!.getByteFrequencyData(freqArray);

		const w = canvas.value!.clientWidth;
		const h = canvas.value!.clientHeight;
		const mid = h * 0.5;

		let amp = 0;
		for (let i = 0; i < waveArray.length; i++) {
			const v = (waveArray[i] - 128) / 128;
			amp += Math.abs(v);
		}
		amp = amp / waveArray.length;
		let bass = 0;
		for (let i = 0; i < 40; i++) {
			bass += freqArray[i] || 0;
		}
		bass /= 40 * 255;

		// Persist hue across rapid src changes: DO NOT reset hue anywhere
		hue = (hue + 15 * bass + 0.2) % 360;
		glow = lerp(glow, (0.4 + 0.9 * amp + 0.6 * bass) * intensity, 0.08);

		// background trail
		ctx!.globalCompositeOperation = "source-over";
		ctx!.fillStyle = `rgba(10,12,28, ${0.12 + 0.08 * glow})`;
		ctx!.fillRect(0, 0, w, h);

		// compute smoothed path
		const density = Math.max(50, Math.min(perfConfig.pointDensity, Math.floor(w / 4)));
		const step = Math.max(1, Math.floor(waveArray.length / density));
		const points: Array<{ x: number; y: number }> = [];
		for (let x = 0, i = 0; i < waveArray.length; i += step) {
			const v = (waveArray[i] - 128) / 128;
			const y = mid + v * (h * 0.34);
			points.push({ x, y });
			x += (w * step) / waveArray.length;
		}

		// glow underlay
		ctx!.save();
		ctx!.shadowColor = `hsla(${hue}, 90%, 65%, ${0.6 * glow})`;
		ctx!.shadowBlur = perfConfig.shadowBlur * glow * lineScale;
		ctx!.strokeStyle = `hsla(${(hue + 30) % 360}, 90%, 65%, ${0.35 + 0.25 * glow})`;
		ctx!.lineWidth = (10 + 22 * glow) * lineScale;
		drawSmoothPath(points);
		ctx!.restore();

		// main bright line
		ctx!.save();
		ctx!.strokeStyle = `hsla(${hue}, 95%, ${60 + 20 * glow}%, 0.85)`;
		ctx!.lineWidth = (2.4 + 1.2 * Math.sin(time * 0.8 + amp * 6)) * lineScale;
		ctx!.shadowColor = `hsla(${hue}, 100%, 55%, 0.55)`;
		ctx!.shadowBlur = (perfConfig.shadowBlur * 0.4 + 18 * glow) * lineScale;
		drawSmoothPath(points);
		ctx!.restore();

		// peak particles (performance-based)
		if (performance !== "low") {
			ctx!.save();
			ctx!.globalCompositeOperation = "lighter";
			// for (let i = 3; i < points.length - 3; i += Math.max(perfConfig.particleSkip, Math.floor(22 - 18 * amp))) {
			// 	const p = points[i];
			// 	const prev = points[i - 3];
			// 	const next = points[i + 3];
			// 	const dy = Math.abs(next.y - prev.y);
			// 	if (dy > h * 0.06) {
			// 		const r = 1 + Math.min(14, dy * 0.06) * lineScale;
			// 		const grad = ctx!.createRadialGradient(p.x, p.y, 0, p.x, p.y, r);
			// 		grad.addColorStop(0, `hsla(${(hue + 10) % 360}, 100%, 65%, 0.85)`);
			// 		grad.addColorStop(1, `hsla(${(hue + 10) % 360}, 100%, 65%, 0)`);
			// 		ctx!.fillStyle = grad;
			// 		ctx!.beginPath();
			// 		ctx!.arc(p.x, p.y, r, 0, Math.PI * 2);
			// 		ctx!.fill();
			// 	}
			// }
			ctx!.restore();
		}

		if (useVignette) {
			drawVignette();
		}
		time += 0.016;
	};
	rafId = requestAnimationFrame(tick);
}

function stopRenderLoop() {
	if (rafId != null) {
		cancelAnimationFrame(rafId);
		rafId = null;
	}
}

function drawSmoothPath(pts: Array<{ x: number; y: number }>) {
	if (!ctx || pts.length < 2) {
		return;
	}
	ctx.beginPath();
	ctx.moveTo(pts[0].x, pts[0].y);
	for (let i = 1; i < pts.length - 2; i++) {
		const xc = (pts[i].x + pts[i + 1].x) / 2;
		const yc = (pts[i].y + pts[i + 1].y) / 2;
		ctx.quadraticCurveTo(pts[i].x, pts[i].y, xc, yc);
	}
	const pen = pts[pts.length - 1];
	const prev = pts[pts.length - 2];
	ctx.quadraticCurveTo(prev.x, prev.y, pen.x, pen.y);
	ctx.stroke();
}

function drawVignette() {
	if (!canvas.value || !ctx) {
		return;
	}
	const w = canvas.value.clientWidth;
	const h = canvas.value.clientHeight;
	const grad = ctx.createRadialGradient(w * 0.5, h * 0.6, Math.min(w, h) * 0.2, w * 0.5, h * 0.6, Math.max(w, h) * 0.7);
	grad.addColorStop(0, "rgba(0,0,0,0)");
	grad.addColorStop(1, "rgba(0,0,0,0.5)");
	ctx.fillStyle = grad;
	ctx.fillRect(0, 0, w, h);
}

function lerp(a: number, b: number, t: number) {
	return a + (b - a) * t;
}

// --- Lifecycle ---
onMounted(async () => {
	await nextTick();
	if (!canvas.value) {
		return;
	}
	ctx = canvas.value.getContext("2d");
	fitCanvas();
	const onResize = () => fitCanvas();
	window.addEventListener("resize", onResize, { passive: true });
	unsubResize = () => window.removeEventListener("resize", onResize);

	if (!props.visible) {
		stopRenderLoop();
		return;
	}

	// Attach to the provided audio element; do not reset hue so colors persist across src changes
	if (props.audioEl) {
		attachToAudioEl(props.audioEl);
	}

	attachToAudioEl(props.audioEl);
	stopRenderLoop();
	fitCanvas();
	startRenderLoop();
});

onBeforeUnmount(() => {
	console.log("WaveformBackground unmounting, stopping render loop");
	stopRenderLoop();

	disconnectAnalysers();
	analyserWave = null;
	analyserFreq = null;

	disconnectActive();

	// Remove source change listener
	if (sourceChangeListener) {
		sourceChangeListener();
		sourceChangeListener = null;
	}

	try {
		// Only disconnect stream sources as they're component-specific
		streamSource && (streamSource as any).disconnect?.();
		streamSource = null;
	} catch {}
	if (unsubResize) {
		unsubResize();
	}
});

// React if parent swaps the audio element instance at runtime
watch(
	() => props.audioEl,
	(el) => {
		if (el) {
			console.log("Audio element prop changed, reattaching visualizer");
			attachToAudioEl(el);
			stopRenderLoop();
			fitCanvas();
			startRenderLoop();
		}
	}
);

// React if visibility changes - just resize canvas when becoming visible
watch(
	() => props.visible,
	(visible) => {
		if (visible) {
			// fitCanvas();
			attachToAudioEl(props.audioEl);
			stopRenderLoop();
			fitCanvas();
			startRenderLoop();
		} else {
			stopRenderLoop();
		}
	}
);

// React if track changes - ensure audio connection exists
watch(
	() => props.currentTrack,
	() => {
		if (!activeSource && props.audioEl) {
			attachToAudioEl(props.audioEl);
			stopRenderLoop();
			fitCanvas();
			startRenderLoop();
		}
	}
);

// Listen for audio source changes to reattach visualizer
let sourceChangeListener: (() => void) | null = null;

// Expose an imperative resize method
function resize() {
	fitCanvas();
}

defineExpose({ resize });
</script>

<template>
	<div
		data-testid="visualizer-overlay"
		:class="['fullscreen-visualizer-overlay', { 'hidden-visualizer': !props.visible }]"
		@click="close"
	>
		<button data-testid="visualizer-close-btn" class="fullscreen-close-btn" @click="close">
			<font-awesome-icon icon="fa-times" />
		</button>

		<div class="waveform-bg fullscreen-visualizer" :class="{ 'is-fixed': fixed }">
			<canvas ref="canvas" data-testid="visualizer-canvas" class="viz" aria-hidden="true"></canvas>
		</div>
		<div class="visualizer-track-info">
			<h2 class="title is-3 has-text-white">{{ props.currentTrack?.title }}</h2>
			<p class="subtitle is-5 has-text-white-bis">{{ props.currentTrack?.artist }}</p>
			<p class="subtitle is-6 has-text-white-ter">{{ props.currentTrack?.album }}</p>
		</div>
	</div>
</template>

<style scoped>
.waveform-bg {
	position: relative;
	width: 100%;
	height: 100%;
}
.waveform-bg.is-fixed {
	position: fixed;
	inset: 0;
	z-index: -1; /* sit behind content */
	background:
		radial-gradient(1200px 800px at 20% 20%, #1b213e 0%, transparent 60%),
		radial-gradient(1000px 700px at 80% 80%, #2a174c 0%, transparent 60%), linear-gradient(180deg, #0b1020, #0d0221);
}
.viz {
	display: block;
	width: 100%;
	height: 100%;
}

/* Diagnostics UI */
.panel {
	position: absolute;
	left: 16px;
	bottom: 16px;
	background: rgba(255, 255, 255, 0.06);
	border: 1px solid rgba(255, 255, 255, 0.2);
	padding: 10px;
	border-radius: 12px;
	backdrop-filter: blur(8px);
	max-width: 520px;
}
.row {
	display: flex;
	gap: 8px;
	flex-wrap: wrap;
	margin-bottom: 8px;
}
.btn {
	appearance: none;
	border: 0;
	cursor: pointer;
	padding: 8px 12px;
	border-radius: 10px;
	color: #0b1020;
	font-weight: 600;
	background: linear-gradient(180deg, #c9d7ff, #90a9ff);
	box-shadow:
		0 6px 16px rgba(51, 89, 255, 0.35),
		inset 0 0 0 1px rgba(255, 255, 255, 0.4);
}
.log {
	white-space: pre-wrap;
	background: rgba(0, 0, 0, 0.35);
	border: 1px solid rgba(255, 255, 255, 0.18);
	padding: 8px;
	border-radius: 8px;
	max-height: 180px;
	overflow: auto;
	color: #e8ecff;
}
.err {
	margin-top: 6px;
	background: rgba(255, 92, 122, 0.14);
	border: 1px solid rgba(255, 92, 122, 0.45);
	color: #ffdce3;
	padding: 8px 10px;
	border-radius: 10px;
	font-size: 0.9rem;
}
/* Visualizer overlay styles */
.fullscreen-visualizer-overlay {
	position: fixed;
	top: 0;
	left: 0;
	width: 100vw;
	height: 100vh;
	background: rgba(0, 0, 0, 0.9);
	z-index: 1000;
	display: flex;
	align-items: center;
	justify-content: center;
}

.hidden-visualizer {
	display: none;
}

.fullscreen-visualizer {
	width: 100vw;
	height: 100vh;
	position: absolute;
	top: 0;
	left: 0;
	z-index: 1;
}

.visualizer-track-info {
	position: absolute;
	bottom: 2rem;
	left: 2rem;
	right: 2rem;
	z-index: 10;
	text-align: center;
	text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8);
	pointer-events: none;
}
</style>
