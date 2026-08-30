import Security
import SimpleKeychain

struct KeychainHelper {
    static let shared = KeychainHelper()

    private let keychain = SimpleKeychain(
        service: "au.edu.vic.woodleigh.WoodGateApp",
        accessibility: .whenUnlockedThisDeviceOnly,
        attributes: [kSecUseDataProtectionKeychain as String: true]
    )

    func save(_ value: String, key: String) {
        if value.isEmpty {
            try? keychain.deleteItem(forKey: key)
            return
        }

        try? keychain.set(value, forKey: key)
    }

    func read(key: String) -> String? {
        try? keychain.string(forKey: key)
    }
}
