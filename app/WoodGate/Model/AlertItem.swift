import Foundation

struct AlertItem: Identifiable {
    // MARK: - Properties

    let id = UUID()
    let title: String
    let message: String
}
