import { app, BrowserWindow } from "electron";
import serve from "electron-serve";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isDev = "development" === process.env.NODE_ENV;
const useSandbox = true;
const useWebSecurity = false; // !isDev;

// __dirname = electron/dist/, so go up two levels to reach project root
const rootDir = path.join(__dirname, "../..");

if (!isDev) {
	// Register the app:// protocol to serve .output/public
	const staticDir = path.join(rootDir, ".output/public");
	console.log("Serving static files from:", staticDir);
	serve({ directory: staticDir });
}

async function createWindow() {
	const iconPath = path.join(rootDir, "assets/ios/AppIcon-512@2x.png");
	const preloadPath = path.join(__dirname, "preload.js");

	const win = new BrowserWindow({
		width: 1280,
		height: 800,
		minWidth: 800,
		minHeight: 600,
		icon: iconPath,
		webPreferences: {
			preload: preloadPath,
			contextIsolation: true,
			nodeIntegration: false,
			sandbox: useSandbox,
			webSecurity: useWebSecurity
		}
	});

	if (isDev) {
		console.log("Loading dev server at http://localhost:3000/ui/");
		await win.loadURL("http://localhost:3000/ui/");
		win.webContents.openDevTools();
	} else {
		console.log("Loading production app");
		try {
			await win.loadURL("app://-/");
		} catch (err) {
			console.error("Failed to load app via electron-serve:", err);
			// Fallback to file protocol
			await win.loadFile(path.join(rootDir, ".output/public/index.html"));
		}
	}
}

app.whenReady().then(createWindow);

app.on("window-all-closed", () => {
	if (process.platform !== "darwin") {
		app.quit();
	}
});

app.on("activate", () => {
	if (0 === BrowserWindow.getAllWindows().length) {
		createWindow();
	}
});
