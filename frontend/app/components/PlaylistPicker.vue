<script setup lang="ts">
import { ref, computed, nextTick } from "vue";
import type { Playlist } from "~/types";

const props = defineProps<{
	playlists: Playlist[];
}>();

const emit = defineEmits<{
	select: [playlist: Playlist];
}>();

const filter = ref("");
const filterInput = ref<HTMLInputElement | null>(null);

const filtered = computed(() => {
	const q = filter.value.toLowerCase().trim();
	if (!q) return props.playlists;
	return props.playlists.filter((p) => p.name.toLowerCase().includes(q));
});

const showSearch = computed(() => props.playlists.length > 5);

const focusInput = () => {
	nextTick(() => filterInput.value?.focus());
};

defineExpose({ focusInput });
</script>

<template>
	<div class="playlist-picker">
		<div v-if="showSearch" class="playlist-picker-search">
			<span class="playlist-picker-search-icon">
				<font-awesome-icon icon="fa-search" />
			</span>
			<input
				ref="filterInput"
				v-model="filter"
				class="playlist-picker-search-input"
				type="text"
				placeholder="Filter playlists..."
			/>
			<button v-if="filter" class="playlist-picker-search-clear" @click="filter = ''">
				<font-awesome-icon icon="fa-times" />
			</button>
		</div>
		<div class="playlist-picker-list">
			<button
				v-for="playlist in filtered"
				:key="playlist.id"
				class="playlist-picker-item"
				@click="emit('select', playlist)"
			>
				<font-awesome-icon icon="fa-music" class="mr-2" />
				{{ playlist.name }}
			</button>
			<div v-if="filtered.length === 0" class="playlist-picker-empty">No playlists found</div>
		</div>
	</div>
</template>

<style scoped>
.playlist-picker {
	display: flex;
	flex-direction: column;
	max-height: 280px;
}

.playlist-picker-search {
	display: flex;
	align-items: center;
	background: var(--clr-surface-higher);
	border-radius: 16px;
	padding: 0 0.5rem;
	height: 32px;
	margin: 0.4rem 0.5rem;
	flex-shrink: 0;
}

.playlist-picker-search-icon {
	color: var(--clr-text-muted);
	font-size: 0.75rem;
	margin-right: 0.4rem;
	flex-shrink: 0;
}

.playlist-picker-search-input {
	background: transparent;
	border: none;
	outline: none;
	color: var(--clr-text-primary);
	font-size: 0.85rem;
	flex: 1;
	min-width: 0;
}

.playlist-picker-search-input::placeholder {
	color: var(--clr-text-muted);
}

.playlist-picker-search-clear {
	background: transparent;
	border: none;
	color: var(--clr-text-muted);
	cursor: pointer;
	font-size: 0.7rem;
	padding: 2px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
}

.playlist-picker-search-clear:hover {
	color: var(--clr-text-primary);
}

.playlist-picker-list {
	overflow-y: auto;
	flex: 1;
	min-height: 0;
}

.playlist-picker-item {
	display: flex;
	align-items: center;
	width: 100%;
	padding: 0.5rem 1rem;
	border: none;
	background: transparent;
	color: var(--clr-text-primary);
	font-size: 0.85rem;
	cursor: pointer;
	text-align: left;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.playlist-picker-item:hover,
.playlist-picker-item:active {
	background: var(--clr-surface-higher);
}

.playlist-picker-empty {
	padding: 0.75rem 1rem;
	color: var(--clr-text-muted);
	font-size: 0.85rem;
	text-align: center;
}
</style>
