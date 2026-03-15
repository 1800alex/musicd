<script setup lang="ts">
import { ref, onMounted, watch, inject } from "vue";
import type { Album, Track, Playlist } from "~/types";
import useAppState from "~/stores/appState";
import awaitAppState from "~/composables/awaitAppState";
import httpService from "~/services/http.service";
import backendService from "~/services/backend.service";
import PlayerService from "~/services/player.service";
import type { IColorPalette } from "~/components/ColorizedHero.vue";
import ColorizedHero from "~/components/ColorizedHero.vue";
import { useImageUrl } from "~/composables/useImageUrl";
import { useToast } from "~/composables/useToast";

// Inject search handlers from layout
const searchFocus = inject("searchFocus") as () => void;
const searchBlur = inject("searchBlur") as () => void;

const route = useRoute();
const router = useRouter();
const appState = useAppState();
const { getImageUrl } = useImageUrl();
const player = ref<PlayerService | null>(null);

// Get album ID from route
const albumID = computed(() => route.params.id as string);

// Reactive variables
const album = ref<Album | null>(null);
const artist = ref<string>("");
const tracks = ref<Track[]>([]);
const loading = ref(false);
const tracksLoading = ref(false);
const currentPage = ref(1);
const totalPages = ref(1);
const searchQuery = ref("");
const showDuplicateConfirm = ref(false);
const pendingTrackAdd = ref<{ track: Track; playlistName: string } | null>(null);

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
	title: computed(() =>
		album.value ? `${album.value.name} - ${album.value.artist} - Albums - Music Player` : "Album - Music Player"
	)
});

// Methods
const fetchAlbum = async (firstRun = false) => {
	if (!albumID.value) {
		return;
	}

	loading.value = true;

	if (true === firstRun) {
		try {
			const response = await backendService.FetchAlbum(albumID.value, {});

			album.value = response.album;
			artist.value = response.artist;
		} catch (error) {
			console.error("Error fetching album:", error);
			album.value = null;
		}
	}

	try {
		const response = await backendService.FetchAlbumTracks(albumID.value, {
			page: currentPage.value,
			pageSize: appState.PageSize,
			search: searchQuery.value
		});

		tracks.value = response.data;
		totalPages.value = response.totalPages;
	} catch (error) {
		console.error("Error fetching album:", error);
		album.value = null;
	} finally {
		loading.value = false;
		tracksLoading.value = false;
	}
};

const playAllTracks = () => {
	if (!albumID.value) {
		return;
	}

	player.value?.PlayAlbum(albumID.value).catch((error) => {
		console.error("Error playing album:", error);
	});
};

const addAllToQueue = () => {
	if (!albumID.value) {
		return;
	}

	player.value?.AddAlbumToQueue(albumID.value).catch((error) => {
		console.error("Error adding album to queue:", error);
	});
};

const handleSearch = (query: string) => {
	searchQuery.value = query;
	currentPage.value = 1;
	tracksLoading.value = true;
	fetchAlbum().catch((error) => {
		console.error("Error during search fetch:", error);
	});
};

const handlePageChange = (page: number) => {
	currentPage.value = page;
	tracksLoading.value = true;
	fetchAlbum().catch((error) => {
		console.error("Error during page fetch:", error);
	});
};

const handlePageSizeChange = (size: number) => {
	player.value?.SetPageSize(size);
	currentPage.value = 1;
	tracksLoading.value = true;
	fetchAlbum().catch((error) => {
		console.error("Error during page size fetch:", error);
	});
};

const handlePlayTrack = (track: Track) => {
	if (!albumID.value) {
		return;
	}

	player.value?.PlayAlbumTrack(track, albumID.value, searchQuery.value).catch((error) => {
		console.error("Error playing track:", error);
	});
};

const handleAddToQueue = (track: Track) => {
	player.value?.AddTrackToQueue(track).catch((error) => {
		console.error("Error adding track to queue:", error);
	});
	console.log("Added to queue:", track.title);
};

const handleAddToPlaylist = async (track: Track, playlistName: string) => {
	const { showSuccess, showError } = useToast();

	try {
		// Find the playlist ID from the name
		const targetPlaylist = appState.Playlists.find((p: Playlist) => p.name === playlistName);
		if (!targetPlaylist) {
			console.error(`Playlist "${playlistName}" not found`);
			showError(`Playlist "${playlistName}" not found`);
			return;
		}

		// Fetch current playlist tracks to check for duplicates
		const playlistTracks = await backendService.FetchPlaylistTracks(targetPlaylist.id, {
			pageSize: 1000
		});

		const isDuplicate = playlistTracks.data.some(
			(t) => t.id === track.id || (t.title === track.title && t.artist === track.artist)
		);

		if (isDuplicate) {
			// Show confirmation modal for duplicate
			pendingTrackAdd.value = { track, playlistName };
			showDuplicateConfirm.value = true;
			return;
		}

		await backendService.AddTrackToPlaylistById(track.id, targetPlaylist.id);
		showSuccess(`Added "${track.title}" to playlist "${playlistName}"`);
		console.log(`Added "${track.title}" to playlist "${playlistName}"`);
	} catch (error: any) {
		const errorMsg = error?.data?.message || error?.message || "Unknown error";
		console.error("Error adding track to playlist:", error);
		showError(`Failed to add "${track.title}" to playlist: ${errorMsg}`);
	}
};

const confirmDuplicateAdd = async () => {
	const { showSuccess, showError } = useToast();

	if (!pendingTrackAdd.value) {
		return;
	}

	const { track, playlistName } = pendingTrackAdd.value;

	try {
		const targetPlaylist = appState.Playlists.find((p: Playlist) => p.name === playlistName);
		if (!targetPlaylist) {
			showError(`Playlist "${playlistName}" not found`);
			return;
		}

		await backendService.AddTrackToPlaylistById(track.id, targetPlaylist.id);
		showSuccess(`Added "${track.title}" to playlist "${playlistName}"`);
		console.log(`Added "${track.title}" to playlist "${playlistName}"`);
	} catch (error: any) {
		const errorMsg = error?.data?.message || error?.message || "Unknown error";
		console.error("Error adding track to playlist:", error);
		showError(`Failed to add "${track.title}" to playlist: ${errorMsg}`);
	} finally {
		showDuplicateConfirm.value = false;
		pendingTrackAdd.value = null;
	}
};

const cancelDuplicateAdd = () => {
	showDuplicateConfirm.value = false;
	pendingTrackAdd.value = null;
};

const duplicateConfirmMessage = computed(() => {
	if (!pendingTrackAdd.value) {
		return "";
	}
	const { track, playlistName } = pendingTrackAdd.value;
	return `"${track.title}" by ${track.artist} is already in "${playlistName}". Add it again?`;
});

const navigateToArtist = async (artistName: string) => {
	try {
		// Search for artist by name to get the artist ID
		const artistsResponse = await backendService.FetchArtists();
		const artist = artistsResponse.data.find((a) => a.name === artistName);
		if (artist) {
			await router.push(`/artist/${artist.id}`);
		} else {
			console.warn(`Artist "${artistName}" not found`);
		}
	} catch (error) {
		console.error("Error navigating to artist:", error);
	}
};

const goToArtist = () => {
	if (!album.value) {
		return;
	}

	if (album.value!.artist) {
		navigateToArtist(album.value.artist).catch((error) => {
			console.error("Error navigating to artist:", error);
		});
	}
};

// Lifecycle
onMounted(async () => {
	await awaitAppState();

	player.value = new PlayerService(appState);

	await fetchAlbum(true);
});

// Watch for route changes
watch(
	() => route.params.id,
	() => {
		if (route.params.id) {
			fetchAlbum().catch((error) => {
				console.error("Error fetching album on route change:", error);
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
		fetchAlbum().catch((error) => {
			console.error("Error fetching album on query change:", error);
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
			<p>Loading album...</p>
		</div>

		<!-- Album Content -->
		<div v-else-if="album">
			<!-- Album Header -->
			<section class="hero">
				<ColorizedHero
					:image-url="album.cover_art_id ? getImageUrl(`/api/cover-art/${album.cover_art_id}`) : null"
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
										v-if="album.cover_art_id"
										:src="getImageUrl(`/api/cover-art/${album.cover_art_id}`)"
										:alt="`${album.name} cover`"
										class="is-rounded"
									/>
									<div
										v-else
										class="has-background-white-ter is-128x128 is-flex is-align-items-center is-justify-content-center is-rounded"
									>
										<font-awesome-icon icon="fa-compact-disc" class="has-text-grey fa-4x" />
									</div>
								</figure>
							</div>
							<div class="column">
								<h1 class="title is-2" :style="{ color: palette.text }">{{ album.name }}</h1>
								<p class="subtitle is-4" :style="{ color: palette.text, opacity: 0.8 }">
									<a class="artist-link" :style="{ color: palette.text }" @click="goToArtist">{{
										album.artist
									}}</a>
								</p>
								<p class="subtitle is-5 artist-info" :style="{ color: palette.text, opacity: 0.8 }">
									{{ album.year || "Unknown Year" }} • {{ album.track_count || album.tracks?.length || 0 }} track{{
										(album.track_count || album.tracks?.length || 0) !== 1 ? "s" : ""
									}}
								</p>
								<div class="page-actions">
									<button class="page-action-btn page-action-btn-primary" @click="playAllTracks">
										<font-awesome-icon icon="fa-play" />
										Play All
									</button>
									<button class="page-action-btn page-action-btn-secondary" @click="addAllToQueue">
										<font-awesome-icon icon="fa-plus" />
										Add All to Queue
									</button>
								</div>
							</div>
						</div>
					</div>
				</ColorizedHero>
			</section>

			<div class="container mt-5">
				<!-- Tracks Section -->
				<div>
					<h2 class="title is-4">
						<font-awesome-icon icon="fa-list" class="mr-2" />
						Tracks
					</h2>
					<TrackList
						:tracks="tracks"
						:loading="tracksLoading"
						:current-page="currentPage"
						:total-pages="totalPages"
						:search-query="searchQuery"
						:page-size="appState.PageSize"
						:show-search="true"
						:show-page-size="false"
						:show-cover="false"
						:show-year="false"
						:show-actions="true"
						:search-placeholder="`Search tracks in ${album.name}...`"
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
		</div>

		<!-- Error State -->
		<div v-else class="has-text-centered p-6">
			<p class="has-text-grey">
				<font-awesome-icon icon="fa-exclamation-triangle" class="fa-3x mb-3"></font-awesome-icon>
			</p>
			<p class="title is-5 has-text-grey">Album not found</p>
			<NuxtLink to="/artists" class="page-action-btn page-action-btn-primary">
				<font-awesome-icon icon="fa-arrow-left" />
				Back to Artists
			</NuxtLink>
		</div>

		<!-- Duplicate Track Confirmation Modal -->
		<ConfirmationModal
			:is-open="showDuplicateConfirm"
			title="Duplicate Track"
			:message="duplicateConfirmMessage"
			confirm-text="Add Again"
			cancel-text="Cancel"
			:is-danger="false"
			@confirm="confirmDuplicateAdd"
			@cancel="cancelDuplicateAdd"
		/>
	</div>
</template>
