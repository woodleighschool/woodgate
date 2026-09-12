import SwiftUI
import UIKit

struct SecretMenuSheet: View {
    // MARK: - Properties

    @Environment(ModelData.self) private var modelData
    @Environment(\.dismiss) private var dismiss

    let session: ActiveSession?

    @State private var isRefreshing = false
    @State private var singleAppModeEnabled = UIAccessibility.isGuidedAccessEnabled
    @State private var isChangingSingleAppMode = false

    // MARK: - Computed Properties

    private var appVersion: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "Unknown"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "Unknown"
        return "Version \(version) (\(build))"
    }

    // MARK: - Body

    var body: some View {
        NavigationStack {
            Form {
                singleAppModeSection
                actionsSection
                debugSection

                Section {
                    EmptyView()
                } footer: {
                    Text(appVersion)
                        .frame(maxWidth: .infinity, alignment: .center)
                        .padding(.top, 8)
                }
            }
            .navigationTitle("Device Menu")
            .navigationBarTitleDisplayMode(.inline)
        }
        .presentationDetents([.medium, .large])
        .modelAlert()
        .task {
            let changes = NotificationCenter.default.notifications(named: UIAccessibility.guidedAccessStatusDidChangeNotification)
                .map { _ in () }
            singleAppModeEnabled = UIAccessibility.isGuidedAccessEnabled
            for await _ in changes {
                guard !Task.isCancelled else { return }
                singleAppModeEnabled = UIAccessibility.isGuidedAccessEnabled
            }
        }
    }

    // MARK: - View Builders

    private var singleAppModeSection: some View {
        Section {
            LabeledContent("Single App Mode", value: singleAppModeEnabled ? "Enabled" : "Disabled")
            Button(singleAppModeEnabled ? "Disable Single App Mode" : "Enable Single App Mode") {
                let enabled = !singleAppModeEnabled
                isChangingSingleAppMode = true
                UIAccessibility.requestGuidedAccessSession(enabled: enabled) { succeeded in
                    isChangingSingleAppMode = false
                    singleAppModeEnabled = UIAccessibility.isGuidedAccessEnabled
                    if !succeeded {
                        modelData.alert = AlertItem(
                            title: "Could Not Change Single App Mode",
                            message: enabled
                                ? "This device must be supervised and MDM must permit this app to use Autonomous Single App Mode."
                                : "The device could not exit Single App Mode. An MDM-enforced lock or manually started Guided Access must be ended using its own controls."
                        )
                    }
                }
            }
            .disabled(isChangingSingleAppMode)
        } header: {
            Text("Single App Mode")
        } footer: {
            Text("Requires a supervised device and permission from MDM.")
        }
    }

    private var actionsSection: some View {
        Section("Actions") {
            if session != nil {
                Button {
                    Task {
                        isRefreshing = true
                        await modelData.refreshSession()
                        isRefreshing = false
                    }
                } label: {
                    HStack {
                        Label("Refresh Configuration", systemImage: "arrow.triangle.2.circlepath")
                        Spacer()
                        if isRefreshing {
                            ProgressView()
                        }
                    }
                }
                .disabled(isRefreshing)

                Button {
                    dismiss()
                    Task {
                        await modelData.beginSwitchLocation()
                    }
                } label: {
                    Label("Switch Location", systemImage: "building.2")
                }
                .disabled(isRefreshing)

                Button(role: .destructive) {
                    dismiss()
                    modelData.forgetPairing()
                } label: {
                    Label("Disconnect", systemImage: "network.slash")
                }
                .disabled(isRefreshing)
            }
        }
    }

    @ViewBuilder
    private var debugSection: some View {
        if let session {
            Section("Debug") {
                Text("Location: \(session.location.name)")
                Text("People cached: \(session.people.count)")
                Text(
                    "Last refresh: \(session.lastSyncedAt.formatted(date: .abbreviated, time: .shortened))"
                )
                Text("Server: \(session.baseURLString)")
            }
            .font(.system(size: 13, weight: .regular))
            .foregroundStyle(.secondary)
        }
    }
}
