import { Capacitor } from "@capacitor/core";

/**
 * Detect whether the app is running inside a native Capacitor shell
 * (iOS / Android) or as a regular web app in a browser.
 */
export const isNative = (): boolean => {
	return Capacitor.isNativePlatform();
};

export const isIOS = (): boolean => {
	return "ios" === Capacitor.getPlatform();
};

export const isAndroid = (): boolean => {
	return "android" === Capacitor.getPlatform();
};

export const isWeb = (): boolean => {
	return "web" === Capacitor.getPlatform();
};

export const getPlatform = (): string => {
	return Capacitor.getPlatform();
};

export const isElectron = (): boolean => {
	return typeof window !== "undefined" && Boolean((window as any).__ELECTRON__);
};

export const isNativeOrElectron = (): boolean => isNative() || isElectron();
