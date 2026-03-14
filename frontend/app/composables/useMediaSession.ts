import { ref, watch } from "vue";
import type { Track } from "~/types";
import useAppState from "~/stores/appState";
import MediaSessionService from "~/services/mediaSession.service";
import { useImageUrl } from "~/composables/useImageUrl";

/**
 * Composable for managing Media Session API integration with the music player.
 * Handles lock screen controls, metadata display, and position tracking.
 * Works with both local playback and remote control scenarios.
 */
export const useMediaSession = () => {
	const appState = useAppState();
	const mediaService = new MediaSessionService();
	const positionUpdateInterval = ref<NodeJS.Timeout | null>(null);

	// Update metadata when track changes
	const updateMetadata = (track: Track | null) => {
		if (!track) {
			mediaService.clearMetadata();
			return;
		}

		// Update metadata with track info
		try {
			if ("mediaSession" in navigator) {
				const { getImageUrl } = useImageUrl();
				const artwork: MediaImage[] = [];

				if (track.cover_art_id) {
					artwork.push({
						src: getImageUrl(`/api/cover-art/${track.cover_art_id}`),
						sizes: "512x512",
						type: "image/jpeg"
					});
				}

				navigator.mediaSession.metadata = new MediaMetadata({
					title: track.title || "Unknown Title",
					artist: track.artist || "Unknown Artist",
					album: track.album || "Unknown Album",
					artwork:
						artwork.length > 0
							? artwork
							: [
									{
										src: `${window.location.origin}/favicon.ico`,
										sizes: "32x32"
									}
								]
				});
			}
		} catch (error) {
			console.warn("Failed to update media metadata:", error);
		}
	};

	// Update position state for lock screen progress
	const updatePosition = () => {
		const track = appState.CurrentTrack;
		if (!track || !("mediaSession" in navigator)) {
			return;
		}

		const duration = track.duration || 0;
		const position = appState.CurrentTime || 0;

		mediaService.setPositionState(duration, 1.0, position);
	};

	// Start position update loop (for lock screen progress bar)
	const startPositionUpdates = () => {
		if (positionUpdateInterval.value) {
			clearInterval(positionUpdateInterval.value);
		}

		// Update position every 1 second while playing
		positionUpdateInterval.value = setInterval(() => {
			if (appState.IsPlaying) {
				updatePosition();
			}
		}, 1000);
	};

	// Stop position updates
	const stopPositionUpdates = () => {
		if (positionUpdateInterval.value) {
			clearInterval(positionUpdateInterval.value);
			positionUpdateInterval.value = null;
		}
	};

	// Setup action handlers (lock screen buttons, headphone controls)
	const setupActionHandlers = (playerService: any) => {
		if (!mediaService.isMediaSessionSupported()) {
			return;
		}

		mediaService.setActionHandlers({
			play: () => {
				playerService?.Play();
			},
			pause: () => {
				playerService?.Pause();
			},
			previoustrack: () => {
				playerService?.PreviousTrack();
			},
			nexttrack: () => {
				playerService?.NextTrack();
			},
			seekto: (details) => {
				if (details.seekTime !== undefined && playerService?.Seek) {
					playerService.Seek(details.seekTime);
				}
			},
			seekforward: (details) => {
				const skipTime = details.skipTime || 15;
				const newTime = Math.min((appState.CurrentTime || 0) + skipTime, appState.CurrentTrack?.duration || 0);
				if (playerService?.Seek) {
					playerService.Seek(newTime);
				}
			},
			seekbackward: (details) => {
				const skipTime = details.skipTime || 15;
				const newTime = Math.max((appState.CurrentTime || 0) - skipTime, 0);
				if (playerService?.Seek) {
					playerService.Seek(newTime);
				}
			}
		});
	};

	// Watch for track changes
	const watchTrackChanges = () => {
		watch(
			() => appState.CurrentTrack,
			(newTrack) => {
				updateMetadata(newTrack);
				updatePosition();
			},
			{ deep: true }
		);
	};

	// Watch for playback state changes
	const watchPlaybackState = () => {
		watch(
			() => appState.IsPlaying,
			(isPlaying) => {
				mediaService.setPlaybackState(isPlaying ? "playing" : "paused");
				if (isPlaying) {
					startPositionUpdates();
				} else {
					stopPositionUpdates();
				}
			}
		);
	};

	// Watch for position changes (from seeking)
	const watchPositionChanges = () => {
		watch(
			() => appState.CurrentTime,
			() => {
				updatePosition();
			}
		);
	};

	// Initialize everything
	const init = (playerService: any) => {
		setupActionHandlers(playerService);
		watchTrackChanges();
		watchPlaybackState();
		watchPositionChanges();
		updateMetadata(appState.CurrentTrack);
	};

	// Cleanup
	const cleanup = () => {
		stopPositionUpdates();
		mediaService.clearMetadata();
	};

	return {
		init,
		cleanup,
		updateMetadata,
		updatePosition,
		setupActionHandlers,
		startPositionUpdates,
		stopPositionUpdates,
		isSupported: mediaService.isMediaSessionSupported()
	};
};
