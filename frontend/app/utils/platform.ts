import { Capacitor } from "@capacitor/core";

/**
 * Detect whether the app is running inside a native Capacitor shell
 * (iOS / Android) or as a regular web app in a browser.
 */
export const isNative = (): boolean => {
	return Capacitor.isNativePlatform();
};

export const isIOS = (): boolean => {
	return Capacitor.getPlatform() === "ios";
};

export const isAndroid = (): boolean => {
	return Capacitor.getPlatform() === "android";
};

export const isWeb = (): boolean => {
	return Capacitor.getPlatform() === "web";
};

export const getPlatform = (): string => {
	return Capacitor.getPlatform();
};

export const isElectron = (): boolean => {
	return typeof window !== "undefined" && !!(window as any).__ELECTRON__;
};

export const isNativeOrElectron = (): boolean => isNative() || isElectron();
