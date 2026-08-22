import Foundation

public enum MobileFeatureState: String, Codable, Sendable {
    case available
    case inProgress
    case unavailable
    case requiresHardware
}

public struct MobileFeature: Identifiable, Codable, Equatable, Sendable {
    public var id: String
    public var name: String
    public var state: MobileFeatureState
    public var note: String
}

public enum MobileFeatures {
    public static let current: [MobileFeature] = [
        .init(id: "profiles", name: "Profiles", state: .available,
              note: "Desktop-compatible JSON import and export"),
        .init(id: "activity", name: "Activity journal", state: .available,
              note: "Append-only local event storage"),
        .init(id: "simulator", name: "Simulator", state: .available,
              note: "Synthetic IQ is always marked as simulated"),
        .init(id: "spectrum", name: "Spectrum", state: .inProgress,
              note: "Live model present; Accelerate FFT optimization pending"),
        .init(id: "rtlsdr", name: "RTL-SDR USB", state: .requiresHardware,
              note: "DriverKit adapter and signing required"),
        .init(id: "hackrf", name: "HackRF USB", state: .inProgress,
              note: "Receive-only DriverKit transport requires hardware validation"),
        .init(id: "analog", name: "AM/NFM/WFM audio", state: .inProgress,
              note: "Mobile audio pipeline under development"),
        .init(id: "p25", name: "P25 Phase 1/2", state: .inProgress,
              note: "Desktop Java engine cannot run unchanged on iPadOS"),
        .init(id: "transmit", name: "Transmit", state: .unavailable,
              note: "GP-SDR mobile remains receive-only")
    ]
}
