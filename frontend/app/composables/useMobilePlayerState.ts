import { reactive, watch, type Ref } from "vue";
import type PlayerService from "~/services/player.service";
import useAppState from "~/stores/appState";
import { useHaptics } from "./useHaptics";
import { useImageUrl } from "./useImageUrl";

export type SwipeDirection = "left" | "right" | null;
export type ArtAnimationPhase = "idle" | "exit" | "enter";

export interface AlbumArtAnimationState {
	phase: ArtAnimationPhase;
	direction: SwipeDirection;
	exitingCoverArtUrl: string | null;
}

export interface FullscreenPlayerDragState {
	isDragging: boolean;
	startY: number;
	startX: number;
	startTime: number;
	offsetY: number;
	offsetX: number;
	opacity: number;
}

export interface MiniPlayerDragState {
	isDragging: boolean;
	startY: number;
	startTime: number;
	offsetY: number;
}

export interface MobilePlayerUIState {
	showFullscreen: boolean;
	showMenu: boolean;
	fullscreenDrag: FullscreenPlayerDragState;
	miniPlayerDrag: MiniPlayerDragState;
	albumArtAnimation: AlbumArtAnimationState;
}

export function useMobilePlayerState(playerRef: Ref<PlayerService | null>) {
	const appState = useAppState();
	const { tap, heavyTap, selectionChanged } = useHaptics();
	const { getImageUrl } = useImageUrl();

	// Swipe debounce
	let lastSwipeTime = 0;
	const SWIPE_DEBOUNCE_MS = 1000;

	// Animation lock
	let animationInProgress = false;
	const ART_ANIMATION_DURATION_MS = 300;

	// Drag commit thresholds
	const SCREEN_PERCENT_THRESHOLD = 0.3; // 30% of screen height for slow drags
	const VELOCITY_THRESHOLD = 0.5; // px/ms — fast swipe commits regardless of distance

	const state = reactive<MobilePlayerUIState>({
		showFullscreen: false,
		showMenu: false,
		fullscreenDrag: {
			isDragging: false,
			startY: 0,
			startX: 0,
			startTime: 0,
			offsetY: 0,
			offsetX: 0,
			opacity: 1
		},
		miniPlayerDrag: {
			isDragging: false,
			startY: 0,
			startTime: 0,
			offsetY: 0
		},
		albumArtAnimation: {
			phase: "idle",
			direction: null,
			exitingCoverArtUrl: null
		}
	});

	function isSwipeDebounced(): boolean {
		const now = Date.now();
		if (now - lastSwipeTime < SWIPE_DEBOUNCE_MS) {
			return true;
		}
		lastSwipeTime = now;
		return false;
	}

	// Track change with album art fly-off animation
	async function swipeTrackChange(direction: "left" | "right"): Promise<void> {
		if (isSwipeDebounced() || animationInProgress) {
			return;
		}

		animationInProgress = true;
		await selectionChanged();

		// Capture current cover art URL before track changes
		const currentTrack = appState.CurrentTrack;
		if (currentTrack?.cover_art_id) {
			state.albumArtAnimation.exitingCoverArtUrl = getImageUrl(`/api/cover-art/${currentTrack.cover_art_id}`);
		} else {
			state.albumArtAnimation.exitingCoverArtUrl = null;
		}

		// Start exit animation
		state.albumArtAnimation.direction = direction;
		state.albumArtAnimation.phase = "exit";

		// Trigger track change
		if ("left" === direction) {
			playerRef.value?.NextTrack();
		} else {
			playerRef.value?.PreviousTrack();
		}

		// Start enter animation shortly after exit begins
		setTimeout(() => {
			state.albumArtAnimation.phase = "enter";
		}, 50);

		// Reset after full animation duration
		setTimeout(() => {
			state.albumArtAnimation.phase = "idle";
			state.albumArtAnimation.direction = null;
			state.albumArtAnimation.exitingCoverArtUrl = null;
			animationInProgress = false;
		}, ART_ANIMATION_DURATION_MS + 50);
	}

	// --- Fullscreen player swipe handlers ---

	async function onFullscreenSwipeDown(): Promise<void> {
		if (isSwipeDebounced()) {
			return;
		}
		console.log("Swiped down on mobile player, closing player");
		await heavyTap();
		state.showFullscreen = false;
	}

	async function onFullscreenSwipeLeft(): Promise<void> {
		console.log("Swiped left on mobile player, going to next track");
		await swipeTrackChange("left");
	}

	async function onFullscreenSwipeRight(): Promise<void> {
		console.log("Swiped right on mobile player, going to previous track");
		await swipeTrackChange("right");
	}

	// --- Mini player swipe handlers ---

	async function onMiniPlayerSwipeLeft(): Promise<void> {
		if (isSwipeDebounced()) {
			return;
		}
		console.log("Swiped left on main player, going to next track");
		await selectionChanged();
		playerRef.value?.NextTrack();
	}

	async function onMiniPlayerSwipeRight(): Promise<void> {
		if (isSwipeDebounced()) {
			return;
		}
		console.log("Swiped right on main player, going to previous track");
		await selectionChanged();
		playerRef.value?.PreviousTrack();
	}

	async function onMiniPlayerSwipeUp(): Promise<void> {
		if (isSwipeDebounced()) {
			return;
		}
		console.log("Swiped up on main player, opening mobile player");
		await tap();
		state.showFullscreen = true;
	}

	// --- Fullscreen player drag handlers ---

	function onFullscreenDragging(e: TouchEvent): void {
		if (!e.touches || 0 === e.touches.length) {
			return;
		}

		const touch = e.touches[0];
		if (!touch) {
			return;
		}

		const currentY = touch.clientY;
		const currentX = touch.clientX;

		// Initialize drag start position on first drag event
		if (!state.fullscreenDrag.isDragging) {
			state.fullscreenDrag.isDragging = true;
			state.fullscreenDrag.startY = currentY;
			state.fullscreenDrag.startX = currentX;
			state.fullscreenDrag.startTime = Date.now();
			state.fullscreenDrag.offsetY = 0;
			state.fullscreenDrag.offsetX = 0;
			state.fullscreenDrag.opacity = 1;
			return;
		}

		const deltaY = currentY - state.fullscreenDrag.startY;
		const deltaX = currentX - state.fullscreenDrag.startX;

		// Threshold before animation starts
		const dragThresholdStart = 100;
		const adjustedDeltaY = Math.max(0, deltaY - dragThresholdStart);

		state.fullscreenDrag.offsetY = adjustedDeltaY;
		state.fullscreenDrag.offsetX = deltaX;

		// Calculate opacity based on vertical drag
		const maxDragY = 200;
		const opacityFade = Math.max(0, 1 - Math.abs(adjustedDeltaY) / maxDragY);
		state.fullscreenDrag.opacity = opacityFade;
	}

	async function onFullscreenDragEnd(): Promise<void> {
		const elapsed = Date.now() - state.fullscreenDrag.startTime;
		const velocity = elapsed > 0 ? state.fullscreenDrag.offsetY / elapsed : 0;
		const screenPercent = state.fullscreenDrag.offsetY / window.innerHeight;

		const shouldCommit = velocity > VELOCITY_THRESHOLD || screenPercent > SCREEN_PERCENT_THRESHOLD;

		if (shouldCommit && state.fullscreenDrag.offsetY > 0) {
			await heavyTap();
			state.showFullscreen = false;
		}

		// Reset drag state with animation
		state.fullscreenDrag.offsetY = 0;
		state.fullscreenDrag.offsetX = 0;
		state.fullscreenDrag.opacity = 1;
		state.fullscreenDrag.isDragging = false;
	}

	// --- Mini player drag handlers ---

	function onMiniPlayerDragging(e: TouchEvent): void {
		if (!e.touches || 0 === e.touches.length) {
			return;
		}

		const touch = e.touches[0];
		if (!touch) {
			return;
		}

		const currentY = touch.clientY;

		if (!state.miniPlayerDrag.isDragging) {
			state.miniPlayerDrag.isDragging = true;
			state.miniPlayerDrag.startY = currentY;
			state.miniPlayerDrag.startTime = Date.now();
			state.miniPlayerDrag.offsetY = 0;
			return;
		}

		const deltaY = currentY - state.miniPlayerDrag.startY;
		// Only allow upward drag (negative delta)
		state.miniPlayerDrag.offsetY = deltaY < 0 ? Math.abs(deltaY) : 0;
	}

	async function onMiniPlayerDragEnd(): Promise<void> {
		const elapsed = Date.now() - state.miniPlayerDrag.startTime;
		const velocity = elapsed > 0 ? state.miniPlayerDrag.offsetY / elapsed : 0;
		const screenPercent = state.miniPlayerDrag.offsetY / window.innerHeight;

		const shouldCommit = velocity > VELOCITY_THRESHOLD || screenPercent > SCREEN_PERCENT_THRESHOLD;

		if (shouldCommit && state.miniPlayerDrag.offsetY > 0) {
			await tap();
			state.showFullscreen = true;
		}

		state.miniPlayerDrag.offsetY = 0;
		state.miniPlayerDrag.isDragging = false;
	}

	// --- Visibility helpers ---

	function open(): void {
		state.showFullscreen = true;
	}

	function close(): void {
		state.showFullscreen = false;
	}

	async function toggle(): Promise<void> {
		await tap();
		state.showFullscreen = !state.showFullscreen;
	}

	function closeMenu(): void {
		state.showMenu = false;
	}

	// Close fullscreen when track becomes null
	watch(
		() => appState.CurrentTrack,
		(track) => {
			if (!track) {
				state.showFullscreen = false;
			}
		}
	);

	return {
		state,

		// Whether the fullscreen player DOM should be present
		// (during drag-up from mini player, we show it peeking from the bottom)
		get shouldShowFullscreen() {
			return state.showFullscreen || (state.miniPlayerDrag.isDragging && state.miniPlayerDrag.offsetY > 0);
		},

		// Dynamic style for the fullscreen player container
		get fullscreenStyle() {
			const isDraggingUp =
				state.miniPlayerDrag.isDragging && state.miniPlayerDrag.offsetY > 0 && !state.showFullscreen;
			const isDragging = state.fullscreenDrag.isDragging || state.miniPlayerDrag.isDragging;

			return {
				transform: isDraggingUp
					? `translateY(calc(100% - ${state.miniPlayerDrag.offsetY}px))`
					: `translateY(${state.fullscreenDrag.offsetY}px)`,
				transition: isDragging ? "none" : "all 0.3s ease-out"
			};
		},

		// Fullscreen player handlers
		onFullscreenSwipeDown,
		onFullscreenSwipeLeft,
		onFullscreenSwipeRight,
		onFullscreenDragging,
		onFullscreenDragEnd,

		// Mini player handlers
		onMiniPlayerSwipeLeft,
		onMiniPlayerSwipeRight,
		onMiniPlayerSwipeUp,
		onMiniPlayerDragging,
		onMiniPlayerDragEnd,

		// Visibility
		open,
		close,
		toggle,
		closeMenu
	};
}
