import { Haptics, ImpactStyle } from "@capacitor/haptics";
import { isNativeOrElectron } from "~/utils/platform";

export const useHaptics = () => {
	// Only enable haptics on native apps (iOS/Android)
	const isEnabled = isNativeOrElectron();

	const tap = async () => {
		if (!isEnabled) {
			return;
		}
		try {
			await Haptics.impact({ style: ImpactStyle.Light });
		} catch (e) {
			// Silently fail if haptics not available
		}
	};

	const heavyTap = async () => {
		if (!isEnabled) {
			return;
		}
		try {
			await Haptics.impact({ style: ImpactStyle.Heavy });
		} catch (e) {
			// Silently fail if haptics not available
		}
	};

	const mediumTap = async () => {
		if (!isEnabled) {
			return;
		}
		try {
			await Haptics.impact({ style: ImpactStyle.Medium });
		} catch (e) {
			// Silently fail if haptics not available
		}
	};

	const selectionChanged = async () => {
		if (!isEnabled) {
			return;
		}
		try {
			await Haptics.selectionChanged();
		} catch (e) {
			// Silently fail if haptics not available
		}
	};

	return {
		tap,
		heavyTap,
		mediumTap,
		selectionChanged
	};
};
