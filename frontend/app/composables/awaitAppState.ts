import useAppState from "@/stores/appState";

// `await awaitAppState()` must be called in onMounted()

export default function (): Promise<void> {
	return new Promise<void>((resolve) => {
		const ready = () => {
			const appState = useAppState();
			if (!appState || appState.Loaded === undefined) {
				// Store not yet initialized
				return false;
			} else if (!appState.Loaded) {
				appState.SetLoaded(true);
			}

			if (appState && appState.Loaded) {
				return true;
			}

			return false;
		};

		if (ready()) {
			resolve();
			return;
		}

		const interval = setInterval(() => {
			if (ready()) {
				clearInterval(interval);
				resolve();
			}
		}, 100);
	});
}
