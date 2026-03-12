<script setup lang="ts">
import { onMounted } from "vue";
import useAppState from "~/stores/appState";
import awaitAppState from "~/composables/awaitAppState";

const router = useRouter();

// Page metadata
useHead({
	title: "Music Player"
});

const appState = useAppState();

// Lifecycle
onMounted(async () => {
	await awaitAppState();

	router.push(`/tracks`);
});
</script>

<template>
	<div class="container mt-5">
		<!-- Loading State -->
		<div class="has-text-centered p-4">
			<div class="is-loading"></div>
			<p>Loading...</p>
		</div>
	</div>
</template>
