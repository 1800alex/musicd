interface WSMessage {
	type: string;
	[key: string]: any;
}

interface PlayerState {
	is_playing: boolean;
	current_track: any | null;
	current_time: number;
	duration: number;
	volume: number;
	muted: boolean;
	shuffle: boolean;
	repeat_mode: string;
	queue: any[];
	temporary_queue: any[];
	current_playlist: any | null;
}

export default class RemoteControlService {
	private ws: WebSocket | null = null;
	private sessionId: string = "";
	private reconnectDelay = 1000;
	private maxReconnectDelay = 30000;
	private reconnectTimeout: NodeJS.Timeout | null = null;

	// Throttle state for high-frequency commands (volume, seek)
	private throttleTimers: Map<string, NodeJS.Timeout> = new Map();
	private throttlePending: Map<string, any> = new Map();
	private static readonly THROTTLE_MS = 100;

	public onStateUpdate: (state: PlayerState) => void = () => {};
	public onConnected: () => void = () => {};
	public onDisconnected: () => void = () => {};
	public onError: (error: string) => void = () => {};

	constructor(serverUrl: string, sessionId: string) {
		this.sessionId = sessionId;
		this.connect(serverUrl);
	}

	private connect(serverUrl: string) {
		try {
			const protocol = serverUrl.includes("https") ? "wss:" : "ws:";
			const url = `${protocol}//${serverUrl}/api/ws/control/${this.sessionId}`;

			this.ws = new WebSocket(url);

			this.ws.onopen = () => {
				console.log("Remote control WebSocket connected");
				this.reconnectDelay = 1000;
				this.onConnected();
			};

			this.ws.onmessage = (event) => {
				try {
					const msg: WSMessage = JSON.parse(event.data);

					if (msg.type === "state") {
						const state: PlayerState = {
							is_playing: msg.is_playing,
							current_track: msg.current_track,
							current_time: msg.current_time,
							duration: msg.duration,
							volume: msg.volume,
							muted: msg.muted,
							shuffle: msg.shuffle,
							repeat_mode: msg.repeat_mode,
							queue: msg.queue || [],
							temporary_queue: msg.temporary_queue || [],
							current_playlist: msg.current_playlist || null
						};
						this.onStateUpdate(state);
					}
				} catch (err) {
					console.error("Error parsing message:", err);
				}
			};

			this.ws.onerror = (error) => {
				console.error("WebSocket error:", error);
				this.onError("Connection error");
			};

			this.ws.onclose = () => {
				console.log("WebSocket disconnected");
				this.onDisconnected();
				this.scheduleReconnect(serverUrl);
			};
		} catch (err) {
			console.error("Error creating WebSocket:", err);
			this.onError(String(err));
			this.scheduleReconnect(serverUrl);
		}
	}

	private scheduleReconnect(serverUrl: string) {
		if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout);
		this.reconnectTimeout = setTimeout(() => {
			console.log(`Reconnecting... (${this.reconnectDelay}ms)`);
			this.connect(serverUrl);
			this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
		}, this.reconnectDelay);
	}

	public sendCommand(action: string, value?: any) {
		if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
			console.warn("WebSocket not connected");
			return;
		}

		const msg: WSMessage = {
			type: "command",
			action
		};

		if (value !== undefined) {
			msg.value = value;
		}

		try {
			this.ws.send(JSON.stringify(msg));
		} catch (err) {
			console.error("Error sending command:", err);
		}
	}

	/**
	 * Throttled send: ensures at most one message per THROTTLE_MS for the given key.
	 * The latest value always wins — if rapid calls arrive, intermediate values are dropped
	 * and the final value is sent when the throttle window expires.
	 */
	private sendThrottled(action: string, value: any) {
		this.throttlePending.set(action, value);

		if (this.throttleTimers.has(action)) {
			return; // timer already running, latest value will be picked up
		}

		// Send immediately for the first call
		this.sendCommand(action, value);
		this.throttlePending.delete(action);

		// Set a timer to flush the latest pending value after the throttle window
		const timer = setTimeout(() => {
			this.throttleTimers.delete(action);
			if (this.throttlePending.has(action)) {
				const pending = this.throttlePending.get(action);
				this.throttlePending.delete(action);
				this.sendCommand(action, pending);
			}
		}, RemoteControlService.THROTTLE_MS);
		this.throttleTimers.set(action, timer);
	}

	public play() {
		this.sendCommand("play");
	}

	public pause() {
		this.sendCommand("pause");
	}

	public togglePlay() {
		this.sendCommand("toggle_play");
	}

	public next() {
		this.sendCommand("next");
	}

	public previous() {
		this.sendCommand("previous");
	}

	public seek(position: number) {
		this.sendThrottled("seek", position);
	}

	public setVolume(volume: number) {
		this.sendThrottled("volume", volume);
	}

	public toggleMute() {
		this.sendCommand("toggle_mute");
	}

	public setShuffle(enabled: boolean) {
		this.sendCommand("set_shuffle", enabled);
	}

	public setRepeat(mode: string) {
		this.sendCommand("set_repeat", mode);
	}

	public playTrack(trackId: string, search?: string) {
		this.sendCommand("play_track", { id: trackId, search });
	}

	public playPlaylist(playlistId: string, playlistName?: string) {
		this.sendCommand("play_playlist", { id: playlistId, name: playlistName });
	}

	public playPlaylistTrack(trackId: string, playlistId: string, playlistName?: string, search?: string) {
		this.sendCommand("play_playlist_track", {
			id: trackId,
			playlist_id: playlistId,
			playlist_name: playlistName,
			search
		});
	}

	public playAlbum(albumId: string) {
		this.sendCommand("play_album", { id: albumId });
	}

	public playAlbumTrack(trackId: string, albumId: string, search?: string) {
		this.sendCommand("play_album_track", { id: trackId, album_id: albumId, search });
	}

	public playArtist(artistId: string) {
		this.sendCommand("play_artist", { id: artistId });
	}

	public playArtistTrack(trackId: string, artistId: string, search?: string) {
		this.sendCommand("play_artist_track", { id: trackId, artist_id: artistId, search });
	}

	public queueAdd(trackId: string) {
		this.sendCommand("queue_add", { id: trackId });
	}

	public queueClear() {
		this.sendCommand("queue_clear");
	}

	public disconnect() {
		if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout);
		for (const timer of this.throttleTimers.values()) clearTimeout(timer);
		this.throttleTimers.clear();
		this.throttlePending.clear();
		if (this.ws) {
			this.ws.close();
			this.ws = null;
		}
	}
}
