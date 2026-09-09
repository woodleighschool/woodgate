package station

import "testing"

func TestParseSubprotocolVersion(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		value       string
		wantVersion int
		wantOK      bool
	}{
		{value: Subprotocol, wantVersion: ProtocolVersion, wantOK: true},
		{value: "woodgate-station.v0", wantVersion: 0, wantOK: true},
		{value: "woodgate-station.v2", wantVersion: 2, wantOK: true},
		{value: "woodgate-station.v-1"},
		{value: "woodgate-station.vnext"},
		{value: "other.v1"},
	} {
		version, ok := parseSubprotocolVersion(test.value)
		if version != test.wantVersion || ok != test.wantOK {
			t.Errorf("parseSubprotocolVersion(%q) = (%d, %t), want (%d, %t)",
				test.value, version, ok, test.wantVersion, test.wantOK)
		}
	}
}

func TestValidBuild(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		value string
		want  bool
	}{
		{value: "1.2.3+42", want: true},
		{value: "dev", want: true},
		{value: ""},
		{value: "contains space"},
		{value: "line\nbreak"},
	} {
		if got := validBuild(test.value); got != test.want {
			t.Errorf("validBuild(%q) = %t, want %t", test.value, got, test.want)
		}
	}
}
