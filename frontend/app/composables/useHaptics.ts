import { Haptics, ImpactStyle } from "@capacitor/haptics";
import { isNativeOrElectron } from "~/utils/platform";

export const useHaptics = () => {
	const isNative = isNativeOrElectron();
	const vibrationSupported = typeof navigator !== "undefined" && "vibrate" in navigator;

	// Helper to vibrate using Web Vibration API
	const vibrateWeb = (pattern: number | number[]) => {
		if (vibrationSupported) {
			try {
				navigator.vibrate(pattern);
			} catch {
				// Silently fail if vibration not available
			}
		}
	};

	const tap = async () => {
		if (isNative) {
			try {
				await Haptics.impact({ style: ImpactStyle.Light });
			} catch {
				// Fall back to web vibration
				vibrateWeb(10);
			}
		} else {
			// PWA in browser
			vibrateWeb(10);
		}
	};

	const heavyTap = async () => {
		if (isNative) {
			try {
				await Haptics.impact({ style: ImpactStyle.Heavy });
			} catch {
				// Fall back to web vibration
				vibrateWeb(50);
			}
		} else {
			// PWA in browser
			vibrateWeb(50);
		}
	};

	const mediumTap = async () => {
		if (isNative) {
			try {
				await Haptics.impact({ style: ImpactStyle.Medium });
			} catch {
				// Fall back to web vibration
				vibrateWeb(30);
			}
		} else {
			// PWA in browser
			vibrateWeb(30);
		}
	};

	const selectionChanged = async () => {
		if (isNative) {
			try {
				await Haptics.selectionChanged();
			} catch {
				// Fall back to web vibration (pattern for selection)
				vibrateWeb([15, 10, 15]);
			}
		} else {
			// PWA in browser (pattern for selection)
			vibrateWeb([15, 10, 15]);
		}
	};

	return {
		tap,
		heavyTap,
		mediumTap,
		selectionChanged
	};
};
