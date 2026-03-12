import type PlayerService from "~/services/player.service";

export interface KeyboardShortcutsOptions {
	// player?: Ref<PlayerService> | null;
	player?: PlayerService;
	onToggleFullscreen?: () => void;
	onToggleVisualizer?: () => void;
	onEscapeFullscreen?: () => void;
	onEscapeVisualizer?: () => void;
	onFocusSearch?: () => void;
	isFullscreenActive?: () => boolean;
	isVisualizerActive?: () => boolean;
	isSearchFocused?: () => boolean;
}

export default function useKeyboardShortcuts(options: KeyboardShortcutsOptions = {}) {
	// Track if any input/textarea/select is focused
	const isAnyInputFocused = () => {
		const activeElement = document.activeElement;
		if (!activeElement) {
			return false;
		}

		const tagName = activeElement.tagName.toLowerCase();
		const isContentEditable = "true" === activeElement.getAttribute("contenteditable");

		return "input" === tagName || "textarea" === tagName || "select" === tagName || isContentEditable;
	};

	const handleKeydown = (event: KeyboardEvent) => {
		// console.log("Key pressed:", event.key, "Code:", event.code);
		// Handle escape key - priority order: fullscreen > visualizer
		if ("Escape" === event.key) {
			if (options.isFullscreenActive?.()) {
				options.onEscapeFullscreen?.();
				event.preventDefault();
				return;
			} else if (options.isVisualizerActive?.()) {
				options.onEscapeVisualizer?.();
				event.preventDefault();
				return;
			}

			// TODO - If search text is non-empty, clear it
		}

		// If any command/ctrl/alt/meta key is pressed, ignore shortcuts
		if (event.metaKey || event.ctrlKey || event.altKey) {
			return;
		}

		// If any input is focused, only handle escape and '/' key
		if (isAnyInputFocused()) {
			// Don't handle other shortcuts when input is focused
			return;
		}

		// Handle '/' key to focus search
		if ("/" === event.key) {
			event.preventDefault();
			options.onFocusSearch?.();
			return;
		}

		// Don't handle other shortcuts if search is focused
		if (options.isSearchFocused?.()) {
			return;
		}

		// Music playback shortcuts
		switch (event.key) {
			case " ":
			case "p":
			case "P":
				event.preventDefault();
				options.player?.TogglePlayback();
				break;

			case "h":
			case "H":
			case "{":
			case "[":
			case "ArrowLeft":
				event.preventDefault();
				options.player?.PreviousTrack();
				break;

			case "l":
			case "L":
			case "}":
			case "]":
			case "ArrowRight":
				event.preventDefault();
				options.player?.NextTrack();
				break;

			case "f":
			case "F":
				event.preventDefault();
				options.onToggleFullscreen?.();
				break;

			case "v":
			case "V":
				event.preventDefault();
				options.onToggleVisualizer?.();
				break;

			case "s":
			case "S":
				event.preventDefault();
				options.player?.ToggleShuffle();
				break;

			case "m":
			case "M":
				event.preventDefault();
				options.player?.ToggleMute();
				break;

			case "r":
			case "R":
				event.preventDefault();
				options.player?.ToggleRepeat();
				break;

			case "ArrowUp":
			case "k":
			case "K":
			case "+":
			case "=":
				event.preventDefault();
				options.player?.VolumeUp();
				break;

			case "ArrowDown":
			case "j":
			case "J":
			case "-":
			case "_":
				event.preventDefault();
				options.player?.VolumeDown();
				break;
		}

		// Handle media keys
		if (
			"MediaPlayPause" === event.code ||
			"MediaPlay" === event.code ||
			"MediaPause" === event.code ||
			"Pause" === event.code
		) {
			event.preventDefault();
			options.player?.TogglePlayback();
		} else if ("MediaTrackNext" === event.code) {
			event.preventDefault();
			options.player?.NextTrack();
		} else if ("MediaTrackPrevious" === event.code) {
			event.preventDefault();
			options.player?.PreviousTrack();
		} else if ("MediaStop" === event.code) {
			event.preventDefault();
			options.player?.Stop();
		}
	};

	return {
		handleKeydown
	};
}
