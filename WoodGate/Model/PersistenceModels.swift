//
//  PersistenceModels.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

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
