import SwiftUI
import GPSDRMobileCore

struct RootView: View {
    @EnvironmentObject private var model: MobileAppModel

    var body: some View {
        NavigationSplitView {
            List(AppSection.allCases, selection: $model.selectedSection) { section in
                Label(section.rawValue, systemImage: section.symbol)
                    .tag(section)
            }
            .navigationTitle("GP-SDR")
        } detail: {
            switch model.selectedSection ?? .live {
            case .live: LiveView()
            case .activity: ActivityView()
            case .profiles: ProfilesView()
            case .mapper: MapperView()
            case .hardware: HardwareView()
            case .status: FeatureStatusView()
            }
        }
        .frame(minWidth: 820, minHeight: 600)
    }
}

private struct LiveView: View {
    @EnvironmentObject private var model: MobileAppModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Live Receiver").font(.largeTitle.bold())
                        Label(model.statusText, systemImage: model.running ? "circle.fill" : "circle")
                            .foregroundStyle(model.running ? .green : .secondary)
                    }
                    Spacer()
                    Button(model.running ? "Stop" : "Start") { model.toggleReceiver() }
                        .buttonStyle(.borderedProminent)
                        .tint(model.running ? .red : .blue)
                }

                SpectrumView(points: model.spectrum)
                    .frame(height: 280)

                GroupBox("Receiver") {
                    Grid(alignment: .leading, horizontalSpacing: 18, verticalSpacing: 14) {
                        GridRow {
                            Text("Frequency")
                            TextField("Hz", value: $model.tuner.centerFrequencyHz, format: .number)
                                .textFieldStyle(.roundedBorder)
                        }
                        GridRow {
                            Text("Sample rate")
                            Picker("Sample rate", selection: $model.tuner.sampleRateHz) {
                                Text("1.024 MS/s").tag(1_024_000)
                                Text("2.4 MS/s").tag(2_400_000)
                            }.labelsHidden()
                        }
                        GridRow {
                            Text("Mode")
                            Picker("Mode", selection: $model.tuner.modulation) {
                                ForEach([Modulation.am, .nfm, .wfm], id: \.self) { Text($0.rawValue.uppercased()).tag($0) }
                            }.pickerStyle(.segmented).labelsHidden()
                        }
                    }.padding(.top, 6)
                }

                HStack(spacing: 24) {
                    Metric(label: "Signal", value: String(format: "%.1f dBFS", model.telemetry.signalDBFS))
                    Metric(label: "Noise", value: String(format: "%.1f dBFS", model.telemetry.noiseDBFS))
                    Metric(label: "Dropped", value: "\(model.telemetry.droppedSamples)")
                    Metric(label: "Samples", value: "\(model.telemetry.receivedSamples)")
                }
            }.padding(24)
        }
        .navigationTitle("Live")
    }
}

private struct SpectrumView: View {
    let points: [SpectrumPoint]

    var body: some View {
        Canvas { context, size in
            guard points.count > 1 else { return }
            let low = points.map(\.powerDBFS).min() ?? -120
            let high = max(low + 1, points.map(\.powerDBFS).max() ?? 0)
            var path = Path()
            for (index, point) in points.enumerated() {
                let x = CGFloat(index) / CGFloat(points.count - 1) * size.width
                let normalized = (point.powerDBFS - low) / (high - low)
                let y = size.height - CGFloat(normalized) * size.height
                index == 0 ? path.move(to: CGPoint(x: x, y: y)) : path.addLine(to: CGPoint(x: x, y: y))
            }
            context.stroke(path, with: .linearGradient(
                Gradient(colors: [.cyan, .blue]), startPoint: .zero,
                endPoint: CGPoint(x: size.width, y: 0)), lineWidth: 2)
        }
        .background(Color.black.opacity(0.45), in: RoundedRectangle(cornerRadius: 14))
        .overlay {
            if points.isEmpty { Text("Start receiver to display spectrum").foregroundStyle(.secondary) }
        }
    }
}

private struct ActivityView: View {
    @EnvironmentObject private var model: MobileAppModel
    var body: some View {
        List(model.events) { event in
            VStack(alignment: .leading) {
                Text(event.label ?? String(format: "%.5f MHz", event.frequencyHz / 1_000_000)).bold()
                Text("\(event.modulation.uppercased()) · \(event.durationSeconds, specifier: "%.1f") seconds")
                    .foregroundStyle(.secondary)
            }
        }
        .overlay { if model.events.isEmpty { EmptyState(title: "No activity", symbol: "waveform.slash", message: "Received events will appear here.") } }
        .navigationTitle("Activity")
    }
}

private struct ProfilesView: View {
    @EnvironmentObject private var model: MobileAppModel
    var body: some View {
        List(model.profiles) { profile in
            VStack(alignment: .leading, spacing: 4) {
                Text(profile.name).font(.headline)
                Text(profile.summary).foregroundStyle(.secondary)
                Text("\(profile.channels.count) channels · \(profile.ranges.count) ranges")
                    .font(.caption).foregroundStyle(.tertiary)
            }
        }.navigationTitle("Profiles")
    }
}

private struct HardwareView: View {
    @EnvironmentObject private var model: MobileAppModel
    var body: some View {
        Form {
            Section("Connected receiver") {
                LabeledContent("Name", value: model.radio?.name ?? "None")
                LabeledContent("Kind", value: model.radio?.kind.rawValue.uppercased() ?? "—")
                LabeledContent("State", value: model.radio?.connected == true ? "Connected" : "Unavailable")
            }
            Section("USB") {
                Text("RTL-SDR and HackRF drivers require a developer-signed DriverKit build and real M-series iPad hardware.")
            }
        }.formStyle(.grouped).navigationTitle("Hardware")
    }
}

private struct MapperView: View {
    @EnvironmentObject private var model: MobileAppModel
    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack {
                VStack(alignment: .leading) {
                    Text("Portable Mapper").font(.largeTitle.bold())
                    Text("Results retain receiver provenance and simulation status.")
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button(model.surveyRunning ? "Stop" : "Start survey") { model.toggleSurvey() }
                    .buttonStyle(.borderedProminent)
                    .tint(model.surveyRunning ? .red : .blue)
            }
            ProgressView(value: model.surveyProgress?.fraction ?? 0) {
                Text(model.surveyProgress?.current?.name ?? (model.surveyProgress?.finished == true ? "Complete" : "Ready"))
            } currentValueLabel: {
                Text("\(model.surveyProgress?.completed ?? 0) / \(model.surveyProgress?.total ?? 0)")
            }
            Table(model.surveyProgress?.observations ?? []) {
                TableColumn("Frequency") { observation in
                    Text(String(format: "%.5f MHz", observation.target.frequencyHz / 1_000_000))
                }
                TableColumn("Mode") { Text($0.target.mode.rawValue.uppercased()) }
                TableColumn("SNR") { Text(String(format: "%.1f dB", $0.signalToNoiseDB)) }
                TableColumn("Source") { Text($0.simulated ? "Simulator" : $0.deviceID) }
            }
        }.padding(24).navigationTitle("Mapper")
    }
}

private struct FeatureStatusView: View {
    var body: some View {
        List(MobileFeatures.current) { feature in
            HStack {
                VStack(alignment: .leading) {
                    Text(feature.name).font(.headline)
                    Text(feature.note).foregroundStyle(.secondary)
                }
                Spacer()
                Text(feature.state.rawValue.replacingOccurrences(of: "([a-z])([A-Z])", with: "$1 $2", options: .regularExpression).capitalized)
                    .font(.caption.bold()).foregroundStyle(color(feature.state))
            }
        }.navigationTitle("Feature Status")
    }

    private func color(_ state: MobileFeatureState) -> Color {
        switch state {
        case .available: .green
        case .inProgress: .orange
        case .requiresHardware: .blue
        case .unavailable: .secondary
        }
    }
}

private struct PlaceholderView: View {
    let title: String
    let symbol: String
    let message: String
    var body: some View {
        EmptyState(title: title, symbol: symbol, message: message)
            .navigationTitle(title)
    }
}

private struct EmptyState: View {
    let title: String
    let symbol: String
    let message: String
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: symbol).font(.system(size: 42)).foregroundStyle(.secondary)
            Text(title).font(.title2.bold())
            Text(message).multilineTextAlignment(.center).foregroundStyle(.secondary)
                .frame(maxWidth: 420)
        }.padding()
    }
}

private struct Metric: View {
    let label: String
    let value: String
    var body: some View {
        VStack(alignment: .leading) {
            Text(label).font(.caption).foregroundStyle(.secondary)
            Text(value).font(.title3.monospacedDigit())
        }
    }
}
