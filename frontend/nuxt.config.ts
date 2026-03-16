// https://nuxt.com/docs/api/configuration/nuxt-config
import type { NitroConfig } from "nitropack";

const apiKey = "";

const baseUrl = process.env.ELECTRON_BUILD || process.env.CAPACITOR_BUILD ? "/" : "/ui/";
const apiPrefix = ""; // TODO: fetch public.apiURL from the runtime config (use node env?)

const routeRules: NitroConfig["routeRules"] = {};
const devProxy: Record<string, any> = {};
let viteProxy: Record<string, any> = {};
// Exposed via runtimeConfig so WS connections can bypass the Nuxt dev proxy entirely
let devBackendHost = "";

if ("development" === process.env.NODE_ENV) {
	// You can automatically set the IP via CLI, like so: backend="localhost:8080" corepack yarn dev
	let backendURL = process.env.backend || "http://127.0.0.1:8080";

	if (backendURL.startsWith("http://") || backendURL.startsWith("https://")) {
		backendURL = backendURL.replace(/\/+$/, ""); // Remove trailing slashes
	} else {
		backendURL = `http://${backendURL}`;
	}

	devBackendHost = backendURL.replace(/^https?:\/\//, "");
	console.log(`Re-routing backend routes to ${backendURL}`);
	routeRules["/api/**"] = { proxy: { to: `${backendURL}/api/**` }, cors: true };
	routeRules["/static/**"] = { proxy: { to: `${backendURL}/static/**` }, cors: true };
	routeRules["/ui/api/**"] = { proxy: { to: `${backendURL}/api/**` }, cors: true };
	routeRules["/ui/static/**"] = { proxy: { to: `${backendURL}/static/**` }, cors: true };

	// WebSocket routes: routeRules proxy strips hop-by-hop headers (Connection: Upgrade) so it
	// can't handle WS upgrades. Vite's proxy registers a proper Node.js 'upgrade' event listener
	// and handles the WebSocket handshake correctly. Target is the server root; the full request
	// path (/api/ws/player, /api/ws/control/:id) is forwarded as-is by http-proxy.
	viteProxy["/api/ws"] = { target: backendURL, ws: true, changeOrigin: true };

	// Keep devProxy as a fallback (may work depending on Nitro version)
	const wsBackendURL = backendURL.replace(/^http/, "ws");
	devProxy["/api/ws"] = { target: wsBackendURL, ws: true, changeOrigin: true };
}

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
	ssr: false,
	pages: true,
	telemetry: {
		enabled: false
	},

	eslint: {
		config: {
			standalone: false
		}
	},

	runtimeConfig: {
		public: {
			baseURL: baseUrl,
			apiPrefix: apiPrefix,
			apiKey: apiKey,
			// Backend server URL for Capacitor native builds.
			// Set via NUXT_PUBLIC_BACKEND_URL env var or override at runtime.
			// When running as a web app this is ignored (relative URLs work).
			backendURL: process.env.NUXT_PUBLIC_BACKEND_URL || ""
		}
	},

	routeRules: routeRules,

	nitro: {
		devProxy: devProxy
	},

	vite: {
		// In dev, WS connections bypass the Nuxt proxy and connect directly to the backend.
		// Vite runs in middleware mode so vite.server.proxy doesn't intercept WS upgrades,
		// and Nitro's devProxy strips hop-by-hop headers (Connection: Upgrade) breaking WS.
		// Instead we inject the backend host as a compile-time constant so client WS code
		// can connect directly to the backend, bypassing the proxy entirely.
		define: { __DEV_BACKEND_HOST__: JSON.stringify(devBackendHost) },
		server: {
			proxy: viteProxy
		}
	},

	app: {
		baseURL: baseUrl,
		// Fix page icon to use basepath
		head: {
			link: [
				{ rel: "shortcut icon", type: "image/x-icon", href: `${baseUrl}favicon.ico` },
				{ rel: "apple-touch-icon", sizes: "180x180", href: `${baseUrl}apple-touch-icon.png` },
				{ rel: "manifest", href: `${baseUrl}manifest.webmanifest` }
			],
			meta: [
				{ name: "mobile-web-app-capable", content: "yes" },
				{ name: "apple-mobile-web-app-status-bar-style", content: "black-translucent" },
				{ name: "apple-mobile-web-app-title", content: "Music Player" },
				{ name: "apple-mobile-web-app-status-bar", content: "black-translucent" },
				{ name: "viewport", content: "width=device-width, initial-scale=1, viewport-fit=cover" },
				{ name: "theme-color", content: "#1a1a1a" },
				// Safe area support for notch
				{ name: "format-detection", content: "telephone=no" }
			]
		}
	},

	components: [
		{
			path: "@/components",
			pathPrefix: false
		}
	],
	modules: ["@pinia/nuxt", "@nuxt/fonts", "@nuxt/eslint"],
	css: [
		"@/assets/css/theme.scss",
		"bulma",
		"buefy/dist/css/buefy.css",
		"@fortawesome/fontawesome-svg-core/styles.css",
		"@/assets/css/site.css"
	],
	// pinia: {
	// 	storesDirs: ["./stores/**"]
	// },

	typescript: {
		typeCheck: false
	}
});
