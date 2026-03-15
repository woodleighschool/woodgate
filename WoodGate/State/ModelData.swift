//
//  ModelData.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

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

  init(modelContext: ModelContext) {
    self.modelContext = modelContext
    startBackgroundRefresh()
    Task { await bootstrap() }
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
          mode: .paired,
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

  // MARK: - Demo

  func beginDemoMode() {
    currentSession = DemoCatalog.session()
    locationSelection = nil
    unavailableState = nil
  }

  func exitDemoMode() {
    guard currentSession?.mode == .demo else { return }

    currentSession = nil
    locationSelection = nil
    unavailableState = nil
  }

  func toggleDemoNotes() {
    guard var session = currentSession, session.mode == .demo else { return }

    session.location = ActiveLocation(
      id: session.location.id,
      name: session.location.name,
      notes: !session.location.notes,
      photo: session.location.photo,
      backgroundAssetID: session.location.backgroundAssetID,
      logoAssetID: session.location.logoAssetID
    )
    currentSession = session
  }

  func toggleDemoPhoto() {
    guard var session = currentSession, session.mode == .demo else { return }

    session.location = ActiveLocation(
      id: session.location.id,
      name: session.location.name,
      notes: session.location.notes,
      photo: !session.location.photo,
      backgroundAssetID: session.location.backgroundAssetID,
      logoAssetID: session.location.logoAssetID
    )
    currentSession = session
  }

  // MARK: - Pairing

  func beginPairing(with payloadText: String) async {
    do {
      let payload = try PairingPayload.parse(json: payloadText)
      try await fetchPairableLocations(using: payload)
    } catch {
      alert = AlertItem(title: "QR Code Not Recognised", message: error.localizedDescription)
    }
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
    let payload = locationSelection!.payload

    isBusy = true
    defer { isBusy = false }

    do {
      guard
        let client = AppSettings.shared.woodGateClient(
          baseURLString: payload.baseURL,
          apiKey: payload.apiKey
        )
      else {
        throw WoodGateError(message: "The QR code does not contain a valid server URL.")
      }
      let location = try await client.getLocation(id: option.id)
      guard location.enabled else {
        throw WoodGateError(message: "That location is currently disabled.")
      }
      let people = try await client.listPeople(locationID: location.id)
      let session = await makeSession(
        mode: .paired,
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
    guard let currentSession, currentSession.mode == .paired else { return }
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

    let session = currentSession!
    let trimmedNotes = notes.trimmingCharacters(in: .whitespacesAndNewlines)
    let photoJPEGData = session.location.photo ? selfie!.jpegData : nil

    if session.mode == .demo {
      try await Task.sleep(for: .milliseconds(500))
      let message =
        "\(person.displayName) was \(direction == .checkIn ? "checked in" : "checked out") in demo mode."
      alert = AlertItem(title: "Submitted", message: message)
      return
    }

    let settings = AppSettings.shared
    let client = settings.woodGateClient(
      baseURLString: session.baseURLString,
      apiKey: settings.apiKey
    )!

    _ = try await client.createCheckin(
      locationID: session.location.id,
      userID: person.id,
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

    if currentSession.isDemo {
      return currentSession.people
        .filter { person in
          person.displayName.localizedStandardContains(q)
            || person.email.localizedStandardContains(q)
        }
        .sorted {
          $0.displayName.localizedCaseInsensitiveCompare($1.displayName) == .orderedAscending
        }
        .prefix(25)
        .map(\.self)
    }

    let predicate = #Predicate<CachedPersonRecord> { person in
      person.displayName.localizedStandardContains(q)
        || person.email.localizedStandardContains(q)
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
    isBusy = true
    defer { isBusy = false }

    guard
      let client = AppSettings.shared.woodGateClient(
        baseURLString: payload.baseURL,
        apiKey: payload.apiKey
      )
    else {
      throw WoodGateError(message: "The QR code does not contain a valid server URL.")
    }
    let auth = try await client.authenticate()

    guard auth.principal.type == "api_key" else {
      throw WoodGateError(message: "That QR code did not authenticate as an API key.")
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
      mode: .paired,
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
        mode: .paired,
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
    mode: SessionMode,
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
        mode: mode,
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
      mode: mode,
      baseURLString: baseURLString,
      location: location,
      people: people,
      lastSyncedAt: Date(),
      client: client,
      previousSession: fallbackSession
    )
  }

  private func makeSession(
    mode: SessionMode,
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
      mode: mode,
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
    if error is URLError {
      return .connectivity
    }

    guard let apiError = error as? WoodGateAPIError else {
      return nil
    }

    switch apiError.statusCode {
    case 401, 403:
      return .authorization
    default:
      return .connectivity
    }
  }
}
