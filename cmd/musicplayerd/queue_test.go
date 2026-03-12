package main

import (
	"fmt"
	"testing"
)

// makeTracks returns n tracks with IDs "1".."n" and titles "Track 1".."Track n".
func makeTracks(n int) []Track {
	tracks := make([]Track, n)
	for i := range tracks {
		id := fmt.Sprintf("%d", i+1)
		tracks[i] = Track{
			ID:    id,
			Title: "Track " + id,
		}
	}
	return tracks
}

func trackIDs(tracks []Track) []string {
	ids := make([]string, len(tracks))
	for i, t := range tracks {
		ids[i] = t.ID
	}
	return ids
}

func containsAll(tracks []Track, ids []string) bool {
	have := map[string]bool{}
	for _, t := range tracks {
		have[t.ID] = true
	}
	for _, id := range ids {
		if !have[id] {
			return false
		}
	}
	return len(tracks) == len(ids)
}

// ── PlayTracks ──────────────────────────────────────────────────────────────

func TestPlayTracks_NoShuffle(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(5)

	// Play starting at index 2 (Track 3).
	selected := q.PlayTracks(tracks, 2)

	if selected.ID != "3" {
		t.Fatalf("expected selected track ID=3, got %s", selected.ID)
	}
	if q.currentTrack == nil || q.currentTrack.ID != "3" {
		t.Fatal("currentTrack should be Track 3")
	}

	// History should be tracks before startIdx: [1, 2]
	histIDs := trackIDs(q.history)
	if len(histIDs) != 2 || histIDs[0] != "1" || histIDs[1] != "2" {
		t.Fatalf("expected history [1,2], got %v", histIDs)
	}

	// Queue should be tracks after startIdx: [4, 5]
	qIDs := trackIDs(q.queue)
	if len(qIDs) != 2 || qIDs[0] != "4" || qIDs[1] != "5" {
		t.Fatalf("expected queue [4,5], got %v", qIDs)
	}
}

func TestPlayTracks_FirstTrack(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(3)

	selected := q.PlayTracks(tracks, 0)

	if selected.ID != "1" {
		t.Fatalf("expected selected track ID=1, got %s", selected.ID)
	}
	if len(q.history) != 0 {
		t.Fatalf("expected empty history, got %v", trackIDs(q.history))
	}
	qIDs := trackIDs(q.queue)
	if len(qIDs) != 2 || qIDs[0] != "2" || qIDs[1] != "3" {
		t.Fatalf("expected queue [2,3], got %v", qIDs)
	}
}

func TestPlayTracks_LastTrack(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(3)

	selected := q.PlayTracks(tracks, 2)

	if selected.ID != "3" {
		t.Fatalf("expected selected track ID=3, got %s", selected.ID)
	}
	if len(q.queue) != 0 {
		t.Fatalf("expected empty queue, got %v", trackIDs(q.queue))
	}
	histIDs := trackIDs(q.history)
	if len(histIDs) != 2 || histIDs[0] != "1" || histIDs[1] != "2" {
		t.Fatalf("expected history [1,2], got %v", histIDs)
	}
}

func TestPlayTracks_Shuffle(t *testing.T) {
	q := NewQueue()
	q.Shuffle = true
	tracks := makeTracks(5)

	selected := q.PlayTracks(tracks, 2)

	if selected.ID != "3" {
		t.Fatalf("expected selected track ID=3, got %s", selected.ID)
	}
	if q.currentTrack == nil || q.currentTrack.ID != "3" {
		t.Fatal("currentTrack should be Track 3")
	}

	// History should be empty when shuffling.
	if len(q.history) != 0 {
		t.Fatalf("expected empty history with shuffle, got %v", trackIDs(q.history))
	}

	// Queue should contain all other tracks (order doesn't matter).
	if !containsAll(q.queue, []string{"1", "2", "4", "5"}) {
		t.Fatalf("expected queue to contain [1,2,4,5], got %v", trackIDs(q.queue))
	}
}

func TestPlayTracks_SingleTrack(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(1)

	selected := q.PlayTracks(tracks, 0)

	if selected.ID != "1" {
		t.Fatalf("expected selected track ID=1, got %s", selected.ID)
	}
	if len(q.queue) != 0 {
		t.Fatalf("expected empty queue, got %v", trackIDs(q.queue))
	}
	if len(q.history) != 0 {
		t.Fatalf("expected empty history, got %v", trackIDs(q.history))
	}
}

// ── Next ────────────────────────────────────────────────────────────────────

func TestNext_Basic(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0)

	// Current=1, Queue=[2,3], History=[]
	next := q.Next()
	if next == nil || next.ID != "2" {
		t.Fatalf("expected next track ID=2, got %v", next)
	}
	if q.currentTrack.ID != "2" {
		t.Fatalf("expected currentTrack=2, got %s", q.currentTrack.ID)
	}

	// History should now contain track 1.
	histIDs := trackIDs(q.history)
	if len(histIDs) != 1 || histIDs[0] != "1" {
		t.Fatalf("expected history [1], got %v", histIDs)
	}

	// Queue should have track 3.
	qIDs := trackIDs(q.queue)
	if len(qIDs) != 1 || qIDs[0] != "3" {
		t.Fatalf("expected queue [3], got %v", qIDs)
	}
}

func TestNext_EmptyQueue_RepeatOff(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(1)
	q.PlayTracks(tracks, 0)

	// Current=1, Queue=[], History=[]
	next := q.Next()
	if next != nil {
		t.Fatalf("expected nil, got %v", next)
	}
	if q.currentTrack != nil {
		t.Fatalf("expected nil currentTrack, got %v", q.currentTrack)
	}

	// Track 1 should be in history.
	if len(q.history) != 1 || q.history[0].ID != "1" {
		t.Fatalf("expected history [1], got %v", trackIDs(q.history))
	}
}

func TestNext_EmptyQueue_RepeatAll(t *testing.T) {
	q := NewQueue()
	q.RepeatMode = "All"
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0)

	// Play through all tracks.
	q.Next() // 1→2
	q.Next() // 2→3

	// Now queue is empty, history=[1,2], current=3.
	// Next should loop back.
	next := q.Next()
	if next == nil {
		t.Fatal("expected a track from repeat-all loop, got nil")
	}

	// All 3 tracks should be accounted for (1 current + queue).
	allIDs := []string{q.currentTrack.ID}
	for _, t := range q.queue {
		allIDs = append(allIDs, t.ID)
	}
	if len(allIDs) != 3 {
		t.Fatalf("expected 3 tracks total, got %d: %v", len(allIDs), allIDs)
	}

	// History should be empty (all tracks moved to queue+current).
	if len(q.history) != 0 {
		t.Fatalf("expected empty history after repeat-all loop, got %v", trackIDs(q.history))
	}
}

func TestNext_EmptyQueue_RepeatAll_Shuffle(t *testing.T) {
	q := NewQueue()
	q.RepeatMode = "All"
	q.Shuffle = true
	tracks := makeTracks(10)
	q.PlayTracks(tracks, 0)

	// Play through all tracks.
	for i := 0; i < 9; i++ {
		q.Next()
	}

	// Now queue is empty. Next should loop with shuffle.
	next := q.Next()
	if next == nil {
		t.Fatal("expected a track from repeat-all shuffle loop, got nil")
	}

	// All 10 tracks should be present (current + queue).
	seen := map[string]bool{q.currentTrack.ID: true}
	for _, tr := range q.queue {
		seen[tr.ID] = true
	}
	if len(seen) != 10 {
		t.Fatalf("expected 10 unique tracks, got %d", len(seen))
	}
}

// ── Previous ────────────────────────────────────────────────────────────────

func TestPrevious_Basic(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 1) // Current=2, History=[1], Queue=[3]

	prev := q.Previous()
	if prev == nil || prev.ID != "1" {
		t.Fatalf("expected previous track ID=1, got %v", prev)
	}
	if q.currentTrack.ID != "1" {
		t.Fatalf("expected currentTrack=1, got %s", q.currentTrack.ID)
	}

	// Old current (2) should be at front of queue.
	qIDs := trackIDs(q.queue)
	if len(qIDs) != 2 || qIDs[0] != "2" || qIDs[1] != "3" {
		t.Fatalf("expected queue [2,3], got %v", qIDs)
	}

	// History should be empty.
	if len(q.history) != 0 {
		t.Fatalf("expected empty history, got %v", trackIDs(q.history))
	}
}

func TestPrevious_EmptyHistory(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0) // Current=1, History=[], Queue=[2,3]

	prev := q.Previous()
	if prev != nil {
		t.Fatalf("expected nil when history empty, got %v", prev)
	}

	// State should be unchanged.
	if q.currentTrack.ID != "1" {
		t.Fatalf("currentTrack should still be 1, got %s", q.currentTrack.ID)
	}
}

func TestPrevious_NoCurrent(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(1)
	q.PlayTracks(tracks, 0)
	q.Next() // current=nil, history=[1], queue=[]

	prev := q.Previous()
	if prev == nil || prev.ID != "1" {
		t.Fatalf("expected previous=1, got %v", prev)
	}

	// No current was pushed to queue since currentTrack was nil.
	if len(q.queue) != 0 {
		t.Fatalf("expected empty queue, got %v", trackIDs(q.queue))
	}
}

// ── Next then Previous round-trip ───────────────────────────────────────────

func TestNextThenPrevious_RoundTrip(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(4)
	q.PlayTracks(tracks, 0) // Current=1, Queue=[2,3,4]

	q.Next() // Current=2, History=[1], Queue=[3,4]
	q.Next() // Current=3, History=[1,2], Queue=[4]

	prev := q.Previous() // Current=2, History=[1], Queue=[3,4]
	if prev == nil || prev.ID != "2" {
		t.Fatalf("expected previous=2, got %v", prev)
	}

	// Queue should be [3, 4] (current 3 was pushed back when we went to 2).
	qIDs := trackIDs(q.queue)
	if len(qIDs) != 2 || qIDs[0] != "3" || qIDs[1] != "4" {
		t.Fatalf("expected queue [3,4], got %v", qIDs)
	}

	// History should be [1].
	histIDs := trackIDs(q.history)
	if len(histIDs) != 1 || histIDs[0] != "1" {
		t.Fatalf("expected history [1], got %v", histIDs)
	}

	// Going next again should give us track 3.
	next := q.Next()
	if next == nil || next.ID != "3" {
		t.Fatalf("expected next=3, got %v", next)
	}
}

// ── OnTrackEnd ──────────────────────────────────────────────────────────────

func TestOnTrackEnd_RepeatOne(t *testing.T) {
	q := NewQueue()
	q.RepeatMode = "One"
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0) // Current=1, Queue=[2,3]

	result := q.OnTrackEnd()
	if result == nil || result.ID != "1" {
		t.Fatalf("expected repeat-one to return current track 1, got %v", result)
	}

	// State should be unchanged.
	if q.currentTrack.ID != "1" {
		t.Fatalf("currentTrack should still be 1, got %s", q.currentTrack.ID)
	}
	if len(q.queue) != 2 {
		t.Fatalf("queue should still have 2 tracks, got %d", len(q.queue))
	}
	if len(q.history) != 0 {
		t.Fatalf("history should still be empty, got %d", len(q.history))
	}
}

func TestOnTrackEnd_RepeatOff(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(2)
	q.PlayTracks(tracks, 0) // Current=1, Queue=[2]

	result := q.OnTrackEnd()
	if result == nil || result.ID != "2" {
		t.Fatalf("expected next track 2, got %v", result)
	}
	if q.currentTrack.ID != "2" {
		t.Fatalf("currentTrack should be 2, got %s", q.currentTrack.ID)
	}

	// Track 1 should be in history.
	if len(q.history) != 1 || q.history[0].ID != "1" {
		t.Fatalf("expected history [1], got %v", trackIDs(q.history))
	}
}

func TestOnTrackEnd_RepeatAll_FullLoop(t *testing.T) {
	q := NewQueue()
	q.RepeatMode = "All"
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0)

	// Play through entire playlist.
	played := []string{q.currentTrack.ID}
	for i := 0; i < 2; i++ {
		result := q.OnTrackEnd()
		if result == nil {
			t.Fatalf("expected track on iteration %d, got nil", i)
		}
		played = append(played, result.ID)
	}

	// All 3 tracks played in order.
	if played[0] != "1" || played[1] != "2" || played[2] != "3" {
		t.Fatalf("expected play order [1,2,3], got %v", played)
	}

	// Now at end. OnTrackEnd should loop.
	result := q.OnTrackEnd()
	if result == nil {
		t.Fatal("expected repeat-all to loop, got nil")
	}

	// All 3 tracks accounted for.
	seen := map[string]bool{q.currentTrack.ID: true}
	for _, tr := range q.queue {
		seen[tr.ID] = true
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 tracks after loop, got %d", len(seen))
	}
}

// ── Add / Clear ─────────────────────────────────────────────────────────────

func TestAdd(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(2)
	q.PlayTracks(tracks, 0) // Current=1, Queue=[2]

	extra := Track{ID: "99", Title: "Extra"}
	q.Add(extra)

	qIDs := trackIDs(q.queue)
	if len(qIDs) != 2 || qIDs[0] != "2" || qIDs[1] != "99" {
		t.Fatalf("expected queue [2,99], got %v", qIDs)
	}
}

func TestClear(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(5)
	q.PlayTracks(tracks, 0)

	q.Clear()

	if len(q.queue) != 0 {
		t.Fatalf("expected empty queue after clear, got %v", trackIDs(q.queue))
	}
	// Current and history should be unaffected.
	if q.currentTrack == nil || q.currentTrack.ID != "1" {
		t.Fatal("currentTrack should still be set after clear")
	}
}

// ── Full playthrough ────────────────────────────────────────────────────────

func TestFullPlaythrough_NoRepeat(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(4)
	q.PlayTracks(tracks, 0)

	played := []string{q.currentTrack.ID}
	for {
		next := q.Next()
		if next == nil {
			break
		}
		played = append(played, next.ID)
	}

	expected := []string{"1", "2", "3", "4"}
	if len(played) != len(expected) {
		t.Fatalf("expected %d tracks played, got %d: %v", len(expected), len(played), played)
	}
	for i, id := range expected {
		if played[i] != id {
			t.Fatalf("expected track %s at position %d, got %s", id, i, played[i])
		}
	}

	// After playthrough, current should be nil.
	if q.currentTrack != nil {
		t.Fatalf("expected nil currentTrack after full playthrough, got %s", q.currentTrack.ID)
	}

	// History should contain all 4 tracks.
	if len(q.history) != 4 {
		t.Fatalf("expected 4 tracks in history, got %d", len(q.history))
	}
}

func TestFullPlaythrough_RepeatAll(t *testing.T) {
	q := NewQueue()
	q.RepeatMode = "All"
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0)

	// Play through 2 full loops (6 tracks total).
	played := []string{q.currentTrack.ID}
	for i := 0; i < 6; i++ {
		next := q.Next()
		if next == nil {
			t.Fatalf("expected track on iteration %d, got nil", i)
		}
		played = append(played, next.ID)
	}

	// Should have played 7 tracks total (1 initial + 6 nexts).
	if len(played) != 7 {
		t.Fatalf("expected 7 plays, got %d: %v", len(played), played)
	}

	// First 3 should be in order.
	if played[0] != "1" || played[1] != "2" || played[2] != "3" {
		t.Fatalf("first loop should be [1,2,3], got %v", played[:3])
	}

	// Second loop (tracks 3-6): all 3 tracks should appear.
	secondLoop := played[3:6]
	seen := map[string]bool{}
	for _, id := range secondLoop {
		seen[id] = true
	}
	if len(seen) != 3 {
		t.Fatalf("second loop should contain all 3 tracks, got %v", secondLoop)
	}
}

// ── State snapshot ──────────────────────────────────────────────────────────

func TestState_Snapshot(t *testing.T) {
	q := NewQueue()
	q.Shuffle = true
	q.RepeatMode = "All"
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0)
	q.Next()

	state := q.State()

	if state.CurrentTrack == nil {
		t.Fatal("expected non-nil currentTrack in state")
	}
	if state.Shuffle != true {
		t.Fatal("expected shuffle=true in state")
	}
	if state.RepeatMode != "All" {
		t.Fatalf("expected repeatMode=All, got %s", state.RepeatMode)
	}

	// Mutating returned slices should not affect queue state.
	state.Queue = nil
	state.History = nil
	qState2 := q.State()
	if qState2.Queue == nil || qState2.History == nil {
		t.Fatal("state mutation leaked into queue internals")
	}
}

// ── Multiple Previous calls ─────────────────────────────────────────────────

func TestMultiplePrevious(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(5)
	q.PlayTracks(tracks, 0)

	// Advance to track 4.
	q.Next() // →2
	q.Next() // →3
	q.Next() // →4

	// Go back 3 times.
	p1 := q.Previous()
	p2 := q.Previous()
	p3 := q.Previous()

	if p1.ID != "3" || p2.ID != "2" || p3.ID != "1" {
		t.Fatalf("expected previous sequence [3,2,1], got [%s,%s,%s]", p1.ID, p2.ID, p3.ID)
	}

	// Should be back at track 1 with queue [2,3,4,5].
	if q.currentTrack.ID != "1" {
		t.Fatalf("expected currentTrack=1, got %s", q.currentTrack.ID)
	}
	qIDs := trackIDs(q.queue)
	if len(qIDs) != 4 || qIDs[0] != "2" || qIDs[1] != "3" || qIDs[2] != "4" || qIDs[3] != "5" {
		t.Fatalf("expected queue [2,3,4,5], got %v", qIDs)
	}

	// No more history.
	p4 := q.Previous()
	if p4 != nil {
		t.Fatalf("expected nil when history exhausted, got %s", p4.ID)
	}
}

// ── PlayTracks resets state ─────────────────────────────────────────────────

func TestPlayTracks_ResetsOldState(t *testing.T) {
	q := NewQueue()
	tracks1 := makeTracks(3)
	q.PlayTracks(tracks1, 0)
	q.Next() // current=2, history=[1], queue=[3]

	// Start a completely new playlist.
	tracks2 := makeTracks(2)
	tracks2[0].ID = "A"
	tracks2[1].ID = "B"
	q.PlayTracks(tracks2, 0)

	if q.currentTrack.ID != "A" {
		t.Fatalf("expected currentTrack=A, got %s", q.currentTrack.ID)
	}
	qIDs := trackIDs(q.queue)
	if len(qIDs) != 1 || qIDs[0] != "B" {
		t.Fatalf("expected queue [B], got %v", qIDs)
	}
	// History should be empty (fresh playlist).
	if len(q.history) != 0 {
		t.Fatalf("expected empty history on new playlist, got %v", trackIDs(q.history))
	}
}

// ── TemporaryQueue (full context for controller UI) ─────────────────────────

func TestTemporaryQueue_FullContext(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(5)
	q.PlayTracks(tracks, 0)

	// Initial: current=1, queue=[2,3,4,5], history=[]
	tq := q.TemporaryQueue()
	tqIDs := trackIDs(tq)
	if len(tqIDs) != 5 {
		t.Fatalf("expected 5 tracks in TemporaryQueue, got %d: %v", len(tqIDs), tqIDs)
	}
	// Should be: [1,2,3,4,5] (history + current + queue)
	if tqIDs[0] != "1" || tqIDs[1] != "2" || tqIDs[4] != "5" {
		t.Fatalf("expected TemporaryQueue [1,2,3,4,5], got %v", tqIDs)
	}

	// After next: current=2, history=[1], queue=[3,4,5]
	q.Next()
	tq = q.TemporaryQueue()
	tqIDs = trackIDs(tq)
	// Should be: [1, 2, 3, 4, 5] (history=[1] + current=2 + queue=[3,4,5])
	if len(tqIDs) != 5 || tqIDs[0] != "1" || tqIDs[1] != "2" || tqIDs[2] != "3" {
		t.Fatalf("expected TemporaryQueue [1,2,3,4,5] after next, got %v", tqIDs)
	}

	// After previous: current=1, history=[], queue=[2,3,4,5]
	q.Previous()
	tq = q.TemporaryQueue()
	tqIDs = trackIDs(tq)
	if len(tqIDs) != 5 || tqIDs[0] != "1" || tqIDs[1] != "2" {
		t.Fatalf("expected TemporaryQueue [1,2,3,4,5] after prev, got %v", tqIDs)
	}
}

func TestTemporaryQueue_Shuffle(t *testing.T) {
	q := NewQueue()
	q.Shuffle = true
	tracks := makeTracks(5)
	q.PlayTracks(tracks, 2) // Select track 3

	tq := q.TemporaryQueue()

	// Should have all 5 tracks (current + shuffled queue)
	if len(tq) != 5 {
		t.Fatalf("expected 5 tracks, got %d", len(tq))
	}
	// First should be current track (3) since history is empty
	if tq[0].ID != "3" {
		t.Fatalf("expected first track in TQ to be current=3, got %s", tq[0].ID)
	}
	// All tracks present
	seen := map[string]bool{}
	for _, tr := range tq {
		seen[tr.ID] = true
	}
	if len(seen) != 5 {
		t.Fatalf("expected 5 unique tracks, got %d", len(seen))
	}
}

// ── SetShuffle mid-playback ──────────────────────────────────────────────────

func TestSetShuffle_MidPlayback_ReshufflesQueue(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(10)
	q.PlayTracks(tracks, 0) // current=1, queue=[2..10] in order

	// Queue should be sequential.
	qIDs := trackIDs(q.queue)
	for i, id := range qIDs {
		expected := fmt.Sprintf("%d", i+2)
		if id != expected {
			t.Fatalf("expected sequential queue, got %v", qIDs)
		}
	}

	// Toggle shuffle ON mid-playback.
	q.SetShuffle(true)

	// Queue should still contain the same 9 tracks.
	if len(q.queue) != 9 {
		t.Fatalf("expected 9 tracks in queue after shuffle, got %d", len(q.queue))
	}
	seen := map[string]bool{}
	for _, tr := range q.queue {
		seen[tr.ID] = true
	}
	for i := 2; i <= 10; i++ {
		id := fmt.Sprintf("%d", i)
		if !seen[id] {
			t.Fatalf("missing track %s after shuffle", id)
		}
	}

	// With 9 tracks, it's astronomically unlikely to stay in original order.
	// Check that at least one track is out of sequential position.
	reshuffled := false
	for i, tr := range q.queue {
		expected := fmt.Sprintf("%d", i+2)
		if tr.ID != expected {
			reshuffled = true
			break
		}
	}
	if !reshuffled {
		t.Fatal("queue appears to still be in sequential order after SetShuffle(true)")
	}

	// Current track should be unaffected.
	if q.currentTrack.ID != "1" {
		t.Fatalf("expected currentTrack=1, got %s", q.currentTrack.ID)
	}
}

func TestSetShuffle_Off_RestoresFromCurrentPosition(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(10)

	// Start with shuffle ON from track 1.
	q.Shuffle = true
	q.PlayTracks(tracks, 0) // current=1, queue=shuffled [2..10]

	// Advance a couple tracks through the shuffled queue.
	q.Next()
	q.Next()
	// Now current is some random track, say track X at original index N.

	currentID := q.currentTrack.ID

	// Turn shuffle OFF — should rebuild queue/history based on
	// the current track's position in the original list.
	q.SetShuffle(false)

	// Find current track's position in original list.
	currentOrigIdx := -1
	for i, tr := range tracks {
		if tr.ID == currentID {
			currentOrigIdx = i
			break
		}
	}

	// Queue should be everything AFTER currentOrigIdx in original order.
	expectedQueue := tracks[currentOrigIdx+1:]
	qIDs := trackIDs(q.queue)
	expectedIDs := trackIDs(expectedQueue)
	if len(qIDs) != len(expectedIDs) {
		t.Fatalf("expected queue %v, got %v", expectedIDs, qIDs)
	}
	for i := range qIDs {
		if qIDs[i] != expectedIDs[i] {
			t.Fatalf("expected queue %v, got %v", expectedIDs, qIDs)
		}
	}

	// History should be everything BEFORE currentOrigIdx in original order.
	expectedHist := tracks[:currentOrigIdx]
	hIDs := trackIDs(q.history)
	expectedHistIDs := trackIDs(expectedHist)
	if len(hIDs) != len(expectedHistIDs) {
		t.Fatalf("expected history %v, got %v", expectedHistIDs, hIDs)
	}
	for i := range hIDs {
		if hIDs[i] != expectedHistIDs[i] {
			t.Fatalf("expected history %v, got %v", expectedHistIDs, hIDs)
		}
	}

	// Next track should be the one right after current in original order.
	if currentOrigIdx < len(tracks)-1 {
		next := q.Next()
		if next == nil || next.ID != tracks[currentOrigIdx+1].ID {
			t.Fatalf("expected next=%s, got %v", tracks[currentOrigIdx+1].ID, next)
		}
	}
}

func TestTemporaryQueue_Empty(t *testing.T) {
	q := NewQueue()
	tq := q.TemporaryQueue()
	if len(tq) != 0 {
		t.Fatalf("expected empty TemporaryQueue, got %d tracks", len(tq))
	}
}

func TestTemporaryQueue_AfterFullPlaythrough(t *testing.T) {
	q := NewQueue()
	tracks := makeTracks(3)
	q.PlayTracks(tracks, 0)
	q.Next() // →2
	q.Next() // →3
	q.Next() // → nil (end of queue)

	tq := q.TemporaryQueue()
	// current=nil, history=[1,2,3], queue=[]
	tqIDs := trackIDs(tq)
	if len(tqIDs) != 3 || tqIDs[0] != "1" || tqIDs[1] != "2" || tqIDs[2] != "3" {
		t.Fatalf("expected TemporaryQueue [1,2,3] from history, got %v", tqIDs)
	}
}
