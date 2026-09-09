import Foundation
import Observation
import SwiftData
import UIKit

@MainActor
@Observable
final class ModelData {
    enum UnavailableState {
        case connectivity
        case authorization
        case locationDisabled
    }

    // MARK: - Properties

    var currentSession: ActiveSession?
    var locationSelection: LocationSelectionState?
    var alert: AlertItem?
    var unavailableState: UnavailableState?
    var isBusy = false

    private let modelContext: ModelContext
    private var refreshTask: Task<Void, Never>?
    private var refreshInFlightTask: Task<Void, Never>?

    // MARK: - Init

    init(modelContext: ModelContext, startsServices: Bool = true) {
        self.modelContext = modelContext
        if startsServices {
            startBackgroundRefresh()
            Task { await bootstrap() }
        }
    }

    // MARK: - Lifecycle

    func bootstrap() async {
        do {
            let settings = AppSettings.shared
            guard
                let locationID = settings.locationID,
                settings.hasPairing
            else {
                try clearStoredSession(removeAPIKey: true)
                currentSession = nil
                return
            }

            let cachedSession = try loadStoredSession(
                locationID: locationID,
                settings: settings
            )

            guard let client = settings.woodGateClient() else {
                currentSession = cachedSession
                unavailableState = nil
                return
            }

            do {
                currentSession = try await buildSession(
                    baseURLString: settings.baseURLString,
                    client: client,
                    locationID: locationID,
                    fallbackSession: cachedSession
                )
                unavailableState = nil
            } catch {
                currentSession = cachedSession
                unavailableState = unavailableState(for: error)
            }
        } catch {
            alert = AlertItem(title: "Could Not Start", message: error.localizedDescription)
        }
    }

    func handleSceneActive() async {
        await refreshSession()
    }

    // MARK: - Pairing

    func beginPairing(with payload: PairingPayload) async throws {
        let normalizedPayload = PairingPayload(
            baseURL: payload.baseURL.trimmingCharacters(in: .whitespacesAndNewlines),
            apiKey: payload.apiKey.trimmingCharacters(in: .whitespacesAndNewlines)
        )
        try await fetchPairableLocations(using: normalizedPayload)
    }

    func beginSwitchLocation() async {
        do {
            let settings = AppSettings.shared
            let payload = PairingPayload(baseURL: settings.baseURLString, apiKey: settings.apiKey)
            try await fetchPairableLocations(using: payload)
        } catch {
            alert = AlertItem(title: "Could Not Load Locations", message: error.localizedDescription)
        }
    }

    func cancelLocationSelection() {
        locationSelection = nil
    }

    func selectLocation(_ option: SessionLocation) async {
        guard !isBusy, let payload = locationSelection?.payload else { return }

        isBusy = true
        defer { isBusy = false }

        do {
            guard
                let client = AppSettings.shared.woodGateClient(
                    baseURLString: payload.baseURL,
                    apiKey: payload.apiKey
                )
            else {
                throw WoodGateError(message: "Enter a valid HTTP or HTTPS server URL.")
            }
            let location = try await client.getLocation(id: option.id)
            guard location.enabled else {
                throw WoodGateError(message: "That location is currently disabled.")
            }
            let people = try await client.listPeople(locationID: location.id)
            let session = await makeSession(
                baseURLString: payload.baseURL,
                location: location,
                people: people,
                lastSyncedAt: Date(),
                client: client,
                previousSession: nil
            )

            try persist(session: session, apiKey: payload.apiKey)
            currentSession = session
            unavailableState = nil
            locationSelection = nil
        } catch {
            alert = AlertItem(title: "Could Not Pair", message: error.localizedDescription)
        }
    }

    func forgetPairing() {
        do {
            try clearStoredSession(removeAPIKey: true)
            currentSession = nil
            locationSelection = nil
            unavailableState = nil
        } catch {
            alert = AlertItem(title: "Could Not Forget Pairing", message: error.localizedDescription)
        }
    }

    // MARK: - Session Refresh

    func refreshSession() async {
        guard let currentSession else { return }
        guard !isBusy, locationSelection == nil else { return }

        if let refreshInFlightTask {
            await refreshInFlightTask.value
            return
        }

        let task = Task { [weak self] in
            guard let self else { return }
            await performRefresh(using: currentSession)
        }

        refreshInFlightTask = task
        await task.value
        refreshInFlightTask = nil
    }

    // MARK: - Checkin

    func submitCheckin(
        session: ActiveSession,
        person: PersonSummary,
        direction: CheckinDirectionChoice,
        notes: String,
        selfie: CapturedSelfie?
    ) async throws {
        isBusy = true
        defer { isBusy = false }

        let settings = AppSettings.shared
        let client = settings.woodGateClient(
            baseURLString: session.baseURLString,
            apiKey: settings.apiKey
        )!

        _ = try await client.createCheckin(
            locationID: session.location.id,
            userID: person.id,
            direction: direction,
            notes: session.location.notes ? notes : nil,
            photoJPEGData: session.location.photo ? selfie?.jpegData : nil
        )
    }

    func handleSubmissionFailure(_ error: Error) {
        if let state = unavailableState(for: error) {
            unavailableState = state
        }
    }

    // MARK: - People

    func searchPeople(matching query: String) -> [PersonSummary] {
        let predicate = #Predicate<CachedPersonRecord> { person in
            person.displayName.localizedStandardContains(query)
                || person.email.localizedStandardContains(query)
        }
        var descriptor = FetchDescriptor<CachedPersonRecord>(
            predicate: predicate,
            sortBy: [SortDescriptor(\.displayName)]
        )
        descriptor.fetchLimit = 25

        let records = (try? modelContext.fetch(descriptor)) ?? []
        return records.map {
            PersonSummary(
                id: $0.userID,
                displayName: $0.displayName,
                email: $0.email
            )
        }
    }

    // MARK: - Private Helpers

    private func fetchPairableLocations(using payload: PairingPayload) async throws {
        guard !isBusy else {
            throw WoodGateError(message: "The station is busy. Please try again.")
        }
        isBusy = true
        defer { isBusy = false }

        let serverURL = try await ServerURL.resolve(payload.baseURL)
        let payload = PairingPayload(baseURL: serverURL.absoluteString, apiKey: payload.apiKey)

        guard
            let client = AppSettings.shared.woodGateClient(
                baseURLString: payload.baseURL,
                apiKey: payload.apiKey
            )
        else {
            throw WoodGateError(message: "Enter a valid HTTP or HTTPS server URL.")
        }
        let auth = try await client.authenticate()

        guard auth.principal.type == "api_key" else {
            throw WoodGateError(message: "These credentials did not authenticate as an API key.")
        }

        let allowedLocationIDs = Set(
            auth.access
                .filter { $0.resource == "checkins" && $0.action == "create" }
                .compactMap(\.locationId)
        )

        let locations = try await client.listLocations()
            .filter { $0.enabled && allowedLocationIDs.contains($0.id) }
            .map { SessionLocation(id: $0.id, name: $0.name) }
            .sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }

        guard !locations.isEmpty else {
            throw WoodGateError(message: "This API key does not have any enabled locations available.")
        }

        try Task.checkCancellation()
        locationSelection = LocationSelectionState(
            options: locations,
            payload: payload
        )
    }

    private func loadPeople() throws -> [PersonSummary] {
        let records = try modelContext.fetch(FetchDescriptor<CachedPersonRecord>())
        return records.map {
            PersonSummary(
                id: $0.userID,
                displayName: $0.displayName,
                email: $0.email
            )
        }
    }

    private func loadStoredSession(
        locationID: UUID,
        settings: AppSettings
    ) throws -> ActiveSession {
        let cachedPeople = try loadPeople()
        return ActiveSession(
            baseURLString: settings.baseURLString,
            location: ActiveLocation(
                id: locationID,
                name: settings.locationName,
                notes: settings.notes,
                photo: settings.photo,
                backgroundAssetID: settings.backgroundAssetID,
                logoAssetID: settings.logoAssetID
            ),
            people: cachedPeople,
            backgroundImage: nil,
            logoImage: nil,
            lastSyncedAt: settings.lastSyncedAt ?? .distantPast
        )
    }

    private func persist(session: ActiveSession, apiKey: String) throws {
        for person in try modelContext.fetch(FetchDescriptor<CachedPersonRecord>()) {
            modelContext.delete(person)
        }

        let settings = AppSettings.shared
        settings.baseURLString = session.baseURLString
        settings.locationID = session.location.id
        settings.locationName = session.location.name
        settings.notes = session.location.notes
        settings.photo = session.location.photo
        settings.backgroundAssetID = session.location.backgroundAssetID
        settings.logoAssetID = session.location.logoAssetID
        settings.lastSyncedAt = session.lastSyncedAt

        for person in session.people {
            modelContext.insert(
                CachedPersonRecord(
                    userID: person.id,
                    displayName: person.displayName,
                    email: person.email
                )
            )
        }

        settings.apiKey = apiKey
        try modelContext.save()
    }

    private func clearStoredSession(removeAPIKey: Bool) throws {
        let settings = AppSettings.shared

        for person in try modelContext.fetch(FetchDescriptor<CachedPersonRecord>()) {
            modelContext.delete(person)
        }

        try modelContext.save()
        settings.clear(removeAPIKey: removeAPIKey)
    }

    private func startBackgroundRefresh() {
        guard refreshTask == nil else { return }

        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(60))
                guard let self else { return }
                await refreshSession()
            }
        }
    }

    private func performRefresh(using session: ActiveSession) async {
        do {
            let settings = AppSettings.shared
            guard
                let client = settings.woodGateClient(
                    baseURLString: session.baseURLString,
                    apiKey: settings.apiKey
                )
            else {
                throw WoodGateError(message: "The saved server details are invalid.")
            }
            let location = try await client.getLocation(id: session.location.id)
            guard location.enabled else {
                unavailableState = .locationDisabled
                return
            }
            let people = try await client.listPeople(locationID: location.id)
            let refreshedSession = await makeSession(
                baseURLString: settings.baseURLString,
                location: location,
                people: people,
                lastSyncedAt: Date(),
                client: client,
                previousSession: session
            )

            try persist(session: refreshedSession, apiKey: settings.apiKey)
            currentSession = refreshedSession
            unavailableState = nil
        } catch {
            unavailableState = unavailableState(for: error)
        }
    }

    private func buildSession(
        baseURLString: String,
        client: WoodGateAPIClient,
        locationID: UUID,
        fallbackSession: ActiveSession?
    ) async throws -> ActiveSession {
        let location = try await client.getLocation(id: locationID)
        guard location.enabled else {
            unavailableState = .locationDisabled
            if let fallbackSession {
                return fallbackSession
            }

            return await makeSession(
                baseURLString: baseURLString,
                location: location,
                people: [],
                lastSyncedAt: Date(),
                client: client,
                previousSession: nil
            )
        }

        let people = try await client.listPeople(locationID: location.id)
        return await makeSession(
            baseURLString: baseURLString,
            location: location,
            people: people,
            lastSyncedAt: Date(),
            client: client,
            previousSession: fallbackSession
        )
    }

    private func makeSession(
        baseURLString: String,
        location: WoodGateLocationResponse,
        people: [PersonSummary],
        lastSyncedAt: Date,
        client: WoodGateAPIClient,
        previousSession: ActiveSession?
    ) async -> ActiveSession {
        async let backgroundImage = loadBrandingImage(
            client: client,
            assetID: location.backgroundAssetId,
            previousAssetID: previousSession?.location.backgroundAssetID,
            previousImage: previousSession?.backgroundImage
        )
        async let logoImage = loadBrandingImage(
            client: client,
            assetID: location.logoAssetId,
            previousAssetID: previousSession?.location.logoAssetID,
            previousImage: previousSession?.logoImage
        )

        return await ActiveSession(
            baseURLString: baseURLString,
            location: ActiveLocation(
                id: location.id,
                name: location.name,
                notes: location.notes,
                photo: location.photo,
                backgroundAssetID: location.backgroundAssetId,
                logoAssetID: location.logoAssetId
            ),
            people: people,
            backgroundImage: backgroundImage,
            logoImage: logoImage,
            lastSyncedAt: lastSyncedAt
        )
    }

    private func loadBrandingImage(
        client: WoodGateAPIClient,
        assetID: UUID?,
        previousAssetID: UUID?,
        previousImage: UIImage?
    ) async -> UIImage? {
        guard let assetID else {
            return nil
        }

        if assetID == previousAssetID, let previousImage {
            return previousImage
        }

        do {
            let data = try await client.getAssetContent(id: assetID)
            return UIImage(data: data) ?? previousImage
        } catch {
            return previousImage
        }
    }

    private func unavailableState(for error: Error) -> UnavailableState? {
        if let error = error as? URLError {
            return error.code == .cancelled ? nil : .connectivity
        }

        guard let apiError = error as? WoodGateAPIError else {
            return nil
        }

        switch apiError.statusCode {
        case 401, 403:
            return .authorization
        case 500 ... 599:
            return .connectivity
        default:
            return nil
        }
    }
}
