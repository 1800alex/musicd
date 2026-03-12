import { resolveBaseURL } from "~/services/http.service";
import { isNativeOrElectron } from "~/utils/platform";

declare const __DEV_BACKEND_HOST__: string;

/**
 * Unified composable for constructing backend URLs across web, Capacitor, and Electron platforms.
 * Intelligently handles:
 * - Native/Electron platform detection
 * - localStorage-configured backend URLs
 * - Development mode backend hosts
 * - Protocol handling (http/https → ws/wss)
 * - URL normalization (trailing slashes, protocol prefixes)
 */
export const useBackendURL = () => {
	/**
	 * Get the base HTTP(S) URL for API calls
	 * @example
	 * getHTTPURL() → "http://192.168.1.10:8080"
	 * getHTTPURL("/api/music/file.mp3") → "http://192.168.1.10:8080/api/music/file.mp3"
	 */
	const getHTTPURL = (path: string = ""): string => {
		const baseURL = resolveBaseURL().replace(/\/$/, "");
		// On web, baseURL is "/" so we just return the path
		if (baseURL === "") {
			return path;
		}
		return `${baseURL}${path}`;
	};

	/**
	 * Get the WebSocket host (domain:port without protocol)
	 * @example
	 * getWSHost() → "192.168.1.10:8080"
	 */
	const getWSHost = (): string => {
		const stored = localStorage.getItem("backendURL");
		if (stored) {
			return stored.replace(/^https?:\/\//, "").replace(/\/$/, "");
		}

		// Fall back to dev or current host
		const host = __DEV_BACKEND_HOST__ || window.location.host;
		return host.replace(/^https?:\/\//, "").replace(/\/$/, "");
	};

	/**
	 * Get the WebSocket protocol (ws or wss)
	 * @example
	 * getWSProtocol() → "wss" (if backend URL starts with https)
	 */
	const getWSProtocol = (): "ws" | "wss" => {
		const stored = localStorage.getItem("backendURL");
		if (stored) {
			return stored.startsWith("https://") ? "wss" : "ws";
		}

		// Fall back to window protocol
		return window.location.protocol === "https:" ? "wss" : "ws";
	};

	/**
	 * Get full WebSocket URL
	 * @example
	 * getWSURL("/ws/session/abc123") → "wss://192.168.1.10:8080/ws/session/abc123"
	 */
	const getWSURL = (path: string): string => {
		const protocol = getWSProtocol();
		const host = getWSHost();
		return `${protocol}://${host}${path}`;
	};

	/**
	 * Get the server host (domain:port, handling protocol)
	 * For RemoteControlService which expects just the host without protocol
	 * @example
	 * getServerHost() → "192.168.1.10:8080"
	 */
	const getServerHost = (): string => {
		return getWSHost();
	};

	return {
		getHTTPURL,
		getWSHost,
		getWSProtocol,
		getWSURL,
		getServerHost
	};
};
