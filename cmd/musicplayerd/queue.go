package main

import (
	"math/rand"
	"sync"
)

// Queue manages the playback queue, history, shuffle, and repeat state.
// All methods are safe for concurrent use. No I/O is performed — callers
// use the returned *Track to decide what to play.
type Queue struct {
	mu             sync.RWMutex
	currentTrack   *Track
	priorityQueue  []Track // explicit user-queued tracks; consumed before context queue
	queue          []Track // context tracks (album/playlist/artist)
	history        []Track
	originalTracks []Track // original ordered list from PlayTracks, for restoring order
	Shuffle        bool
	RepeatMode     string // "Off" | "One" | "All"
}

// QueueState is a snapshot of the queue for broadcasting.
type QueueState struct {
	CurrentTrack  *Track
	PriorityQueue []Track // user-added tracks, play before context queue
	Queue         []Track // context tracks
	History       []Track
	Shuffle       bool
	RepeatMode    string
}

// NewQueue creates a queue with default settings.
func NewQueue() *Queue {
	return &Queue{
		priorityQueue: []Track{},
		queue:         []Track{},
		history:       []Track{},
		RepeatMode:    "Off",
	}
}

// PlayTracks sets up playback from a track list starting at startIdx.
// If shuffle is enabled, remaining tracks are shuffled.
// Returns the selected track.
func (q *Queue) PlayTracks(tracks []Track, startIdx int) Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Starting fresh playback always clears the user-managed priority queue.
	q.priorityQueue = []Track{}

	selected := tracks[startIdx]
	q.originalTracks = append([]Track{}, tracks...)

	if q.Shuffle {
		rest := make([]Track, 0, len(tracks)-1)
		for i, t := range tracks {
			if i != startIdx {
				rest = append(rest, t)
			}
		}
		rand.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
		q.queue = rest
		q.history = []Track{}
	} else {
		q.history = append([]Track{}, tracks[:startIdx]...)
		q.queue = append([]Track{}, tracks[startIdx+1:]...)
	}

	q.currentTrack = &selected
	return selected
}

// Next advances to the next track in the queue.
// Priority-queued tracks (explicitly added by the user) are consumed first,
// then the context queue. Pushes the current track to history.
// On empty queue: Repeat All rebuilds from history, Repeat Off returns nil.
func (q *Queue) Next() *Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Push current track to history.
	if q.currentTrack != nil {
		q.history = append(q.history, *q.currentTrack)
		if len(q.history) > 200 {
			q.history = q.history[len(q.history)-200:]
		}
	}

	// User-queued tracks take priority over the context queue.
	if len(q.priorityQueue) > 0 {
		next := q.priorityQueue[0]
		q.priorityQueue = q.priorityQueue[1:]
		q.currentTrack = &next
		return &next
	}

	if len(q.queue) == 0 {
		if q.RepeatMode == "All" && len(q.history) > 0 {
			all := append([]Track{}, q.history...)
			q.history = []Track{}
			if q.Shuffle {
				rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
			}
			q.queue = all[1:]
			next := all[0]
			q.currentTrack = &next
			return &next
		}
		q.currentTrack = nil
		return nil
	}

	next := q.queue[0]
	q.queue = q.queue[1:]
	q.currentTrack = &next
	return &next
}

// Previous goes back to the previous track.
// Pops from history and pushes the current track to the front of the queue.
// Returns nil if history is empty.
func (q *Queue) Previous() *Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.history) == 0 {
		return nil
	}

	prev := q.history[len(q.history)-1]
	q.history = q.history[:len(q.history)-1]

	if q.currentTrack != nil {
		q.queue = append([]Track{*q.currentTrack}, q.queue...)
	}

	q.currentTrack = &prev
	return &prev
}

// OnTrackEnd handles the natural end of a track.
// Repeat One: returns the current track without modifying state.
// Otherwise: delegates to Next().
func (q *Queue) OnTrackEnd() *Track {
	q.mu.RLock()
	repeat := q.RepeatMode
	current := q.currentTrack
	q.mu.RUnlock()

	if repeat == "One" && current != nil {
		t := *current
		return &t
	}
	return q.Next()
}

// Add appends a track to the priority queue (user-managed).
// Priority tracks play before context tracks, in the order they were added.
func (q *Queue) Add(track Track) {
	q.mu.Lock()
	q.priorityQueue = append(q.priorityQueue, track)
	q.mu.Unlock()
}

// Clear empties both the priority queue and the context queue.
func (q *Queue) Clear() {
	q.mu.Lock()
	q.priorityQueue = []Track{}
	q.queue = []Track{}
	q.mu.Unlock()
}

// SetCurrent sets the current track (called after mpv loads the file).
func (q *Queue) SetCurrent(track Track) {
	q.mu.Lock()
	q.currentTrack = &track
	q.mu.Unlock()
}

// Current returns the current track, or nil.
func (q *Queue) Current() *Track {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.currentTrack
}

// SetShuffle sets the shuffle mode.
// When turning shuffle ON, the remaining queue is immediately reshuffled.
// When turning shuffle OFF, the queue and history are rebuilt from the
// current track's position in the original list — so next/previous
// continue in the original order from where you are now.
func (q *Queue) SetShuffle(on bool) {
	q.mu.Lock()
	q.Shuffle = on
	if on && len(q.queue) > 1 {
		rand.Shuffle(len(q.queue), func(i, j int) { q.queue[i], q.queue[j] = q.queue[j], q.queue[i] })
	} else if !on && q.currentTrack != nil && len(q.originalTracks) > 0 {
		// Find the current track's position in the original list.
		currentIdx := -1
		for i, t := range q.originalTracks {
			if t.ID == q.currentTrack.ID {
				currentIdx = i
				break
			}
		}
		if currentIdx >= 0 {
			q.history = append([]Track{}, q.originalTracks[:currentIdx]...)
			q.queue = append([]Track{}, q.originalTracks[currentIdx+1:]...)
		}
	}
	q.mu.Unlock()
}

// SetRepeatMode sets the repeat mode ("Off", "One", "All").
func (q *Queue) SetRepeatMode(mode string) {
	q.mu.Lock()
	q.RepeatMode = mode
	q.mu.Unlock()
}

// TemporaryQueue returns the full playback context in order:
// history + [currentTrack] + priorityQueue + queue.
// This matches the frontend's TemporaryQueue format used for display and navigation.
func (q *Queue) TemporaryQueue() []Track {
	q.mu.RLock()
	defer q.mu.RUnlock()

	total := len(q.history) + len(q.priorityQueue) + len(q.queue)
	if q.currentTrack != nil {
		total++
	}
	tq := make([]Track, 0, total)
	tq = append(tq, q.history...)
	if q.currentTrack != nil {
		tq = append(tq, *q.currentTrack)
	}
	tq = append(tq, q.priorityQueue...)
	tq = append(tq, q.queue...)
	return tq
}

// IsShuffle returns the current shuffle state.
func (q *Queue) IsShuffle() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.Shuffle
}

// GetRepeatMode returns the current repeat mode.
func (q *Queue) GetRepeatMode() string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.RepeatMode
}

// State returns a snapshot of the queue state for broadcasting.
func (q *Queue) State() QueueState {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return QueueState{
		CurrentTrack:  q.currentTrack,
		PriorityQueue: append([]Track{}, q.priorityQueue...),
		Queue:         append([]Track{}, q.queue...),
		History:       append([]Track{}, q.history...),
		Shuffle:       q.Shuffle,
		RepeatMode:    q.RepeatMode,
	}
}
