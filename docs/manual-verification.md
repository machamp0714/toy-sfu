# Manual verification procedure

Run once after any change to `internal/room` or `internal/signaling` that
touches track forwarding or renegotiation.

## Setup

1. `cd ~/repo/toy-sfu && go run ./cmd/sfu`
2. Open `http://localhost:8080` in two separate browser windows (not just
   tabs — some browsers throttle camera access in background tabs).
   Grant camera/mic permission in both.

## Two-participant fan-out

3. In each window, confirm the "Local" video shows your own camera.
4. Within a few seconds, confirm each window's "Remote" section shows the
   *other* window's video and plays audio (mute one to avoid feedback).
5. Open `chrome://webrtc-internals` in each window and confirm:
   - `iceConnectionState` reaches `completed` and `connectionState` reaches
     `connected`
   - the selected `candidate-pair` entry has non-zero
     `bytesReceived`/`bytesSent` that keep increasing
   - `inbound-rtp` shows `packetsReceived` increasing for both an audio and
     a video stream

## Late joiner

6. Open a third window/tab to `http://localhost:8080` and grant permission.
7. Confirm the third participant's "Remote" section immediately shows
   *both* existing participants' video, without needing a page reload on
   the other two windows.
8. Confirm the first two windows now also show the third participant's
   video in their "Remote" section.

## Leave

9. Close the third window.
10. Confirm the remaining two windows keep working normally. The third
    participant's video element in the other two windows will freeze on
    its last frame rather than disappearing — this is the documented
    simplification in the design doc (senders for a departed participant
    are not explicitly removed/renegotiated away).

## If something doesn't work

Use the existing `signalingState`/`iceConnectionState` diagnostic tables in
the `WebRTC 1.0 - Real-time Communication Between Browsers` Obsidian note
("3つの状態機械を統合して読む" section) to narrow down whether the problem is
in signaling (stuck in `have-local-offer`/`have-remote-offer`) or in the
media/ICE layer (`checking`→`failed`).
