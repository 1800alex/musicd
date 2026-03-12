<template>
	<div class="modal" :class="{ 'is-active': visible }">
		<div class="modal-background" @click="$emit('close')"></div>
		<div class="modal-card">
			<header class="modal-card-head">
				<p class="modal-card-title">
					<font-awesome-icon icon="fa-music" class="mr-2" />
					Lyrics
				</p>
				<button class="delete" @click="$emit('close')"></button>
			</header>
			<section class="modal-card-body">
				<div v-if="currentTrack" class="track-info mb-4">
					<h5 class="title is-5">{{ currentTrack.title }}</h5>
					<h6 class="subtitle is-6">by {{ currentTrack.artist }}</h6>
					<p v-if="currentTrack.album" class="has-text-grey">from {{ currentTrack.album }}</p>
				</div>

				<!-- Loading State -->
				<div v-if="loading" class="has-text-centered p-4">
					<div class="is-loading mb-3"></div>
					<p>Searching for lyrics...</p>
				</div>

				<!-- Error State -->
				<div v-else-if="error" class="notification is-warning">
					<p><strong>Lyrics not found</strong></p>
					<p class="is-size-7">{{ error }}</p>
				</div>

				<!-- Lyrics Display -->
				<div v-else-if="lyrics" class="lyrics-content">
					<div class="lyrics-text">
						<pre>{{ lyrics }}</pre>
					</div>
				</div>

				<!-- No lyrics loaded yet -->
				<div v-else class="has-text-centered p-4">
					<p class="has-text-grey">Click "Search Lyrics" to find lyrics for this track</p>
				</div>
			</section>
			<footer class="modal-card-foot">
				<button class="button" @click="$emit('close')">Close</button>
				<button
					class="button is-primary"
					:class="{ 'is-loading': loading }"
					:disabled="!currentTrack || loading"
					@click="searchLyrics"
				>
					<font-awesome-icon icon="fa-search" class="mr-1" />
					Search Lyrics
				</button>
			</footer>
		</div>
	</div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { Track } from "~/types";
import httpService from "~/services/http.service";

interface Props {
	visible: boolean;
	currentTrack?: Track | null;
}

const props = defineProps<Props>();
const emit = defineEmits<{
	close: [];
}>();

const loading = ref(false);
const lyrics = ref<string>("");
const error = ref<string>("");

const searchLyrics = async () => {
	if (!props.currentTrack) {
		return;
	}

	loading.value = true;
	error.value = "";
	lyrics.value = "";

	try {
		const response = await httpService.get<{ lyrics: string }>("/api/lyrics", {
			params: {
				artist: props.currentTrack.artist,
				title: props.currentTrack.title
			}
		});

		if (response.data?.lyrics) {
			lyrics.value = response.data.lyrics;
		} else {
			error.value = "No lyrics found for this track";
		}
	} catch (err: any) {
		error.value = err?.data?.message || "Failed to search for lyrics";
		console.error("Error fetching lyrics:", err);
	} finally {
		loading.value = false;
	}
};

// Clear lyrics when track changes
watch(
	() => props.currentTrack,
	() => {
		lyrics.value = "";
		error.value = "";
	}
);

// Auto-search when modal opens if we have a track
watch(
	() => [props.visible, props.currentTrack],
	([newVisible, newTrack]) => {
		if (newVisible && newTrack && !lyrics.value && !loading.value) {
			searchLyrics();
		}
	}
);
</script>

<style scoped>
.lyrics-content {
	max-height: 60vh;
	overflow-y: auto;
}

.lyrics-text pre {
	white-space: pre-wrap;
	word-wrap: break-word;
	line-height: 1.6;
	font-family: inherit;
	background: none;
	border: none;
	padding: 0;
	margin: 0;
	color: inherit;
}

.track-info {
	border-bottom: 1px solid var(--clr-surface-higher);
	padding-bottom: 1rem;
}
</style>
