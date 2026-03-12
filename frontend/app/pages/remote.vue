<script setup lang="ts">
import { ref, onMounted } from "vue";
import RemoteControlService from "~/services/remoteControl.service";
import useAppState from "~/stores/appState";
import { useBackendURL } from "~/composables/useBackendURL";

definePageMeta({
	layout: "default" // Use the full layout with all features
});

const appState = useAppState();
const router = useRouter();
const { getHTTPURL, getServerHost } = useBackendURL();

// Session state
const sessions = ref<any[]>([]);
const selectedSession = ref<string>("");
const loading = ref(false);
const connected = ref(!!appState.RemoteControl);

// UI State
const showSessionModal = ref(false);

const getSessionIdFromUrl = () => {
	if (typeof window === "undefined") return "";
	const params = new URLSearchParams(window.location.search);
	return params.get("session") || "";
};

const fetchSessions = async () => {
	try {
		loading.value = true;
		const response = await fetch(getHTTPURL("/api/sessions"));
		if (response.ok) {
			sessions.value = await response.json();
		}
	} catch (err) {
		console.error("Error fetching sessions:", err);
	} finally {
		loading.value = false;
	}
};

const connectToSession = async (sessionId: string) => {
	if (!sessionId) return;

	try {
		disconnectSession();

		const rc = new RemoteControlService(getServerHost(), sessionId);

		rc.onConnected = () => {
			connected.value = true;
			selectedSession.value = sessionId;
			localStorage.setItem("remoteControlSessionId", sessionId);
			// Store session name for the banner in default.vue
			const sess = sessions.value.find((s: any) => s.id === sessionId);
			localStorage.setItem("remoteControlSessionName", sess?.name || sessionId);
			showSessionModal.value = false;
			// Navigate away from the session picker to the main UI
			router.push("/");
		};

		rc.onDisconnected = () => {
			connected.value = false;
		};

		rc.onStateUpdate = (state: any) => {
			// Sync remote state to local app state for display in player bar
			appState.SetIsPlaying(state.is_playing);
			appState.SetCurrentTrack(state.current_track);
			appState.SetCurrentTime(state.current_time);
			appState.SetDuration(state.duration);
			appState.SetVolume(state.volume);
			appState.SetMuted(state.muted);
			appState.SetShuffle(state.shuffle);
			appState.SetRepeatMode(state.repeat_mode);
			appState.SetQueue(state.queue || []);
			appState.SetTemporaryQueue(state.temporary_queue || []);
			appState.SetCurrentPlaylist(state.current_playlist || null);
		};

		rc.onError = (err: any) => {
			console.error("Remote error:", err);
		};

		// Set the remote control service in the store so PlayerService routes through it
		appState.SetRemoteControl(rc);
	} catch (err) {
		console.error("Error connecting:", err);
	}
};

const disconnectSession = () => {
	const rc = appState.RemoteControl;
	if (rc) {
		rc.disconnect();
	}
	appState.SetRemoteControl(null);
	connected.value = false;
	selectedSession.value = "";
	localStorage.removeItem("remoteControlSessionId");
	// Clear app state
	appState.SetCurrentTrack(null);
	appState.SetIsPlaying(false);
};

const disconnectAndShowModal = () => {
	disconnectSession();
	showSessionModal.value = true;
};

const onMountedInit = async () => {
	await fetchSessions();

	// Already connected from a previous visit (connection persists across navigations)
	if (appState.RemoteControl) {
		router.push("/");
		return;
	}

	// Try to connect from URL param
	const sessionFromUrl = getSessionIdFromUrl();
	if (sessionFromUrl) {
		await connectToSession(sessionFromUrl);
		return;
	}

	// Or try to restore previous session
	const savedSessionId = localStorage.getItem("remoteControlSessionId");
	if (savedSessionId) {
		await connectToSession(savedSessionId);
		return;
	}

	// Otherwise show modal
	showSessionModal.value = true;
};

onMounted(() => {
	onMountedInit();
});

// NOTE: Do NOT disconnect on unmount — the connection persists across page navigations.
// Disconnect is only triggered by the user clicking "Disconnect" (here or in the navbar).
</script>

<template>
	<div v-if="!connected" class="modal is-active">
		<div class="modal-background" @click="showSessionModal = false"></div>
		<div class="modal-card">
			<header class="modal-card-head">
				<p class="modal-card-title">Connect to Session</p>
			</header>
			<section class="modal-card-body">
				<div v-if="loading" class="content">
					<p>Loading sessions...</p>
				</div>

				<div v-else-if="sessions.length === 0" class="content">
					<p class="has-text-danger">No active music sessions found.</p>
					<p class="is-size-7 has-text-grey mt-3">Make sure the music app is open on another device.</p>
				</div>

				<div v-else class="content">
					<p>Select a session to control:</p>
					<div class="mt-4">
						<button
							v-for="sess in sessions.filter((s) => s.has_player)"
							:key="sess.id"
							class="button is-primary is-fullwidth mb-2"
							@click="connectToSession(sess.id)"
						>
							<font-awesome-icon icon="fa-music" class="mr-2"></font-awesome-icon>
							{{ sess.name }}
						</button>
					</div>

					<div v-if="sessions.some((s) => !s.has_player)" class="mt-4">
						<p class="is-size-7 has-text-grey">Offline Sessions:</p>
						<div class="mt-2">
							<p
								v-for="sess in sessions.filter((s) => !s.has_player)"
								:key="sess.id"
								class="is-size-7 has-text-grey"
							>
								{{ sess.name }} (offline)
							</p>
						</div>
					</div>
				</div>
			</section>
			<footer class="modal-card-foot">
				<button class="button" @click="fetchSessions">Refresh</button>
			</footer>
		</div>
	</div>

	<!-- Full UI when connected as remote controller -->
	<div v-else>
		<!-- Show that we're in remote control mode -->
		<div class="remote-control-bar">
			<div class="is-flex is-justify-content-space-between is-align-items-center">
				<div>
					<font-awesome-icon icon="fa-mobile-alt" class="mr-2"></font-awesome-icon>
					Controlling: <strong>{{ sessions.find((s) => s.id === selectedSession)?.name }}</strong>
				</div>
				<div class="is-flex" style="gap: 0.5rem">
					<button class="page-action-btn page-action-btn-secondary" @click="showSessionModal = true">
						<font-awesome-icon icon="fa-exchange-alt"></font-awesome-icon>
						Switch
					</button>
					<button class="page-action-btn page-action-btn-secondary" @click="disconnectAndShowModal">
						<font-awesome-icon icon="fa-sign-out-alt"></font-awesome-icon>
						Disconnect
					</button>
				</div>
			</div>
		</div>

		<!-- Render the normal slot - the app will use the remote control service for playback -->
		<slot />
	</div>
</template>

<style scoped>
.modal-card {
	min-width: 350px;
}

.button.is-fullwidth {
	width: 100%;
}

.remote-control-bar {
	background-color: var(--clr-surface-elevated);
	color: var(--clr-text-primary);
	padding: 0.75rem 1.25rem;
	border-bottom: 1px solid var(--clr-surface-higher);
}
</style>
