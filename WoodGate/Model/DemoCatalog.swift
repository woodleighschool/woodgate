//
//  DemoCatalog.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import Foundation

enum DemoCatalog {
  static let reviewLocation = SessionLocation(
    id: UUID(uuidString: "31F63EC7-41E6-43D2-8D9E-2F4B4AA3D3B1")!,
    name: "Demo Location"
  )

  static func session() -> ActiveSession {
    ActiveSession(
      mode: .demo,
      baseURLString: "Demo Mode",
      location: ActiveLocation(
        id: reviewLocation.id,
        name: reviewLocation.name,
        notes: true,
        photo: true,
        backgroundAssetID: nil,
        logoAssetID: nil
      ),
      people: people,
      backgroundImage: nil,
      logoImage: nil,
      lastSyncedAt: Date()
    )
  }

  static let people: [PersonSummary] = demoPeople(
    in: [
      // https://www.randomlists.com/random-names
      "Angelina Kelley", "Conor Phelps", "Hamza Stewart", "Roberto Hill", "Magdalena Beck",
      "Gaven Trujillo", "Clayton Lyons", "Victoria Ritter", "Rowan Morales", "Mylie Hood",
      "Kamren Fletcher", "Nikhil Cabrera", "Charity Hinton", "Julien Cortez", "Quinten Weaver",
      "Kyla Bender", "Kelton Dalton", "Damari Zuniga", "Barbara Valdez", "June Villanueva",
    ]
  )

  private static func demoPeople(in names: [String]) -> [PersonSummary] {
    names.enumerated().map { offset, name in
      let uuid =
        UUID(uuidString: String(format: "00000000-0000-0000-0000-%012d", offset + 1)) ?? UUID()
      return PersonSummary(
        id: uuid,
        displayName: name,
        email:
        name
          .lowercased()
          .replacingOccurrences(of: " ", with: ".")
          .appending("@example.com")
      )
    }
  }
}
