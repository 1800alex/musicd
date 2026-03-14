<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick, inject } from "vue";
import type { Playlist, Track, PlaylistResponse } from "~/types";
import useAppState from "~/stores/appState";
import awaitAppState from "~/composables/awaitAppState";
import httpService from "~/services/http.service";
import backendService from "~/services/backend.service";
import PlayerService from "~/services/player.service";
import type { IColorPalette } from "~/components/ColorizedHero.vue";
import ColorizedHero from "~/components/ColorizedHero.vue";
import { useImageUrl } from "~/composables/useImageUrl";

// Inject search handlers from layout
const searchFocus = inject("searchFocus") as () => void;
const searchBlur = inject("searchBlur") as () => void;

const route = useRoute();
const router = useRouter();
const appState = useAppState();
const { getImageUrl } = useImageUrl();
const player = ref<PlayerService | null>(null);

// Get playlist ID from route
const playlistID = computed(() => route.params.id as string);

// Reactive variables
const playlist = ref<Playlist | null>(null);
const tracks = ref<Track[]>([]);
const loading = ref(false);
const tracksLoading = ref(false);
const currentPage = ref(1);
const totalPages = ref(1);
const searchQuery = ref("");
const totalTracks = ref(0);

// Color analysis variables
const palette = ref<IColorPalette>({
	background: "#000000",
	text: "#ffffff",
	whiteContrast: "4.5",
	blackContrast: "4.5",
	colors: ["#000000", "#ffffff"]
});

// Page metadata
useHead({
	title: computed(() => (playlist.value ? `${playlist.value.name} - Playlists - Music Player` : "Playlist - Music Player"))
});

// Methods
const fetchPlaylist = async (firstRun = false) => {
	if (!playlistID.value) {
		return;
	}

	loading.value = true;
	if (true === firstRun || !playlist.value) {
		try {
			const response = await backendService.FetchPlaylist(playlistID.value, {});

			playlist.value = response;
		} catch (error) {
			console.error("Error fetching playlist:", error);
			playlist.value = null;
		}
	}

	try {
		const response = await backendService.FetchPlaylistTracks(playlistID.value, {
			page: currentPage.value,
			pageSize: appState.PageSize,
			search: searchQuery.value
		});

		tracks.value = response.data;
		totalPages.value = response.totalPages;
		totalTracks.value = response.total;
	} catch (error) {
		console.error("Error fetching playlist tracks:", error);
		playlist.value = null;
	} finally {
		loading.value = false;
		tracksLoading.value = false;
	}
};

const playPlaylist = async (shuffle?: boolean) => {
	if (!playlist.value) {
		return;
	}

	if (true === shuffle) {
		player.value?.SetShuffle(true);
	}

	player.value?.PlayPlaylist(playlist.value).catch((error) => {
		console.error("Error playing playlist:", error);
	});
};

const addAllToQueue = () => {
	if (!playlist.value) {
		return;
	}

	player.value?.AddPlaylistToQueue(playlist.value).catch((error) => {
		console.error("Error adding playlist to queue:", error);
	});
};

const shufflePlay = () => {
	return playPlaylist(true);
};

const handleSearch = (query: string) => {
	searchQuery.value = query;
	currentPage.value = 1;
	tracksLoading.value = true;
	fetchPlaylist().catch((error) => {
		console.error("Error during search fetch:", error);
	});
};

const handlePageChange = (page: number) => {
	currentPage.value = page;
	tracksLoading.value = true;
	fetchPlaylist().catch((error) => {
		console.error("Error during page fetch:", error);
	});
};

const handlePageSizeChange = (size: number) => {
	player.value?.SetPageSize(size);
	currentPage.value = 1;
	tracksLoading.value = true;
	fetchPlaylist().catch((error) => {
		console.error("Error during page size fetch:", error);
	});
};

const handlePlayTrack = async (track: Track) => {
	if (!playlist.value) {
		return;
	}

	player.value?.PlayPlaylistTrack(track, playlist.value, searchQuery.value).catch((error) => {
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
			console.log("Added to queue:", track.title);
		});
};

const handleAddToPlaylist = async (track: Track, targetPlaylistName: string) => {
	try {
		await httpService.post(`/api/playlist/${encodeURIComponent(targetPlaylistName)}/add/${track.id}`, {});
		console.log(`Added "${track.title}" to playlist "${targetPlaylistName}"`);
	} catch (error) {
		console.error("Error adding track to playlist:", error);
	}
};

// Lifecycle
onMounted(async () => {
	await awaitAppState();

	player.value = new PlayerService(appState);

	await fetchPlaylist(true);
});

// Watch for route changes
watch(
	() => route.params.id,
	() => {
		if (route.params.id) {
			fetchPlaylist().catch((error) => {
				console.error("Error fetching playlist on route change:", error);
			});
		}
	}
);

// Watch for query changes
watch(
	() => route.query,
	(newQuery) => {
		if (newQuery.page) {
			currentPage.value = parseInt(newQuery.page as string) || 1;
		}
		fetchPlaylist().catch((error) => {
			console.error("Error fetching playlist on query change:", error);
		});
	},
	{ immediate: true }
);

// Update URL when state changes
watch(
	[currentPage],
	() => {
		router.replace({
			query: {
				page: currentPage.value > 1 ? currentPage.value.toString() : undefined
			}
		});
	},
	{ deep: true }
);
</script>

<template>
	<div class="music-page">
		<!-- Loading State -->
		<div v-if="loading" class="has-text-centered p-6">
			<div class="is-loading"></div>
			<p>Loading playlist...</p>
		</div>

		<!-- Playlist Content -->
		<div v-else-if="playlist">
			<!-- Playlist Header -->
			<section class="hero">
				<ColorizedHero
					:image-url="playlist.cover_art_id ? getImageUrl(`/api/cover-art/${playlist.cover_art_id}`) : null"
					@colors="
						(v) => {
							palette = v;
						}
					"
				>
					<div class="container">
						<div class="columns is-vcentered">
							<div class="column is-narrow">
								<figure class="image is-128x128">
									<img
										v-if="playlist.cover_art_id"
										:src="getImageUrl(`/api/cover-art/${playlist.cover_art_id}`)"
										:alt="`${playlist.name} cover`"
										class="is-rounded"
									/>
									<div
										v-else
										class="has-background-white-ter is-128x128 is-flex is-align-items-center is-justify-content-center is-rounded"
									>
										<font-awesome-icon icon="fa-music" class="has-text-grey fa-4x" />
									</div>
								</figure>
							</div>
							<div class="column">
								<p class="subtitle is-6" :style="{ color: palette.text, opacity: 0.8 }">Playlist</p>
								<h1 class="title is-2" :style="{ color: palette.text }">
									{{ playlist.name }}
								</h1>
								<p class="subtitle is-5" :style="{ color: palette.text, opacity: 0.8 }">
									{{ totalTracks }} track{{ totalTracks !== 1 ? "s" : "" }}
								</p>
								<div class="page-actions">
									<button
										class="page-action-btn page-action-btn-primary"
										:disabled="totalTracks === 0"
										@click="playPlaylist()"
									>
										<font-awesome-icon icon="fa-play" />
										Play Playlist
									</button>
									<button
										class="page-action-btn page-action-btn-secondary"
										:disabled="totalTracks === 0"
										@click="addAllToQueue"
									>
										<font-awesome-icon icon="fa-plus" />
										Add All to Queue
									</button>
									<button
										class="page-action-btn page-action-btn-secondary"
										:disabled="totalTracks === 0"
										@click="shufflePlay"
									>
										<font-awesome-icon icon="fa-random" />
										Shuffle
									</button>
								</div>
							</div>
						</div>
					</div>
				</ColorizedHero>
			</section>

			<div class="container mt-5">
				<!-- Tracks Section -->
				<TrackList
					:tracks="tracks"
					:loading="tracksLoading"
					:current-page="currentPage"
					:total-pages="totalPages"
					:search-query="searchQuery"
					:page-size="appState.PageSize"
					:show-search="true"
					:show-page-size="true"
					:show-cover="true"
					:show-year="true"
					:show-actions="true"
					:search-placeholder="`Search tracks in ${playlist.name}...`"
					@search="handleSearch"
					@page-change="handlePageChange"
					@page-size-change="handlePageSizeChange"
					@play-track="handlePlayTrack"
					@add-to-queue="handleAddToQueue"
					@add-to-playlist="handleAddToPlaylist"
					@search-focus="searchFocus"
					@search-blur="searchBlur"
				/>

				<!-- Empty Playlist State -->
				<div v-if="totalTracks === 0" class="has-text-centered p-6 mt-4">
					<p class="has-text-grey">
						<font-awesome-icon icon="fa-music" class="fa-3x mb-3"></font-awesome-icon>
					</p>
					<p class="title is-5 has-text-grey">This playlist is empty</p>
					<p class="has-text-grey mb-4">Add some tracks to get started</p>
					<NuxtLink to="/tracks" class="page-action-btn page-action-btn-primary">
						<font-awesome-icon icon="fa-plus" />
						Browse Tracks
					</NuxtLink>
				</div>
			</div>
		</div>

		<!-- Error State -->
		<div v-else class="has-text-centered p-6">
			<p class="has-text-grey">
				<font-awesome-icon icon="fa-exclamation-triangle" class="fa-3x mb-3"></font-awesome-icon>
			</p>
			<p class="title is-5 has-text-grey">Playlist not found</p>
			<NuxtLink to="/playlists" class="page-action-btn page-action-btn-primary">
				<font-awesome-icon icon="fa-arrow-left" />
				Back to Playlists
			</NuxtLink>
		</div>
	</div>
</template>
