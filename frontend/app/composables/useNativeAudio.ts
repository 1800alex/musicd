import { watch } from "vue";
import { App } from "@capacitor/app";
import { isNative, isIOS } from "~/utils/platform";
import useAppState from "~/stores/appState";

/**
 * Composable that handles native-specific audio behaviour.
 *
 * On iOS / Android (via Capacitor) it:
 *  - Keeps the audio session active when the app is backgrounded so playback
 *    continues on the lock screen.
 *  - Resumes playback when the app returns to the foreground after an
 *    interruption (e.g. phone call).
 *
 * On the web it is a no-op so it is always safe to call.
 */
export const useNativeAudio = () => {
	if (!isNative()) {
		// Web — nothing to do; the existing Media Session API handles
		// lock-screen metadata and controls.
		return;
	}

	const appState = useAppState();

	// Track whether playback was active before an interruption so we can
	// decide whether to auto-resume.
	let wasPlayingBeforeBackground = false;

	// ── App lifecycle listeners ──────────────────────────────────────────

	// `appStateChange` fires when the app moves to/from background.
	App.addListener("appStateChange", ({ isActive }) => {
		if (!isActive) {
			// Going to background — remember current state.
			// Audio will keep playing thanks to the UIBackgroundModes=audio
			// entitlement configured in Info.plist.
			wasPlayingBeforeBackground = appState.IsPlaying;
		} else {
			// Coming back to foreground.
			// If the OS paused audio while backgrounded (e.g. phone call)
			// and we were playing before, try to resume.
			if (wasPlayingBeforeBackground && !appState.IsPlaying) {
				const el = appState.AudioElement;
				if (el && appState.CurrentTrack) {
					el.play().catch((err) => {
						console.warn("[NativeAudio] Auto-resume failed:", err);
					});
					appState.SetIsPlaying(true);
				}
			}
		}
	});

	// On iOS handle audio interruptions (phone calls, Siri, etc.)
	if (isIOS()) {
		// When iOS interrupts audio (e.g. incoming call) the HTMLAudioElement
		// fires a 'pause' event.  We listen for it so the UI stays in sync.
		watch(
			() => appState.AudioElement,
			(el) => {
				if (!el) {
					return;
				}

				el.addEventListener("pause", () => {
					// Only update state if the pause wasn't triggered by us
					if (appState.IsPlaying) {
						appState.SetIsPlaying(false);
					}
				});

				el.addEventListener("play", () => {
					if (!appState.IsPlaying) {
						appState.SetIsPlaying(true);
					}
				});
			},
			{ immediate: true }
		);
	}
};
