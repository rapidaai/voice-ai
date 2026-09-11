package webrtc_internal

import "testing"

func TestOutputAudioFramePreservesResponseAcrossPeerGeneration(t *testing.T) {
	state := &SessionState{}
	mediaSessionID := state.StartMediaSession()
	playback := &OutputPlayback{ID: "response-a", MediaSessionID: mediaSessionID, Generation: 7, HasAudio: true}
	queued := []OutputAudioFrame{
		{Audio: []byte{1, 2}, Playback: playback},
		{Playback: playback, Terminal: true},
	}
	nextMediaSessionID := state.StartMediaSession()
	state.SetPeerConnected(true)
	state.StampPacedAssistantFrame(nextMediaSessionID)
	if !state.CanWritePacedAssistantFrame() {
		t.Fatal("new peer's paced frame must be writable")
	}
	for _, frame := range queued {
		if frame.Playback.ID != "response-a" || frame.Playback.Generation != 7 {
			t.Fatal("queued frames lost their response generation")
		}
		if state.IsActiveMediaSession(frame.Playback.MediaSessionID) {
			t.Fatal("restamping the pacer must not make old queued response frames current")
		}
	}
	queued[0].Playback.Failed = true
	if !queued[1].Terminal || !queued[1].Playback.Failed {
		t.Fatal("a dropped audio frame must invalidate its response's terminal marker")
	}
	next := OutputAudioFrame{Playback: &OutputPlayback{ID: "response-b", MediaSessionID: nextMediaSessionID, Generation: 8}}
	if next.Playback.Failed {
		t.Fatal("an earlier response failure must not invalidate a new response")
	}
}
