<script setup lang="ts">
import { ref, watch } from "vue";

interface Props {
	isOpen: boolean;
	isLoading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
	isLoading: false
});

const emit = defineEmits<{
	close: [];
	create: [name: string, location: string, customPath: string];
}>();

// Form state
const playlistName = ref("");
const playlistLocation = ref("playlists");
const playlistCustomPath = ref("");

// Reset form when modal closes
watch(
	() => props.isOpen,
	(newValue) => {
		if (!newValue) {
			playlistName.value = "";
			playlistLocation.value = "playlists";
			playlistCustomPath.value = "";
		}
	}
);

const handleCreate = () => {
	if (!playlistName.value.trim()) {
		return;
	}

	emit("create", playlistName.value.trim(), playlistLocation.value, playlistCustomPath.value);
};

const handleClose = () => {
	emit("close");
};

const handleKeyUp = (event: KeyboardEvent) => {
	if (event.key === "Enter" && playlistName.value.trim()) {
		handleCreate();
	}
};
</script>

<template>
	<div v-if="isOpen" class="playlist-modal">
		<div class="playlist-modal-content">
			<h3 class="title is-5 has-text-white">Create New Playlist</h3>
			<div class="field">
				<label class="label has-text-white">Playlist Name</label>
				<div class="control">
					<input
						v-model="playlistName"
						class="input"
						type="text"
						placeholder="Enter playlist name..."
						@keyup="handleKeyUp"
					/>
				</div>
			</div>
			<div class="field">
				<label class="label has-text-white">Location</label>
				<div class="control">
					<div class="select is-fullwidth">
						<select v-model="playlistLocation">
							<option value="playlists">Playlists folder</option>
							<option value="music">Music folder</option>
							<option value="custom">Custom folder</option>
						</select>
					</div>
				</div>
			</div>
			<div v-if="playlistLocation === 'custom'" class="field">
				<label class="label has-text-white">Custom Path</label>
				<div class="control">
					<input
						v-model="playlistCustomPath"
						class="input"
						type="text"
						placeholder="folder/subfolder"
					/>
				</div>
			</div>
			<div class="playlist-modal-buttons">
				<button class="button" @click="handleClose">Cancel</button>
				<button
					class="button is-primary"
					:class="{ 'is-loading': isLoading }"
					:disabled="!playlistName.trim()"
					@click="handleCreate"
				>
					Create
				</button>
			</div>
		</div>
	</div>
</template>

<style scoped>
.playlist-modal {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
}

.playlist-modal-content {
	background: var(--clr-surface);
	border-radius: 8px;
	padding: 2rem;
	width: 90%;
	max-width: 400px;
	box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

.field {
	margin-bottom: 1.5rem;
}

.label {
	display: block;
	margin-bottom: 0.75rem;
	font-weight: 500;
	font-size: 0.95rem;
}

.input,
.select select {
	width: 100%;
	padding: 0.75rem;
	border: 1px solid var(--clr-surface-higher);
	border-radius: 4px;
	background: var(--clr-surface-elevated);
	color: var(--clr-text-primary);
	font-size: 1rem;
	line-height: 1.5;
	height: auto;
	min-height: 2.5rem;
}

.select {
	position: relative;
}

.select select {
	appearance: none;
	padding-right: 2.5rem;
}

.input:focus,
.input:focus,
.select select:focus {
	outline: none;
	border-color: var(--clr-primary);
	box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.1);
}

.playlist-modal-buttons {
	display: flex;
	gap: 1rem;
	justify-content: flex-end;
	margin-top: 1.5rem;
}

.button {
	padding: 0.75rem 1.5rem;
	border: none;
	border-radius: 4px;
	font-weight: 500;
	cursor: pointer;
	transition: all 0.2s ease;
}

.button:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.is-primary {
	background: var(--clr-primary);
	color: white;
}

.is-primary:hover:not(:disabled) {
	background: var(--clr-primary-hover, #0066cc);
}
</style>
