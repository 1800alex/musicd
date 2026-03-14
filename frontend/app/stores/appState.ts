import { ref } from "vue";
import type { Ref } from "vue";
import { defineStore } from "pinia";
import type { Track, Playlist } from "~/types";

export const RepeatModeOff = "Off";
export const RepeatModeOne = "One";
export const RepeatModeAll = "All";
export const RepeatMode = {
	Off: RepeatModeOff,
	One: RepeatModeOne,
	All: RepeatModeAll
};

export type RepeatModeType = "Off" | "One" | "All";

export interface IAppState {
	Loaded: Ref<boolean>;
	SetLoaded: (val: boolean) => void;
	AudioElement: Ref<HTMLAudioElement | null>;
	SetAudioElement: (el: HTMLAudioElement | null) => void;
	RemoteControl: Ref<any>;
	SetRemoteControl: (rc: any) => void;
	Shuffle: Ref<boolean>;
	SetShuffle: (val: boolean) => void;
	RepeatMode: Ref<RepeatModeType>;
	SetRepeatMode: (val: RepeatModeType) => void;
	Volume: Ref<number>;
	SetVolume: (val: number) => void;
	Muted: Ref<boolean>;
	SetMuted: (val: boolean) => void;
	VolumeBeforeMute: Ref<number>;
	SetVolumeBeforeMute: (val: number) => void;
	CurrentTrack: Ref<Track | null>;
	SetCurrentTrack: (track: Track | null) => void;
	CurrentPlaylist: Ref<Playlist | null>;
	SetCurrentPlaylist: (playlist: Playlist | null) => void;
	Playlists: Ref<Playlist[]>;
	SetPlaylists: (playlists: Playlist[]) => void;
	TemporaryQueue: Ref<Track[]>;
	SetTemporaryQueue: (tracks: Track[]) => void;
	OriginalQueue: Ref<Track[]>;
	SetOriginalQueue: (tracks: Track[]) => void;
	Queue: Ref<Track[]>;
	SetQueue: (tracks: Track[]) => void;
	AddToQueue: (track: Track) => void;
	IsPlaying: Ref<boolean>;
	SetIsPlaying: (playing: boolean) => void;
	CurrentTime: Ref<number>;
	SetCurrentTime: (time: number) => void;
	Duration: Ref<number>;
	SetDuration: (duration: number) => void;
	PageSize: Ref<number>;
	SetPageSize: (size: number) => void;
	IsScanning: Ref<boolean>;
	SetIsScanning: (val: boolean) => void;
}

const useAppState = defineStore("musicPlayer", (): IAppState => {
	const Loaded = ref(false);
	const AudioElement = ref<HTMLAudioElement | null>(null);
	const RemoteControl = ref<any>(null);
	const Shuffle = ref(false);
	const RepeatMode = ref(<RepeatModeType>RepeatModeOff);
	const Volume = ref(50);
	const Muted = ref(false);
	const VolumeBeforeMute = ref(50);
	const CurrentTrack = ref<Track | null>(null);
	const CurrentPlaylist = ref<Playlist | null>(null);
	const TemporaryQueue = ref<Track[]>([]);
	const OriginalQueue = ref<Track[]>([]);
	const Queue = ref<Track[]>([]);
	const IsPlaying = ref(false);
	const CurrentTime = ref(0);
	const Duration = ref(0);
	const PageSize = ref(25);
	const Playlists = ref<Playlist[]>([]);
	const IsScanning = ref(false);

	function SetLoaded(val: boolean) {
		Loaded.value = val;
	}

	function SetAudioElement(el: HTMLAudioElement | null) {
		AudioElement.value = el;
	}

	function SetRemoteControl(rc: any) {
		RemoteControl.value = rc;
	}

	function SetShuffle(val: boolean) {
		Shuffle.value = val;
	}

	function SetRepeatMode(val: RepeatModeType) {
		RepeatMode.value = val as RepeatModeType;
	}

	function SetVolume(val: number) {
		Volume.value = val;
	}

	function SetMuted(val: boolean) {
		Muted.value = val;
	}

	function SetVolumeBeforeMute(val: number) {
		VolumeBeforeMute.value = val;
	}

	function SetCurrentTrack(track: Track | null) {
		CurrentTrack.value = track;
	}

	function SetCurrentPlaylist(playlist: Playlist | null) {
		CurrentPlaylist.value = playlist;
	}

	function SetPlaylists(playlists: Playlist[]) {
		Playlists.value = playlists;
	}

	function SetTemporaryQueue(tracks: Track[]) {
		TemporaryQueue.value = tracks;
	}

	function SetOriginalQueue(tracks: Track[]) {
		OriginalQueue.value = [...tracks];
	}

	function SetQueue(tracks: Track[]) {
		Queue.value = tracks;
	}

	function AddToQueue(track: Track) {
		Queue.value.push(track);
	}

	function SetIsPlaying(playing: boolean) {
		IsPlaying.value = playing;
	}

	function SetCurrentTime(time: number) {
		CurrentTime.value = time;
	}

	function SetDuration(duration: number) {
		Duration.value = duration;
	}

	function SetPageSize(size: number) {
		PageSize.value = size;
	}

	function SetIsScanning(val: boolean) {
		IsScanning.value = val;
	}

	return {
		Loaded,
		SetLoaded,
		AudioElement,
		SetAudioElement,
		RemoteControl,
		SetRemoteControl,
		Shuffle,
		SetShuffle,
		RepeatMode,
		SetRepeatMode,
		Volume,
		SetVolume,
		Muted,
		SetMuted,
		VolumeBeforeMute,
		SetVolumeBeforeMute,
		CurrentTrack,
		SetCurrentTrack,
		CurrentPlaylist,
		SetCurrentPlaylist,
		Playlists,
		SetPlaylists,
		TemporaryQueue,
		SetTemporaryQueue,
		OriginalQueue,
		SetOriginalQueue,
		Queue,
		SetQueue,
		AddToQueue,
		IsPlaying,
		SetIsPlaying,
		CurrentTime,
		SetCurrentTime,
		Duration,
		SetDuration,
		PageSize,
		SetPageSize,
		IsScanning,
		SetIsScanning
	};
});

export default useAppState;
