import SwiftData
import UIKit

@MainActor
enum PreviewFixtures {
    static let plainSession = makeSession()
    static let brandedSession = makeSession(
        backgroundImage: wallpaper,
        logoImage: UIImage(systemName: "person.crop.circle.fill")?.withTintColor(
            .white,
            renderingMode: .alwaysOriginal
        )
    )

    static func modelData(session: ActiveSession? = nil) -> ModelData {
        let schema = Schema([CachedStationPersonRecord.self])
        let configuration = ModelConfiguration(schema: schema, isStoredInMemoryOnly: true)
        let container = try! ModelContainer(for: schema, configurations: [configuration])
        let modelData = ModelData(
            modelContext: container.mainContext,
            startsServices: false
        )

        if let session {
            for person in session.people {
                container.mainContext.insert(
                    CachedStationPersonRecord(
                        userID: person.id,
                        displayName: person.displayName,
                        email: person.email
                    )
                )
            }
            try! container.mainContext.save()
            modelData.currentSession = session
        }

        return modelData
    }

    private static let people = [
        PersonSummary(id: 1, displayName: "Avery Example", email: "avery@example.invalid"),
        PersonSummary(id: 2, displayName: "Jordan Sample", email: "jordan@example.invalid"),
        PersonSummary(id: 3, displayName: "Morgan Test", email: "morgan@example.invalid"),
    ]

    private static let wallpaper = UIGraphicsImageRenderer(
        size: CGSize(width: 1200, height: 1600)
    ).image { rendererContext in
        let colors = [
            UIColor(red: 0.08, green: 0.24, blue: 0.19, alpha: 1).cgColor,
            UIColor(red: 0.32, green: 0.54, blue: 0.43, alpha: 1).cgColor,
        ] as CFArray
        let gradient = CGGradient(
            colorsSpace: CGColorSpaceCreateDeviceRGB(),
            colors: colors,
            locations: [0, 1]
        )!
        rendererContext.cgContext.drawLinearGradient(
            gradient,
            start: .zero,
            end: CGPoint(x: 1200, y: 1600),
            options: []
        )
    }

    private static func makeSession(
        backgroundImage: UIImage? = nil,
        logoImage: UIImage? = nil
    ) -> ActiveSession {
        ActiveSession(
            baseURLString: "https://woodgate.invalid",
            stationID: 1,
            stationName: "Preview Station",
            location: ActiveLocation(
                id: 1,
                name: "Reception",
                enabled: true,
                notes: true,
                photo: true,
                backgroundObjectID: backgroundImage == nil ? nil : 1,
                logoObjectID: logoImage == nil ? nil : 1
            ),
            people: people,
            backgroundImage: backgroundImage,
            logoImage: logoImage,
            lastSyncedAt: .distantPast
        )
    }
}
