import { ref, onMounted } from "vue";

/**
 * Composable for PWA functionality including service worker registration
 * and install prompts for iOS/Android.
 */
export const usePWA = () => {
	const isInstallable = ref(false);
	const isInstalled = ref(false);
	const deferredPrompt = ref<any>(null);
	const swRegistration = ref<ServiceWorkerRegistration | null>(null);

	// Register service worker
	const registerServiceWorker = async () => {
		if (!("serviceWorker" in navigator)) {
			console.warn("Service Workers are not supported");
			return;
		}

		try {
			const registration = await navigator.serviceWorker.register("/ui/sw.js", {
				scope: "/ui/"
			});
			swRegistration.value = registration;
			console.log("Service Worker registered:", registration);

			// Check for updates periodically
			setInterval(() => {
				registration.update();
			}, 60000); // Check every minute

			// Listen for updates
			registration.addEventListener("updatefound", () => {
				const newWorker = registration.installing;
				if (!newWorker) return;

				newWorker.addEventListener("statechange", () => {
					if (newWorker.state === "activated") {
						console.log("New Service Worker activated");
						notifyUpdate();
					}
				});
			});
		} catch (error) {
			console.error("Service Worker registration failed:", error);
		}
	};

	// Notify user of available update
	const notifyUpdate = () => {
		if (swRegistration.value?.waiting) {
			// You could show a notification here
			console.log("Service Worker update available");
		}
	};

	// Check if app is installed as PWA
	const checkIfInstalled = () => {
		// Check if running as standalone PWA
		if (window.matchMedia("(display-mode: standalone)").matches) {
			isInstalled.value = true;
			return;
		}

		// Check if running in fullscreen mode (iOS)
		if (navigator.standalone === true) {
			isInstalled.value = true;
			return;
		}

		// Check if running in web app mode
		if (window.matchMedia("(display-mode: fullscreen)").matches) {
			isInstalled.value = true;
			return;
		}
	};

	// Handle install prompt
	const handleBeforeInstallPrompt = (event: any) => {
		event.preventDefault();
		deferredPrompt.value = event;
		isInstallable.value = true;
		console.log("Install prompt captured");
	};

	// Trigger install prompt
	const promptInstall = async () => {
		if (!deferredPrompt.value) {
			console.warn("Install prompt not available");
			return;
		}

		deferredPrompt.value.prompt();
		const { outcome } = await deferredPrompt.value.userChoice;
		console.log(`User response to install prompt: ${outcome}`);

		if (outcome === "accepted") {
			isInstalled.value = true;
		}

		deferredPrompt.value = null;
		isInstallable.value = false;
	};

	// Initialize PWA features
	const init = () => {
		if (typeof window === "undefined") {
			return;
		}

		checkIfInstalled();
		registerServiceWorker();

		// Listen for install prompt
		window.addEventListener("beforeinstallprompt", handleBeforeInstallPrompt);

		// Listen for app installed
		window.addEventListener("appinstalled", () => {
			console.log("PWA was installed");
			isInstalled.value = true;
			deferredPrompt.value = null;
		});

		// Cleanup function
		return () => {
			window.removeEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
		};
	};

	onMounted(() => {
		init();
	});

	return {
		isInstallable,
		isInstalled,
		swRegistration,
		promptInstall,
		registerServiceWorker
	};
};
