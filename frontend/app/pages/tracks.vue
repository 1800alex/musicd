<script setup lang="ts">
import { ref, onMounted, watch, inject } from "vue";
import type { Track, Playlist } from "~/types";
import useAppState, { RepeatMode } from "~/stores/appState";
import awaitAppState from "~/composables/awaitAppState";
import backendService from "~/services/backend.service";
import PlayerService from "~/services/player.service";

// Page metadata
useHead({
	title: "All Tracks - Music Player"
});

// Inject search handlers from layout
const searchFocus = inject("searchFocus") as () => void;
const searchBlur = inject("searchBlur") as () => void;

const appState = useAppState();
const player = ref<PlayerService | null>(null);

// Reactive variables
const tracks = ref<Track[]>([]);
const loading = ref(false);
const currentPage = ref(1);
const totalPages = ref(1);
const searchQuery = ref("");

// Methods
const fetchTracks = async () => {
	loading.value = true;
	try {
		const response = await backendService.FetchTracks({
			page: currentPage.value,
			pageSize: appState.PageSize,
			search: searchQuery.value
		});

		tracks.value = response.data;
		totalPages.value = response.totalPages;
	} catch (error) {
		console.error("Error fetching tracks:", error);
		// TODO: Show error notification
	} finally {
		loading.value = false;
	}
};

const handleSearch = (query: string) => {
	searchQuery.value = query;
	currentPage.value = 1; // Reset to first page when searching
	fetchTracks().catch((error) => {
		console.error("Error during search fetch:", error);
	});
};

const handlePageChange = (page: number) => {
	currentPage.value = page;
	fetchTracks().catch((error) => {
		console.error("Error during page fetch:", error);
	});
};

const handlePageSizeChange = (size: number) => {
	appState.SetPageSize(size);
	currentPage.value = 1; // Reset to first page when changing page size
	fetchTracks().catch((error) => {
		console.error("Error during page size fetch:", error);
	});
};

const handlePlayTrack = async (track: Track) => {
	player.value?.PlayTrackFromAllTracks(track, searchQuery.value).catch((error) => {
		console.error("Error playing track:", error);
	});
};

const handleAddToQueue = (track: Track) => {
	player.value
		?.AddTrackToQueue(track)
		.catch((error) => {
			console.error("Error adding track to queue:", error);
		})
		.finally(() => {
			// TODO: Show success notification
			console.log("Added to queue:", track.title);
		});
};

const handleAddToPlaylist = async (track: Track, playlistName: string) => {
	try {
		await backendService.AddTrackToPlaylist(track, playlistName);
		// TODO: Show success notification
		console.log(`Added "${track.title}" to playlist "${playlistName}"`);
	} catch (error) {
		console.error("Error adding track to playlist:", error);
		// TODO: Show error notification
	}
};

// Lifecycle
onMounted(async () => {
	await awaitAppState();

	player.value = new PlayerService(appState);

	await fetchTracks();
});

// Watch for route query changes (for deep linking)
const route = useRoute();
watch(
	() => route.query,
	(newQuery) => {
		if (newQuery.page) {
			currentPage.value = parseInt(newQuery.page as string) || 1;
		}
		if (newQuery.pageSize) {
			player.value?.SetPageSize(parseInt(newQuery.pageSize as string) || 25);
		}
		if (newQuery.search) {
			searchQuery.value = newQuery.search as string;
		}
		fetchTracks();
	},
	{ immediate: true }
);

// Update URL when state changes
watch(
	[currentPage],
	() => {
		const router = useRouter();
		router.replace({
			query: {
				page: currentPage.value > 1 ? currentPage.value.toString() : undefined
			}
		});
	},
	{ deep: true }
);

// Watch for search query changes and update URL
watch(
	[searchQuery],
	() => {
		const router = useRouter();
		router.replace({
			query: {
				search: searchQuery.value || undefined
			}
		});
	},
	{ deep: true }
);
</script>

<template>
	<div class="music-page">
		<section class="hero hero-music-page">
			<div class="hero-body">
				<div class="container">
					<h1 class="title">
						<font-awesome-icon icon="music" class="mr-2" />
						All Tracks
					</h1>
					<p class="subtitle">Browse and play your music library</p>
				</div>
			</div>
		</section>

		<div class="container mt-5">
			<TrackList
				:tracks="tracks"
				:loading="loading"
				:current-page="currentPage"
				:total-pages="totalPages"
				:search-query="searchQuery"
				:page-size="appState.PageSize"
				:show-search="true"
				:show-page-size="true"
				:show-cover="true"
				:show-year="true"
				:show-actions="true"
				search-placeholder="Search tracks by title, artist, or album..."
				@search="handleSearch"
				@page-change="handlePageChange"
				@page-size-change="handlePageSizeChange"
				@play-track="handlePlayTrack"
				@add-to-queue="handleAddToQueue"
				@add-to-playlist="handleAddToPlaylist"
				@search-focus="searchFocus"
				@search-blur="searchBlur"
			/>
		</div>
	</div>
</template>
