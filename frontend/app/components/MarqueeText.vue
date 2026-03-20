<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, watch } from "vue";

const container = ref<HTMLElement | null>(null);
const inner = ref<HTMLElement | null>(null);
const scale = ref(1);
const scrolling = ref(false);
const overflowPx = ref(0);
const animDuration = ref("6s");

const MIN_SCALE = 0.85;

const measure = () => {
	if (!container.value || !inner.value) {
		return;
	}

	// Reset state
	scale.value = 1;
	scrolling.value = false;

	// Container width is the available space
	const cw = container.value.clientWidth;
	if (cw <= 0) {
		return;
	}

	// Measure the natural text width using a Range — this gives the true
	// content width regardless of overflow/clipping in the ancestor chain.
	const range = document.createRange();
	range.selectNodeContents(inner.value);
	const sw = range.getBoundingClientRect().width;

	// Scale to minimum, then scroll the remaining overflow
	scale.value = MIN_SCALE;
	const scaledWidth = sw * MIN_SCALE;
	const overflow = scaledWidth - cw;
	overflowPx.value = overflow + 16;
	const dur = Math.max(4, overflow / 40); // ~40px/s
	animDuration.value = `${dur.toFixed(1)}s`;
	scrolling.value = true;
};

let resizeObs: ResizeObserver | null = null;
let mutationObs: MutationObserver | null = null;

onMounted(() => {
	// Delay initial measure to ensure layout is settled
	requestAnimationFrame(() => measure());

	if (container.value) {
		resizeObs = new ResizeObserver(() => measure());
		resizeObs.observe(container.value);
	}
	if (inner.value) {
		mutationObs = new MutationObserver(() => {
			nextTick(() => measure());
		});
		mutationObs.observe(inner.value, { childList: true, characterData: true, subtree: true });
	}
});

onBeforeUnmount(() => {
	resizeObs?.disconnect();
	mutationObs?.disconnect();
});
</script>

<template>
	<div ref="container" class="marquee-outer">
		<span
			ref="inner"
			class="marquee-inner"
			:class="{ 'marquee-scrolling': scrolling }"
			:style="{
				fontSize: scale < 1 ? `${scale}em` : undefined,
				'--marquee-overflow': `-${overflowPx}px`,
				'--marquee-duration': animDuration
			}"
		>
			<slot />
		</span>
	</div>
</template>

<style scoped>
.marquee-outer {
	overflow: hidden;
	white-space: nowrap;
}

.marquee-inner {
	display: inline-block;
	white-space: nowrap;
	transform-origin: left center;
}

.marquee-scrolling {
	animation: marquee-slide var(--marquee-duration, 6s) ease-in-out infinite alternate;
}

@keyframes marquee-slide {
	0%,
	15% {
		transform: translateX(0);
	}
	85%,
	100% {
		transform: translateX(var(--marquee-overflow, 0px));
	}
}
</style>
