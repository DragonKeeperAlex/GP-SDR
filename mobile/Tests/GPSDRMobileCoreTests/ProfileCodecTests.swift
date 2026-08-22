import Foundation
import Testing
@testable import GPSDRMobileCore

@Test func desktopProfileRoundTrip() throws {
    let profile = ScanProfile(
        id: "portable-gmrs",
        name: "Portable GMRS",
        channels: [ChannelDefinition(id: "gmrs-20", name: "GMRS 20",
                                     frequencyHz: 462_675_000, bandwidthHz: 12_500,
                                     mode: "nfm")]
    )
    let codec = ProfileCodec()
    let data = try codec.encode(profile)
    #expect(try codec.decode(data) == profile)
}

@Test func rejectsOutOfRangeFrequency() {
    let profile = ScanProfile(
        id: "invalid", name: "Invalid",
        channels: [ChannelDefinition(id: "bad", name: "Bad", frequencyHz: -1,
                                     bandwidthHz: 12_500, mode: "nfm")]
    )
    #expect(throws: ProfileCodecError.invalidFrequency) {
        try ProfileCodec().validate(profile)
    }
}

@Test func simulatorProducesClearlySyntheticIQ() async throws {
    let receiver = SimulatedReceiver()
    let stream = try await receiver.start()
    var iterator = stream.makeAsyncIterator()
    let frame = try await iterator.next()
    #expect(frame?.unsignedInterleavedIQ.count == 8_192)
    #expect(frame?.sampleRateHz == 2_400_000)
    await receiver.stop()
    let telemetry = await receiver.telemetry()
    #expect(telemetry.receivedSamples > 0)
}

@Test func spectrumOutputIsBounded() {
    let bytes = (0..<2_048).map { UInt8(truncatingIfNeeded: $0) }
    let frame = IQFrame(unsignedInterleavedIQ: bytes, sampleRateHz: 2_400_000,
                        centerFrequencyHz: 100_000_000, sequence: 1)
    let points = SpectrumAnalyzer().analyze(frame, bins: 128)
    #expect(points.count == 128)
    #expect(points.allSatisfy { $0.powerDBFS.isFinite })
}

@Test func eventStoreAppendsAndReturnsNewestFirst() async throws {
    let directory = FileManager.default.temporaryDirectory
        .appendingPathComponent(UUID().uuidString, isDirectory: true)
    defer { try? FileManager.default.removeItem(at: directory) }
    let store = EventStore(directory: directory)
    let first = TransmissionEvent(durationSeconds: 1, frequencyHz: 100_000_000,
                                  bandwidthHz: 12_500, signalDBFS: -40, noiseDBFS: -80,
                                  modulation: "nfm", deviceID: "test")
    let second = TransmissionEvent(durationSeconds: 2, frequencyHz: 101_000_000,
                                   bandwidthHz: 12_500, signalDBFS: -35, noiseDBFS: -80,
                                   modulation: "nfm", deviceID: "test")
    try await store.append(first)
    try await store.append(second)
    let events = try await store.recent()
    #expect(events.map(\.id) == [second.id, first.id])
}

@Test func scanPlannerExpandsRangesAndHonorsPriority() throws {
    let profile = ScanProfile(
        id: "scan", name: "Scan",
        ranges: [ScanRange(id: "r", name: "Range", startHz: 100_000_000,
                           endHz: 100_025_000, stepHz: 12_500,
                           dwellMilliseconds: 250, preferredMode: "nfm")],
        channels: [ChannelDefinition(id: "priority", name: "Priority",
                                     frequencyHz: 162_550_000, bandwidthHz: 12_500,
                                     mode: "nfm", priority: 10)]
    )
    let targets = try ScanPlanner().targets(for: profile)
    #expect(targets.count == 4)
    #expect(targets.first?.id == "priority")
}

@Test func nfmDemodulatorProducesAudioAtBoundedRate() {
    let sampleRate = 240_000
    var bytes: [UInt8] = []
    for index in 0..<4_800 {
        let phase = 2 * Double.pi * 12_000 * Double(index) / Double(sampleRate)
        bytes.append(UInt8(clamping: Int(127.5 + cos(phase) * 100)))
        bytes.append(UInt8(clamping: Int(127.5 + sin(phase) * 100)))
    }
    let iq = IQFrame(unsignedInterleavedIQ: bytes, sampleRateHz: sampleRate,
                     centerFrequencyHz: 100_000_000, sequence: 0)
    let audio = AnalogDemodulator().demodulate(iq, mode: .nfm, outputRateHz: 48_000)
    #expect(audio.sampleRateHz == 48_000)
    #expect(audio.samples.count == 960)
    #expect(audio.samples.contains { $0 != 0 })
}

@Test func signalDetectorSeparatesNonzeroSignalFromFloor() {
    let iq = IQFrame(unsignedInterleavedIQ: [255, 127, 127, 255, 0, 127, 127, 0],
                     sampleRateHz: 2_400_000, centerFrequencyHz: 100_000_000, sequence: 0)
    let levels = SignalDetector().levels(iq)
    #expect(levels.signalDBFS.isFinite)
    #expect(levels.signalDBFS >= levels.noiseDBFS)
}

@Test func surveyEngineLabelsSimulatorEvidence() async throws {
    let profile = ScanProfile(
        id: "survey", name: "Survey",
        channels: [ChannelDefinition(id: "one", name: "One",
                                     frequencyHz: 100_000_000, bandwidthHz: 12_500,
                                     mode: "nfm")],
        settings: SurveySettings(noiseMarginDB: 0, revisitSeconds: 1,
                                 recordAudio: false, recordIQForUnknown: false,
                                 transcribeVoice: false, maxRecordingDays: 1,
                                 p25SampleRateHz: nil)
    )
    let engine = SurveyEngine()
    let stream = try await engine.run(profile: profile, receiver: SimulatedReceiver())
    var final: SurveyProgress?
    for try await update in stream { final = update }
    #expect(final?.finished == true)
    #expect(final?.observations.first?.simulated == true)
}

private actor TestRTLTransport: RTLSDRTransport {
    private var streaming = false
    func open() -> RadioDescriptor {
        RadioDescriptor(id: "rtl-test", name: "RTL test", kind: .rtlSDR,
                        serial: "1", connected: true, sampleRateLimit: 3_200_000)
    }
    func configure(_ configuration: TunerConfiguration) throws {
        if configuration.sampleRateHz <= 0 { throw ReceiverError.invalidConfiguration("rate") }
    }
    func startStream() { streaming = true }
    func readIQ(maximumBytes: Int) -> [UInt8] {
        guard streaming else { return [] }
        return [255, 127, 127, 255, 0, 127, 127, 0]
    }
    func stats() -> RTLSDRTransportStats { .init(writtenBytes: 8, droppedBytes: 0) }
    func stopStream() { streaming = false }
    func close() {}
}

@Test func rtlReceiverBridgesTransportIntoIQFrames() async throws {
    let receiver = RTLSDRReceiver(transport: TestRTLTransport())
    try await receiver.connect()
    try await receiver.configure(.mobileDefault)
    let stream = try await receiver.start()
    var iterator = stream.makeAsyncIterator()
    let frame = try await iterator.next()
    #expect(frame?.unsignedInterleavedIQ.count == 8)
    await receiver.stop()
    #expect(await receiver.telemetry().receivedSamples == 4)
}

private actor TestHackRFTransport: HackRFTransport {
    private var streaming = false
    func open() -> RadioDescriptor {
        RadioDescriptor(id: "hackrf-test", name: "HackRF test", kind: .hackRF,
                        serial: "1", connected: true, sampleRateLimit: 10_000_000)
    }
    func configure(_ configuration: TunerConfiguration) throws {}
    func startRX() { streaming = true }
    func readSignedIQ(maximumBytes: Int) -> [Int8] {
        guard streaming else { return [] }
        return [-128, 0, 0, 127]
    }
    func stats() -> HackRFTransportStats { .init(receivedBytes: 4, droppedBytes: 0) }
    func stopRX() { streaming = false }
    func close() {}
}

@Test func hackRFReceiverConvertsSignedTransportIQ() async throws {
    let receiver = HackRFReceiver(transport: TestHackRFTransport())
    try await receiver.connect()
    try await receiver.configure(.mobileDefault)
    let stream = try await receiver.start()
    var iterator = stream.makeAsyncIterator()
    let frame = try await iterator.next()
    #expect(frame?.unsignedInterleavedIQ == [0, 128, 128, 255])
    await receiver.stop()
    #expect(await receiver.telemetry().receivedSamples == 2)
}
