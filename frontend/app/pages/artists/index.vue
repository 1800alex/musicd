<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import type { Artist } from "~/types";
import useAppState from "~/stores/appState";
import awaitAppState from "~/composables/awaitAppState";
import backendService from "~/services/backend.service";
import PlayerService from "~/services/player.service";
import { useImageUrl } from "~/composables/useImageUrl";

// Page metadata
useHead({
	title: "Artists - Music Player"
});

const appState = useAppState();
const router = useRouter();
const route = useRoute();
const { getImageUrl } = useImageUrl();
const player = ref<PlayerService | null>(null);

// Reactive variables
const artists = ref<Artist[]>([]);
const loading = ref(false);
const currentPage = ref(1);
const totalPages = ref(1);
const searchQuery = ref("");
const searchTimeout = ref<ReturnType<typeof setTimeout> | null>(null);
const pageSize = ref(25);

// Computed properties
const visiblePages = computed(() => {
	const pages: (number | string)[] = [];
	const total = totalPages.value;
	const current = currentPage.value;

	if (total <= 7) {
		for (let i = 1; i <= total; i++) {
			pages.push(i);
		}
	} else {
		pages.push(1);

		if (current > 4) {
			pages.push("...");
		}

		const start = Math.max(2, current - 1);
		const end = Math.min(total - 1, current + 1);

		for (let i = start; i <= end; i++) {
			pages.push(i);
		}

		if (current < total - 3) {
			pages.push("...");
		}

		if (total > 1) {
			pages.push(total);
		}
	}

	return pages;
});

// Methods
const fetchArtists = async () => {
	loading.value = true;
	try {
		const response = await backendService.FetchArtists({
			page: currentPage.value,
			pageSize: pageSize.value,
			search: searchQuery.value
		});

		artists.value = response.data;
		totalPages.value = response.totalPages;
	} catch (error) {
		console.error("Error fetching artists:", error);
	} finally {
		loading.value = false;
	}
};

const performSearch = () => {
	currentPage.value = 1;
	fetchArtists();
};

const clearSearch = () => {
	searchQuery.value = "";
	performSearch();
};

const onSearchInput = () => {
	if (searchTimeout.value) clearTimeout(searchTimeout.value);
	searchTimeout.value = setTimeout(() => {
		performSearch();
	}, 800);
};

const onPageSizeChange = () => {
	currentPage.value = 1;
	fetchArtists();
};

const goToPage = (page: number) => {
	if (page >= 1 && page <= totalPages.value && page !== currentPage.value) {
		currentPage.value = page;
		fetchArtists();
	}
};

const goToArtist = (artist: Artist) => {
	router.push(`/artist/${artist.id}`);
};

const playArtist = async (artist: Artist) => {
	player.value?.PlayArtist(artist.id).catch((error) => {
		console.error("Error playing artist:", error);
	});
};

const addArtistToQueue = async (artist: Artist) => {
	player.value?.AddArtistToQueue(artist.id).catch((error) => {
		console.error("Error adding artist to queue:", error);
	});
};

// Lifecycle
onMounted(async () => {
	await awaitAppState();

	player.value = new PlayerService(appState);

	// Get initial state from route query
	if (route.query.page) {
		currentPage.value = parseInt(route.query.page as string) || 1;
	}

	await fetchArtists();
});

// Watch for state changes and update URL
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
		<section class="hero hero-music-page">
			<div class="hero-body">
				<div class="container">
					<h1 class="title">
						<font-awesome-icon icon="fa-users" class="mr-2" />
						Artists
					</h1>
					<p class="subtitle">Browse music by artist</p>
				</div>
			</div>
		</section>

		<div class="container mt-5">
			<!-- Search + Page Size -->
			<div class="track-list-toolbar">
				<div class="nav-search">
					<span class="nav-search-icon">
						<font-awesome-icon icon="fa-search"></font-awesome-icon>
					</span>
					<input
						v-model="searchQuery"
						class="nav-search-input"
						type="text"
						placeholder="Search artists..."
						@input="onSearchInput"
						@keydown.enter="performSearch"
					/>
					<button v-if="searchQuery" class="nav-search-clear" @click="clearSearch">
						<font-awesome-icon icon="fa-times"></font-awesome-icon>
					</button>
				</div>
				<select v-model="pageSize" class="page-size-select" @change="onPageSizeChange">
					<option :value="25">25 per page</option>
					<option :value="50">50 per page</option>
					<option :value="100">100 per page</option>
				</select>
			</div>

			<!-- Loading State -->
			<div v-if="loading" class="has-text-centered p-4">
				<div class="is-loading"></div>
				<p>Loading artists...</p>
			</div>

			<!-- Artists Grid -->
			<div v-else-if="artists.length > 0" data-testid="artist-grid" class="columns is-multiline">
				<div
					v-for="artist in artists"
					:key="artist.name"
					class="column is-one-quarter-desktop is-one-third-tablet is-half-mobile"
				>
					<div
						data-testid="artist-card"
						:data-artist-id="artist.id"
						class="card music-card"
						@click="goToArtist(artist)"
					>
						<div class="card-content">
							<div class="media">
								<div class="media-left">
									<figure class="image is-48x48">
										<!-- Use first album cover as artist image, or default -->
										<img
											v-if="artist.albums.length > 0 && artist.albums[0]?.cover_art_id"
											:src="getImageUrl(`/api/cover-art/${artist.albums[0].cover_art_id}`)"
											:alt="`${artist.name} cover`"
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
									<p class="title is-6">{{ artist.name }}</p>
									<p class="subtitle is-7 has-text-grey">
										{{ artist.albums?.length || 0 }} album{{
											(artist.albums?.length || 0) !== 1 ? "s" : ""
										}}
										• {{ artist.track_count || artist.tracks?.length || 0 }} track{{
											(artist.track_count || artist.tracks?.length || 0) !== 1 ? "s" : ""
										}}
									</p>
								</div>
							</div>
						</div>
						<div class="card-action-footer">
							<button data-testid="artist-play-btn" class="card-action-btn" @click.stop="playArtist(artist)">
								<font-awesome-icon icon="fa-play" />
								Play All
							</button>
							<button
								data-testid="artist-queue-btn"
								class="card-action-btn"
								@click.stop="addArtistToQueue(artist)"
							>
								<font-awesome-icon icon="fa-plus" />
								Queue
							</button>
						</div>
					</div>
				</div>
			</div>

			<!-- Empty State -->
			<div v-else class="has-text-centered p-6">
				<p class="has-text-grey">
					<font-awesome-icon icon="fa-users" class="fa-3x mb-3"></font-awesome-icon>
				</p>
				<p class="title is-5 has-text-grey">No artists found</p>
				<p v-if="searchQuery" class="has-text-grey">Try adjusting your search criteria</p>
			</div>

			<!-- Pagination -->
			<nav v-if="totalPages > 1" class="pagination is-centered mt-6" role="navigation">
				<button
					data-testid="pagination-prev-btn"
					class="pagination-previous"
					:disabled="currentPage === 1"
					@click="goToPage(currentPage - 1)"
				>
					Previous
				</button>
				<button
					data-testid="pagination-next-btn"
					class="pagination-next"
					:disabled="currentPage === totalPages"
					@click="goToPage(currentPage + 1)"
				>
					Next
				</button>
				<ul class="pagination-list">
					<li v-for="page in visiblePages" :key="page">
						<button v-if="page === '...'" class="pagination-ellipsis" disabled>&hellip;</button>
						<button
							v-else
							class="pagination-link"
							:class="{ 'is-current': page === currentPage }"
							@click="goToPage(page)"
						>
							{{ page }}
						</button>
					</li>
				</ul>
			</nav>
		</div>
	</div>
</template>
