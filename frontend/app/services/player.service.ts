import type { Track, Playlist } from "~/types";
import useAppState, { RepeatMode } from "~/stores/appState";
import type { RepeatModeType } from "~/stores/appState";
import backendService from "~/services/backend.service";
import MediaSessionService from "~/services/mediaSession.service";
import { resolveBaseURL } from "~/services/http.service";
import { isNativeOrElectron } from "~/utils/platform";

/** Fisher-Yates shuffle — mutates the array in place. */
function shuffleArray<T>(arr: T[]): T[] {
	for (let i = arr.length - 1; i > 0; i--) {
		const j = Math.floor(Math.random() * (i + 1));
		[arr[i], arr[j]] = [arr[j]!, arr[i]!];
	}
	return arr;
}

// TODO - When things change we need to save them to local storage and restore them on startup, but we need a way to identify who is the "parent" so that only the top level default.vue does this

class PlayerService {
	private appState: ReturnType<typeof useAppState>;
	private mediaSession: MediaSessionService;
	private topLevel = false;

	constructor(state?: ReturnType<typeof useAppState>) {
		if (!state) {
			state = useAppState();
		}
		this.appState = state;
		this.mediaSession = new MediaSessionService();
	}

	SetTopLevel(val: boolean) {
		this.topLevel = val;
		if (val) {
			this.setupMediaSessionHandlers();
		}
	}

	/** Find the index of the current track in the queue, handling duplicates via playlist_position_id */
	private findCurrentTrackIndex(): number {
		const currentTrack = this.appState.CurrentTrack;
		if (!currentTrack) return -1;

		// If current track has a playlist_position_id, use that for comparison (handles duplicates)
		if (currentTrack.playlist_position_id) {
			return this.appState.TemporaryQueue.findIndex(
				(t) => t.playlist_position_id === currentTrack.playlist_position_id
			);
		}

		// Otherwise, fall back to using track id
		return this.appState.TemporaryQueue.findIndex((t) => t.id === currentTrack.id);
	}

	UpdateMediaSessionArtwork(artwork: { src?: string; type?: string; sizes?: string }) {
		if (!this.topLevel) {
			return; // Only update media session for top-level player
		}

		this.mediaSession.updateArtwork(artwork);
	}

	private setupMediaSessionHandlers() {
		if (!this.topLevel) {
			return; // Only set up media session for top-level player
		}

		this.mediaSession.setActionHandlers({
			play: () => {
				this.appState.SetIsPlaying(true);
			},
			pause: () => {
				this.appState.SetIsPlaying(false);
			},
			previoustrack: () => {
				this.PreviousTrack();
			},
			nexttrack: () => {
				this.NextTrack();
			},
			seekbackward: (details) => {
				const currentTime = this.appState.AudioElement?.currentTime || 0;
				const seekOffset = details.seekOffset || 10;
				const newTime = Math.max(0, currentTime - seekOffset);
				if (this.appState.AudioElement) {
					this.appState.AudioElement.currentTime = newTime;
				}
			},
			seekforward: (details) => {
				const currentTime = this.appState.AudioElement?.currentTime || 0;
				const duration = this.appState.AudioElement?.duration || 0;
				const seekOffset = details.seekOffset || 10;
				const newTime = Math.min(duration, currentTime + seekOffset);
				if (this.appState.AudioElement) {
					this.appState.AudioElement.currentTime = newTime;
				}
			},
			seekto: (details) => {
				const seekTime = details.seekTime || 0;
				const duration = this.appState.AudioElement?.duration || 0;
				const newTime = Math.min(duration, Math.max(0, seekTime));
				if (this.appState.AudioElement) {
					this.appState.AudioElement.currentTime = newTime;
				}
			}
		});
	}

	LoadFromLocalStorage() {
		if (!this.topLevel) {
			console.warn("Not loading from local storage as not top level player");
			return;
		}

		const loaded = localStorage.getItem("musicPlayerLoaded");
		if ("true" === loaded) {
			const volume = localStorage.getItem("musicPlayerVolume");
			if (volume) {
				this.appState.SetVolume(parseInt(volume, 10) || 50);
			}

			const muted = localStorage.getItem("musicPlayerMuted");
			if (muted) {
				this.appState.SetMuted("true" === muted);
			}

			const shuffle = localStorage.getItem("musicPlayerShuffle");
			if (shuffle) {
				this.appState.SetShuffle("true" === shuffle);
			}

			const repeatMode = localStorage.getItem("musicPlayerRepeatMode");
			if (
				repeatMode &&
				(repeatMode === RepeatMode.Off || repeatMode === RepeatMode.One || repeatMode === RepeatMode.All)
			) {
				this.appState.SetRepeatMode(<RepeatModeType>repeatMode);
			}

			const volumeBeforeMute = localStorage.getItem("musicPlayerVolumeBeforeMute");
			if (volumeBeforeMute) {
				this.appState.SetVolumeBeforeMute(parseInt(volumeBeforeMute, 10) || 50);
			}

			const pageSize = localStorage.getItem("musicPlayerPageSize");
			if (pageSize) {
				this.appState.SetPageSize(parseInt(pageSize, 10) || 25);
			}

			this.appState.SetLoaded(true);
			console.log("Loaded player state from local storage");

			this.SetAudioPlayerVolume();
		}
	}

	SaveToLocalStorage() {
		if (!this.topLevel) {
			return;
		}

		localStorage.setItem("musicPlayerLoaded", "true");
		localStorage.setItem("musicPlayerVolume", this.appState.Volume.toString());
		localStorage.setItem("musicPlayerMuted", this.appState.Muted ? "true" : "false");
		localStorage.setItem("musicPlayerShuffle", this.appState.Shuffle ? "true" : "false");
		localStorage.setItem("musicPlayerRepeatMode", this.appState.RepeatMode);
		localStorage.setItem("musicPlayerVolumeBeforeMute", this.appState.VolumeBeforeMute.toString());
		localStorage.setItem("musicPlayerPageSize", this.appState.PageSize.toString());
	}

	PlayStateChanged(playing: boolean) {
		// In remote mode, don't control local audio — state comes from remote
		if (this.appState.RemoteControl) {
			return;
		}

		if (!this.appState.AudioElement) {
			return;
		}

		if (playing && this.appState.CurrentTrack) {
			this.appState.AudioElement.play().catch((error) => {
				console.error("Error playing audio:", error);
			});
		} else {
			this.appState.AudioElement.pause();
		}
		this.appState.SetIsPlaying(playing);

		// Update media session playback state
		if (this.topLevel) {
			this.mediaSession.setPlaybackState(playing ? "playing" : "paused");
			if (this.appState.CurrentTrack) {
				this.mediaSession.updateFromTrack(this.appState.CurrentTrack).catch((error) => {
					console.error("Error updating media session metadata:", error);
				});
			}
		}

		this.SaveToLocalStorage();
	}

	VolumeChanged(volume: number) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.setVolume(volume);
			return;
		}

		if (!this.appState.AudioElement) {
			return;
		}

		this.appState.AudioElement.volume = volume / 100;
		if (this.appState.Muted && volume > 0) {
			this.appState.SetMuted(false);
		}
		this.appState.SetVolume(volume);
		this.SaveToLocalStorage();
	}

	HandleTrackEnd() {
		if (this.appState.RemoteControl) {
			return; // Parent handles track endings
		}

		if (this.appState.RepeatMode === RepeatMode.One) {
			if (this.appState.AudioElement) {
				this.appState.AudioElement.currentTime = 0;
				this.appState.AudioElement.play().catch((error) => {
					console.error("Error playing audio:", error);
				});
			}
		} else {
			this.NextTrack();
		}
	}

	SetCurrentTrack(track: Track) {
		console.log("Current track changed:", track?.file_path);

		this.SetAudioPlayerVolume();
		this.appState.SetCurrentTrack(track);

		// Update media session metadata
		if (this.appState.CurrentTrack && this.topLevel) {
			this.mediaSession.updateFromTrack(this.appState.CurrentTrack).catch((error) => {
				console.error("Error updating media session metadata:", error);
			});
		}

		if (track && this.appState.AudioElement) {
			const baseURL = isNativeOrElectron() ? resolveBaseURL().replace(/\/$/, "") : "";
			this.appState.AudioElement.src = `${baseURL}/api/music/${track.file_path}`;
			if (this.appState.IsPlaying) {
				this.appState.AudioElement.play().catch((error) => {
					console.error("Error playing audio:", error);
				});
			}
		} else {
			console.log("No track to play or audio player not initialized");
			this.appState.SetIsPlaying(false);
			if (this.appState.AudioElement) {
				this.appState.AudioElement.pause();
			}
		}
	}

	SetAudioPlayerVolume() {
		if (this.appState.AudioElement) {
			if (this.appState.Muted) {
				this.appState.AudioElement.volume = 0;
				this.appState.SetVolume(0);
			} else {
				this.appState.AudioElement.volume = this.appState.Volume / 100;
			}
		}
	}

	NextTrack() {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.next();
			return;
		}

		// User-queued tracks (explicitly added via "Add to Queue") play before the current context
		if (this.appState.Queue.length > 0) {
			const nextTrack = this.appState.Queue[0]!;
			this.appState.SetQueue(this.appState.Queue.slice(1));
			// Splice into TemporaryQueue after the current position so Previous still works
			const currentIndex = this.findCurrentTrackIndex();
			const newTQ = [...this.appState.TemporaryQueue];
			newTQ.splice(currentIndex >= 0 ? currentIndex + 1 : newTQ.length, 0, nextTrack);
			this.appState.SetTemporaryQueue(newTQ);
			this.SetCurrentTrack(nextTrack);
			return;
		}

		console.log(`Next track requested from queue of ${this.appState.TemporaryQueue.length} tracks`);

		if (this.appState.TemporaryQueue.length > 0) {
			if (this.appState.RepeatMode === RepeatMode.One) {
				if (this.appState.AudioElement) {
					this.appState.AudioElement.currentTime = 0;
					this.appState.AudioElement.play().catch((error) => {
						console.error("Error playing audio:", error);
					});
				}
				return;
			}

			if (this.appState.RepeatMode === RepeatMode.All) {
				const currentIndex = this.findCurrentTrackIndex();
				if (currentIndex >= 0) {
					const nextIndex = (currentIndex + 1) % this.appState.TemporaryQueue.length;
					this.SetCurrentTrack(this.appState.TemporaryQueue[nextIndex]);
					return;
				}
			}

			const currentIndex = this.findCurrentTrackIndex();
			if (currentIndex >= 0 && currentIndex < this.appState.TemporaryQueue.length - 1) {
				this.SetCurrentTrack(this.appState.TemporaryQueue[currentIndex + 1]);
				return;
			}
		}
	}

	PreviousTrack() {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.previous();
			return;
		}

		if (this.appState.TemporaryQueue.length > 0) {
			if (this.appState.RepeatMode === RepeatMode.One) {
				if (this.appState.AudioElement) {
					this.appState.AudioElement.currentTime = 0;
					this.appState.AudioElement.play().catch((error) => {
						console.error("Error playing audio:", error);
					});
				}
				return;
			}

			if (this.appState.RepeatMode === RepeatMode.All) {
				const currentIndex = this.findCurrentTrackIndex();
				if (currentIndex > 0) {
					this.SetCurrentTrack(this.appState.TemporaryQueue[currentIndex - 1]);
				} else {
					this.SetCurrentTrack(this.appState.TemporaryQueue[this.appState.TemporaryQueue.length - 1]);
				}
				return;
			}

			const currentIndex = this.findCurrentTrackIndex();
			if (currentIndex > 0) {
				this.SetCurrentTrack(this.appState.TemporaryQueue[currentIndex - 1]);
				return;
			}
		}
	}

	SetShuffle(val: boolean) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.setShuffle(val);
			return;
		}

		this.appState.SetShuffle(val);

		if (this.appState.TemporaryQueue.length > 1 && this.appState.CurrentTrack) {
			if (val) {
				// Turning shuffle ON: shuffle tracks after the current track.
				const currentIndex = this.findCurrentTrackIndex();
				if (currentIndex >= 0) {
					const before = this.appState.TemporaryQueue.slice(0, currentIndex + 1);
					const after = this.appState.TemporaryQueue.slice(currentIndex + 1);
					shuffleArray(after);
					this.appState.SetTemporaryQueue([...before, ...after]);
				}
			} else if (this.appState.OriginalQueue.length > 0) {
				// Turning shuffle OFF: restore original order from current position.
				const currentId = this.appState.CurrentTrack.id;
				const originalIndex = this.appState.OriginalQueue.findIndex((t) => t.id === currentId);
				if (originalIndex >= 0) {
					this.appState.SetTemporaryQueue([...this.appState.OriginalQueue]);
				}
			}
		}

		this.SaveToLocalStorage();
	}

	SetPageSize(size: number) {
		this.appState.SetPageSize(size);
		this.SaveToLocalStorage();
	}

	UpdateVolume() {
		if (this.appState.RemoteControl) {
			this.appState.RemoteControl.setVolume(this.appState.Volume);
			return;
		}

		if (this.appState.AudioElement) {
			this.appState.AudioElement.volume = this.appState.Volume / 100;
			if (this.appState.Muted && this.appState.Volume > 0) {
				this.appState.Muted = false;
			}
		}

		this.SaveToLocalStorage();
	}

	CycleRepeatMode() {
		const rc = this.appState.RemoteControl;
		if (rc) {
			let nextMode: string;
			switch (this.appState.RepeatMode) {
				case RepeatMode.Off:
					nextMode = RepeatMode.All;
					break;
				case RepeatMode.All:
					nextMode = RepeatMode.One;
					break;
				default:
					nextMode = RepeatMode.Off;
					break;
			}
			rc.setRepeat(nextMode);
			return;
		}

		switch (this.appState.RepeatMode) {
			case RepeatMode.Off:
				this.appState.SetRepeatMode(<RepeatModeType>RepeatMode.All);
				break;
			case RepeatMode.All:
				this.appState.SetRepeatMode(<RepeatModeType>RepeatMode.One);
				break;
			default:
				this.appState.SetRepeatMode(<RepeatModeType>RepeatMode.Off);
				break;
		}
		this.SaveToLocalStorage();
	}

	TogglePlay() {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.togglePlay();
			return;
		}

		if (!this.appState.AudioElement || !this.appState.CurrentTrack) {
			return;
		}

		if (this.appState.IsPlaying) {
			this.appState.AudioElement.pause();
			this.appState.SetIsPlaying(false);
		} else {
			this.appState.AudioElement.play().catch((error) => {
				console.error("Error playing audio:", error);
			});
			this.appState.SetIsPlaying(true);

			// Update media session metadata
			if (this.appState.CurrentTrack && this.topLevel) {
				this.mediaSession.updateFromTrack(this.appState.CurrentTrack).catch((error) => {
					console.error("Error updating media session metadata:", error);
				});
			}
		}
		this.SaveToLocalStorage();
	}

	ToggleMute() {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.toggleMute();
			return;
		}

		if (this.appState.Muted) {
			this.appState.SetMuted(false);
			this.appState.SetVolume(this.appState.VolumeBeforeMute);
		} else {
			this.appState.SetMuted(true);
			this.appState.SetVolumeBeforeMute(this.appState.Volume);
			this.appState.SetVolume(0);
		}
		this.UpdateVolume();
	}

	ToggleShuffle() {
		this.SetShuffle(!this.appState.Shuffle);
	}

	ToggleRepeat() {
		this.CycleRepeatMode();
	}

	VolumeUp() {
		const newVolume = Math.min(100, this.appState.Volume + 5);
		this.VolumeChanged(newVolume);
	}

	VolumeDown() {
		const newVolume = Math.max(0, this.appState.Volume - 5);
		this.VolumeChanged(newVolume);
	}

	TogglePlayback() {
		this.TogglePlay();
	}

	UpdatePositionState() {
		if (!this.topLevel || !this.appState.AudioElement || !this.appState.CurrentTrack) {
			return;
		}

		const audioElement = this.appState.AudioElement;
		const duration = audioElement.duration || this.appState.CurrentTrack.duration || 0;
		const position = audioElement.currentTime || 0;
		const playbackRate = audioElement.playbackRate || 1.0;

		this.mediaSession.setPositionState(duration, playbackRate, position);
	}

	Stop() {
		if (!this.appState.IsPlaying) {
			return;
		}

		this.TogglePlay();
	}

	async AddTrackToQueue(track: Track) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.queueAdd(track.id);
			return;
		}

		this.appState.AddToQueue(track);
		console.log("Added to queue:", track.title);

		// If nothing is currently playing, auto-start the newly queued track
		if (!this.appState.CurrentTrack && !this.appState.IsPlaying) {
			this.appState.SetIsPlaying(true);
			this.SetCurrentTrack(track);
		}
	}

	async PlayTrackFromAllTracks(track: Track, search?: string) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.playTrack(track.id, search);
			return;
		}
		this.appState.SetQueue([]);

		// Set IsPlaying=true before SetCurrentTrack so that SetCurrentTrack sees IsPlaying=true
		// and calls play() synchronously, rather than relying on the async Vue watch firing later.
		this.appState.SetIsPlaying(true);
		this.appState.SetCurrentPlaylist(null);
		this.SetCurrentTrack(track);
		console.log("Playing track:", track.title);

		try {
			// Fetch full playlist tracks in case of search filter
			const response = await backendService.FetchTracks({
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play");
				return;
			}

			console.log(`Fetched ${response.data.length} tracks for queue context`);

			let startIndex = 0;
			// Find the index of the selected track
			for (let i = 0; i < response.data.length; i++) {
				const t = response.data[i];
				if (t.id === track.id) {
					startIndex = i;
					break;
				}
			}

			// Set the queue starting from the selected track
			if (RepeatMode.One === this.appState.RepeatMode) {
				// If repeating one, set queue to just the selected track
				this.appState.SetTemporaryQueue([track]);
				return;
			}

			if (RepeatMode.All === this.appState.RepeatMode) {
				// If repeating all, set queue to all tracks starting from selected track and looping back
				const newQueue = response.data.slice(startIndex).concat(response.data.slice(0, startIndex));
				this.appState.SetTemporaryQueue(newQueue);
				return;
			}

			this.appState.SetOriginalQueue(response.data);

			if (this.appState.Shuffle) {
				const queue = [...response.data];
				shuffleArray(queue);
				// Move selected track to front
				const idx = queue.findIndex((t) => t.id === track.id);
				if (idx > 0) {
					const [selected] = queue.splice(idx, 1);
					queue.unshift(selected!);
				}
				this.appState.SetTemporaryQueue(queue);
			} else {
				this.appState.SetTemporaryQueue(response.data.slice(startIndex));
			}

			console.log(`Added all tracks to temporary queue`);
		} catch (error) {
			console.error("Error fetching all tracks for playback:", error);
			return;
		}
	}

	async PlayPlaylist(playlist: Playlist, shuffle = false) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.playPlaylist(playlist.id, playlist.name);
			return;
		}
		this.appState.SetQueue([]);

		try {
			// Fetch full playlist tracks in case of search filter
			const response = await backendService.FetchPlaylistTracks(playlist.id, {});

			if (0 === response.data.length) {
				console.warn("No tracks available to play in the playlist");
				return;
			}

			if (true === shuffle) {
				this.appState.SetShuffle(true);
			}

			this.appState.SetOriginalQueue(response.data);

			if (this.appState.Shuffle) {
				const queue = [...response.data];
				shuffleArray(queue);
				this.appState.SetTemporaryQueue(queue);
			} else {
				this.appState.SetTemporaryQueue(response.data);
			}

			this.SetCurrentTrack(this.appState.TemporaryQueue[0]!);
			this.appState.SetCurrentPlaylist(playlist);
			this.appState.SetIsPlaying(true);
			console.log(`Playing playlist ${playlist.name}`);
		} catch (error) {
			console.error("Error fetching playlist for playback:", error);
			return;
		}
	}

	async PlayPlaylistTrack(track: Track, playlist: Playlist, search?: string) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.playPlaylistTrack(track.id, playlist.id, playlist.name, search, track.playlist_position_id);
			return;
		}
		this.appState.SetQueue([]);

		this.SetCurrentTrack(track);
		this.appState.SetCurrentPlaylist(playlist);
		this.appState.SetIsPlaying(true);
		console.log("Playing track:", track.title);

		try {
			// Fetch full playlist tracks in case of search filter
			const response = await backendService.FetchPlaylistTracks(playlist.id, {
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play in the playlist");
				return;
			}

			this.appState.SetOriginalQueue(response.data);

			if (this.appState.Shuffle) {
				const queue = [...response.data];
				shuffleArray(queue);
				// Move selected track to front
				const idx = queue.findIndex((t) => t.id === track.id);
				if (idx > 0) {
					const [selected] = queue.splice(idx, 1);
					queue.unshift(selected!);
				}
				this.appState.SetTemporaryQueue(queue);
			} else {
				this.appState.SetTemporaryQueue(response.data);
			}

			console.log(`Added all tracks from ${playlist.name} to temporary queue`);
		} catch (error) {
			console.error("Error fetching playlist for playback:", error);
			return;
		}
	}

	async AddPlaylistToQueue(playlist: Playlist) {
		try {
			// Fetch full playlist tracks in case of search filter
			const response = await backendService.FetchPlaylistTracks(playlist.id, {});

			if (0 === response.data.length) {
				console.warn("No tracks available to play in the playlist");
				return;
			}

			for (const track of response.data) {
				await this.AddTrackToQueue(track);
			}
			console.log(`Added all tracks from ${playlist.name} to queue`);
		} catch (error) {
			console.error("Error fetching playlist for playback:", error);
			return;
		}
	}

	async PlayAlbum(id: string, search?: string) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.playAlbum(id);
			return;
		}
		this.appState.SetQueue([]);

		try {
			// Fetch full album tracks in case of search filter
			const response = await backendService.FetchAlbumTracks(id, {
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play in the album");
				return;
			}

			this.appState.SetOriginalQueue(response.data);

			if (this.appState.Shuffle) {
				const queue = [...response.data];
				shuffleArray(queue);
				this.appState.SetTemporaryQueue(queue);
			} else {
				this.appState.SetTemporaryQueue(response.data);
			}

			this.SetCurrentTrack(this.appState.TemporaryQueue[0]!);
			this.appState.SetCurrentPlaylist(null);
			this.appState.SetIsPlaying(true);
			console.log(`Playing album ${id}`);
		} catch (error) {
			console.error("Error fetching album for playback:", error);
			return;
		}
	}

	async PlayAlbumTrack(track: Track, id: string, search?: string) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.playAlbumTrack(track.id, id, search);
			return;
		}
		this.appState.SetQueue([]);

		this.SetCurrentTrack(track);
		this.appState.SetCurrentPlaylist(null);
		this.appState.SetIsPlaying(true);
		console.log("Playing track:", track.title);

		try {
			// Fetch full artist tracks in case of search filter
			const response = await backendService.FetchAlbumTracks(id, {
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play in this album");
				return;
			}

			this.appState.SetOriginalQueue(response.data);

			if (this.appState.Shuffle) {
				const queue = [...response.data];
				shuffleArray(queue);
				const idx = queue.findIndex((t) => t.id === track.id);
				if (idx > 0) {
					const [selected] = queue.splice(idx, 1);
					queue.unshift(selected!);
				}
				this.appState.SetTemporaryQueue(queue);
			} else {
				this.appState.SetTemporaryQueue(response.data);
			}

			console.log(`Added all tracks from ${id} to temporary queue`);
		} catch (error) {
			console.error("Error fetching album for playback:", error);
			return;
		}
	}

	async AddAlbumToQueue(id: string, search?: string) {
		try {
			const response = await backendService.FetchAlbumTracks(id, {
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play in the album");
				return;
			}

			for (const track of response.data) {
				await this.AddTrackToQueue(track);
			}
			console.log(`Added all tracks from ${id} to queue`);
		} catch (error) {
			console.error("Error fetching album for playback:", error);
			return;
		}
	}

	async PlayArtist(id: string, search?: string) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.playArtist(id);
			return;
		}
		this.appState.SetQueue([]);

		try {
			// Fetch full artist tracks in case of search filter
			const response = await backendService.FetchArtistTracks(id, {
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play by this artist");
				return;
			}

			this.appState.SetOriginalQueue(response.data);

			if (this.appState.Shuffle) {
				const queue = [...response.data];
				shuffleArray(queue);
				this.appState.SetTemporaryQueue(queue);
			} else {
				this.appState.SetTemporaryQueue(response.data);
			}

			this.SetCurrentTrack(this.appState.TemporaryQueue[0]!);
			this.appState.SetCurrentPlaylist(null);
			this.appState.SetIsPlaying(true);
			console.log(`Playing artist ${id}`);
		} catch (error) {
			console.error("Error fetching artist for playback:", error);
			return;
		}
	}

	async PlayArtistTrack(track: Track, id: string, search?: string) {
		const rc = this.appState.RemoteControl;
		if (rc) {
			rc.playArtistTrack(track.id, id, search);
			return;
		}
		this.appState.SetQueue([]);

		this.SetCurrentTrack(track);
		this.appState.SetCurrentPlaylist(null);
		this.appState.SetIsPlaying(true);
		console.log("Playing track:", track.title);

		try {
			// Fetch full artist tracks in case of search filter
			const response = await backendService.FetchArtistTracks(id, {
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play by this artist");
				return;
			}

			this.appState.SetOriginalQueue(response.data);

			if (this.appState.Shuffle) {
				const queue = [...response.data];
				shuffleArray(queue);
				const idx = queue.findIndex((t) => t.id === track.id);
				if (idx > 0) {
					const [selected] = queue.splice(idx, 1);
					queue.unshift(selected!);
				}
				this.appState.SetTemporaryQueue(queue);
			} else {
				this.appState.SetTemporaryQueue(response.data);
			}

			console.log(`Added all tracks from ${id} to temporary queue`);
		} catch (error) {
			console.error("Error fetching artist for playback:", error);
			return;
		}
	}

	async AddArtistToQueue(id: string, search?: string) {
		try {
			const response = await backendService.FetchArtistTracks(id, {
				search: search
			});

			if (0 === response.data.length) {
				console.warn("No tracks available to play by this artist");
				return;
			}

			for (const track of response.data) {
				await this.AddTrackToQueue(track);
			}
			console.log(`Added all tracks from ${id} to queue`);
		} catch (error) {
			console.error("Error fetching artist for playback:", error);
			return;
		}
	}
}

export default PlayerService;
