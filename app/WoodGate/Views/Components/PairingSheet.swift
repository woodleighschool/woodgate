import Foundation
import SwiftUI
import Vision
import VisionKit

struct PairingSheet: View {
    // MARK: - Properties

    @Environment(ModelData.self) private var modelData
    @Environment(\.dismiss) private var dismiss

    @State private var pendingPayload: PairingPayload?

    private var isBusy: Bool {
        pendingPayload != nil || modelData.isBusy
    }

    @State private var canScan = DataScannerViewController.isSupported && DataScannerViewController.isAvailable
    @State private var method: PairingMethod
    @State private var scannerID = UUID()

    init(initialMethod: PairingMethod = .scan) {
        _method = State(initialValue: DataScannerViewController.isSupported && DataScannerViewController.isAvailable ? initialMethod : .manual)
    }

    // MARK: - Body

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("Pairing Method", selection: $method) {
                    Text("Scan").tag(PairingMethod.scan)
                    Text("Manual").tag(PairingMethod.manual)
                }
                .pickerStyle(.segmented)
                .disabled(isBusy || !canScan)
                .padding(.horizontal, 24)
                .padding(.vertical, 16)

                switch method {
                case .scan:
                    ScanPairingView(isBusy: isBusy, onPayload: scan, onUnavailable: {
                        canScan = false
                        method = .manual
                    })
                    .id(scannerID)
                case .manual:
                    ManualPairingView(isBusy: isBusy, onPayload: pair)
                }
            }
            .task(id: pendingPayload) {
                guard let payload = pendingPayload else { return }
                do {
                    try await modelData.beginPairing(with: payload)
                    dismiss()
                } catch {
                    guard !Task.isCancelled else { return }
                    modelData.alert = AlertItem(title: "Could Not Pair", message: error.localizedDescription)
                }
                pendingPayload = nil
            }
            .onChange(of: modelData.alert?.id) { _, alertID in
                if alertID == nil {
                    scannerID = UUID()
                }
            }
            .interactiveDismissDisabled(isBusy)
            .navigationTitle("Pair Device")
            .navigationBarTitleDisplayMode(.inline)
        }
        .modelAlert()
    }

    // MARK: - Private Helpers

    private func scan(_ text: String) {
        guard !isBusy else { return }
        do {
            try pair(PairingPayload.parse(urlString: text))
        } catch {
            modelData.alert = AlertItem(title: "Could Not Pair", message: error.localizedDescription)
        }
    }

    private func pair(_ payload: PairingPayload) {
        guard !isBusy else { return }
        pendingPayload = payload
    }
}

// MARK: - Private Components

enum PairingMethod {
    case scan
    case manual
}

private struct ScanPairingView: View {
    let isBusy: Bool
    let onPayload: (String) -> Void
    let onUnavailable: () -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text("Scan Configuration QR")
                    .font(.title2.weight(.bold))

                Text(
                    "Scan the Station configuration QR code. Its server and location will be applied automatically."
                )
                .font(.callout)
                .foregroundStyle(.secondary)

                QRScannerView(onPayload: onPayload, onUnavailable: onUnavailable)
                    .frame(maxWidth: .infinity)
                    .frame(height: 420)
                    .clipShape(RoundedRectangle(cornerRadius: 28, style: .continuous))
                    .shadow(color: .black.opacity(0.1), radius: 10, y: 4)
                    .allowsHitTesting(!isBusy)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(24)
        }
    }
}

private struct ManualPairingView: View {
    // MARK: - Properties

    @State private var baseURL = ""
    @State private var stationKey = ""

    let isBusy: Bool
    let onPayload: (PairingPayload) -> Void

    private var isPairingDisabled: Bool {
        isBusy
            || baseURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            || stationKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    // MARK: - Body

    var body: some View {
        Form {
            Section {
                TextField("Server URL", text: $baseURL)
                    .textContentType(.URL)
                    .keyboardType(.URL)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()

                SecureField("Station Key", text: $stationKey)
                    .textContentType(.password)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onSubmit(pair)
            } header: {
                Text("Server and Key")
            } footer: {
                Text("Enter the server URL and Station key to start pairing.")
            }

            Section {
                Button(action: pair) {
                    HStack {
                        Label("Pair Device", systemImage: "link")
                        Spacer()
                        if isBusy {
                            ProgressView()
                        }
                    }
                }
                .disabled(isPairingDisabled)
            }
        }
        .disabled(isBusy)
    }

    // MARK: - Private Helpers

    private func pair() {
        guard !isPairingDisabled else {
            return
        }

        onPayload(PairingPayload(baseURL: baseURL, stationKey: stationKey))
    }
}

private struct QRScannerView: UIViewControllerRepresentable {
    // MARK: - Properties

    let onPayload: (String) -> Void
    let onUnavailable: () -> Void

    // MARK: - UIViewControllerRepresentable

    func makeCoordinator() -> Coordinator {
        Coordinator(onPayload: onPayload, onUnavailable: onUnavailable)
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

    func updateUIViewController(_ controller: DataScannerViewController, context: Context) {
        guard !controller.isScanning, !context.coordinator.hasScanned else { return }
        do {
            try controller.startScanning()
        } catch {
            Task { onUnavailable() }
        }
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
        private let onUnavailable: () -> Void
        private(set) var hasScanned = false

        // MARK: - Lifecycle

        init(onPayload: @escaping (String) -> Void, onUnavailable: @escaping () -> Void) {
            self.onPayload = onPayload
            self.onUnavailable = onUnavailable
        }

        // MARK: - DataScannerViewControllerDelegate

        func dataScanner(
            _ dataScanner: DataScannerViewController,
            becameUnavailableWithError _: DataScannerViewController.ScanningUnavailable
        ) {
            dataScanner.stopScanning()
            onUnavailable()
        }

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

#Preview("Pairing - Manual") {
    PairingSheet(initialMethod: .manual)
        .environment(PreviewFixtures.modelData())
}
