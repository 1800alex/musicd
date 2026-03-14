import { ref, watch, onBeforeUnmount } from "vue";
import type { Ref } from "vue";
import type PlayerService from "~/services/player.service";
import backendService from "~/services/backend.service";
import useAppState from "~/stores/appState";
import { useBackendURL } from "~/composables/useBackendURL";

interface WSMessage {
	type: string;
	[key: string]: any;
}

export function useRemoteSync(player: PlayerService, appState: any, audioEl: Ref<HTMLAudioElement | null>) {
	const sessionName = ref<string>("");
	const sessionId = ref<string>("");
	const controllerCount = ref<number>(0);
	const enabled = ref<boolean>("true" === localStorage.getItem("remoteSyncEnabled"));

	let ws: WebSocket | null = null;
	let reconnectTimeout: NodeJS.Timeout | null = null;
	let reconnectDelay = 1000; // start at 1s
	const maxReconnectDelay = 30000; // max 30s
	let intentionalClose = false;

	// State debouncing
	let stateDebounceTimer: NodeJS.Timeout | null = null;
	let lastSentState: any = null;

	const connectWebSocket = () => {
		if (!enabled.value) {
			return;
		}
		intentionalClose = false;

		try {
			const { getWSURL } = useBackendURL();
			const wsUrl = getWSURL("/api/ws/player");
			ws = new WebSocket(wsUrl);

			ws.onopen = () => {
				console.log("Player WebSocket connected");
				reconnectDelay = 1000; // Reset delay on successful connection

				// Send register message with stored session_id, hostname, and custom name
				const storedSessionId = localStorage.getItem("remoteSyncSessionId") || "";
				const storedSessionName = localStorage.getItem("remoteSyncSessionName") || "";
				const registerMsg: WSMessage = {
					type: "register",
					session_id: storedSessionId,
					client_hostname: window.location.hostname,
					session_name: storedSessionName
				};
				ws?.send(JSON.stringify(registerMsg));
			};

			ws.onmessage = (event) => {
				try {
					const msg: WSMessage = JSON.parse(event.data);

					switch (msg.type) {
						case "session_ack":
							sessionId.value = msg.session_id;
							sessionName.value = msg.session_name;
							localStorage.setItem("remoteSyncSessionId", msg.session_id);
							console.log(`Session registered: ${msg.session_name} (${msg.session_id})`);
							break;

						case "command":
							handleCommand(msg);
							break;

						case "controllers_update":
							controllerCount.value = msg.count || 0;
							break;
					}
				} catch (err) {
					console.error("Error parsing WebSocket message:", err);
				}
			};

			ws.onerror = (error) => {
				console.error("WebSocket error:", error);
			};

			ws.onclose = () => {
				if (intentionalClose) {
					return;
				}
				console.log("WebSocket disconnected, will reconnect...");
				scheduleReconnect();
			};
		} catch (err) {
			console.error("Error creating WebSocket:", err);
			scheduleReconnect();
		}
	};

	const disconnectWebSocket = () => {
		intentionalClose = true;
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
			reconnectTimeout = null;
		}
		if (stateDebounceTimer) {
			clearTimeout(stateDebounceTimer);
			stateDebounceTimer = null;
		}
		if (ws) {
			ws.close();
			ws = null;
		}
		sessionName.value = "";
		sessionId.value = "";
		controllerCount.value = 0;
		lastSentState = null;
	};

	const scheduleReconnect = () => {
		if (!enabled.value) {
			return;
		}
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
		}
		reconnectTimeout = setTimeout(() => {
			console.log(`Reconnecting WebSocket (${reconnectDelay}ms delay)...`);
			connectWebSocket();
			reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
		}, reconnectDelay);
	};

	const sendStateUpdate = (force = false) => {
		if (!ws || ws.readyState !== WebSocket.OPEN) {
			return;
		}

		const currentState = {
			is_playing: appState.IsPlaying,
			current_track: appState.CurrentTrack,
			current_time: audioEl.value?.currentTime || 0,
			duration: audioEl.value?.duration || 0,
			volume: appState.Volume,
			muted: appState.Muted,
			shuffle: appState.Shuffle,
			repeat_mode: appState.RepeatMode,
			queue: appState.Queue,
			temporary_queue: appState.TemporaryQueue,
			current_playlist: appState.CurrentPlaylist
		};

		// Check if state actually changed
		if (!force && JSON.stringify(currentState) === JSON.stringify(lastSentState)) {
			return;
		}

		lastSentState = currentState;
		const msg: WSMessage = {
			type: "state",
			...currentState
		};

		try {
			ws.send(JSON.stringify(msg));
		} catch (err) {
			console.error("Error sending state update:", err);
		}
	};

	const handleCommand = async (msg: WSMessage) => {
		const action = msg.action as string;
		const value = msg.value;

		try {
			switch (action) {
				case "play":
					appState.SetIsPlaying(true);
					break;
				case "pause":
					appState.SetIsPlaying(false);
					break;
				case "toggle_play":
					appState.SetIsPlaying(!appState.IsPlaying);
					break;
				case "next":
					player.NextTrack();
					break;
				case "previous":
					player.PreviousTrack();
					break;
				case "seek":
					if (audioEl.value) {
						audioEl.value.currentTime = value;
					}
					break;
				case "volume":
					player.VolumeChanged(value);
					break;
				case "toggle_mute":
					player.ToggleMute();
					break;
				case "set_shuffle":
					player.SetShuffle(value);
					break;
				case "set_repeat":
					appState.SetRepeatMode(value);
					break;
				case "play_track": {
					const trackId = value?.id || value;
					const track = await backendService.FetchTrackById(trackId);
					if (track) {
						player.PlayTrackFromAllTracks(track, value?.search);
					}
					break;
				}
				case "play_playlist": {
					const playlistId = value?.id || value;
					const playlist = await backendService.FetchPlaylist(playlistId);
					if (playlist) {
						player.PlayPlaylist(playlist);
					}
					break;
				}
				case "play_playlist_track": {
					const trackId = value?.id || value;
					const playlistId = value?.playlist_id;
					const track = await backendService.FetchTrackById(trackId);
					const playlist = await backendService.FetchPlaylist(playlistId);
					if (track && playlist) {
						player.PlayPlaylistTrack(track, playlist, value?.search);
					}
					break;
				}
				case "play_album": {
					const albumId = value?.id || value;
					player.PlayAlbum(albumId);
					break;
				}
				case "play_album_track": {
					const trackId = value?.id || value;
					const albumId = value?.album_id;
					const track = await backendService.FetchTrackById(trackId);
					if (track) {
						player.PlayAlbumTrack(track, albumId, value?.search);
					}
					break;
				}
				case "play_artist": {
					const artistId = value?.id || value;
					player.PlayArtist(artistId);
					break;
				}
				case "play_artist_track": {
					const trackId = value?.id || value;
					const artistId = value?.artist_id;
					const track = await backendService.FetchTrackById(trackId);
					if (track) {
						player.PlayArtistTrack(track, artistId, value?.search);
					}
					break;
				}
				case "queue_add": {
					const queueTrackId = value?.id || value;
					const queueTrack = await backendService.FetchTrackById(queueTrackId);
					if (queueTrack) {
						player.AddTrackToQueue(queueTrack);
					}
					break;
				}
				case "queue_clear":
					appState.SetQueue([]);
					appState.SetTemporaryQueue([]);
					break;
				case "remove_track_from_playlist": {
					// Handle remote playlist track removal
					const trackId = value?.id || value;
					const playlistId = value?.playlist_id;
					if (trackId && playlistId) {
						await backendService.RemoveTrackFromPlaylistById(trackId, playlistId);
					}
					break;
				}
				default:
					console.warn("Unknown command:", action);
			}
		} catch (err) {
			console.error("Error handling command:", err);
		}
	};

	const renameSession = (newName: string) => {
		sessionName.value = newName;
		localStorage.setItem("remoteSyncSessionName", newName);
		if (ws && ws.readyState === WebSocket.OPEN) {
			ws.send(JSON.stringify({ type: "rename", session_name: newName }));
		}
	};

	const enable = () => {
		enabled.value = true;
		localStorage.setItem("remoteSyncEnabled", "true");
		reconnectDelay = 1000;
		connectWebSocket();
		setupAudioTimeTracking();
	};

	const disable = () => {
		enabled.value = false;
		localStorage.setItem("remoteSyncEnabled", "false");
		disconnectWebSocket();
	};

	// Watch for state changes and debounce
	watch(
		[
			() => appState.IsPlaying,
			() => appState.CurrentTrack,
			() => appState.Volume,
			() => appState.Shuffle,
			() => appState.RepeatMode,
			() => appState.Queue,
			() => appState.TemporaryQueue,
			() => appState.CurrentPlaylist
		],
		() => {
			if (stateDebounceTimer) {
				clearTimeout(stateDebounceTimer);
			}
			// Increased debounce to 500ms to reduce update frequency during playback
			stateDebounceTimer = setTimeout(() => {
				sendStateUpdate();
			}, 500);
		}
	);

	// Watch audio element currentTime
	let timeUpdateHandler: (() => void) | null = null;
	let lastTimeSent = 0;
	const setupAudioTimeTracking = () => {
		if (!audioEl.value) {
			return;
		}

		// Remove old listener if it exists (prevent duplicate listeners)
		if (timeUpdateHandler && audioEl.value) {
			audioEl.value.removeEventListener("timeupdate", timeUpdateHandler);
		}

		// Create and store the handler so we can remove it later
		timeUpdateHandler = () => {
			// Throttle: only update every 5 seconds during playback to reduce CPU usage
			const now = Date.now();
			if (now - lastTimeSent < 5000) {
				return;
			}
			lastTimeSent = now;
			sendStateUpdate();
		};

		audioEl.value.addEventListener("timeupdate", timeUpdateHandler);
	};

	onBeforeUnmount(() => {
		// Remove timeupdate listener
		if (timeUpdateHandler && audioEl.value) {
			audioEl.value.removeEventListener("timeupdate", timeUpdateHandler);
		}
		disconnectWebSocket();
	});

	// Connect on mount only if enabled
	if (enabled.value) {
		connectWebSocket();
		setupAudioTimeTracking();
	}

	return {
		sessionName,
		sessionId,
		controllerCount,
		enabled,
		renameSession,
		enable,
		disable
	};
}
