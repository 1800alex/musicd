<script setup lang="ts">
import { ref, computed, onMounted, watch, inject } from "vue";
import type { Artist, Track, Album, Playlist } from "~/types";
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

// Get artist ID from route
const artistID = computed(() => route.params.id as string);

// Reactive variables
const artist = ref<Artist | null>(null);
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
	title: computed(() => (artist.value ? `${artist.value.name} - Artists - Music Player` : "Artist - Music Player"))
});

// Methods
const fetchArtist = async (firstRun = false) => {
	if (!artistID.value) {
		return;
	}

	loading.value = true;

	if (true === firstRun) {
		try {
			const response = await backendService.FetchArtist(artistID.value, {});

			artist.value = response.artist;
		} catch (error) {
			console.error("Error fetching artist:", error);
			artist.value = null;
		}
	}

	try {
		const response = await backendService.FetchArtistTracks(artistID.value, {
			page: currentPage.value,
			pageSize: appState.PageSize,
			search: searchQuery.value
		});

		tracks.value = response.data;
		totalPages.value = response.totalPages;
	} catch (error) {
		console.error("Error fetching artist:", error);
		artist.value = null;
	} finally {
		loading.value = false;
		tracksLoading.value = false;
	}
};

const playAllTracks = () => {
	if (!artistID.value) {
		return;
	}

	player.value?.PlayArtist(artistID.value).catch((error) => {
		console.error("Error playing artist:", error);
	});
};

const addAllToQueue = () => {
	if (!artistID.value) {
		return;
	}

	player.value?.AddArtistToQueue(artistID.value).catch((error) => {
		console.error("Error adding artist to queue:", error);
	});
};

const playAlbum = (album: Album) => {
	player.value?.PlayAlbum(album.id).catch((error) => {
		console.error("Error playing album:", error);
	});
};

const addAlbumToQueue = (album: Album) => {
	player.value?.AddAlbumToQueue(album.id).catch((error) => {
		console.error("Error adding album to queue:", error);
	});
};

const goToAlbum = (album: Album) => {
	router.push(`/album/${album.id}`);
};

const handleSearch = (query: string) => {
	searchQuery.value = query;
	currentPage.value = 1;
	tracksLoading.value = true;
	fetchArtist().catch((error) => {
		console.error("Error during search fetch:", error);
	});
};

const handlePageChange = (page: number) => {
	currentPage.value = page;
	tracksLoading.value = true;
	fetchArtist().catch((error) => {
		console.error("Error during page fetch:", error);
	});
};

const handlePageSizeChange = (size: number) => {
	player.value?.SetPageSize(size);
	currentPage.value = 1;
	tracksLoading.value = true;
	fetchArtist().catch((error) => {
		console.error("Error during page size fetch:", error);
	});
};

const handlePlayTrack = (track: Track) => {
	if (!artistID.value) {
		return;
	}

	player.value?.PlayArtistTrack(track, artistID.value, searchQuery.value).catch((error) => {
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
	} catch (error) {
		console.error("Error adding track to playlist:", error);
		showError(`Failed to add "${track.title}" to playlist`);
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
	} catch (error) {
		console.error("Error adding track to playlist:", error);
		showError(`Failed to add "${track.title}" to playlist`);
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

// Lifecycle
onMounted(async () => {
	await awaitAppState();

	player.value = new PlayerService(appState);

	await fetchArtist(true);
});

// Watch for route changes
watch(
	() => route.params.id,
	() => {
		if (route.params.id) {
			fetchArtist().catch((error) => {
				console.error("Error fetching artist on route change:", error);
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
		fetchArtist().catch((error) => {
			console.error("Error fetching artist on query change:", error);
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
			<p>Loading artist...</p>
		</div>

		<!-- Artist Content -->
		<div v-else-if="artist">
			<!-- Artist Header -->
			<section class="hero">
				<ColorizedHero
					:image-url="
						artist.albums[0]?.cover_art_id
							? getImageUrl(`/api/cover-art/${artist.albums[0].cover_art_id}`)
							: null
					"
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
										v-if="artist.albums.length > 0 && artist.albums[0]?.cover_art_id"
										:src="getImageUrl(`/api/cover-art/${artist.albums[0].cover_art_id}`)"
										:alt="`${artist.name} cover`"
										class="is-rounded"
									/>
									<div
										v-else
										class="has-background-white-ter is-128x128 is-flex is-align-items-center is-justify-content-center is-rounded"
									>
										<font-awesome-icon icon="fa-user" class="has-text-grey fa-4x" />
									</div>
								</figure>
							</div>
							<div class="column">
								<h1 class="title is-2" :style="{ color: palette.text }">{{ artist.name }}</h1>
								<p class="subtitle is-5" :style="{ color: palette.text, opacity: 0.8 }">
									{{ artist.albums.length }} album{{ artist.albums.length !== 1 ? "s" : "" }} •
									{{ artist.tracks.length }} track{{ artist.tracks.length !== 1 ? "s" : "" }}
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
				<!-- Albums Section -->
				<div v-if="artist.albums.length > 0" class="mb-6">
					<h2 class="title is-4">
						<font-awesome-icon icon="fa-compact-disc" class="mr-2" />
						Albums
					</h2>
					<div class="columns is-multiline">
						<div
							v-for="album in artist.albums"
							:key="album.id"
							class="column is-one-quarter-desktop is-one-third-tablet is-half-mobile"
						>
							<div class="card music-card" @click="goToAlbum(album)">
								<div class="card-content">
									<div class="media">
										<div class="media-left">
											<figure class="image is-48x48">
												<img
													v-if="album.cover_art_id"
													:src="getImageUrl(`/api/cover-art/${album.cover_art_id}`)"
													:alt="`${album.name} cover`"
													class="is-rounded"
												/>
												<div
													v-else
													class="has-background-grey-lighter is-48x48 is-flex is-align-items-center is-justify-content-center is-rounded"
												>
													<font-awesome-icon icon="fa-user" class="has-text-grey fa-lg" />
												</div>
											</figure>
										</div>
										<div class="media-content">
											<p class="title is-6">{{ album.name }}</p>
											<p class="subtitle is-7 has-text-grey">
												{{ album.year || "Unknown Year" }} • {{ album.tracks.length }} track{{
													album.tracks.length !== 1 ? "s" : ""
												}}
											</p>
										</div>
									</div>
								</div>
								<div class="card-action-footer">
									<button class="card-action-btn" @click.stop="playAlbum(album)">
										<font-awesome-icon icon="fa-play" />
										Play All
									</button>
									<button class="card-action-btn" @click.stop="addAlbumToQueue(album)">
										<font-awesome-icon icon="fa-plus" />
										Queue
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- All Tracks Section -->
				<div>
					<h2 class="title is-4">
						<font-awesome-icon icon="fa-list" class="mr-2" />
						All Tracks
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
						:show-cover="true"
						:show-year="true"
						:show-actions="true"
						:search-placeholder="`Search tracks by ${artist.name}...`"
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
			<p class="title is-5 has-text-grey">Artist not found</p>
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
