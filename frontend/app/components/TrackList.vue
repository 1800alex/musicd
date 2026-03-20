<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import type { Track } from "~/types";
import useAppState from "~/stores/appState";
import awaitAppState from "~/composables/awaitAppState";
import backendService from "~/services/backend.service";
import { useImageUrl } from "~/composables/useImageUrl";

const props = defineProps<{
	tracks: Track[];
	loading?: boolean;
	currentPage?: number;
	totalPages?: number;
	searchQuery?: string;
	pageSize?: number;
	showSearch?: boolean;
	showPageSize?: boolean;
	showCover?: boolean;
	showYear?: boolean;
	showActions?: boolean;
	searchPlaceholder?: string;
	playlistId?: string;
}>();

const emit = defineEmits<{
	search: [query: string];
	pageChange: [page: number];
	pageSizeChange: [size: number];
	playTrack: [track: Track];
	addToQueue: [track: Track];
	addToPlaylist: [track: Track, playlist: string];
	removeFromPlaylist: [track: Track];
	searchFocus: [];
	searchBlur: [];
}>();

const appState = useAppState();
const { getImageUrl } = useImageUrl();

// Local reactive variables
const searchQuery = ref(props.searchQuery || "");
const pageSize = ref(props.pageSize || 25);
const searchInputRef = ref<HTMLInputElement | null>(null);

// Computed properties
const currentPage = computed(() => props.currentPage || 1);
const totalPages = computed(() => props.totalPages || 1);
const showSearch = computed(() => props.showSearch !== false);
const showPageSize = computed(() => props.showPageSize !== false);
const showCover = computed(() => props.showCover !== false);
const showYear = computed(() => props.showYear !== false);
const showActions = computed(() => props.showActions !== false);
const searchPlaceholder = computed(() => props.searchPlaceholder || "Search tracks...");

// Calculate visible page numbers for pagination
const visiblePages = computed(() => {
	const pages: (number | string)[] = [];
	const total = totalPages.value;
	const current = currentPage.value;

	if (total <= 7) {
		// Show all pages if total is 7 or less
		for (let i = 1; i <= total; i++) {
			pages.push(i);
		}
	} else {
		// Always show first page
		pages.push(1);

		if (current > 4) {
			pages.push("...");
		}

		// Show pages around current page
		const start = Math.max(2, current - 1);
		const end = Math.min(total - 1, current + 1);

		for (let i = start; i <= end; i++) {
			pages.push(i);
		}

		if (current < total - 3) {
			pages.push("...");
		}

		// Always show last page
		if (total > 1) {
			pages.push(total);
		}
	}

	return pages;
});

// Methods
const performSearch = () => {
	emit("search", searchQuery.value);
};

const clearSearch = () => {
	searchQuery.value = "";
	emit("search", "");
};

const onSearchInput = () => {
	// Debounce search
	if (searchTimeout.value) {
		clearTimeout(searchTimeout.value);
	}
	searchTimeout.value = setTimeout(() => {
		performSearch();
	}, 800);
};

const searchTimeout = ref<ReturnType<typeof setTimeout> | null>(null);

// Track actions menu
const openTrackMenu = ref<string | null>(null);
const menuStyle = ref<Record<string, string>>({});

const toggleTrackMenu = (trackId: string, event?: MouseEvent) => {
	if (openTrackMenu.value === trackId) {
		openTrackMenu.value = null;
		return;
	}
	openTrackMenu.value = trackId;
	if (event) {
		const btn = event.currentTarget as HTMLElement;
		const rect = btn.getBoundingClientRect();
		const spaceBelow = window.innerHeight - rect.bottom;
		// If less than 320px below the button, open upward
		if (spaceBelow < 320) {
			menuStyle.value = {
				position: "fixed",
				bottom: `${window.innerHeight - rect.top + 4}px`,
				right: `${window.innerWidth - rect.right}px`,
				top: "auto",
				maxHeight: `${rect.top - 8}px`
			};
		} else {
			menuStyle.value = {
				position: "fixed",
				top: `${rect.bottom + 4}px`,
				right: `${window.innerWidth - rect.right}px`,
				bottom: "auto",
				maxHeight: `${spaceBelow - 8}px`
			};
		}
	}
};

const closeTrackMenu = () => {
	openTrackMenu.value = null;
};

const handleDocumentClick = () => {
	closeTrackMenu();
};

onMounted(async () => {
	await awaitAppState();
	document.addEventListener("click", handleDocumentClick);
});

onUnmounted(() => {
	document.removeEventListener("click", handleDocumentClick);
});

const onPageSizeChange = () => {
	emit("pageSizeChange", pageSize.value);
};

const goToPage = (page: number) => {
	if (page >= 1 && page <= totalPages.value && page !== currentPage.value) {
		emit("pageChange", page);
	}
};

const playTrack = (track: Track) => {
	emit("playTrack", track);
};

const addToQueue = (track: Track) => {
	emit("addToQueue", track);
};

const addToPlaylist = (track: Track, playlistName: string) => {
	emit("addToPlaylist", track, playlistName);
};

const removeFromPlaylist = (track: Track) => {
	emit("removeFromPlaylist", track);
};

const router = useRouter();

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

const navigateToAlbum = async (albumName: string, artistName: string) => {
	try {
		// Search for album by name and artist
		const artistsResponse = await backendService.FetchArtists();
		const artist = artistsResponse.data.find((a) => a.name === artistName);
		if (artist) {
			const album = artist.albums.find((a) => a.name === albumName);
			if (album) {
				await router.push(`/album/${album.id}`);
			} else {
				console.warn(`Album "${albumName}" by "${artistName}" not found`);
			}
		} else {
			console.warn(`Artist "${artistName}" not found`);
		}
	} catch (error) {
		console.error("Error navigating to album:", error);
	}
};

// Search focus/blur handlers
const handleSearchFocus = () => {
	emit("searchFocus");
};

const handleSearchBlur = () => {
	emit("searchBlur");
};

// Watch for prop changes
watch(
	() => props.searchQuery,
	(newValue) => {
		if (newValue !== undefined) {
			searchQuery.value = newValue;
		}
	}
);

watch(
	() => props.pageSize,
	(newValue) => {
		if (newValue !== undefined) {
			pageSize.value = newValue;
		}
	}
);
</script>

<template>
	<div class="track-list" data-testid="track-list">
		<!-- Search and Controls Toolbar -->
		<div v-if="showSearch || showPageSize" class="track-list-toolbar">
			<div v-if="showSearch" class="nav-search">
				<span class="nav-search-icon">
					<font-awesome-icon icon="fa-search"></font-awesome-icon>
				</span>
				<input
					ref="searchInputRef"
					v-model="searchQuery"
					class="nav-search-input"
					type="text"
					:placeholder="searchPlaceholder"
					@input="onSearchInput"
					@keydown.enter="performSearch"
					@focus="handleSearchFocus"
					@blur="handleSearchBlur"
				/>
				<button v-if="searchQuery" class="nav-search-clear" data-testid="track-search-clear" @click="clearSearch">
					<font-awesome-icon icon="fa-times"></font-awesome-icon>
				</button>
			</div>
			<select
				v-if="showPageSize"
				v-model="pageSize"
				data-testid="track-page-size-select"
				class="page-size-select"
				@change="onPageSizeChange"
			>
				<option :value="25">25 per page</option>
				<option :value="50">50 per page</option>
				<option :value="100">100 per page</option>
			</select>
		</div>

		<!-- Loading State -->
		<div v-if="loading" class="has-text-centered p-4">
			<div class="is-loading"></div>
			<p>Loading tracks...</p>
		</div>

		<!-- Track Table -->
		<div v-else-if="tracks.length > 0" class="table-container">
			<table class="table is-fullwidth is-hoverable">
				<thead>
					<tr>
						<th v-if="showCover" class="cover-column">Cover</th>
						<th>Title</th>
						<th>Artist</th>
						<th>Album</th>
						<th v-if="showYear">Year</th>
						<th v-if="showActions">Actions</th>
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="track in tracks"
						:key="track.id"
						data-testid="track-row"
						:data-track-id="track.id"
						:class="appState.CurrentTrack?.id === track.id ? 'track-row currently-playing' : 'track-row'"
						@dblclick="playTrack(track)"
					>
						<td v-if="showCover" class="is-narrow cover-column">
							<figure class="track-cover">
								<img
									v-if="track.cover_art_id"
									:src="getImageUrl(`/api/cover-art/${track.cover_art_id}`)"
									:alt="`${track.album} cover`"
									loading="lazy"
								/>
								<div
									v-else
									class="has-background-grey-lighter is-32x32 is-flex is-align-items-center is-justify-content-center"
								>
									<font-awesome-icon icon="fa-music" class="has-text-grey"></font-awesome-icon>
								</div>
							</figure>
						</td>
						<td class="track-row-text">
							<span class="clickable-link" @click="playTrack(track)">{{ track.title }}</span>
						</td>
						<td class="track-row-text">
							<span class="clickable-link" @click="navigateToArtist(track.artist)">{{ track.artist }}</span>
						</td>
						<td class="track-row-text">
							<span class="clickable-link" @click="navigateToAlbum(track.album, track.artist)">{{
								track.album
							}}</span>
						</td>
						<td v-if="showYear" class="track-row-text">{{ track.year || "—" }}</td>
						<td v-if="showActions" class="track-actions-cell">
							<div class="track-actions-menu-wrapper">
								<button class="track-actions-btn" @click.stop="toggleTrackMenu(track.id, $event)">
									<font-awesome-icon icon="fa-ellipsis-vertical" />
								</button>
								<div
									v-if="openTrackMenu === track.id"
									class="track-actions-menu"
									:style="menuStyle"
									@click.stop
								>
									<button
										class="track-actions-menu-item"
										data-testid="track-play-btn"
										@click="
											playTrack(track);
											closeTrackMenu();
										"
									>
										<font-awesome-icon icon="fa-play" class="mr-2" />
										Play
									</button>
									<button
										class="track-actions-menu-item"
										data-testid="track-queue-btn"
										@click="
											addToQueue(track);
											closeTrackMenu();
										"
									>
										<font-awesome-icon icon="fa-plus" class="mr-2" />
										Add to queue
									</button>
									<button
										class="track-actions-menu-item"
										@click="
											navigateToArtist(track.artist);
											closeTrackMenu();
										"
									>
										<font-awesome-icon icon="fa-user" class="mr-2" />
										Go to artist
									</button>
									<button
										class="track-actions-menu-item"
										@click="
											navigateToAlbum(track.album, track.artist);
											closeTrackMenu();
										"
									>
										<font-awesome-icon icon="fa-folder" class="mr-2" />
										Go to album
									</button>
									<template v-if="appState.Playlists.length > 0">
										<div class="track-actions-divider"></div>
										<div class="track-actions-submenu-label">
											<font-awesome-icon icon="fa-list" class="mr-2" />
											Add to playlist
										</div>
										<PlaylistPicker
											:playlists="appState.Playlists"
											@select="
												(p) => {
													addToPlaylist(track, p.name);
													closeTrackMenu();
												}
											"
										/>
									</template>
									<template v-if="playlistId">
										<div class="track-actions-divider"></div>
										<button
											class="track-actions-menu-item track-actions-menu-item-danger"
											@click="
												removeFromPlaylist(track);
												closeTrackMenu();
											"
										>
											<font-awesome-icon icon="fa-trash" class="mr-2" />
											Remove from Playlist
										</button>
									</template>
								</div>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
		</div>

		<!-- Empty State -->
		<div v-else class="has-text-centered p-6">
			<p class="has-text-grey">
				<font-awesome-icon icon="fa-music" class="fa-3x mb-3"></font-awesome-icon>
			</p>
			<p class="title is-5 has-text-grey">No tracks found</p>
			<p v-if="searchQuery" class="has-text-grey">Try adjusting your search criteria</p>
		</div>

		<!-- Pagination -->
		<nav v-if="totalPages > 1" class="pagination is-centered mt-4" role="navigation">
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
</template>

<style scoped>
.track-row {
	cursor: pointer;
	user-select: none;
	-webkit-user-select: none;
	-moz-user-select: none;
	-ms-user-select: none;
}
/* Track row styling handled by theme.scss */

.track-row-text {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
	max-width: calc(100px + 10vw);
}

.table-container {
	overflow-x: auto;
}

.track-actions-cell {
	width: 48px;
	text-align: center;
}

.track-actions-menu-wrapper {
	position: relative;
	display: inline-flex;
	justify-content: center;
}

.track-actions-btn {
	width: 32px;
	height: 32px;
	border: none;
	background: transparent;
	color: var(--clr-text-secondary);
	cursor: pointer;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	transition:
		background-color 0.15s ease,
		color 0.15s ease;
}

.track-actions-btn:hover {
	background: var(--clr-surface-higher);
	color: var(--clr-text-primary);
}

.track-actions-menu {
	background: var(--clr-surface-elevated);
	border: 1px solid var(--clr-surface-higher);
	border-radius: 8px;
	padding: 0.25rem 0;
	min-width: 180px;
	z-index: 9999;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
	overflow-y: auto;
}

.track-actions-menu-item {
	display: flex;
	align-items: center;
	width: 100%;
	padding: 0.6rem 1rem;
	border: none;
	background: transparent;
	color: var(--clr-text-primary);
	font-size: 0.9rem;
	cursor: pointer;
	text-align: left;
}

.track-actions-menu-item:hover,
.track-actions-menu-item:active {
	background: var(--clr-surface-higher);
}

.track-actions-menu-item-danger {
	color: var(--clr-error, #ff6b6b);
}

.track-actions-menu-item-danger:hover,
.track-actions-menu-item-danger:active {
	background: rgba(255, 107, 107, 0.1);
}

.track-actions-divider {
	height: 1px;
	background: var(--clr-surface-higher);
	margin: 0.25rem 0;
}

.track-actions-submenu-label {
	display: flex;
	align-items: center;
	padding: 0.4rem 1rem 0.2rem;
	color: var(--clr-text-muted);
	font-size: 0.75rem;
	text-transform: uppercase;
	letter-spacing: 0.03em;
}

.track-cover {
	width: 32px;
	height: 32px;
	object-fit: cover;
	border-radius: 4px;
}

/* Mobile breakpoint */
@media screen and (max-width: 768px) {
	/* Hide album art column on mobile for more space */
	.cover-column {
		display: none;
	}

	/* Compact table styling */
	.table {
		font-size: 0.85rem;
	}

	.table th {
		padding: 0.5rem 0.25rem;
		font-size: 0.8rem;
	}

	.table td {
		padding: 0.5rem 0.25rem;
	}

	/* More compact track info */
	.track-row-text {
		max-width: none;
		font-size: 0.8rem;
	}

	/* Smaller title text */
	.table td strong {
		font-size: 0.9rem;
	}

	/* Compact pagination */
	.pagination-link,
	.pagination-previous,
	.pagination-next {
		font-size: 0.8rem;
		padding: 0.25rem 0.5rem;
	}
}

@media screen and (max-width: 480px) {
	/* Even more compact on small phones */
	.table {
		font-size: 0.8rem;
	}

	.table th {
		padding: 0.375rem 0.125rem;
		font-size: 0.75rem;
	}

	.table td {
		padding: 0.375rem 0.125rem;
	}

	.track-row-text {
		font-size: 0.75rem;
	}

	.table td strong {
		font-size: 0.85rem;
	}
}
</style>
