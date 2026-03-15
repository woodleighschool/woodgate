//
//  KeychainHelper.swift
//  WoodGate
//
//  Created by Alexander Hyde on 8/2/2026.
//

import Foundation
import Security

struct KeychainHelper {
  // MARK: - Properties

  static let shared = KeychainHelper()

  private let serviceName = "au.edu.woodleigh.WoodGate"

  // MARK: - Public Methods

  func save(_ data: Data, service: String, account: String) {
    let query =
      [
        kSecValueData: data,
        kSecClass: kSecClassGenericPassword,
        kSecAttrService: service,
        kSecAttrAccount: account,
      ] as [CFString: Any]

    let status = SecItemAdd(query as CFDictionary, nil)

    if status == errSecDuplicateItem {
      let query =
        [
          kSecAttrService: service,
          kSecAttrAccount: account,
          kSecClass: kSecClassGenericPassword,
        ] as [CFString: Any]

      let attributesToUpdate = [kSecValueData: data] as [CFString: Any]

      SecItemUpdate(query as CFDictionary, attributesToUpdate as CFDictionary)
    }
  }

  func read(service: String, account: String) -> Data? {
    let query =
      [
        kSecAttrService: service,
        kSecAttrAccount: account,
        kSecClass: kSecClassGenericPassword,
        kSecReturnData: true,
      ] as [CFString: Any]

    var result: AnyObject?
    SecItemCopyMatching(query as CFDictionary, &result)

    return result as? Data
  }

  func delete(service: String, account: String) {
    let query =
      [
        kSecAttrService: service,
        kSecAttrAccount: account,
        kSecClass: kSecClassGenericPassword,
      ] as [CFString: Any]

    SecItemDelete(query as CFDictionary)
  }

  // MARK: - String wrappers

  func save(_ value: String, key: String) {
    if value.isEmpty {
      delete(service: serviceName, account: key)
    } else {
      if let data = value.data(using: .utf8) {
        save(data, service: serviceName, account: key)
      }
    }
  }

  func read(key: String) -> String? {
    if let data = read(service: serviceName, account: key) {
      return String(data: data, encoding: .utf8)
    }
    return nil
  }
}
