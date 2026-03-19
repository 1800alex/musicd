<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick, provide } from "vue";
import { throttle } from "lodash";
import useAppState, { RepeatMode } from "~/stores/appState";
import awaitAppState from "@/composables/awaitAppState";
import backendService from "~/services/backend.service";
import httpService from "~/services/http.service";
import PlayerService from "~/services/player.service";
import useKeyboardShortcuts from "~/composables/useKeyboardShortcuts";
import { useRemoteSync } from "../composables/useRemoteSync";
import RemoteControlService from "~/services/remoteControl.service";
import { useAppInitialization } from "~/composables/useAppInitialization";
import { useNativeAudio } from "~/composables/useNativeAudio";
import { isNativeOrElectron } from "@/utils/platform";
import { useImageUrl } from "~/composables/useImageUrl";
import { useBackendURL } from "~/composables/useBackendURL";
import { useHaptics } from "../composables/useHaptics";
import { useMobilePlayerState } from "~/composables/useMobilePlayerState";

const appState = useAppState();
const { tap, selectionChanged } = useHaptics();
const router = useRouter();
const { getImageUrl } = useImageUrl();
const { getHTTPURL } = useBackendURL();
const player = ref<PlayerService | null>(null);
const mobilePlayer = useMobilePlayerState(player);

const BACKGROUND_COLOR_DEFAULT = "#282828";
const BACKGROUND_COLOR_MOBILE_PLAYER = "#121212";

// Remote sync state
const remoteSessionName = ref<string>("");
const remoteSessionId = ref<string>("");
const remoteControllerCount = ref<number>(0);
const remoteSyncEnabled = ref<boolean>(false);
const remoteSessions = ref<any[]>([]);
const editingSessionName = ref(false);
const editSessionNameInput = ref("");
const remoteControlledSessionName = ref("");
let remoteRenameSession: ((name: string) => void) | null = null;
let remoteEnable: (() => void) | null = null;
let remoteDisable: (() => void) | null = null;
let appInitCleanup: (() => void) | null = null;

// Reactive data
const showNavbarMenu = ref(false);
const showCreatePlaylistModal = ref(false);
const searchQuery = ref("");
const showVisualizerOverlay = ref(false);
const showLyricsModal = ref(false);
const isLoading = ref(false);
const isSearchFocused = ref(false);
const showRemoteDropdown = ref(false);
const showPlaylistDropdown = ref(false);
const backgroundColor = ref(BACKGROUND_COLOR_DEFAULT); // Default background color

// Server connectivity and remote server URL config (native/electron only)
const serverURL = ref(localStorage.getItem("backendURL") || "");
const editingServerURL = ref(false);
const serverURLInput = ref("");
const serverConnected = ref(true);
let connectivityFailures = 0;

const updateThemeColor = (color: string) => {
	// Update meta theme-color (older iOS Safari)
	let meta = document.querySelector('meta[name="theme-color"]') as HTMLMetaElement | null;
	if (!meta) {
		meta = document.createElement("meta");
		meta.name = "theme-color";
		document.head.appendChild(meta);
	}
	meta.content = color;
	// Update body background-color (newer iOS Safari uses this for the status bar)
	document.body.style.backgroundColor = color;
};

const closeRemoteDropdown = () => {
	showRemoteDropdown.value = false;
};
const closeMobilePlayerMenu = () => {
	mobilePlayer.closeMenu();
};

const closePlaylistDropdown = () => {
	showPlaylistDropdown.value = false;
};

// Search focus handlers
const handleSearchFocus = () => {
	isSearchFocused.value = true;
};

const handleSearchBlur = () => {
	isSearchFocused.value = false;
};

// Provide search handlers to child components
provide("searchFocus", handleSearchFocus);
provide("searchBlur", handleSearchBlur);

// Focus search function that will be called by keyboard shortcut
const focusSearch = () => {
	// Try to find and focus the search input on the current page
	const searchInput = document.querySelector('.track-list input[type="text"]') as HTMLInputElement;
	if (searchInput) {
		searchInput.focus();
	}
};

// Keyboard shortcuts will be set up after player is ready
let keyboardShortcuts: any = null;

// Audio player refs
const audioPlayer = ref<HTMLAudioElement | null>(null);
const coverArtImage = ref<HTMLImageElement | null>(null);

// Audio player state
const currentTime = ref(0);
const duration = ref(0);
const seekPosition = ref(0);
const seeking = ref(false);

// Smooth progress animation
let progressAnimationId: number | null = null;
// Remote mode interpolation: last received server time and the wall-clock instant it arrived
let remoteTimeBase = 0;
let remoteTimeBaseAt = 0;

// Methods
const toggleNavbarMenu = () => {
	showNavbarMenu.value = !showNavbarMenu.value;
};

const fetchPlaylists = async () => {
	try {
		appState.SetPlaylists(await backendService.FetchPlaylists());
	} catch (error) {
		console.error("Error fetching playlists:", error);
	}
};
const handleCreatePlaylist = async (name: string, location: string, customPath: string) => {
	isLoading.value = true;
	try {
		const payload = {
			name: name,
			location: location,
			customPath: "custom" === location ? customPath : ""
		};

		await httpService.post("/api/playlist/create", payload);

		// Close modal (component will reset form)
		showCreatePlaylistModal.value = false;

		// Refresh playlists
		await fetchPlaylists();

		console.log(`Created playlist "${payload.name}"`);
	} catch (error) {
		console.error("Error creating playlist:", error);
	} finally {
		isLoading.value = false;
	}
};

const handleCloseCreateModal = () => {
	showCreatePlaylistModal.value = false;
};

const performSearch = () => {
	if (searchQuery.value.trim()) {
		router.push(`/tracks?search=${encodeURIComponent(searchQuery.value.trim())}`).catch((error) => {
			console.error("Error navigating to search results:", error);
		});
		showNavbarMenu.value = false;
		searchQuery.value = "";
	}
};

const clearSearch = () => {
	searchQuery.value = "";
};

const toggleRemoteSync = () => {
	if (remoteSyncEnabled.value) {
		if (remoteDisable) {
			remoteDisable();
		}
	} else {
		if (remoteEnable) {
			remoteEnable();
		}
	}
};

const startEditingSessionName = () => {
	editSessionNameInput.value = remoteSessionName.value;
	editingSessionName.value = true;
};

const saveSessionName = () => {
	const name = editSessionNameInput.value.trim();
	if (name && remoteRenameSession) {
		remoteRenameSession(name);
		remoteSessionName.value = name;
	}
	editingSessionName.value = false;
};

const cancelEditSessionName = () => {
	editingSessionName.value = false;
};

const saveServerURL = () => {
	const url = serverURLInput.value.trim().replace(/\/$/, "");
	if (url) {
		localStorage.setItem("backendURL", url);
		serverURL.value = url;
		editingServerURL.value = false;
		// Force reconnect of remote sync WS if enabled
		if (remoteDisable && remoteSyncEnabled.value) {
			remoteDisable();
			setTimeout(() => remoteEnable?.(), 100);
		}
	}
};

const cancelEditServerURL = () => {
	editingServerURL.value = false;
};

const startEditingServerURL = () => {
	serverURLInput.value = serverURL.value;
	editingServerURL.value = true;
};

const fetchRemoteSessions = async () => {
	try {
		const response = await fetch(getHTTPURL("/api/sessions"));
		if (response.ok) {
			remoteSessions.value = await response.json();
			// Update displayed name if we're connected and sessions just loaded
			const connectedId = appState.RemoteControl ? localStorage.getItem("remoteControlSessionId") : null;
			if (connectedId) {
				const sess = remoteSessions.value.find((s: any) => s.id === connectedId);
				if (sess?.name) {
					remoteControlledSessionName.value = sess.name;
					localStorage.setItem("remoteControlSessionName", sess.name);
				}
			}
			// Connection succeeded
			connectivityFailures = 0;
			serverConnected.value = true;
		}
	} catch (err) {
		console.error("Error fetching remote sessions:", err);
	}
};

const disconnectRemote = () => {
	const rc = appState.RemoteControl;
	if (rc) {
		rc.disconnect();
	}
	appState.SetRemoteControl(null);
	remoteControlledSessionName.value = "";
	localStorage.removeItem("remoteControlSessionId");
	localStorage.removeItem("remoteControlSessionName");
	appState.SetCurrentTrack(null);
	appState.SetIsPlaying(false);
};

const connectToRemoteSession = (sessionId: string) => {
	if (!sessionId) {
		return;
	}

	// Capture stored name before disconnectRemote clears localStorage
	const knownName = localStorage.getItem("remoteControlSessionName");

	disconnectRemote();

	const { getServerHost } = useBackendURL();
	const rc = new RemoteControlService(getServerHost(), sessionId);

	rc.onConnected = () => {
		localStorage.setItem("remoteControlSessionId", sessionId);
		// Find the session name from the list — sessions may not be loaded yet on auto-reconnect,
		// so fall back to previously stored name rather than corrupting localStorage with the UUID.
		const sess = remoteSessions.value.find((s: any) => s.id === sessionId);
		if (sess?.name) {
			remoteControlledSessionName.value = sess.name;
			localStorage.setItem("remoteControlSessionName", sess.name);
		} else {
			// Sessions not loaded yet (race on page refresh) — use name saved before disconnect
			remoteControlledSessionName.value = knownName || sessionId;
			if (knownName) {
				localStorage.setItem("remoteControlSessionName", knownName);
			}
		}
	};

	rc.onDisconnected = () => {
		// Will auto-reconnect via RemoteControlService
	};

	rc.onStateUpdate = (state: any) => {
		// Anchor interpolation to the freshly received server time
		remoteTimeBase = state.current_time;
		remoteTimeBaseAt = performance.now();
		appState.SetIsPlaying(state.is_playing);
		appState.SetCurrentTrack(state.current_track);
		appState.SetCurrentTime(state.current_time);
		appState.SetDuration(state.duration);
		appState.SetVolume(state.volume);
		appState.SetMuted(state.muted);
		appState.SetShuffle(state.shuffle);
		appState.SetRepeatMode(state.repeat_mode);
		appState.SetQueue(state.queue || []);
		appState.SetTemporaryQueue(state.temporary_queue || []);
		appState.SetCurrentPlaylist(state.current_playlist || null);

		// Ensure smooth progress animation is running while playing.
		// The IsPlaying watcher only fires on changes, so if is_playing is
		// already true the animation loop may never have been started.
		if (state.is_playing) {
			startSmoothProgressAnimation();
		}
	};

	rc.onError = (err: any) => {
		console.error("Remote error:", err);
	};

	appState.SetRemoteControl(rc);
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

// Rescan state
const isRescanLoading = ref(false);
const showRescanConfirm = ref(false);
const showCompletionToast = ref(false);
let scanStatusPollInterval: ReturnType<typeof setInterval> | null = null;
let remoteSessionsPollInterval: ReturnType<typeof setInterval> | null = null;
let scanStartTime: number | null = null;
const MIN_SCAN_DISPLAY_TIME = 2000; // 2 seconds minimum

const confirmRescan = () => {
	showRescanConfirm.value = true;
};

const triggerRescan = async () => {
	showRescanConfirm.value = false;
	isRescanLoading.value = true;
	scanStartTime = Date.now(); // Record when scan started
	appState.SetIsScanning(true); // Show scanning indicator immediately
	try {
		await backendService.TriggerRescan();
		// Start polling more frequently while rescanning
		if (scanStatusPollInterval) {
			clearInterval(scanStatusPollInterval);
		}
		scanStatusPollInterval = setInterval(() => {
			pollScanStatus();
		}, 1000); // Poll every 1 second during scan
	} catch (error) {
		console.error("Error triggering rescan:", error);
		isRescanLoading.value = false;
		scanStartTime = null;
		appState.SetIsScanning(false); // Hide indicator if error
	}
};

const pollScanStatus = async () => {
	try {
		const isScanning = await backendService.FetchScanStatus();
		appState.SetIsScanning(isScanning);
		// Connection succeeded - reset failure counter
		connectivityFailures = 0;
		serverConnected.value = true;
		if (!isScanning && isRescanLoading.value && scanStartTime) {
			// Ensure minimum display time has elapsed
			const elapsedTime = Date.now() - scanStartTime;
			if (elapsedTime < MIN_SCAN_DISPLAY_TIME) {
				// Wait before finishing the loading state
				const remainingTime = MIN_SCAN_DISPLAY_TIME - elapsedTime;
				setTimeout(() => {
					finishScanLoading();
				}, remainingTime);
				return;
			}
			finishScanLoading();
		}
	} catch (error) {
		console.error("Error fetching scan status:", error);
		// Track connectivity failures
		connectivityFailures++;
		if (connectivityFailures >= 2) {
			serverConnected.value = false;
		}
	}
};

const finishScanLoading = () => {
	isRescanLoading.value = false;
	scanStartTime = null;
	// Show completion toast
	showCompletionToast.value = true;
	setTimeout(() => {
		showCompletionToast.value = false;
	}, 3000);
	// Revert to normal polling rate
	if (scanStatusPollInterval) {
		clearInterval(scanStatusPollInterval);
		scanStatusPollInterval = setInterval(() => {
			pollScanStatus();
		}, 2000); // Normal rate: every 2 seconds
	}
};

const loadPlaylist = (playlistID: string) => {
	if (!playlistID) {
		router.push("/playlists").catch((error) => {
			console.error("Error navigating to playlists:", error);
		});
		showNavbarMenu.value = false;
		return;
	}

	router.push(`/playlists/${encodeURIComponent(playlistID)}`).catch((error) => {
		console.error("Error navigating to playlist:", error);
	});
	showNavbarMenu.value = false;
};

const navigateToAllTracks = () => {
	router.push("/tracks").catch((error) => {
		console.error("Error navigating to all tracks:", error);
	});
	showNavbarMenu.value = false;
};

// Audio player methods
const togglePlay = async () => {
	await tap();
	player.value?.TogglePlay();
};

// TODO Ideally we should make a class for the queue management
// and handle all the logic there instead of in the component

const nextTrack = async () => {
	await selectionChanged();
	player.value?.NextTrack();
};

const previousTrack = async () => {
	await selectionChanged();
	player.value?.PreviousTrack();
};

const toggleShuffle = async () => {
	await tap();
	player.value?.SetShuffle(!appState.Shuffle);
};

const toggleRepeat = async () => {
	await tap();
	player.value?.CycleRepeatMode();
};

let volumeDebounceTimer: ReturnType<typeof setTimeout> | null = null;
const updateVolume = () => {
	// Update audio element immediately for responsiveness
	if (!appState.RemoteControl && appState.AudioElement) {
		appState.AudioElement.volume = appState.Volume / 100;
		if (appState.Muted && appState.Volume > 0) {
			appState.Muted = false;
		}
	}

	// Debounce the full UpdateVolume (which includes SaveToLocalStorage / remote send)
	if (volumeDebounceTimer) {
		clearTimeout(volumeDebounceTimer);
	}
	volumeDebounceTimer = setTimeout(() => {
		player.value?.UpdateVolume();
		volumeDebounceTimer = null;
	}, 250);
};

const toggleMute = () => {
	player.value?.ToggleMute();
};

// This function is throttled to run at most once every 5 seconds during playback to reduce CPU usage,
// If you don't do this, the timeupdate event can fire dozens of times per second and cause performance issues.
const updateProgress = throttle(() => {
	if (!seeking.value && audioPlayer.value) {
		currentTime.value = audioPlayer.value.currentTime;
		if (duration.value > 0) {
			seekPosition.value = (currentTime.value / duration.value) * 100;
		}
	}
}, 5000);

// Throttle Media Session position updates (expensive API calls)
const updateMediaSessionPosition = throttle(() => {
	player.value?.UpdatePositionState();
}, 1000); // Update position state at most once per second

const startSmoothProgressAnimation = () => {
	if (progressAnimationId) {
		cancelAnimationFrame(progressAnimationId);
	}

	const animate = () => {
		if (!seeking.value && duration.value > 0) {
			if (appState.RemoteControl) {
				// Interpolate: advance time by wall-clock elapsed since last server update.
				// Stop advancing if no update has arrived in 5 s (lost connection).
				const sinceLastUpdate = performance.now() - remoteTimeBaseAt;
				if (sinceLastUpdate < 5000) {
					const elapsed = sinceLastUpdate / 1000;
					const interpolated = Math.min(remoteTimeBase + elapsed, duration.value);
					currentTime.value = interpolated;
					seekPosition.value = (interpolated / duration.value) * 100;
				}
			} else if (audioPlayer.value && !audioPlayer.value.paused) {
				currentTime.value = audioPlayer.value.currentTime;
				seekPosition.value = (currentTime.value / duration.value) * 100;
			}
		}
		progressAnimationId = requestAnimationFrame(animate);
	};

	progressAnimationId = requestAnimationFrame(animate);
};

const stopSmoothProgressAnimation = () => {
	if (progressAnimationId) {
		cancelAnimationFrame(progressAnimationId);
		progressAnimationId = null;
	}
};

const updateDuration = () => {
	if (audioPlayer.value) {
		duration.value = audioPlayer.value.duration || 0;
	}
};

const seekToPosition = () => {
	const rc = appState.RemoteControl;
	if (rc && duration.value > 0) {
		const newTime = (seekPosition.value / 100) * duration.value;
		rc.seek(newTime);
		currentTime.value = newTime;
		remoteTimeBase = newTime;
		remoteTimeBaseAt = performance.now();
		seeking.value = false;
		return;
	}

	if (audioPlayer.value && duration.value > 0) {
		const newTime = (seekPosition.value / 100) * duration.value;
		audioPlayer.value.currentTime = newTime;
		currentTime.value = newTime;
	}
	seeking.value = false;
	// Restart smooth animation after seeking if playing
	if (appState.IsPlaying) {
		startSmoothProgressAnimation();
	}
};

const handleTrackEnd = () => {
	player.value?.HandleTrackEnd();
};

const formatTime = (seconds: number) => {
	const mins = Math.floor(seconds / 60);
	const secs = Math.floor(seconds % 60);
	return `${mins}:${secs.toString().padStart(2, "0")}`;
};

const toggleMobilePlayer = async () => {
	await mobilePlayer.toggle();
};

const toggleVisualizerOverlay = () => {
	showVisualizerOverlay.value = !showVisualizerOverlay.value;
};

const toggleLyricsModal = () => {
	showLyricsModal.value = !showLyricsModal.value;
};

const navigateToCurrentArtist = async () => {
	if (!appState.CurrentTrack) {
		return;
	}

	try {
		// Search for artist by name to get the artist ID
		const artistsResponse = await backendService.FetchArtists();
		const artist = artistsResponse.data.find((a) => a.name === appState.CurrentTrack!.artist);
		if (artist) {
			await router.push(`/artist/${artist.id}`);
		} else {
			console.warn(`Artist "${appState.CurrentTrack.artist}" not found`);
		}
	} catch (error) {
		console.error("Error navigating to current artist:", error);
	}
};

const navigateToCurrentAlbum = async () => {
	if (!appState.CurrentTrack) {
		return;
	}

	try {
		// Search for album by name and artist
		const artistsResponse = await backendService.FetchArtists();
		const artist = artistsResponse.data.find((a) => a.name === appState.CurrentTrack!.artist);
		if (artist) {
			const album = artist.albums.find((a) => a.name === appState.CurrentTrack!.album);
			if (album) {
				await router.push(`/album/${album.id}`);
			} else {
				console.warn(`Album "${appState.CurrentTrack.album}" by "${appState.CurrentTrack.artist}" not found`);
			}
		} else {
			console.warn(`Artist "${appState.CurrentTrack.artist}" not found`);
		}
	} catch (error) {
		console.error("Error navigating to current album:", error);
	}
};

const coverArtLoaded = () => {
	console.log("Cover art image loaded, updating Media Session artwork");
	// player.value?.SetCoverArtImageRef(coverArtImage);

	try {
		const img = coverArtImage.value;
		if (!img) {
			player.value?.UpdateMediaSessionArtwork({}); // Clear artwork
			throw new Error("Cover art image element not found");
		}

		if (img.complete && img.naturalHeight !== 0) {
			player.value?.UpdateMediaSessionArtwork({
				src: img.src
			});
		}
	} catch (error) {
		console.warn("Failed to use cached image, falling back to fetch:", error);
	}
};

let handleKeyDown: () => void = () => {};

onMounted(async () => {
	updateThemeColor(backgroundColor.value); // Set initial theme color on mount

	// `await awaitAppState()` must be called in onMounted()
	await awaitAppState();

	const svc = new PlayerService(appState);
	player.value = svc;
	player.value.SetTopLevel(true); // Indicate this is the top-level player
	player.value.LoadFromLocalStorage();
	appState.SetAudioElement(audioPlayer.value);

	// Initialize iOS PWA features and Media Session API
	const appInit = useAppInitialization(player.value);
	appInitCleanup = appInit.cleanup;

	// Initialize native audio handling (Capacitor background playback)
	useNativeAudio();

	// Set up keyboard shortcuts after player is ready
	keyboardShortcuts = useKeyboardShortcuts({
		player: svc,
		onToggleFullscreen: () => {
			void toggleMobilePlayer();
		},
		onToggleVisualizer: () => {
			showVisualizerOverlay.value = !showVisualizerOverlay.value;
		},
		onEscapeFullscreen: () => {
			mobilePlayer.close();
		},
		onEscapeVisualizer: () => {
			showVisualizerOverlay.value = false;
		},
		onFocusSearch: focusSearch,
		isFullscreenActive: () => mobilePlayer.state.showFullscreen,
		isVisualizerActive: () => showVisualizerOverlay.value,
		isSearchFocused: () => isSearchFocused.value
	});

	handleKeyDown = keyboardShortcuts.handleKeydown;
	document.addEventListener("keydown", handleKeyDown);
	document.addEventListener("click", closeRemoteDropdown);
	document.addEventListener("click", closeMobilePlayerMenu);
	document.addEventListener("click", closePlaylistDropdown);

	// Watch for theme changes
	// iOS theme-color: directly update DOM meta tag for reliable Safari support
	watch(
		() => mobilePlayer.state.showFullscreen,
		(open) => {
			backgroundColor.value = open ? BACKGROUND_COLOR_MOBILE_PLAYER : BACKGROUND_COLOR_DEFAULT;
		}
	);

	watch(backgroundColor, (val) => {
		updateThemeColor(val);
	});

	// Set up remote control WebSocket sync
	const remoteSyncResult = useRemoteSync(svc, appState, audioPlayer);
	remoteSessionName.value = remoteSyncResult.sessionName.value;
	remoteSessionId.value = remoteSyncResult.sessionId.value;
	remoteControllerCount.value = remoteSyncResult.controllerCount.value;
	remoteSyncEnabled.value = remoteSyncResult.enabled.value;
	remoteRenameSession = remoteSyncResult.renameSession;
	remoteEnable = remoteSyncResult.enable;
	remoteDisable = remoteSyncResult.disable;

	// Watch for changes
	watch(remoteSyncResult.sessionName, (val) => {
		remoteSessionName.value = val;
	});
	watch(remoteSyncResult.sessionId, (val) => {
		remoteSessionId.value = val;
	});
	watch(remoteSyncResult.controllerCount, (val) => {
		remoteControllerCount.value = val;
	});
	watch(remoteSyncResult.enabled, (val) => {
		remoteSyncEnabled.value = val;
	});

	// Restore remote-controlled session name from localStorage
	if (appState.RemoteControl) {
		remoteControlledSessionName.value = localStorage.getItem("remoteControlSessionName") || "";
	}

	// Auto-reconnect to previously selected remote session after a page refresh
	const savedRemoteSessionId = localStorage.getItem("remoteControlSessionId");
	if (savedRemoteSessionId && !appState.RemoteControl) {
		remoteControlledSessionName.value = localStorage.getItem("remoteControlSessionName") || "";
		connectToRemoteSession(savedRemoteSessionId);
	}

	await fetchPlaylists();
	void fetchRemoteSessions();

	// Start polling remote sessions and scan status
	remoteSessionsPollInterval = setInterval(() => {
		void fetchRemoteSessions();
	}, 3000); // Poll every 3 seconds

	scanStatusPollInterval = setInterval(() => {
		pollScanStatus();
	}, 2000); // Poll every 2 seconds

	if (!audioPlayer.value) {
		console.warn("Audio player not found");
	}

	await nextTick(() => {
		// Watch for play state changes
		watch(
			() => appState.IsPlaying,
			(playState) => {
				player.value?.PlayStateChanged(playState);
				// Start/stop smooth progress animation based on play state
				if (playState) {
					startSmoothProgressAnimation();
				} else {
					stopSmoothProgressAnimation();
				}
			}
		);

		// Set up audio element event listeners for media session
		if (audioPlayer.value && player.value) {
			audioPlayer.value.addEventListener("timeupdate", () => {
				updateMediaSessionPosition();
			});

			audioPlayer.value.addEventListener("loadedmetadata", () => {
				player.value?.UpdatePositionState();
			});

			audioPlayer.value.addEventListener("durationchange", () => {
				player.value?.UpdatePositionState();
			});
		}

		// Watch for volume changes
		watch(
			() => appState.Volume,
			(volume) => {
				player.value?.VolumeChanged(volume);
			}
		);

		// In remote mode, keep duration in sync and snap position when paused.
		// When playing, the animation loop interpolates smoothly — no need to update here.
		watch([() => appState.CurrentTime, () => appState.Duration], ([time, dur]) => {
			if (appState.RemoteControl) {
				duration.value = dur;
				if (!appState.IsPlaying) {
					currentTime.value = time;
					if (dur > 0) {
						seekPosition.value = (time / dur) * 100;
					}
				}
			}
		});
	});
});

onBeforeUnmount(() => {
	// Clean up app initialization (Media Session, PWA features)
	if (appInitCleanup) {
		appInitCleanup();
	}

	// Clean up smooth progress animation
	stopSmoothProgressAnimation();

	// Clean up keyboard event listener
	document.removeEventListener("keydown", handleKeyDown);
	document.removeEventListener("click", closeRemoteDropdown);
	document.removeEventListener("click", closeMobilePlayerMenu);
	document.removeEventListener("click", closePlaylistDropdown);

	// Clean up polling intervals
	if (scanStatusPollInterval) {
		clearInterval(scanStatusPollInterval);
	}
	if (remoteSessionsPollInterval) {
		clearInterval(remoteSessionsPollInterval);
	}

	// Clean up remote control connection on layout destroy (tab close)
	disconnectRemote();
});
</script>

<template>
	<div class="music-app">
		<!-- Navigation -->
		<nav class="navbar is-dark is-fixed-top">
			<div class="navbar-brand">
				<div class="navbar-item is-disabled navbar-item-no-hover">
					<h1 class="title is-4 has-text-white">
						<img src="/assets/icons/waveform.svg" alt="Logo" style="vertical-align: top" />
					</h1>
				</div>
				<div
					v-if="appState.RemoteControl"
					class="navbar-item remote-indicator has-dropdown"
					:class="{ 'is-active': showRemoteDropdown }"
				>
					<a class="navbar-link is-arrowless" @click.stop="showRemoteDropdown = !showRemoteDropdown">
						<font-awesome-icon icon="fa-tower-broadcast" class="has-text-success"></font-awesome-icon>
					</a>
					<div class="navbar-dropdown" @click.stop>
						<div class="navbar-item">
							<div>
								<p class="is-size-7 has-text-grey">Remote controlling</p>
								<p class="has-text-weight-bold">{{ remoteControlledSessionName || "Unknown session" }}</p>
							</div>
						</div>
						<hr class="navbar-divider" />
						<a
							class="navbar-item"
							@click="
								disconnectRemote();
								showRemoteDropdown = false;
							"
						>
							<font-awesome-icon icon="fa-sign-out-alt" class="mr-2"></font-awesome-icon>
							Disconnect
						</a>
					</div>
				</div>
				<a
					role="button"
					class="navbar-burger burger"
					aria-label="menu"
					aria-expanded="false"
					:class="{ 'is-active': showNavbarMenu }"
					@click="toggleNavbarMenu()"
				>
					<span aria-hidden="true"></span>
					<span aria-hidden="true"></span>
					<span aria-hidden="true"></span>
				</a>
			</div>
			<div class="navbar-menu" :class="{ 'is-active': showNavbarMenu }">
				<div class="navbar-start">
					<!-- Navigation Dropdown -->
					<div data-testid="nav-browse-dropdown" class="navbar-item has-dropdown is-hoverable">
						<a class="navbar-link">
							<font-awesome-icon icon="fa-compass" class="mr-2"></font-awesome-icon>
							Browse
						</a>
						<div class="navbar-dropdown">
							<a
								data-testid="nav-all-tracks-link"
								class="navbar-item"
								@click="
									navigateToAllTracks();
									showNavbarMenu = false;
								"
							>
								<font-awesome-icon icon="fa-list" class="mr-2"></font-awesome-icon>
								All Tracks
							</a>
							<NuxtLink
								to="/artists"
								data-testid="nav-artists-link"
								class="navbar-item"
								@click="showNavbarMenu = false"
							>
								<font-awesome-icon icon="fa-users" class="mr-2"></font-awesome-icon>
								Artists
							</NuxtLink>
						</div>
					</div>
					<div class="navbar-item has-dropdown" :class="{ 'is-active': showPlaylistDropdown }">
						<a class="navbar-link" @click.stop="showPlaylistDropdown = !showPlaylistDropdown">
							<font-awesome-icon icon="fa-folder" class="mr-2"></font-awesome-icon>
							Playlists
						</a>
						<div class="navbar-dropdown" @click.stop>
							<a
								class="navbar-item"
								@click="
									showCreatePlaylistModal = true;
									showPlaylistDropdown = false;
									showNavbarMenu = false;
								"
							>
								<font-awesome-icon icon="fa-plus" class="mr-2"></font-awesome-icon>
								Create Playlist
							</a>
							<a
								class="navbar-item"
								@click="
									loadPlaylist('');
									showPlaylistDropdown = false;
									showNavbarMenu = false;
								"
							>
								<font-awesome-icon icon="fa-folder" class="mr-2"></font-awesome-icon>
								All Playlists
							</a>
							<hr v-if="appState.Playlists.length > 0" class="navbar-divider" />
							<PlaylistPicker
								v-if="appState.Playlists.length > 0"
								:playlists="appState.Playlists"
								@select="
									(p) => {
										loadPlaylist(p.id);
										showPlaylistDropdown = false;
										showNavbarMenu = false;
									}
								"
							/>
							<div v-if="appState.Playlists.length === 0" class="navbar-item">
								<span class="has-text-grey">No playlists found</span>
							</div>
						</div>
					</div>
					<div class="navbar-item has-dropdown is-hoverable">
						<a class="navbar-link">
							<font-awesome-icon icon="fa-library" class="mr-2"></font-awesome-icon>
							Library
						</a>
						<div class="navbar-dropdown">
							<a
								class="navbar-item"
								:disabled="isRescanLoading || appState.IsScanning"
								title="Rescan library"
								@click="
									confirmRescan();
									showNavbarMenu = false;
								"
							>
								<font-awesome-icon icon="fa-sync" class="mr-2"></font-awesome-icon>
								Rescan Library
							</a>
						</div>
					</div>
					<div class="navbar-item has-dropdown is-hoverable">
						<a class="navbar-link">
							<font-awesome-icon icon="fa-mobile-screen" class="mr-2"></font-awesome-icon>
							Remote
							<span v-if="remoteControllerCount > 0" class="tag is-success is-small ml-2">
								{{ remoteControllerCount }}
							</span>
						</a>
						<div
							class="navbar-dropdown is-scrollable"
							style="max-height: 400px; overflow-y: auto; min-width: 300px"
						>
							<!-- Server URL config (native/electron only) -->
							<template v-if="isNativeOrElectron">
								<div class="navbar-item">
									<div style="width: 100%">
										<p class="is-size-7 has-text-grey">Server URL</p>
										<div v-if="editingServerURL" class="field has-addons mt-1">
											<div class="control is-expanded">
												<input
													v-model="serverURLInput"
													class="input is-small"
													type="url"
													placeholder="http://192.168.1.x:8080"
													@keyup.enter="saveServerURL"
													@keyup.escape="cancelEditServerURL"
												/>
											</div>
											<div class="control">
												<button class="button is-small is-success" @click="saveServerURL">
													<font-awesome-icon icon="fa-check"></font-awesome-icon>
												</button>
											</div>
											<div class="control">
												<button class="button is-small" @click="cancelEditServerURL">
													<font-awesome-icon icon="fa-times"></font-awesome-icon>
												</button>
											</div>
										</div>
										<div v-else class="is-flex is-align-items-center mt-1">
											<p class="is-family-monospace is-size-7">{{ serverURL || "Not set" }}</p>
											<a class="ml-2" @click.stop="startEditingServerURL">
												<font-awesome-icon icon="fa-pen" size="xs" class="has-text-grey" />
											</a>
										</div>
									</div>
								</div>
								<hr class="navbar-divider" />
							</template>

							<!-- Share toggle -->
							<div class="navbar-item">
								<div
									class="is-flex is-align-items-center is-justify-content-space-between"
									style="width: 100%"
								>
									<span class="is-size-7">Share this session &nbsp;</span>
									<button
										class="button is-small"
										:class="remoteSyncEnabled ? 'is-success' : ''"
										@click.stop="toggleRemoteSync"
									>
										{{ remoteSyncEnabled ? "On" : "Off" }}
									</button>
								</div>
							</div>

							<!-- Session details (only when enabled) -->
							<template v-if="remoteSyncEnabled">
								<div class="navbar-item">
									<div style="width: 100%">
										<p class="is-size-7 has-text-grey">Session Name</p>
										<div v-if="editingSessionName" class="field has-addons mt-1">
											<div class="control is-expanded">
												<input
													v-model="editSessionNameInput"
													class="input is-small"
													type="text"
													placeholder="Session name"
													@keyup.enter="saveSessionName"
													@keyup.escape="cancelEditSessionName"
												/>
											</div>
											<div class="control">
												<button class="button is-small is-success" @click="saveSessionName">
													<font-awesome-icon icon="fa-check"></font-awesome-icon>
												</button>
											</div>
											<div class="control">
												<button class="button is-small" @click="cancelEditSessionName">
													<font-awesome-icon icon="fa-times"></font-awesome-icon>
												</button>
											</div>
										</div>
										<div v-else class="is-flex is-align-items-center mt-1">
											<p class="has-text-weight-bold is-family-monospace">
												{{ remoteSessionName || "Connecting..." }}
											</p>
											<a
												v-if="remoteSessionName"
												class="ml-2"
												title="Rename session"
												@click.stop="startEditingSessionName"
											>
												<font-awesome-icon
													icon="fa-pen"
													size="xs"
													class="has-text-grey"
												></font-awesome-icon>
											</a>
										</div>
										<p class="is-size-7 has-text-grey mt-1">
											{{ remoteControllerCount }} remote(s) connected
										</p>
									</div>
								</div>
								<hr class="navbar-divider" />
							</template>

							<!-- Other Sessions -->
							<div v-if="remoteSessions.filter((s) => s.id !== remoteSessionId).length > 0">
								<p class="navbar-item is-size-7 has-text-grey">Other Sessions</p>
								<a
									v-for="sess in remoteSessions.filter((s) => s.id !== remoteSessionId)"
									:key="sess.id"
									class="navbar-item"
									style="font-size: 0.875rem"
									@click="connectToRemoteSession(sess.id)"
								>
									<span v-if="sess.has_player" class="icon is-small has-text-success">
										<font-awesome-icon icon="fa-circle" size="xs"></font-awesome-icon>
									</span>
									<span v-else class="icon is-small has-text-grey">
										<font-awesome-icon icon="fa-circle" size="xs"></font-awesome-icon>
									</span>
									&nbsp; {{ sess.name }}
									<span v-if="sess.controller_count > 0" class="tag is-small is-info ml-2">
										{{ sess.controller_count }}
									</span>
								</a>
								<hr class="navbar-divider" />
							</div>

							<!-- Connected remote indicator + disconnect -->
							<a v-if="appState.RemoteControl" class="navbar-item" @click="disconnectRemote">
								<font-awesome-icon icon="fa-sign-out-alt" class="mr-2"></font-awesome-icon>
								Disconnect Remote
							</a>
							<hr v-if="appState.RemoteControl" class="navbar-divider" />
						</div>
					</div>
					<div class="navbar-item is-hidden-touch navbar-item-no-hover">
						<div class="nav-search">
							<span class="nav-search-icon">
								<font-awesome-icon icon="fa-search"></font-awesome-icon>
							</span>
							<input
								v-model="searchQuery"
								data-testid="nav-search-input"
								class="nav-search-input"
								type="text"
								placeholder="Search tracks..."
								@input="onSearchInput"
								@keydown.enter="performSearch"
							/>
							<button
								v-if="searchQuery"
								class="nav-search-clear"
								data-testid="nav-search-btn"
								@click="clearSearch"
							>
								<font-awesome-icon icon="fa-times"></font-awesome-icon>
							</button>
						</div>
					</div>
					<!-- Scan Progress Indicator -->
					<transition name="slide-fade">
						<div v-if="appState.IsScanning" class="navbar-item scan-indicator">
							<span class="scan-badge">
								<font-awesome-icon icon="fa-sync" class="fa-spin"></font-awesome-icon>
								Scanning Library
							</span>
						</div>
					</transition>
				</div>
			</div>
		</nav>

		<!-- Main Content -->
		<div class="content">
			<slot />
		</div>

		<!-- Audio Player or Connectivity Lost Overlay -->
		<template v-if="isNativeOrElectron && !serverConnected">
			<!-- Connectivity Lost Overlay -->
			<div data-testid="connectivity-overlay" class="connectivity-lost-bar">
				<font-awesome-icon icon="fa-wifi" class="mr-2 has-text-warning" />
				<span>Connection lost</span>
				<span v-if="serverURL" class="ml-2 is-size-7 has-text-grey">{{ serverURL }}</span>
				<span class="ml-3 is-size-7 has-text-grey">Reconnecting...</span>
			</div>
		</template>

		<!-- Regular Bottom Player -->
		<div
			v-else-if="appState.CurrentTrack && !mobilePlayer.state.showFullscreen"
			data-testid="audio-player"
			class="audio-player"
		>
			<div
				v-touch:swipe.left="mobilePlayer.onMiniPlayerSwipeLeft"
				v-touch:swipe.right="mobilePlayer.onMiniPlayerSwipeRight"
				v-touch:swipe.up="mobilePlayer.onMiniPlayerSwipeUp"
				class="audio-player-controls"
			>
				<div class="audio-player-left">
					<div class="level-item">
						<figure
							data-testid="player-cover-art"
							class="image is-48x48 clickable-cover"
							@click="toggleMobilePlayer()"
						>
							<img
								v-if="appState.CurrentTrack.cover_art_id"
								ref="coverArtImage"
								:src="getImageUrl(`/api/cover-art/${appState.CurrentTrack.cover_art_id}`)"
								:alt="`${appState.CurrentTrack.album} cover`"
								loading="lazy"
								@load="coverArtLoaded()"
							/>
							<div v-else class="player-cover-placeholder">
								<font-awesome-icon icon="fa-music" />
							</div>
						</figure>
						<div class="ml-3 track-info">
							<MarqueeText data-testid="player-track-title" class="has-text-white has-text-weight-semibold">
								{{ appState.CurrentTrack.title }}
							</MarqueeText>
							<MarqueeText class="has-text-grey-light">
								<span
									data-testid="player-track-artist"
									class="clickable-artist"
									@click="navigateToCurrentArtist()"
									>{{ appState.CurrentTrack.artist }}</span
								>
								<span class="mx-1">•</span>
								<span
									data-testid="player-track-album"
									class="clickable-artist"
									@click="navigateToCurrentAlbum()"
									>{{ appState.CurrentTrack.album }}</span
								>
							</MarqueeText>
						</div>
					</div>
				</div>
				<div class="audio-player-center">
					<div class="player-controls-group">
						<button
							data-testid="player-shuffle-btn"
							class="player-btn"
							:class="{ active: appState.Shuffle }"
							@click="toggleShuffle()"
						>
							<font-awesome-icon icon="fa-random" />
						</button>
						<button data-testid="player-prev-btn" class="player-btn" @click="previousTrack()">
							<font-awesome-icon icon="fa-step-backward" />
						</button>
						<button data-testid="player-play-btn" class="player-btn player-btn-play" @click="togglePlay()">
							<font-awesome-icon :icon="appState.IsPlaying ? 'fa-pause' : 'fa-play'" />
						</button>
						<button data-testid="player-next-btn" class="player-btn" @click="nextTrack()">
							<font-awesome-icon icon="fa-step-forward" />
						</button>
						<button
							data-testid="player-repeat-btn"
							class="player-btn"
							:class="{ active: appState.RepeatMode !== RepeatMode.Off }"
							@click="toggleRepeat()"
						>
							<font-awesome-icon :icon="appState.RepeatMode === RepeatMode.One ? 'fa-redo' : 'fa-repeat'" />
						</button>
						<button
							v-if="!appState.RemoteControl"
							data-testid="player-visualizer-btn"
							class="player-btn"
							:class="{ 'active-info': showVisualizerOverlay }"
							@click="toggleVisualizerOverlay()"
						>
							<font-awesome-icon icon="fa-wave-square" />
						</button>
						<button
							data-testid="player-lyrics-btn"
							class="player-btn"
							:class="{ 'active-info': showLyricsModal }"
							title="Show Lyrics"
							@click="toggleLyricsModal()"
						>
							<font-awesome-icon icon="fa-align-left" />
						</button>
					</div>
				</div>
				<div class="audio-player-right">
					<input
						v-model="appState.Volume"
						data-testid="player-volume-slider"
						type="range"
						min="0"
						max="100"
						class="slider"
						:style="{ '--volume-percent': appState.Volume + '%' }"
						@input="updateVolume()"
					/>
					<font-awesome-icon
						data-testid="player-mute-btn"
						class="has-text-white ml-2 volume-icon"
						:icon="
							appState.Muted || appState.Volume == 0
								? 'fa-volume-mute'
								: appState.Volume <= 50
									? 'fa-volume-down'
									: 'fa-volume-up'
						"
						@click="toggleMute()"
					></font-awesome-icon>
				</div>
			</div>

			<!-- Progress Bar -->
			<div class="progress-container">
				<div class="progress-bar-container">
					<input
						v-model="seekPosition"
						data-testid="player-progress-bar"
						type="range"
						min="0"
						max="100"
						step="0.001"
						class="progress-bar"
						:style="{ '--progress-percent': seekPosition + '%' }"
						@mousedown="
							seeking = true;
							stopSmoothProgressAnimation();
						"
						@mouseup="seekToPosition()"
						@touchstart="
							seeking = true;
							stopSmoothProgressAnimation();
						"
						@touchend="seekToPosition()"
					/>
					<div class="time-info">
						<span data-testid="player-current-time" class="has-text-white">{{ formatTime(currentTime) }}</span>
						<span data-testid="player-duration" class="has-text-white">{{ formatTime(duration) }}</span>
					</div>
				</div>
			</div>
		</div>

		<!-- Mobile Mini Player Bar -->
		<div
			v-if="appState.CurrentTrack && !mobilePlayer.state.showFullscreen && (serverConnected || !isNativeOrElectron)"
			v-touch:swipe.left="mobilePlayer.onMiniPlayerSwipeLeft"
			v-touch:swipe.right="mobilePlayer.onMiniPlayerSwipeRight"
			v-touch:swipe.up="mobilePlayer.onMiniPlayerSwipeUp"
			data-testid="mobile-mini-player"
			class="mobile-mini-player"
			:style="{
				transform: `translateY(${-mobilePlayer.state.miniPlayerDrag.offsetY}px)`,
				transition: mobilePlayer.state.miniPlayerDrag.isDragging ? 'none' : 'all 0.3s ease-out'
			}"
			@click="mobilePlayer.open()"
			@touchstart="mobilePlayer.onMiniPlayerDragging"
			@touchmove="mobilePlayer.onMiniPlayerDragging"
			@touchend="mobilePlayer.onMiniPlayerDragEnd"
		>
			<div class="mobile-mini-progress" :style="{ width: seekPosition + '%' }"></div>
			<figure class="mobile-mini-cover">
				<img
					v-if="appState.CurrentTrack.cover_art_id"
					:src="getImageUrl(`/api/cover-art/${appState.CurrentTrack.cover_art_id}`)"
					:alt="`${appState.CurrentTrack.album} cover`"
				/>
				<div v-else class="mobile-mini-cover-placeholder">
					<font-awesome-icon icon="fa-music" />
				</div>
			</figure>
			<div class="mobile-mini-info">
				<p class="mobile-mini-title">{{ appState.CurrentTrack.title }}</p>
				<p class="mobile-mini-artist">{{ appState.CurrentTrack.artist }}</p>
			</div>
			<button data-testid="mobile-mini-play" class="mobile-mini-play" @click.stop="togglePlay()">
				<font-awesome-icon :icon="appState.IsPlaying ? 'fa-pause' : 'fa-play'" />
			</button>
		</div>

		<!-- Mobile Full-Screen Player -->
		<transition name="mobile-player-slide">
			<div
				v-if="mobilePlayer.shouldShowFullscreen && appState.CurrentTrack && (serverConnected || !isNativeOrElectron)"
				v-touch:swipe.down="mobilePlayer.onFullscreenSwipeDown"
				v-touch:swipe.left="mobilePlayer.onFullscreenSwipeLeft"
				v-touch:swipe.right="mobilePlayer.onFullscreenSwipeRight"
				v-touch:drag="mobilePlayer.onFullscreenDragging"
				data-testid="mobile-player"
				class="mobile-fullscreen-player"
				:style="mobilePlayer.fullscreenStyle"
				@touchend="mobilePlayer.onFullscreenDragEnd"
			>
				<div class="mobile-player-header">
					<button
						data-testid="mobile-player-collapse"
						class="mobile-player-collapse"
						@click="mobilePlayer.close()"
					>
						<font-awesome-icon icon="fa-chevron-down" />
					</button>
					<span class="mobile-player-header-title">Now Playing</span>
					<div class="mobile-player-menu-wrapper">
						<button
							data-testid="mobile-player-menu-btn"
							class="mobile-player-collapse"
							@click.stop="mobilePlayer.state.showMenu = !mobilePlayer.state.showMenu"
						>
							<font-awesome-icon icon="fa-ellipsis-vertical" />
						</button>
						<div v-if="mobilePlayer.state.showMenu" class="mobile-player-menu" @click="mobilePlayer.closeMenu()">
							<button
								data-testid="mobile-player-go-artist"
								class="mobile-player-menu-item"
								@click="
									navigateToCurrentArtist();
									mobilePlayer.close();
								"
							>
								<font-awesome-icon icon="fa-user" class="mr-2" />
								Go to artist
							</button>
							<button
								data-testid="mobile-player-go-album"
								class="mobile-player-menu-item"
								@click="
									navigateToCurrentAlbum();
									mobilePlayer.close();
								"
							>
								<font-awesome-icon icon="fa-folder" class="mr-2" />
								Go to album
							</button>
							<button
								v-if="appState.CurrentPlaylist"
								class="mobile-player-menu-item"
								@click="
									router.push(`/playlists/${appState.CurrentPlaylist?.id}`);
									mobilePlayer.close();
								"
							>
								<font-awesome-icon icon="fa-list" class="mr-2" />
								Go to playlist
							</button>
						</div>
					</div>
				</div>
				<div
					data-testid="mobile-player-cover"
					class="mobile-player-cover"
					:style="{
						transform: `translateX(${mobilePlayer.state.fullscreenDrag.offsetX * 0.15}px)`,
						transition: mobilePlayer.state.fullscreenDrag.isDragging ? 'none' : 'transform 0.3s ease-out'
					}"
				>
					<!-- Exiting art (old track, flies off screen) -->
					<div
						v-if="
							mobilePlayer.state.albumArtAnimation.phase !== 'idle' &&
							mobilePlayer.state.albumArtAnimation.exitingCoverArtUrl
						"
						class="album-art-layer album-art-exiting"
						:class="{
							'fly-out-left': mobilePlayer.state.albumArtAnimation.direction === 'left',
							'fly-out-right': mobilePlayer.state.albumArtAnimation.direction === 'right'
						}"
					>
						<img
							:src="mobilePlayer.state.albumArtAnimation.exitingCoverArtUrl"
							class="mobile-player-cover-image"
						/>
					</div>
					<!-- Current art (new track, flies in) -->
					<div
						class="album-art-layer"
						:class="{
							'fly-in-from-right':
								mobilePlayer.state.albumArtAnimation.phase === 'enter' &&
								mobilePlayer.state.albumArtAnimation.direction === 'left',
							'fly-in-from-left':
								mobilePlayer.state.albumArtAnimation.phase === 'enter' &&
								mobilePlayer.state.albumArtAnimation.direction === 'right'
						}"
					>
						<img
							v-if="appState.CurrentTrack.cover_art_id"
							:src="getImageUrl(`/api/cover-art/${appState.CurrentTrack.cover_art_id}`)"
							:alt="`${appState.CurrentTrack.album} cover`"
							data-testid="mobile-player-cover-image"
							class="mobile-player-cover-image"
						/>
						<div v-else class="mobile-player-cover-placeholder">
							<font-awesome-icon icon="fa-music" />
						</div>
					</div>
				</div>
				<div class="mobile-player-info">
					<MarqueeText class="mobile-player-title">{{ appState.CurrentTrack.title }}</MarqueeText>
					<MarqueeText class="mobile-player-artist">
						<span
							class="clickable-artist"
							@click="
								navigateToCurrentArtist();
								mobilePlayer.close();
							"
							>{{ appState.CurrentTrack.artist }}</span
						>
						<span class="mx-1">&bull;</span>
						<span
							class="clickable-artist"
							@click="
								navigateToCurrentAlbum();
								mobilePlayer.close();
							"
							>{{ appState.CurrentTrack.album }}</span
						>
					</MarqueeText>
				</div>
				<div class="mobile-player-seek">
					<input
						v-model="seekPosition"
						type="range"
						min="0"
						max="100"
						step="0.001"
						data-testid="mobile-player-seek"
						class="mobile-seek-bar"
						:style="{ '--progress-percent': seekPosition + '%' }"
						@mousedown="
							seeking = true;
							stopSmoothProgressAnimation();
						"
						@mouseup="seekToPosition()"
						@touchstart="
							seeking = true;
							stopSmoothProgressAnimation();
						"
						@touchend="seekToPosition()"
					/>
					<div class="mobile-player-times">
						<span>{{ formatTime(currentTime) }}</span>
						<span>{{ formatTime(duration) }}</span>
					</div>
				</div>
				<div class="mobile-player-controls">
					<button
						data-testid="mobile-player-shuffle"
						class="mobile-control-btn"
						:class="{ active: appState.Shuffle }"
						@click="toggleShuffle()"
					>
						<font-awesome-icon icon="fa-random" />
					</button>
					<button data-testid="mobile-player-prev" class="mobile-control-btn" @click="previousTrack()">
						<font-awesome-icon icon="fa-step-backward" />
					</button>
					<button
						data-testid="mobile-player-play"
						class="mobile-control-btn mobile-play-btn"
						@click="togglePlay()"
					>
						<font-awesome-icon :icon="appState.IsPlaying ? 'fa-pause' : 'fa-play'" />
					</button>
					<button data-testid="mobile-player-next" class="mobile-control-btn" @click="nextTrack()">
						<font-awesome-icon icon="fa-step-forward" />
					</button>
					<button
						data-testid="mobile-player-repeat"
						class="mobile-control-btn"
						:class="{ active: appState.RepeatMode !== RepeatMode.Off }"
						@click="toggleRepeat()"
					>
						<font-awesome-icon :icon="appState.RepeatMode === RepeatMode.One ? 'fa-redo' : 'fa-repeat'" />
					</button>
				</div>
				<div class="mobile-player-volume">
					<font-awesome-icon
						data-testid="mobile-player-mute"
						class="mobile-volume-icon"
						:icon="
							appState.Muted || appState.Volume == 0
								? 'fa-volume-mute'
								: appState.Volume <= 50
									? 'fa-volume-down'
									: 'fa-volume-up'
						"
						@click="toggleMute()"
					/>
					<input
						v-model="appState.Volume"
						type="range"
						min="0"
						max="100"
						data-testid="mobile-player-volume"
						class="mobile-volume-slider"
						:style="{ '--volume-percent': appState.Volume + '%' }"
						@input="updateVolume()"
					/>
				</div>
			</div>
		</transition>

		<!-- HTML5 Audio Element -->
		<audio
			ref="audioPlayer"
			preload="metadata"
			controls="false"
			:playsinline="true"
			style="display: none"
			@ended="handleTrackEnd()"
			@timeupdate="updateProgress()"
			@loadedmetadata="updateDuration()"
			@play="startSmoothProgressAnimation()"
			@pause="stopSmoothProgressAnimation()"
			@seeking="stopSmoothProgressAnimation()"
			@seeked="() => appState.IsPlaying && startSmoothProgressAnimation()"
		></audio>

		<!-- Fullscreen Visualizer Overlay -->
		<WaveformBackground
			v-if="audioPlayer"
			:audio-el="audioPlayer"
			:fixed="true"
			vignette
			:visible="showVisualizerOverlay"
			:current-track="appState.CurrentTrack"
			performance="medium"
			@close="showVisualizerOverlay = false"
		/>

		<!-- Lyrics Modal -->
		<LyricsModal :visible="showLyricsModal" :current-track="appState.CurrentTrack" @close="showLyricsModal = false" />

		<!-- Scan Completion Toast -->
		<transition name="slide-fade">
			<div v-if="showCompletionToast" class="scan-toast">
				<div class="scan-toast-content">
					<font-awesome-icon icon="fa-check-circle" class="scan-toast-icon" />
					<span>Library scan completed</span>
				</div>
			</div>
		</transition>

		<!-- Create Playlist Modal -->
		<CreatePlaylistModal
			:is-open="showCreatePlaylistModal"
			:is-loading="isLoading"
			@close="handleCloseCreateModal"
			@create="handleCreatePlaylist"
		/>

		<!-- Rescan Confirmation Modal -->
		<div :class="{ modal: true, 'is-active': showRescanConfirm }">
			<div class="modal-background" @click="showRescanConfirm = false"></div>
			<div class="modal-card">
				<header class="modal-card-head">
					<p class="modal-card-title">Rescan Library</p>
					<button class="delete" aria-label="close" @click="showRescanConfirm = false"></button>
				</header>
				<section class="modal-card-body">
					<p>Are you sure you want to rescan the music library? This may take a while.</p>
				</section>
				<footer class="modal-card-foot">
					<button class="button" @click="showRescanConfirm = false">Cancel</button>
					<button class="button is-primary" @click="triggerRescan()">Confirm</button>
				</footer>
			</div>
		</div>
	</div>
</template>

<style>
/* Layout-specific styles only */

.connectivity-lost-bar {
	position: fixed;
	bottom: 0;
	left: 0;
	right: 0;
	height: 60px;
	display: flex;
	align-items: center;
	padding: 0 1rem;
	background-color: #1a1a1a;
	border-top: 1px solid #404040;
	z-index: 999;
	color: #fff;
	gap: 0.5rem;
}

.control-button {
	transition: background-color 0.2s ease;
	width: 48px;
}

/* Toast notification styles */
.scan-toast {
	position: fixed;
	bottom: 2rem;
	right: 2rem;
	z-index: 1000;
	max-width: 400px;
	animation: slideInUp 0.3s ease-out;
}

.scan-toast-content {
	display: inline-flex;
	align-items: center;
	gap: 0.6rem;
	padding: 0.6rem 1.2rem;
	border-radius: 999px;
	font-size: 0.85rem;
	font-weight: 600;
	color: var(--clr-success);
	background-color: var(--clr-surface-elevated);
	border: 1px solid var(--clr-success);
	box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}

.scan-toast-icon {
	font-size: 1rem;
}

@keyframes slideInUp {
	from {
		transform: translateY(100%);
		opacity: 0;
	}
	to {
		transform: translateY(0);
		opacity: 1;
	}
}

.slide-fade-enter-active,
.slide-fade-leave-active {
	transition: all 0.3s ease;
}

.slide-fade-enter-from {
	transform: translateY(100%);
	opacity: 0;
}

.slide-fade-leave-to {
	transform: translateY(100%);
	opacity: 0;
}

/* Scan progress badge next to search */
.scan-indicator {
	display: flex;
	align-items: center;
	margin-left: 1rem;
}

.scan-badge {
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.35rem 0.9rem;
	border-radius: 999px;
	font-size: 0.8rem;
	font-weight: 600;
	color: var(--clr-warning);
	background-color: var(--clr-surface-elevated);
	border: 1px solid var(--clr-warning);
	animation: pulse 1.5s ease-in-out infinite;
}

/* Fullscreen cover styles */
.fullscreen-cover-content {
	padding-top: 4rem;
	padding-bottom: 150px;
	background: var(--clr-surface-base);
	min-height: 100vh;
	position: relative;
	display: flex;
	flex-direction: column;
}

.fullscreen-cover-container {
	flex: 1;
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 2rem;
	min-height: 0; /* Allow flex child to shrink */
}

.fullscreen-cover-image {
	width: 100%;
	height: 100%;
	max-width: calc(100vh - 12rem); /* Constrain by viewport height minus navbar/player/padding */
	max-height: calc(100vh - 18rem);
	object-fit: contain;
	border-radius: 8px;
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}

.fullscreen-close-btn {
	position: absolute;
	top: 5.5rem;
	right: 2rem;
	background: rgba(255, 255, 255, 0.2);
	border: none;
	color: white;
	width: 48px;
	height: 48px;
	border-radius: 50%;
	cursor: pointer;
	font-size: 1.5rem;
	transition: background-color 0.2s ease;
	z-index: 10;
}

.fullscreen-close-btn:hover {
	background: rgba(255, 255, 255, 0.3);
}

/* Playlist modal styles */
.playlist-modal {
	position: fixed;
	top: 0;
	left: 0;
	width: 100vw;
	height: 100vh;
	background: rgba(0, 0, 0, 0.5);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 9998;
}

.playlist-modal-content {
	padding: 2rem;
	border-radius: 8px;
	max-width: 500px;
	width: 90%;
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.playlist-modal-buttons {
	margin-top: 1.5rem;
	display: flex;
	gap: 1rem;
	justify-content: flex-end;
}

/* Mobile navbar styles */
.navbar-burger {
	display: none;
}

@media screen and (max-width: 1023px) {
	.navbar-burger {
		display: block;
	}

	.navbar-menu {
		display: none;
	}

	.navbar-menu.is-active {
		display: block;
		padding-bottom: 5rem;
	}
}

/* ═══ Mobile Player ═══ */

/* Hide desktop player on mobile, show mobile mini-bar instead */
@media screen and (max-width: 768px) {
	.audio-player {
		display: none !important;
	}

	.content {
		padding-bottom: 80px !important;
	}
}

/* Hide mini-player bar on desktop (fullscreen player is available everywhere) */
@media screen and (min-width: 769px) {
	.mobile-mini-player {
		display: none !important;
	}
}

/* Mobile Mini Player Bar */
.mobile-mini-player {
	position: fixed;
	bottom: 0;
	left: 0;
	right: 0;
	height: 64px;
	background: linear-gradient(135deg, var(--clr-surface-elevated) 0%, var(--clr-surface-higher) 100%);
	border-top: 2px solid var(--clr-primary);
	display: flex;
	align-items: center;
	padding: 0 1rem;
	gap: 0.75rem;
	z-index: 20;
	cursor: pointer;
	user-select: none;
	touch-action: none;
}

.mobile-mini-progress {
	position: absolute;
	top: 0;
	left: 0;
	height: 2px;
	background: var(--clr-primary);
	transition: width 0.5s linear;
}

.mobile-mini-cover {
	width: 44px;
	height: 44px;
	flex-shrink: 0;
	margin: 0;
}

.mobile-mini-cover img {
	width: 100%;
	height: 100%;
	object-fit: cover;
	border-radius: 4px;
}

.mobile-mini-cover-placeholder {
	width: 100%;
	height: 100%;
	background: var(--clr-surface-elevated);
	border-radius: 4px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.5rem;
	color: var(--clr-text-muted);
}

.mobile-mini-info {
	flex: 1;
	min-width: 0;
	overflow: hidden;
}

.mobile-mini-title {
	color: white;
	font-weight: 600;
	font-size: 0.9rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	margin: 0;
	line-height: 1.3;
}

.mobile-mini-artist {
	color: var(--clr-text-secondary);
	font-size: 0.8rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	margin: 0;
	line-height: 1.3;
}

.mobile-mini-play {
	width: 40px;
	height: 40px;
	border-radius: 50%;
	border: none;
	background: var(--clr-primary);
	color: white;
	font-size: 1rem;
	cursor: pointer;
	flex-shrink: 0;
	display: flex;
	align-items: center;
	justify-content: center;
}

/* Mobile Full-Screen Player */
.mobile-fullscreen-player {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	z-index: 100;
	background: var(--clr-surface-base);
	display: flex;
	flex-direction: column;
	padding: 0 1rem;
	padding-bottom: env(safe-area-inset-bottom, 1rem);
	overflow-y: auto;
	touch-action: none;
}

.mobile-player-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 0.5rem 0;
	flex-shrink: 0;
}

.mobile-player-collapse {
	width: 48px;
	height: 48px;
	border: none;
	background: transparent;
	color: var(--clr-text-secondary);
	font-size: 1.25rem;
	cursor: pointer;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
}

.mobile-player-collapse:active {
	background: var(--clr-surface-elevated);
}

.mobile-player-header-title {
	color: var(--clr-text-secondary);
	font-size: 0.85rem;
	font-weight: 600;
	text-transform: uppercase;
	letter-spacing: 0.05em;
}

.mobile-player-menu-wrapper {
	position: relative;
	width: 48px;
	display: flex;
	justify-content: center;
}

.mobile-player-menu {
	position: absolute;
	top: 100%;
	right: 0;
	background: var(--clr-surface-elevated);
	border: 1px solid var(--clr-surface-higher);
	border-radius: 8px;
	padding: 0.25rem 0;
	min-width: 180px;
	z-index: 110;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.mobile-player-menu-item {
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

.mobile-player-menu-item:hover,
.mobile-player-menu-item:active {
	background: var(--clr-surface-higher);
}

.mobile-player-cover {
	flex: 1;
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 0.5rem 1.5rem;
	min-height: 0;
	position: relative;
	overflow: hidden;
}

/* Album art animation layers */
.album-art-layer {
	position: absolute;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	display: flex;
	align-items: center;
	justify-content: center;
}

.album-art-layer:only-child {
	position: relative;
}

.album-art-exiting {
	z-index: 2;
}

.fly-out-left {
	animation: flyOutLeft 300ms ease-in forwards;
}

.fly-out-right {
	animation: flyOutRight 300ms ease-in forwards;
}

.fly-in-from-right {
	animation: flyInFromRight 300ms ease-out forwards;
}

.fly-in-from-left {
	animation: flyInFromLeft 300ms ease-out forwards;
}

@keyframes flyOutLeft {
	from {
		transform: translateX(0);
		opacity: 1;
	}
	to {
		transform: translateX(-120%);
		opacity: 0;
	}
}

@keyframes flyOutRight {
	from {
		transform: translateX(0);
		opacity: 1;
	}
	to {
		transform: translateX(120%);
		opacity: 0;
	}
}

@keyframes flyInFromRight {
	from {
		transform: translateX(120%);
		opacity: 0;
	}
	to {
		transform: translateX(0);
		opacity: 1;
	}
}

@keyframes flyInFromLeft {
	from {
		transform: translateX(-120%);
		opacity: 0;
	}
	to {
		transform: translateX(0);
		opacity: 1;
	}
}

.mobile-player-cover-image {
	width: 100%;
	max-width: 360px;
	aspect-ratio: 1;
	object-fit: cover;
	border-radius: 8px;
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}

.mobile-player-cover-placeholder {
	width: 80vw;
	max-width: 360px;
	aspect-ratio: 1;
	background: var(--clr-surface-elevated);
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 4rem;
	color: var(--clr-text-muted);
}

.mobile-player-info {
	text-align: center;
	padding: 0.75rem 1.5rem;
	flex-shrink: 0;
}

.mobile-player-title {
	color: white;
	font-size: 1.25rem;
	font-weight: 700;
	margin: 0 0 0.25rem 0;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.mobile-player-artist {
	color: var(--clr-text-secondary);
	font-size: 1rem;
	margin: 0;
}

.mobile-player-seek {
	padding: 0 1.5rem;
	flex-shrink: 0;
}

.mobile-seek-bar {
	width: 100%;
	height: 6px;
	outline: none;
	border-radius: 3px;
	appearance: none;
	cursor: pointer;
	background: linear-gradient(
		to right,
		var(--clr-primary) 0%,
		var(--clr-primary) var(--progress-percent, 0%),
		var(--clr-surface-higher) var(--progress-percent, 0%),
		var(--clr-surface-higher) 100%
	);
}

.mobile-seek-bar::-webkit-slider-thumb {
	appearance: none;
	width: 16px;
	height: 16px;
	border-radius: 50%;
	background: var(--clr-primary);
	cursor: pointer;
}

.mobile-seek-bar::-moz-range-thumb {
	width: 16px;
	height: 16px;
	border-radius: 50%;
	background: var(--clr-primary);
	cursor: pointer;
	border: none;
}

.mobile-player-times {
	display: flex;
	justify-content: space-between;
	font-size: 0.8rem;
	color: var(--clr-text-secondary);
	margin-top: 0.5rem;
}

.mobile-player-controls {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 1.5rem;
	padding: 1.5rem 0;
	flex-shrink: 0;
}

.mobile-control-btn {
	width: 48px;
	height: 48px;
	border: none;
	background: transparent;
	color: white;
	font-size: 1.25rem;
	cursor: pointer;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	transition: background-color 0.15s ease;
}

.mobile-control-btn:active {
	background: var(--clr-surface-elevated);
}

.mobile-control-btn.active {
	color: var(--clr-warning);
}

.mobile-play-btn {
	width: 64px;
	height: 64px;
	background: var(--clr-primary) !important;
	font-size: 1.5rem;
}

.mobile-play-btn:active {
	opacity: 0.8;
}

.mobile-player-volume {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 1rem;
	padding: 0 2rem 2rem;
	flex-shrink: 0;
}

.mobile-volume-icon {
	color: var(--clr-text-secondary);
	width: 20px;
	cursor: pointer;
}

.mobile-volume-slider {
	flex: 1;
	max-width: 240px;
	height: 4px;
	outline: none;
	border-radius: 2px;
	appearance: none;
	cursor: pointer;
	background: linear-gradient(
		to right,
		var(--clr-primary) 0%,
		var(--clr-primary) var(--volume-percent, 50%),
		var(--clr-surface-higher) var(--volume-percent, 50%),
		var(--clr-surface-higher) 100%
	);
}

.mobile-volume-slider::-webkit-slider-thumb {
	appearance: none;
	width: 14px;
	height: 14px;
	border-radius: 50%;
	background: var(--clr-primary);
	cursor: pointer;
}

.mobile-volume-slider::-moz-range-thumb {
	width: 14px;
	height: 14px;
	border-radius: 50%;
	background: var(--clr-primary);
	cursor: pointer;
	border: none;
}

/* Slide-up transition for mobile fullscreen player */
.mobile-player-slide-enter-active,
.mobile-player-slide-leave-active {
	transition: transform 0.3s ease;
}

.mobile-player-slide-enter-from,
.mobile-player-slide-leave-to {
	transform: translateY(100%);
}

/* Remote indicator dropdown — always hidden unless clicked */
.navbar-brand .remote-indicator .navbar-dropdown {
	display: none;
}

.navbar-brand .remote-indicator.is-active .navbar-dropdown {
	display: block;
}

@media screen and (max-width: 1023px) {
	.navbar-brand .remote-indicator .navbar-dropdown {
		position: absolute;
		left: 0;
		top: 100%;
		min-width: 220px;
		background-color: var(--clr-surface-elevated);
		border: 1px solid var(--clr-surface-higher);
		border-radius: 0 0 4px 4px;
		box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3);
		padding: 0.5rem 0;
	}
}

/* Player cover placeholder styles */
.player-cover-placeholder {
	width: 100%;
	height: 100%;
	background: var(--clr-surface-elevated);
	border-radius: 4px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.5rem;
	color: var(--clr-text-muted);
}

.clickable-cover {
	cursor: pointer;
	transition: opacity 0.2s ease;
}

.clickable-cover:hover {
	opacity: 0.8;
}
</style>
