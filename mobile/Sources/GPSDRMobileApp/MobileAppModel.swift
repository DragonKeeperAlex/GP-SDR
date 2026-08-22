import Foundation
import GPSDRMobileCore

@MainActor
final class MobileAppModel: ObservableObject {
    @Published var selectedSection: AppSection? = .live
    @Published var radio: RadioDescriptor?
    @Published var tuner = TunerConfiguration.mobileDefault
    @Published var spectrum: [SpectrumPoint] = []
    @Published var telemetry = ReceiverTelemetry.idle
    @Published var profiles: [ScanProfile] = []
    @Published var events: [TransmissionEvent] = []
    @Published var running = false
    @Published var statusText = "Ready"
    @Published var lastError: String?
    @Published var surveyProgress: SurveyProgress?
    @Published var surveyRunning = false

    private let simulator = SimulatedReceiver()
    private let surveyEngine = SurveyEngine()
    private var streamTask: Task<Void, Never>?
    private var surveyTask: Task<Void, Never>?

    init() {
        radio = RadioDescriptor(id: "mobile-simulator", name: "Mobile simulator",
                                kind: .simulator, serial: nil, connected: true,
                                sampleRateLimit: 2_400_000,
                                note: "Synthetic IQ; no received RF")
        profiles = [
            ScanProfile(
                id: "mobile-quick-tune", name: "Quick Tune",
                summary: "Portable direct receiver",
                channels: [ChannelDefinition(id: "quick", name: "Listen",
                                             frequencyHz: tuner.centerFrequencyHz,
                                             bandwidthHz: Double(tuner.bandwidthHz), mode: tuner.modulation.rawValue)],
                builtIn: true
            )
        ]
    }

    func toggleReceiver() {
        if running { stopReceiver() } else { startReceiver() }
    }

    func startReceiver() {
        guard !running else { return }
        lastError = nil
        statusText = "Starting…"
        streamTask = Task {
            do {
                try await simulator.configure(tuner)
                let stream = try await simulator.start()
                running = true
                statusText = "Simulated IQ"
                let analyzer = SpectrumAnalyzer()
                for try await frame in stream {
                    guard !Task.isCancelled else { break }
                    spectrum = analyzer.analyze(frame)
                    telemetry = await simulator.telemetry()
                }
            } catch {
                lastError = error.localizedDescription
                statusText = "Start failed"
                running = false
            }
        }
    }

    func stopReceiver() {
        streamTask?.cancel()
        streamTask = nil
        Task { await simulator.stop() }
        running = false
        statusText = "Stopped"
    }

    func toggleSurvey() {
        if surveyRunning {
            surveyTask?.cancel()
            surveyTask = nil
            Task { await surveyEngine.stop() }
            surveyRunning = false
            return
        }
        stopReceiver()
        guard let profile = profiles.first else { return }
        surveyTask = Task {
            do {
                surveyRunning = true
                let stream = try await surveyEngine.run(profile: profile, receiver: simulator)
                for try await update in stream {
                    surveyProgress = update
                }
                surveyRunning = false
            } catch {
                lastError = error.localizedDescription
                surveyRunning = false
            }
        }
    }
}

enum AppSection: String, CaseIterable, Identifiable {
    case live = "Live"
    case activity = "Activity"
    case profiles = "Profiles"
    case mapper = "Mapper"
    case hardware = "Hardware"
    case status = "Status"
    var id: String { rawValue }

    var symbol: String {
        switch self {
        case .live: "waveform"
        case .activity: "clock.arrow.circlepath"
        case .profiles: "list.bullet.rectangle"
        case .mapper: "scope"
        case .hardware: "cable.connector"
        case .status: "checklist"
        }
    }
}
