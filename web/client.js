const ROOM = "default";

const pc = new RTCPeerConnection({
  iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
});

const localStreamPromise = navigator.mediaDevices.getUserMedia({
  audio: true,
  video: true,
});

localStreamPromise
  .then((stream) => {
    document.getElementById("local").srcObject = stream;
  })
  .catch((err) => {
    console.error("getUserMedia failed", err);
  });

const remotesEl = document.getElementById("remotes");

pc.ontrack = (event) => {
  const stream = event.streams[0];
  let video = document.getElementById(`remote-${stream.id}`);
  if (!video) {
    video = document.createElement("video");
    video.id = `remote-${stream.id}`;
    video.autoplay = true;
    video.playsInline = true;
    remotesEl.appendChild(video);
  }
  video.srcObject = stream;
};

const ws = new WebSocket(`ws://${location.host}/ws`);

ws.onopen = () => {
  ws.send(JSON.stringify({ type: "join", room: ROOM }));
};

pc.onicecandidate = (event) => {
  if (!event.candidate) {
    return;
  }
  ws.send(
    JSON.stringify({
      type: "ice-candidate",
      candidate: event.candidate.toJSON(),
    })
  );
};

ws.onmessage = async (event) => {
  const msg = JSON.parse(event.data);

  switch (msg.type) {
    case "offer": {
      await pc.setRemoteDescription({ type: "offer", sdp: msg.sdp });

      const stream = await localStreamPromise;
      for (const track of stream.getTracks()) {
        const alreadyAdded = pc
          .getSenders()
          .some((sender) => sender.track === track);
        if (!alreadyAdded) {
          pc.addTrack(track, stream);
        }
      }

      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      ws.send(JSON.stringify({ type: "answer", sdp: answer.sdp }));
      break;
    }
    case "ice-candidate": {
      if (msg.candidate) {
        await pc.addIceCandidate(msg.candidate);
      }
      break;
    }
    default:
      console.warn("unexpected message type", msg.type);
  }
};

ws.onclose = () => {
  console.log("signaling socket closed");
};
