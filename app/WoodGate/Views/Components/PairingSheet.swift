import SwiftUI
import Vision
import VisionKit

struct PairingSheet: View {
    // MARK: - Properties

    @Environment(ModelData.self) private var modelData
    @Environment(\.dismiss) private var dismiss

    @State private var method = PairingMethod.scan
    @State private var scannerID = UUID()

    init(initialMethod: PairingMethod = .scan) {
        _method = State(initialValue: initialMethod)
    }

    // MARK: - Body

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("Pairing Method", selection: $method) {
                    ForEach(PairingMethod.allCases) { method in
                        Text(method.title)
                            .tag(method)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, 24)
                .padding(.vertical, 16)

                switch method {
                case .scan:
                    ScanPairingView(isBusy: modelData.isBusy, onPayload: pair)
                        .id(scannerID)
                case .manual:
                    ManualPairingView()
                }
            }
            .navigationTitle("Pair Device")
            .navigationBarTitleDisplayMode(.inline)
        }
        .modelAlert()
    }

    // MARK: - Private Helpers

    private func pair(_ payload: String) {
        guard !modelData.isBusy else { return }

        Task {
            if await modelData.beginPairing(with: payload) {
                dismiss()
            } else {
                scannerID = UUID()
            }
        }
    }
}

// MARK: - Private Components

enum PairingMethod: CaseIterable, Identifiable {
    case scan
    case manual

    var id: Self {
        self
    }

    var title: LocalizedStringKey {
        switch self {
        case .scan:
            "Scan"
        case .manual:
            "Manual"
        }
    }
}

#Preview("Pairing — Manual") {
    PairingSheet(initialMethod: .manual)
        .environment(PreviewFixtures.modelData())
}

private struct ScanPairingView: View {
    let isBusy: Bool
    let onPayload: (String) -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                QRScannerView(onPayload: onPayload)
                    .frame(maxWidth: .infinity)
                    .frame(height: 420)
                    .clipShape(.rect(cornerRadius: 28))
                    .shadow(color: .black.opacity(0.1), radius: 10, y: 4)
                    .allowsHitTesting(!isBusy)

                VStack(alignment: .leading, spacing: 8) {
                    Text("Scan Configuration QR")
                        .font(.title2.weight(.bold))

                    Text(
                        "Scan the Station configuration QR code. Its server and location will be applied automatically."
                    )
                    .font(.callout)
                    .foregroundStyle(.secondary)
                }
            }
            .padding(.horizontal, 24)
            .padding(.bottom, 8)
        }
    }
}

private struct ManualPairingView: View {
    @Environment(ModelData.self) private var modelData
    @Environment(\.dismiss) private var dismiss

    @State private var baseURL = ""
    @State private var stationKey = ""

    private var isPairingDisabled: Bool {
        modelData.isBusy
            || baseURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            || stationKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        Form {
            Section {
                TextField("Server URL", text: $baseURL)
                    .textContentType(.URL)
                    .keyboardType(.URL)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()

                TextField("Station Key", text: $stationKey)
                    .textContentType(.password)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onSubmit(pair)
            } header: {
                Text("Station Details")
            } footer: {
                Text("Enter the server address and Station key supplied for this device.")
            }

            Section {
                Button(action: pair) {
                    HStack {
                        Label("Pair Device", systemImage: "link")
                        Spacer()
                        if modelData.isBusy {
                            ProgressView()
                        }
                    }
                }
                .disabled(isPairingDisabled)
            }
        }
    }

    private func pair() {
        guard !isPairingDisabled else { return }

        Task {
            let paired = await modelData.beginPairing(
                with: PairingPayload(baseURL: baseURL, stationKey: stationKey)
            )
            if paired {
                dismiss()
            }
        }
    }
}

private struct QRScannerView: UIViewControllerRepresentable {
    // MARK: - Properties

    let onPayload: (String) -> Void

    // MARK: - UIViewControllerRepresentable

    func makeCoordinator() -> Coordinator {
        Coordinator(onPayload: onPayload)
    }

    func makeUIViewController(context: Context) -> DataScannerViewController {
        let controller = DataScannerViewController(
            recognizedDataTypes: [.barcode(symbologies: [.qr])],
            qualityLevel: .balanced,
            recognizesMultipleItems: false,
            isHighFrameRateTrackingEnabled: false,
            isHighlightingEnabled: true
        )
        controller.delegate = context.coordinator
        return controller
    }

    func updateUIViewController(_ controller: DataScannerViewController, context _: Context) {
        guard !controller.isScanning else { return }

        try? controller.startScanning()
    }

    static func dismantleUIViewController(
        _ controller: DataScannerViewController,
        coordinator _: Coordinator
    ) {
        controller.stopScanning()
    }

    // MARK: - Coordinator

    final class Coordinator: NSObject, DataScannerViewControllerDelegate {
        // MARK: - Properties

        private let onPayload: (String) -> Void
        private var hasScanned = false

        // MARK: - Lifecycle

        init(onPayload: @escaping (String) -> Void) {
            self.onPayload = onPayload
        }

        // MARK: - DataScannerViewControllerDelegate

        func dataScanner(
            _ dataScanner: DataScannerViewController,
            didAdd addedItems: [RecognizedItem],
            allItems _: [RecognizedItem]
        ) {
            guard !hasScanned else { return }

            for item in addedItems {
                guard case let .barcode(barcode) = item,
                      let payload = barcode.payloadStringValue
                else {
                    continue
                }

                hasScanned = true
                dataScanner.stopScanning()
                onPayload(payload)
                return
            }
        }
    }
}
