import Security
import SimpleKeychain

struct KeychainHelper {
    static let shared = KeychainHelper()

    private let keychain = SimpleKeychain(
        service: "au.edu.vic.woodleigh.WoodGateApp",
        accessibility: .whenUnlockedThisDeviceOnly,
        attributes: [kSecUseDataProtectionKeychain as String: true]
    )
    private let legacyKeychain = SimpleKeychain(service: "au.edu.woodleigh.WoodGate")

    func save(_ value: String, key: String) {
        if value.isEmpty {
            try? keychain.deleteItem(forKey: key)
            try? legacyKeychain.deleteItem(forKey: key)
            return
        }

        do {
            try keychain.set(value, forKey: key)
        } catch {
            return
        }

        try? legacyKeychain.deleteItem(forKey: key)
    }

    func read(key: String) -> String? {
        do {
            let value = try keychain.string(forKey: key)
            try? legacyKeychain.deleteItem(forKey: key)
            return value
        } catch SimpleKeychainError.itemNotFound {
            return migrateLegacyValue(forKey: key)
        } catch {
            return nil
        }
    }

    func delete(key: String) {
        try? keychain.deleteItem(forKey: key)
        try? legacyKeychain.deleteItem(forKey: key)
    }

    func migrate(from oldKey: String, to newKey: String) -> String? {
        if let current = read(key: newKey) {
            delete(key: oldKey)
            return current
        }

        guard let value = read(key: oldKey) else {
            return nil
        }

        do {
            try keychain.set(value, forKey: newKey)
        } catch {
            return value
        }

        delete(key: oldKey)
        return value
    }

    private func migrateLegacyValue(forKey key: String) -> String? {
        guard let value = try? legacyKeychain.string(forKey: key) else {
            return nil
        }

        do {
            try keychain.set(value, forKey: key)
        } catch {
            return value
        }

        try? legacyKeychain.deleteItem(forKey: key)
        return value
    }
}
