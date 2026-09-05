import SwiftData
import SwiftUI

@main
struct WoodGateApp: App {
    private let modelTypes: [any PersistentModel.Type] = [
        CachedStationPersonRecord.self,
    ]

    private let container: ModelContainer
    private let modelData: ModelData

    // MARK: - Init

    init() {
        let schema = Schema(modelTypes)
        container = try! ModelContainer(for: schema)
        modelData = ModelData(modelContext: container.mainContext)
    }

    // MARK: - Body

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(modelData)
        }
        .modelContainer(container)
    }
}
