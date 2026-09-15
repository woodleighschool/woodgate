import Foundation
import Testing
@testable import WoodGate

@Suite("Display settings")
@MainActor
struct DisplaySettingsTests {
    private var calendar: Calendar {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "Australia/Melbourne")!
        return calendar
    }

    @Test("disabled schedules stay open")
    func disabledSchedule() {
        #expect(DisplaySchedule().isOpen(at: date("2026-09-12T03:00:00+10:00"), calendar: calendar))
    }

    @Test("daytime hours include opening and exclude closing", arguments: [
        ("2026-09-11T07:59:59+10:00", false),
        ("2026-09-11T08:00:00+10:00", true),
        ("2026-09-11T15:59:59+10:00", true),
        ("2026-09-11T16:00:00+10:00", false),
        ("2026-09-12T12:00:00+10:00", false),
    ])
    func daytimeHours(value: String, open: Bool) {
        var schedule = DisplaySchedule()
        schedule.enabled = true
        #expect(schedule.isOpen(at: date(value), calendar: calendar) == open)
    }

    @Test("overnight hours belong to their opening day", arguments: [
        ("2026-09-11T21:59:59+10:00", false),
        ("2026-09-11T22:00:00+10:00", true),
        ("2026-09-12T05:59:59+10:00", true),
        ("2026-09-12T06:00:00+10:00", false),
        ("2026-09-12T22:00:00+10:00", false),
        ("2026-09-13T05:00:00+10:00", false),
        ("2026-09-14T05:00:00+10:00", true),
    ])
    func overnightHours(value: String, open: Bool) {
        var schedule = DisplaySchedule()
        schedule.enabled = true
        schedule.days = [.friday, .sunday]
        schedule.start = "22:00"
        schedule.end = "06:00"
        #expect(schedule.isOpen(at: date(value), calendar: calendar) == open)
    }

    @Test("hours follow the local clock across daylight saving", arguments: [
        "2026-10-04T03:00:00+11:00",
        "2026-04-05T02:30:00+11:00",
        "2026-04-05T02:30:00+10:00",
    ])
    func daylightSaving(value: String) {
        var schedule = DisplaySchedule()
        schedule.enabled = true
        schedule.days = [.sunday]
        schedule.start = "01:00"
        schedule.end = "04:00"
        #expect(schedule.isOpen(at: date(value), calendar: calendar))
    }

    @Test("invalid hours leave the display open", arguments: [
        "8:00", "24:00", "08:60", "aa:00", "08:00:00", "08:", "-1:00", " 8:00", "16:00",
    ])
    func invalidHours(start: String) {
        var schedule = DisplaySchedule()
        schedule.enabled = true
        schedule.start = start
        #expect(schedule.validationMessage != nil)
        #expect(schedule.isOpen(at: date("2026-09-11T12:00:00+10:00"), calendar: calendar))
    }

    @Test("a schedule requires at least one opening day")
    func missingDays() {
        var schedule = DisplaySchedule()
        schedule.days = []
        #expect(schedule.validationMessage != nil)
    }

    @Test("an enabled schedule prevents sleep regardless of the local awake setting")
    func schedulePreventsSleep() {
        var settings = DisplaySettings()
        settings.keepAwake = false
        #expect(!settings.preventsSleep)
        settings.schedule.enabled = true
        #expect(settings.preventsSleep)
    }

    @Test("local settings survive persistence")
    func persistence() throws {
        var settings = DisplaySettings()
        settings.keepAwake = false
        settings.schedule.enabled = true
        let data = try PropertyListEncoder().encode(settings)
        #expect(try PropertyListDecoder().decode(DisplaySettings.self, from: data) == settings)
    }

    private func date(_ value: String) -> Date {
        ISO8601DateFormatter().date(from: value)!
    }
}
