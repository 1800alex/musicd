<script setup lang="ts">
import { ref, onMounted } from "vue";
import type { Playlist } from "~/types";
import useAppState from "~/stores/appState";
import awaitAppState from "~/composables/awaitAppState";
import httpService from "~/services/http.service";
import backendService from "~/services/backend.service";
import PlayerService from "~/services/player.service";
import { useImageUrl } from "~/composables/useImageUrl";

// Page metadata
useHead({
	title: "Playlists - Music Player"
});

const appState = useAppState();
const router = useRouter();
const { getImageUrl } = useImageUrl();
const player = ref<PlayerService | null>(null);

// Reactive variables
const playlists = ref<Playlist[]>([]);
const loading = ref(false);
const showCreateModal = ref(false);
const creating = ref(false);
const newPlaylistName = ref("");
const newPlaylistLocation = ref("playlists");
const newPlaylistCustomPath = ref("");

// Methods
const fetchPlaylists = async () => {
	loading.value = true;
	try {
		const response = await backendService.FetchPlaylists({
			// page: currentPage.value,
			pageSize: appState.PageSize
			// search: searchQuery.value
		});
		playlists.value = response;
	} catch (error) {
		console.error("Error fetching playlists:", error);
	} finally {
		loading.value = false;
	}
};

const goToPlaylist = (playlist: Playlist) => {
	if (playlist.id) {
		router.push(`/playlists/${encodeURIComponent(playlist.id)}`).catch(console.error);
	}
};

const createPlaylist = async () => {
	if (!newPlaylistName.value.trim()) {
		return;
	}

	creating.value = true;
	try {
		const payload = {
			name: newPlaylistName.value.trim(),
			location: newPlaylistLocation.value,
			customPath: "custom" === newPlaylistLocation.value ? newPlaylistCustomPath.value : ""
		};

		await httpService.post("/api/playlist/create", payload);

		// Reset form
		newPlaylistName.value = "";
		newPlaylistLocation.value = "playlists";
		newPlaylistCustomPath.value = "";
		showCreateModal.value = false;

		// Refresh playlists
		await fetchPlaylists();

		console.log(`Created playlist "${payload.name}"`);
	} catch (error) {
		console.error("Error creating playlist:", error);
	} finally {
		creating.value = false;
	}
};

// Lifecycle
onMounted(async () => {
	await awaitAppState();

	player.value = new PlayerService(appState);

	await fetchPlaylists();
});
</script>

<template>
	<div class="music-page">
		<section class="hero hero-music-page">
			<div class="hero-body">
				<div class="container">
					<h1 class="title">
						<font-awesome-icon icon="fa-list" class="mr-2" />
						Playlists
					</h1>
					<p class="subtitle">Your custom music collections</p>
				</div>
			</div>
		</section>

		<div class="container mt-5">
			<!-- Create Playlist Button -->
			<div class="mb-4">
				<button
					data-testid="create-playlist-btn"
					class="page-action-btn page-action-btn-primary"
					@click="showCreateModal = true"
				>
					<font-awesome-icon icon="fa-plus" />
					Create New Playlist
				</button>
			</div>

			<!-- Loading State -->
			<div v-if="loading" class="has-text-centered p-4">
				<div class="is-loading"></div>
				<p>Loading playlists...</p>
			</div>

			<!-- Playlists Grid -->
			<div v-else-if="playlists.length > 0" data-testid="playlist-grid" class="columns is-multiline">
				<div
					v-for="playlist in playlists"
					:key="playlist.name"
					class="column is-one-quarter-desktop is-one-third-tablet is-half-mobile"
				>
					<div
						data-testid="playlist-card"
						:data-playlist-id="playlist.id"
						class="card music-card"
						@click="goToPlaylist(playlist)"
					>
						<div class="card-content">
							<div class="media">
								<div class="media-left">
									<figure class="image is-48x48">
										<!-- Use first track cover as playlist image, or default -->
										<img
											v-if="playlist.cover_art_id"
											:src="getImageUrl(`/api/cover-art/${playlist.cover_art_id}`)"
											:alt="`${playlist.name} cover`"
											class="is-rounded"
										/>
										<div
											v-else
											class="has-background-grey-lighter is-48x48 is-flex is-align-items-center is-justify-content-center is-rounded"
										>
											<font-awesome-icon icon="music" class="has-text-grey fa-lg" />
										</div>
									</figure>
								</div>
								<div class="media-content">
									<p class="title is-6">{{ playlist.name }}</p>
									<p class="subtitle is-7 has-text-grey">
										{{ playlist.track_count }} track{{ playlist.track_count !== 1 ? "s" : "" }}
									</p>
								</div>
							</div>
						</div>
						<div class="card-action-footer">
							<button
								data-testid="playlist-play-btn"
								class="card-action-btn"
								@click.stop="player?.PlayPlaylist(playlist, false)"
							>
								<font-awesome-icon icon="fa-play" />
								Play
							</button>
							<button
								data-testid="playlist-queue-btn"
								class="card-action-btn"
								@click.stop="player?.AddPlaylistToQueue(playlist)"
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
					<font-awesome-icon icon="fa-list" class="fa-3x mb-3"></font-awesome-icon>
				</p>
				<p class="title is-5 has-text-grey">No playlists yet</p>
				<p class="has-text-grey mb-4">Create your first playlist to get started</p>
			</div>
		</div>

		<!-- Create Playlist Modal -->
		<div data-testid="create-playlist-modal" class="modal" :class="{ 'is-active': showCreateModal }">
			<div class="modal-background" @click="showCreateModal = false"></div>
			<div class="modal-card">
				<header class="modal-card-head">
					<p class="modal-card-title">Create New Playlist</p>
					<button class="delete" @click="showCreateModal = false"></button>
				</header>
				<section class="modal-card-body">
					<div class="field">
						<label class="label">Playlist Name</label>
						<div class="control">
							<input
								v-model="newPlaylistName"
								class="input"
								type="text"
								placeholder="Enter playlist name"
								@keyup.enter="createPlaylist"
							/>
						</div>
					</div>
					<div class="field">
						<label class="label">Location</label>
						<div class="control">
							<div class="select is-fullwidth">
								<select v-model="newPlaylistLocation">
									<option value="playlists">Playlists folder</option>
									<option value="music">Music folder</option>
									<option value="custom">Custom folder</option>
								</select>
							</div>
						</div>
					</div>
					<div v-if="newPlaylistLocation === 'custom'" class="field">
						<label class="label">Custom Path</label>
						<div class="control">
							<input
								v-model="newPlaylistCustomPath"
								class="input"
								type="text"
								placeholder="folder/subfolder"
							/>
						</div>
					</div>
				</section>
				<footer class="modal-card-foot">
					<button class="button" @click="showCreateModal = false">Cancel</button>
					<button
						class="button is-primary"
						:disabled="!newPlaylistName.trim()"
						:class="{ 'is-loading': creating }"
						@click="createPlaylist"
					>
						Create
					</button>
				</footer>
			</div>
		</div>
	</div>
</template>
