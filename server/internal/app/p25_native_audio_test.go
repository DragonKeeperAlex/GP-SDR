package app

import (
	"encoding/binary"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// Produced by the standalone C++ header/packet encoder test, not the Go parser.
const nativeAudioFixture = "475053445032350001000300010000000100000064000000c80000002301ff800100feff"

func TestP25NativeAudioWireAndFailClosed(t *testing.T) {
	packet, err := hex.DecodeString(nativeAudioFixture)
	if err != nil {
		t.Fatal(err)
	}
	f, err := decodeP25NativeAudio(packet, 2, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if f.Native.CallID != "fixture:1" || f.Native.GroupID != 100 || f.Native.SourceID != 200 || f.Native.Slot != -1 || len(f.Samples) != 2 || f.Samples[1] != -2 {
		t.Fatalf("bad native frame: %+v", f)
	}
	for _, mutation := range []string{"unknown-identity", "unknown-crypto", "encrypted", "decrypted-encrypted"} {
		p := append([]byte(nil), packet...)
		switch mutation {
		case "unknown-identity":
			p[10] = 2
		case "unknown-crypto":
			p[10] = 1
		case "encrypted":
			p[10] = 7
			p[31] = 0x84
		case "decrypted-encrypted":
			p[31] = 0x84
		}
		frame, err := decodeP25NativeAudio(p, 0, "fixture")
		if err != nil || len(frame.Samples) != 0 || frame.Control != "drain" {
			t.Fatal("unsafe PCM was not suppressed", mutation, err)
		}
	}
	for _, p := range [][]byte{packet[:16], append(append([]byte(nil), packet...), 1)} {
		if _, err := decodeP25NativeAudio(p, 0, "fixture"); err == nil {
			t.Fatal("malformed packet accepted")
		}
	}
	packet[8] = 2
	if _, err = decodeP25NativeAudio(packet, 0, "fixture"); err == nil {
		t.Fatal("unknown version accepted")
	}
}
func TestP25NativeRecorderSplitsCallIdentityAndRejectsMismatchedMetadata(t *testing.T) {
	p, calls := pcmRecorderFixture(t)
	p.requireNative = true
	p.accept(pcmPacket(0, 1, 1))
	if len(p.recordings) != 0 {
		t.Fatal("required-native recorder accepted bare PCM")
	}
	packet, _ := hex.DecodeString(nativeAudioFixture)
	packet[30] = byte(*calls[0].TDMASlot)
	binary.LittleEndian.PutUint32(packet[20:24], calls[0].Grant.GroupID)
	binary.LittleEndian.PutUint32(packet[24:28], calls[0].Grant.SourceID)
	accept := func(seq, call uint32) {
		binary.LittleEndian.PutUint32(packet[12:16], seq)
		binary.LittleEndian.PutUint32(packet[16:20], call)
		f, err := decodeP25NativeAudio(packet, 0, "session")
		if err != nil {
			t.Fatal(err)
		}
		f.ReceivedAt = pcmFixtureTime()
		p.accept(f)
	}
	accept(2, 1)
	accept(3, 2)
	p.finishAll()
	events := p.runtime.Events.Recent(10)
	if len(events) != 2 || events[0].P25NativeCallID == events[1].P25NativeCallID || events[0].P25NativeCallID == "" {
		t.Fatal("native calls merged or lost identity")
	}
	binary.LittleEndian.PutUint32(packet[20:24], 9999)
	accept(4, 3)
	if len(p.recordings) != 0 {
		t.Fatal("polled identity replaced a mismatching native talkgroup")
	}
}

func TestOP25NativeEmitterTransport(t *testing.T) {
	emitter := os.Getenv("GPSDR_TEST_NATIVE_EMITTER")
	if emitter == "" {
		t.Skip("set GPSDR_TEST_NATIVE_EMITTER for linked OP25 socket-only check")
	}
	t.Setenv("GPSDR_P25_TAGGED_AUDIO", "1")
	hub := NewAudioHub()
	pcm, unsubscribe := hub.Subscribe()
	defer unsubscribe()
	recorder, stopRecorder := hub.subscribeP25()
	defer stopRecorder()
	m := &OP25Manager{audioHub: hub}
	if err := m.startOP25Audio(3); err != nil {
		t.Fatal(err)
	}
	defer m.closeOP25Audio()
	for i := 0; i < 3; i++ {
		cmd := exec.Command(emitter, strconv.Itoa(23456+i*10))
		cmd.Env = append(os.Environ(), "GPSDR_P25_TAGGED_AUDIO=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatal(err, string(output))
		}
	}
	for i := 0; i < 6; i++ {
		select {
		case f := <-pcm:
			if f.Native == nil || f.Native.Encrypted || len(f.Samples) != 2 {
				t.Fatal("invalid playback frame")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("missing attributed PCM")
		}
	}
	select {
	case <-pcm:
		t.Fatal("encrypted/control frame reached playback")
	case <-time.After(100 * time.Millisecond):
	}
	for i := 0; i < 12; i++ {
		select {
		case f := <-recorder:
			if f.Native == nil {
				t.Fatal("native facts lost")
			}
		case <-time.After(time.Second):
			t.Fatal("missing recorder frame")
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.audioFrames != 6 || m.nativeAudioPackets != 12 || m.nativeAudioRejected != 3 {
		t.Fatalf("counters frames=%d tagged=%d suppressed=%d", m.audioFrames, m.nativeAudioPackets, m.nativeAudioRejected)
	}
}

func TestP25NativeIdentitySurvivesCheckpointRecovery(t *testing.T) {
	p, calls := pcmRecorderFixture(t)
	p.requireNative = true
	frame := pcmPacket(0, 1, 7)
	frame.Native = &P25NativeAudio{CallID: "fixture-session:9", GroupID: calls[0].Grant.GroupID, SourceID: calls[0].Grant.SourceID, Slot: *calls[0].TDMASlot, Algorithm: 0x80, Valid: true, CryptoKnown: true}
	p.accept(frame)
	p.checkpoint()
	events, err := NewEventStore(filepath.Join(p.runtime.dataDirectory, "Data"))
	if err != nil {
		t.Fatal(err)
	}
	recovered := &Runtime{dataDirectory: p.runtime.dataDirectory, Events: events}
	if err := recovered.recoverP25CallSpools(); err != nil {
		t.Fatal(err)
	}
	all := events.Recent(10)
	if len(all) != 1 || all[0].P25NativeCallID != "fixture-session:9" || !all[0].Recovered {
		t.Fatal("native identity lost across crash checkpoint recovery")
	}
}

func TestOP25AudioRejectsPreviousSessionDatagrams(t *testing.T) {
	hub := NewAudioHub()
	frames, stop := hub.Subscribe()
	defer stop()
	m := &OP25Manager{audioHub: hub, audioSession: "current"}
	frame := AudioFrame{ChannelID: "p25-stream-0", SampleRate: 8000, Samples: []int16{1, 2}, ReceivedAt: time.Now()}
	m.publishOP25Audio("previous", frame, true, false)
	m.rejectOP25Audio("previous", true)
	select {
	case <-frames:
		t.Fatal("stale session audio leaked")
	default:
	}
	m.publishOP25Audio("current", frame, true, false)
	<-frames
	m.closeOP25Audio()
	m.publishOP25Audio("current", frame, true, false)
	select {
	case <-frames:
		t.Fatal("closed session audio leaked")
	default:
	}
	if m.audioFrames != 1 || m.nativeAudioPackets != 1 || m.nativeAudioRejected != 0 {
		t.Fatal("stale frames polluted counters")
	}
}

func TestP25TaggedBrowserRoutingAndMixerPreserveRecording(t *testing.T) {
	m := &OP25Manager{taggedTalkgroupAudio: true, audioSession: "active"}
	r := &Runtime{op25: m, mixer: []MixerChannel{{ID: "p25:100", TalkgroupID: ptr(uint32(100)), Muted: true, Volume: .8}, {ID: "p25:101", TalkgroupID: ptr(uint32(101)), Volume: .7}}}
	frame := AudioFrame{ChannelID: "p25-stream-0", TransportSession: "active", Native: &P25NativeAudio{CallID: "receiver-session:1", GroupID: 100, Valid: true, CryptoKnown: true, Algorithm: 0x80}}
	if id := r.liveAudioIdentifier(frame); id != "p25:100:receiver-session:1" {
		t.Fatal(id)
	}
	if frame.ChannelID != "p25-stream-0" {
		t.Fatal("browser routing changed recorder stream")
	}
	frame.Native.CallID = "receiver-session:2"
	if id := r.liveAudioIdentifier(frame); id != "p25:100:receiver-session:2" {
		t.Fatal("same-group calls share a browser clock")
	}
	if _, err := r.UpdateMixer("p25:101", nil, ptr(true), nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := r.UpdateMixer("p25:101", nil, ptr(false), nil, nil); err != nil {
		t.Fatal(err)
	}
	if !r.mixer[0].Muted {
		t.Fatal("solo off erased another talkgroup's manual mute")
	}
	if _, err := r.UpdateMixer("p25:100", ptr(false), nil, ptr(.3), ptr(-.5)); err != nil {
		t.Fatal("native controls must not call decoder mute", err)
	}
	if r.mixer[0].Volume != .3 || r.mixer[0].Pan != -.5 {
		t.Fatal("talkgroup controls not stored")
	}
	frame.Native.Encrypted = true
	if r.liveAudioIdentifier(frame) != "" {
		t.Fatal("encrypted frame escaped")
	}
	frame.Native.Encrypted = false
	frame.TransportSession = "previous"
	if r.liveAudioIdentifier(frame) != "" {
		t.Fatal("buffered old-session audio escaped after replacement")
	}
	frame.TransportSession = "active"
	m.taggedTalkgroupAudio = false
	if r.liveAudioIdentifier(frame) != "p25-stream-0" {
		t.Fatal("multi-system legacy fallback changed")
	}
}
