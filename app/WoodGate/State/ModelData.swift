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
    var alert: AlertItem?
    var unavailableState: UnavailableState?
    var isBusy = false

    private let modelContext: ModelContext
    private var refreshTask: Task<Void, Never>?
    private var refreshInFlightTask: Task<Void, Never>?
    private var controlTask: Task<Void, Never>?
    private var controlSocket: URLSessionWebSocketTask?
    private var controlGeneration = 0

    // MARK: - Init

    init(modelContext: ModelContext) {
        self.modelContext = modelContext
        startBackgroundRefresh()
        Task { await bootstrap() }
    }

    // MARK: - Lifecycle

    func bootstrap() async {
        do {
            let settings = AppSettings.shared
            guard settings.hasPairing else {
                try clearStoredSession(removeStationSecret: true)
                currentSession = nil
                stopControlConnection()
                return
            }

            let cachedSession = try loadStoredSession(settings: settings)
            guard let client = settings.woodGateClient() else {
                currentSession = cachedSession
                unavailableState = cachedSession == nil ? .connectivity : nil
                return
            }

            do {
                let session = try await buildSession(
                    baseURLString: settings.baseURLString,
                    client: client,
                    fallbackSession: cachedSession
                )
                try persist(session: session, stationSecret: settings.stationSecret)
                currentSession = session
                unavailableState = session.location.enabled ? nil : .locationDisabled
            } catch {
                currentSession = cachedSession
                unavailableState = unavailableState(for: error)
            }
            startControlConnection()
        } catch {
            alert = AlertItem(title: "Could Not Start", message: error.localizedDescription)
        }
    }

    func handleSceneActive() async {
        startControlConnection()
        await refreshSession()
    }

    // MARK: - Pairing

    @discardableResult
    func beginPairing(with payloadText: String) async -> Bool {
        do {
            return try await beginPairing(with: PairingPayload.parse(json: payloadText))
        } catch {
            alert = AlertItem(title: "Could Not Pair", message: error.localizedDescription)
            return false
        }
    }

    @discardableResult
    func beginPairing(with payload: PairingPayload) async -> Bool {
        isBusy = true
        defer { isBusy = false }

        do {
            let baseURL = payload.baseURL.trimmingCharacters(in: .whitespacesAndNewlines)
            let stationSecret = payload.stationSecret.trimmingCharacters(in: .whitespacesAndNewlines)
            guard
                let client = AppSettings.shared.woodGateClient(
                    baseURLString: baseURL,
                    stationSecret: stationSecret
                )
            else {
                throw WoodGateError(message: "Enter a valid server URL and Station secret.")
            }

            let session = try await buildSession(
                baseURLString: baseURL,
                client: client,
                fallbackSession: nil
            )
            try persist(session: session, stationSecret: stationSecret)
            currentSession = session
            unavailableState = session.location.enabled ? nil : .locationDisabled
            startControlConnection()
            return true
        } catch {
            alert = AlertItem(title: "Could Not Pair", message: error.localizedDescription)
            return false
        }
    }

    func forgetPairing() {
        do {
            stopControlConnection()
            try clearStoredSession(removeStationSecret: true)
            currentSession = nil
            unavailableState = nil
        } catch {
            alert = AlertItem(title: "Could Not Forget Pairing", message: error.localizedDescription)
        }
    }

    // MARK: - Session Refresh

    func refreshSession() async {
        guard let currentSession, AppSettings.shared.hasPairing else { return }
        guard !isBusy else { return }

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
        person: PersonSummary,
        direction: CheckinDirectionChoice,
        notes: String,
        selfie: CapturedSelfie?
    ) async throws {
        isBusy = true
        defer { isBusy = false }

        if let refreshInFlightTask {
            await refreshInFlightTask.value
        }

        guard let session = currentSession else {
            throw WoodGateError(message: "The Station configuration is unavailable.")
        }
        let trimmedNotes = notes.trimmingCharacters(in: .whitespacesAndNewlines)
        let photoJPEGData: Data?
        if session.location.photo {
            guard let selfie else {
                throw WoodGateError(message: "Add a selfie to continue.")
            }
            photoJPEGData = selfie.jpegData
        } else {
            photoJPEGData = nil
        }

        guard let client = AppSettings.shared.woodGateClient() else {
            throw WoodGateError(message: "The saved Station pairing is invalid.")
        }
        _ = try await client.createCheckin(
            personID: person.id,
            direction: direction,
            notes: session.location.notes ? trimmedNotes : nil,
            photoJPEGData: photoJPEGData
        )

        var submittedSession = session
        submittedSession.lastSyncedAt = Date()
        currentSession = submittedSession
        alert = AlertItem(
            title: "Submitted",
            message: "\(person.displayName) was \(direction == .checkIn ? "checked in" : "checked out")."
        )
    }

    func handleSubmissionFailure(_ error: Error) {
        if let state = unavailableState(for: error) {
            unavailableState = state
            return
        }

        alert = AlertItem(title: "Could Not Submit", message: error.localizedDescription)
    }

    // MARK: - People

    func searchPeople(matching query: String) -> [PersonSummary] {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !q.isEmpty, let currentSession else {
            return []
        }

        let predicate = #Predicate<CachedStationPersonRecord> { person in
            person.displayName.localizedStandardContains(q)
                || person.email.localizedStandardContains(q)
        }
        var descriptor = FetchDescriptor<CachedStationPersonRecord>(
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

    private func loadPeople() throws -> [PersonSummary] {
        let records = try modelContext.fetch(FetchDescriptor<CachedStationPersonRecord>())
        return records.map {
            PersonSummary(
                id: $0.userID,
                displayName: $0.displayName,
                email: $0.email
            )
        }
    }

    private func loadStoredSession(settings: AppSettings) throws -> ActiveSession? {
        guard let stationID = settings.stationID, let locationID = settings.locationID else {
            return nil
        }
        return try ActiveSession(
            baseURLString: settings.baseURLString,
            stationID: stationID,
            stationName: settings.stationName,
            location: ActiveLocation(
                id: locationID,
                name: settings.locationName,
                enabled: settings.locationEnabled,
                notes: settings.notes,
                photo: settings.photo,
                backgroundObjectID: settings.backgroundObjectID,
                logoObjectID: settings.logoObjectID
            ),
            people: loadPeople(),
            backgroundImage: nil,
            logoImage: nil,
            lastSyncedAt: settings.lastSyncedAt ?? .distantPast
        )
    }

    private func persist(session: ActiveSession, stationSecret: String) throws {
        for person in try modelContext.fetch(FetchDescriptor<CachedStationPersonRecord>()) {
            modelContext.delete(person)
        }

        let settings = AppSettings.shared
        settings.baseURLString = session.baseURLString
        settings.stationID = session.stationID
        settings.stationName = session.stationName
        settings.locationID = session.location.id
        settings.locationName = session.location.name
        settings.locationEnabled = session.location.enabled
        settings.notes = session.location.notes
        settings.photo = session.location.photo
        settings.backgroundObjectID = session.location.backgroundObjectID
        settings.logoObjectID = session.location.logoObjectID
        settings.lastSyncedAt = session.lastSyncedAt

        for person in session.people {
            modelContext.insert(
                CachedStationPersonRecord(
                    userID: person.id,
                    displayName: person.displayName,
                    email: person.email
                )
            )
        }

        settings.stationSecret = stationSecret
        try modelContext.save()
    }

    private func clearStoredSession(removeStationSecret: Bool) throws {
        let settings = AppSettings.shared

        for person in try modelContext.fetch(FetchDescriptor<CachedStationPersonRecord>()) {
            modelContext.delete(person)
        }

        try modelContext.save()
        settings.clear(removeStationSecret: removeStationSecret)
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
            guard let client = AppSettings.shared.woodGateClient() else {
                throw WoodGateError(message: "The saved Station pairing is invalid.")
            }
            let refreshed = try await buildSession(
                baseURLString: session.baseURLString,
                client: client,
                fallbackSession: session
            )
            try persist(session: refreshed, stationSecret: AppSettings.shared.stationSecret)
            currentSession = refreshed
            unavailableState = refreshed.location.enabled ? nil : .locationDisabled
        } catch {
            unavailableState = unavailableState(for: error)
        }
    }

    private func buildSession(
        baseURLString: String,
        client: WoodGateAPIClient,
        fallbackSession: ActiveSession?
    ) async throws -> ActiveSession {
        let configuration = try await client.getConfiguration()
        let people = configuration.location.enabled
            ? try await client.listPeople()
            : fallbackSession?.people ?? []
        return await makeSession(
            baseURLString: baseURLString,
            configuration: configuration,
            people: people,
            lastSyncedAt: Date(),
            client: client,
            previousSession: fallbackSession
        )
    }

    private func makeSession(
        baseURLString: String,
        configuration: WoodGateStationConfigurationResponse,
        people: [PersonSummary],
        lastSyncedAt: Date,
        client: WoodGateAPIClient,
        previousSession: ActiveSession?
    ) async -> ActiveSession {
        async let backgroundImage = loadBrandingImage(
            objectID: configuration.location.backgroundObjectId,
            previousObjectID: previousSession?.location.backgroundObjectID,
            previousImage: previousSession?.backgroundImage,
            load: client.getLocationBackground
        )
        async let logoImage = loadBrandingImage(
            objectID: configuration.location.logoObjectId,
            previousObjectID: previousSession?.location.logoObjectID,
            previousImage: previousSession?.logoImage,
            load: client.getLocationLogo
        )

        return await ActiveSession(
            baseURLString: baseURLString,
            stationID: configuration.stationId,
            stationName: configuration.stationName,
            location: ActiveLocation(
                id: configuration.location.id,
                name: configuration.location.name,
                enabled: configuration.location.enabled,
                notes: configuration.location.notes,
                photo: configuration.location.photo,
                backgroundObjectID: configuration.location.backgroundObjectId,
                logoObjectID: configuration.location.logoObjectId
            ),
            people: people,
            backgroundImage: backgroundImage,
            logoImage: logoImage,
            lastSyncedAt: lastSyncedAt
        )
    }

    private func loadBrandingImage(
        objectID: Int64?,
        previousObjectID: Int64?,
        previousImage: UIImage?,
        load: @Sendable () async throws -> Data
    ) async -> UIImage? {
        guard let objectID else {
            return nil
        }

        if objectID == previousObjectID, let previousImage {
            return previousImage
        }

        do {
            let data = try await load()
            return UIImage(data: data) ?? previousImage
        } catch {
            return previousImage
        }
    }

    private func startControlConnection() {
        guard controlTask == nil, currentSession != nil, AppSettings.shared.hasPairing else {
            return
        }
        controlGeneration += 1
        let generation = controlGeneration
        controlTask = Task { [weak self] in
            await self?.runControlConnection(generation: generation)
        }
    }

    private func stopControlConnection() {
        controlGeneration += 1
        controlTask?.cancel()
        controlTask = nil
        controlSocket?.cancel(with: .goingAway, reason: nil)
        controlSocket = nil
    }

    private func runControlConnection(generation: Int) async {
        var activeSocket: URLSessionWebSocketTask?
        defer {
            activeSocket?.cancel(with: .goingAway, reason: nil)
            if controlGeneration == generation {
                controlSocket = nil
                controlTask = nil
            }
        }

        while !Task.isCancelled, controlGeneration == generation, AppSettings.shared.hasPairing {
            do {
                guard let client = AppSettings.shared.woodGateClient() else { return }
                let socket = try client.makeControlTask(appBuild: Self.appBuild)
                activeSocket = socket
                controlSocket = socket
                socket.resume()
                try await sendPresence(to: socket)

                while !Task.isCancelled {
                    let message = try await receiveControlMessage(from: socket)
                    switch message.type {
                    case "hello":
                        try await sendPresence(to: socket)
                    case "configuration_changed":
                        await refreshSession()
                    default:
                        continue
                    }
                }
            } catch {
                activeSocket?.cancel(with: .goingAway, reason: nil)
                activeSocket = nil
                if controlGeneration == generation {
                    controlSocket = nil
                }
                guard !Task.isCancelled else { return }
                do {
                    try await Task.sleep(for: .seconds(5))
                } catch {
                    return
                }
            }
        }
    }

    private func receiveControlMessage(
        from socket: URLSessionWebSocketTask
    ) async throws -> WoodGateStationControlMessage {
        let message = try await socket.receive()
        let data = switch message {
        case let .data(data):
            data
        case let .string(text):
            Data(text.utf8)
        @unknown default:
            throw WoodGateError(message: "The Station control message was invalid.")
        }
        return try JSONDecoder().decode(WoodGateStationControlMessage.self, from: data)
    }

    private func sendPresence(to socket: URLSessionWebSocketTask) async throws {
        try await socket.send(.string(#"{"type":"presence"}"#))
    }

    private static var appBuild: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "0"
        return "\(version)+\(build)"
    }

    private func unavailableState(for error: Error) -> UnavailableState? {
        if error is URLError {
            return .connectivity
        }

        guard let apiError = error as? WoodGateAPIError else {
            return nil
        }

        switch apiError.statusCode {
        case 401, 403:
            return .authorization
        case 409:
            return .locationDisabled
        default:
            return .connectivity
        }
    }
}
