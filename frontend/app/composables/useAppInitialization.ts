import { useMediaSession } from "./useMediaSession";
import { usePWA } from "./usePWA";

/**
 * Master initialization function that sets up:
 * - PWA features (service worker, install prompts)
 * - Media Session API (lock screen controls, metadata)
 * - iOS-specific optimizations
 */
export const useAppInitialization = (playerService: any) => {
	const { isInstallable, isInstalled, promptInstall } = usePWA();
	const mediaSession = useMediaSession();

	// Initialize Media Session with player service
	if (playerService) {
		mediaSession.init(playerService);
	}

	// iOS-specific optimizations
	if (typeof window !== "undefined") {
		// Prevent double-tap zoom on iOS
		document.addEventListener(
			"touchstart",
			(e) => {
				if (e.touches.length > 1) {
					e.preventDefault();
				}
			},
			{ passive: false }
		);

		// Prevent default zoom on iOS
		document.addEventListener(
			"touchmove",
			(e) => {
				if (e.touches.length > 1) {
					e.preventDefault();
				}
			},
			{ passive: false }
		);

		// Keep screen awake when playing music (via WakeLock API)
		setupWakeLock();
	}

	function setupWakeLock() {
		if (!("wakeLock" in navigator)) {
			console.log("Wake Lock API not supported");
			return;
		}

		let wakeLockSentinel: any = null;

		const acquireWakeLock = async () => {
			try {
				wakeLockSentinel = await (navigator as any).wakeLock.request("screen");
				console.log("Wake Lock acquired");

				wakeLockSentinel.addEventListener("release", () => {
					console.log("Wake Lock released");
				});
			} catch (err) {
				console.warn("Wake Lock request failed:", err);
			}
		};

		const releaseWakeLock = async () => {
			if (wakeLockSentinel !== null) {
				try {
					await wakeLockSentinel.release();
					wakeLockSentinel = null;
					console.log("Wake Lock released");
				} catch (err) {
					console.warn("Wake Lock release failed:", err);
				}
			}
		};

		// Request wake lock when playing, release when paused
		if (typeof window !== "undefined") {
			window.addEventListener("focus", () => {
				if (document.hidden === false && wakeLockSentinel === null) {
					acquireWakeLock();
				}
			});

			document.addEventListener("visibilitychange", async () => {
				if ((document as any).visible === false) {
					releaseWakeLock();
				} else {
					acquireWakeLock();
				}
			});
		}
	}

	const cleanup = () => {
		mediaSession.cleanup();
	};

	return {
		isInstallable,
		isInstalled,
		promptInstall,
		mediaSession,
		cleanup
	};
};
