import type { Ref } from "vue";
import type { Track } from "~/types";

export interface MediaSessionTrack {
	title: string;
	artist: string;
	album: string;
	artwork?: MediaImage[];
	duration?: number;
}

export class MediaSessionService {
	private isSupported: boolean = false;

	private currentTrack: MediaMetadataInit | null = null;

	constructor() {
		this.isSupported = "mediaSession" in navigator;
		if (this.isSupported) {
			this.setupDefaultHandlers();
		} else {
			console.warn("Media Session API is not supported in this browser");
		}
	}

	private setupDefaultHandlers() {
		if (!this.isSupported) {
			return;
		}

		// For iOS, we don't set default handlers initially
		// iOS requires handlers to be properly set with real functionality
		// to enable the buttons. We'll let the actual application set them
		// through setActionHandlers() method.
	}

	updateArtwork(artwork?: { src?: string; type?: string; sizes?: string }) {
		if (!this.isSupported) {
			return;
		}

		if (!this.currentTrack) {
			this.currentTrack = {};
		}

		this.currentTrack.title = this.currentTrack?.title || "Unknown Title";
		this.currentTrack.artist = this.currentTrack?.artist || "Unknown Artist";
		this.currentTrack.album = this.currentTrack?.album || "Unknown Album";

		if (!artwork || !artwork.src) {
			this.currentTrack.artwork = [];
		} else {
			this.currentTrack.artwork = [
				{
					src: artwork.src,
					sizes: artwork.sizes,
					type: artwork.type
				}
			];
		}

		this.setMetadata();
	}

	setMetadata() {
		if (!this.isSupported) {
			return;
		}

		if (!this.currentTrack) {
			navigator.mediaSession.metadata = null;
			return;
		}

		const artwork: MediaImage[] = this.currentTrack.artwork || [];

		// If no artwork provided, add a default placeholder
		if (0 === artwork.length) {
			artwork.push({
				src: `${window.location.origin}/favicon.ico` // Fallback to favicon
			});
		}

		try {
			// console.log("Updated Media Session:", JSON.stringify(this.currentTrack));

			navigator.mediaSession.metadata = new MediaMetadata({
				title: this.currentTrack.title,
				artist: this.currentTrack.artist,
				album: this.currentTrack.album,
				artwork: artwork
			});
		} catch (error) {
			console.error("Failed to set MediaMetadata:", error);
		}
	}

	setPlaybackState(state: "playing" | "paused" | "none") {
		if (!this.isSupported) {
			return;
		}
		navigator.mediaSession.playbackState = state;
	}

	setPositionState(duration?: number, playbackRate: number = 1.0, position: number = 0) {
		if (!this.isSupported) {
			return;
		}

		try {
			navigator.mediaSession.setPositionState({
				duration: duration || 0,
				playbackRate,
				position: Math.min(position, duration || 0)
			});
		} catch (error) {
			console.warn("Failed to set position state:", error);
		}
	}

	setActionHandlers(handlers: {
		play?: () => void;
		pause?: () => void;
		previoustrack?: () => void;
		nexttrack?: () => void;
		seekbackward?: (details: MediaSessionActionDetails) => void;
		seekforward?: (details: MediaSessionActionDetails) => void;
		seekto?: (details: MediaSessionActionDetails) => void;
	}) {
		if (!this.isSupported) {
			return;
		}

		Object.entries(handlers).forEach(([action, handler]) => {
			try {
				navigator.mediaSession.setActionHandler(action as MediaSessionAction, handler || null);
			} catch (error) {
				console.warn(`Failed to set action handler for ${action}:`, error);
			}
		});
	}

	clearMetadata() {
		if (!this.isSupported) {
			return;
		}
		navigator.mediaSession.metadata = null;
	}

	async updateFromTrack(track: Track) {
		if (!this.isSupported) {
			console.warn("Media Session API not supported, cannot update metadata");
			return;
		}
		if (!this.currentTrack) {
			this.currentTrack = {};
		}

		this.currentTrack.title = track.title || "Unknown Title";
		this.currentTrack.artist = track.artist || "Unknown Artist";
		this.currentTrack.album = track.album || "Unknown Album";
		// this.currentTrack.duration = track.duration || 0;

		// this.setMetadata();
	}

	isMediaSessionSupported(): boolean {
		return this.isSupported;
	}
}

export default MediaSessionService;
