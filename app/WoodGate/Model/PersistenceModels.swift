import Foundation
import SwiftData

@Model
final class CachedPersonRecord {
    @Attribute(.unique) var userID: UUID
    var displayName: String
    var email: String

    init(
        userID: UUID,
        displayName: String,
        email: String
    ) {
        self.userID = userID
        self.displayName = displayName
        self.email = email
    }
}
