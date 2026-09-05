import Foundation
import SwiftData

@Model
final class CachedStationPersonRecord {
    @Attribute(.unique) var userID: Int64
    var displayName: String
    var email: String

    init(
        userID: Int64,
        displayName: String,
        email: String
    ) {
        self.userID = userID
        self.displayName = displayName
        self.email = email
    }
}
