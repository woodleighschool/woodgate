import Foundation
import SwiftUI
import Vision
import VisionKit

struct PairingSheet: View {
    // MARK: - Properties

    let onPayload: (PairingPayload) async throws -> Void

    @State private var step = PairingStep.entry
    @State private var baseURL = ""
    @State private var stationKey = ""

    private var pendingPayload: PairingPayload? {
        if case let .connecting(payload) = step {
            payload
        } else {
            nil
        }
    }

    private var isBusy: Bool {
        pendingPayload != nil
    }

    @State private var canScan = DataScannerViewController.isSupported && DataScannerViewController.isAvailable
    @State private var method: PairingMethod = DataScannerViewController.isSupported && DataScannerViewController.isAvailable ? .scan : .manual

    // MARK: - Body

    var body: some View {
        VStack(spacing: 0) {
            Picker("Setup Method", selection: $method) {
                Text("Scan").tag(PairingMethod.scan)
                Text("Manual").tag(PairingMethod.manual)
            }
            .pickerStyle(.segmented)
            .disabled(isBusy || !canScan)
            .padding(.horizontal, 24)
            .padding(.vertical, 16)

            switch step {
            case .entry:
                Text(method == .scan ? "Scan Configuration QR" : "Manually Enter Details")
                    .font(.title2.weight(.bold))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 24)
                    .padding(.top, 8)
                    .padding(.bottom, 12)

                switch method {
                case .scan:
                    ScanPairingView(onPayload: scan, onUnavailable: scannerUnavailable)
                case .manual:
                    ManualPairingView(baseURL: $baseURL, stationKey: $stationKey, onPayload: pair)
                }
            case .connecting:
                ProgressView("Connecting...")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            case let .failed(invalidCode, message):
                ContentUnavailableView {
                    Label(
                        invalidCode ? "Invalid QR Code" : "Could Not Connect",
                        systemImage: invalidCode ? "qrcode" : "network.slash"
                    )
                } description: {
                    Text(message)
                } actions: {
                    Button(method == .scan ? "Rescan" : "Edit Details") {
                        step = .entry
                    }
                    .buttonStyle(.glassProminent)
                    .controlSize(.large)
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .onChange(of: method) {
            step = .entry
        }
        .task(id: pendingPayload) {
            guard let payload = pendingPayload else { return }
            do {
                try await onPayload(payload)
            } catch {
                guard !Task.isCancelled else { return }
                step = .failed(invalidCode: false, message: error.localizedDescription)
            }
        }
        .interactiveDismissDisabled(isBusy)
        .navigationTitle("Set Up")
        .navigationBarTitleDisplayMode(.inline)
    }

    // MARK: - Private Helpers

    private func scan(_ text: String) {
        guard case .entry = step else { return }
        do {
            try pair(PairingPayload.parse(urlString: text))
        } catch {
            step = .failed(
                invalidCode: true,
                message: "This QR code does not contain valid server and Station key details."
            )
        }
    }

    private func pair(_ payload: PairingPayload) {
        guard !isBusy else { return }
        baseURL = payload.baseURL
        stationKey = payload.stationKey
        step = .connecting(payload)
    }

    private func scannerUnavailable() {
        guard case .entry = step else { return }
        canScan = false
        method = .manual
    }
}

// MARK: - Private Components

private enum PairingMethod {
    case scan
    case manual
}

private enum PairingStep {
    case entry
    case connecting(PairingPayload)
    case failed(invalidCode: Bool, message: String)
}

private struct ScanPairingView: View {
    let onPayload: (String) -> Void
    let onUnavailable: () -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                Text(
                    "Scan the Station’s QR code to set up this device."
                )
                .font(.callout)
                .foregroundStyle(.secondary)

                QRScannerView(onPayload: onPayload, onUnavailable: onUnavailable)
                    .frame(maxWidth: .infinity)
                    .frame(height: 420)
                    .clipShape(RoundedRectangle(cornerRadius: 28, style: .continuous))
                    .shadow(color: .black.opacity(0.1), radius: 10, y: 4)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 24)
            .padding(.bottom, 24)
        }
    }
}

private struct ManualPairingView: View {
    // MARK: - Properties

    @Binding var baseURL: String
    @Binding var stationKey: String

    let onPayload: (PairingPayload) -> Void

    private var isPairingDisabled: Bool {
        AppSettings.serverURL(baseURL) == nil
            || stationKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    // MARK: - Body

    var body: some View {
        Form {
            Section("Connection") {
                TextField("https://woodgate.example.com", text: $baseURL)
                    .textContentType(.URL)
                    .keyboardType(.URL)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()

                TextField("Station Key", text: $stationKey)
                    .textContentType(nil)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onSubmit(pair)
            }

            Section {
                Button("Continue", action: pair)
                    .disabled(isPairingDisabled)
            }
        }
        .contentMargins(.horizontal, 24, for: .scrollContent)
        .contentMargins(.top, 0, for: .scrollContent)
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
