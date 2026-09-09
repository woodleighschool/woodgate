import SwiftUI

private struct ModelAlertModifier: ViewModifier {
    @Environment(ModelData.self) private var modelData

    let isEnabled: Bool

    private var alertBinding: Binding<AlertItem?> {
        Binding(
            get: { isEnabled ? modelData.alert : nil },
            set: { modelData.alert = $0 }
        )
    }

    func body(content: Content) -> some View {
        content
            .alert(item: alertBinding) { alert in
                Alert(
                    title: Text(alert.title),
                    message: Text(alert.message),
                    dismissButton: .default(Text("OK"))
                )
            }
    }
}

extension View {
    func modelAlert(isEnabled: Bool = true) -> some View {
        modifier(ModelAlertModifier(isEnabled: isEnabled))
    }
}
