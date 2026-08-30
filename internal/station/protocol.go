package station

import (
	"strconv"
	"strings"
)

const (
	// ProtocolVersion is the only Station protocol spoken by this server build.
	ProtocolVersion = 1
	// Subprotocol is the WebSocket subprotocol for Station v1.
	Subprotocol = "woodgate-station.v1"
	// ProtocolHeader identifies the required subprotocol after a rejected upgrade.
	ProtocolHeader = "Woodgate-Station-Protocol"
	// AppBuildHeader carries the companion app build during a WebSocket upgrade.
	AppBuildHeader = "Woodgate-Station-Build"
	// ServerBuildHeader carries the server build during a WebSocket upgrade.
	ServerBuildHeader = "Woodgate-Version"

	maxBuildLength    = 128
	subprotocolPrefix = "woodgate-station.v"
)

const (
	messageHello                = "hello"
	messageConfigurationChanged = "configuration_changed"
	messagePresence             = "presence"
)

type serverMessage struct {
	Type            string           `json:"type"`
	Station         *stationIdentity `json:"station,omitempty"`
	ProtocolVersion int              `json:"protocol_version,omitempty"`
	ServerBuild     string           `json:"server_build,omitempty"`
}

type stationIdentity struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	LocationID int64  `json:"location_id"`
}

type clientMessage struct {
	Type string `json:"type"`
}

func validBuild(value string) bool {
	if value == "" || len(value) > maxBuildLength {
		return false
	}
	for i := range len(value) {
		if value[i] < '!' || value[i] > '~' {
			return false
		}
	}
	return true
}

func parseSubprotocolVersion(subprotocol string) (int, bool) {
	value, ok := strings.CutPrefix(subprotocol, subprotocolPrefix)
	if !ok {
		return 0, false
	}
	version, err := strconv.Atoi(value)
	if err != nil || version < 0 {
		return 0, false
	}
	return version, true
}

func offeredSubprotocols(headerValues []string) []string {
	var protocols []string
	for _, value := range headerValues {
		for protocol := range strings.SplitSeq(value, ",") {
			protocols = append(protocols, strings.TrimSpace(protocol))
		}
	}
	return protocols
}
