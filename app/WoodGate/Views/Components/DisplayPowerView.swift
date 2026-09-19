import SwiftUI
import UIKit

/// UIKit controls brightness on the display attached to this scene's window.
struct DisplayPowerView: UIViewRepresentable {
    let keepAwake: Bool
    let dimmed: Bool

    func makeUIView(context _: Context) -> PowerView {
        PowerView()
    }

    func updateUIView(_ view: PowerView, context _: Context) {
        view.keepAwake = keepAwake
        view.dimmed = dimmed
        view.apply()
    }

    static func dismantleUIView(_ view: PowerView, coordinator _: ()) {
        view.restoreBrightness()
        UIApplication.shared.isIdleTimerDisabled = false
    }

    final class PowerView: UIView {
        var keepAwake = false
        var dimmed = false
        private var dimmedScreen: UIScreen?
        private var originalBrightness: CGFloat?

        override func didMoveToWindow() {
            super.didMoveToWindow()
            restoreBrightness()
            apply()
        }

        func apply() {
            UIApplication.shared.isIdleTimerDisabled = keepAwake && window != nil
            guard dimmed, let screen = window?.windowScene?.screen else {
                restoreBrightness()
                return
            }
            if dimmedScreen !== screen {
                restoreBrightness()
                dimmedScreen = screen
                originalBrightness = screen.brightness
            }
            screen.brightness = 0
        }

        func restoreBrightness() {
            if let dimmedScreen, let originalBrightness {
                dimmedScreen.brightness = originalBrightness
            }
            dimmedScreen = nil
            originalBrightness = nil
        }
    }
}
