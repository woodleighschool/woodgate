import SwiftUI

struct SecretMenuSheet: View {
    // MARK: - Properties

    @Environment(ModelData.self) private var modelData
    @Environment(\.dismiss) private var dismiss

    let session: ActiveSession?

    @State private var isRefreshing = false

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
        .modelAlert()
    }

    // MARK: - View Builders

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
            }

            Button(role: .destructive) {
                dismiss()
                modelData.forgetPairing()
            } label: {
                Label("Forget Pairing", systemImage: "trash")
            }
            .disabled(isRefreshing)
        }
    }

    @ViewBuilder
    private var debugSection: some View {
        if let session {
            Section("Debug") {
                Text("Station: \(session.stationName)")
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
