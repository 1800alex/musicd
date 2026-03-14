import type { CapacitorConfig } from "@capacitor/cli";

const config: CapacitorConfig = {
	appId: "com.musicplayer.app",
	appName: "Music Player",
	webDir: ".output/public",
	server: {
		// In production, the app loads from the bundled web assets.
		// For development, uncomment the url below and set it to your dev server:
		// url: "http://YOUR_LOCAL_IP:3000/ui/",
		androidScheme: "https"
	},
	ios: {
		// Allow inline media playback (required for background audio)
		allowsLinkPreview: false,
		backgroundColor: "#1a1a1a",
		contentInset: "automatic",
		preferredContentMode: "mobile",
		scheme: "capacitor"
	},
	android: {
		backgroundColor: "#1a1a1a"
	},
	plugins: {
		App: {
			// Keep web view alive in background
		}
	}
};

export default config;
