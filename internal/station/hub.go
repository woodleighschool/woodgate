package station

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	presenceInterval = 20 * time.Second
	presenceTimeout  = 10 * time.Second
	controlBuffer    = 4
)

var errHubClosed = errors.New("station control hub closed")

type deviceStore interface {
	authenticate(context.Context, string) (*Station, error)
	observeClient(context.Context, int64, string, *int, string) error
}

// Hub owns this process's live Station control connections.
type Hub struct {
	store  deviceStore
	logger *slog.Logger
	build  string

	mu     sync.Mutex
	conns  map[int64]*controlConnection
	closed bool
}

type controlConnection struct {
	stationID  int64
	locationID int64
	secret     string
	appBuild   string
	ws         *websocket.Conn
	send       chan []byte
}

func newHub(store deviceStore, build string, logger *slog.Logger) *Hub {
	return &Hub{
		store:  store,
		logger: logger,
		build:  build,
		conns:  make(map[int64]*controlConnection),
	}
}

// Close disconnects every Station and refuses new connections.
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	connections := make([]*controlConnection, 0, len(h.conns))
	for _, connection := range h.conns {
		connections = append(connections, connection)
	}
	h.conns = make(map[int64]*controlConnection)
	h.mu.Unlock()

	for _, connection := range connections {
		_ = connection.ws.Close(websocket.StatusGoingAway, "server shutting down")
	}
}

func (h *Hub) serve(
	parent context.Context,
	ws *websocket.Conn,
	station *Station,
	secret string,
	appBuild string,
) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	connection := &controlConnection{
		stationID:  station.ID,
		locationID: station.LocationID,
		secret:     secret,
		appBuild:   appBuild,
		ws:         ws,
		send:       make(chan []byte, controlBuffer),
	}
	if !h.register(connection) {
		return errHubClosed
	}
	defer h.unregister(connection)

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.writeLoop(ctx, cancel, connection)
	}()

	if err := h.enqueueJSON(connection, serverMessage{
		Type: messageHello,
		Station: &stationIdentity{
			ID:         station.ID,
			Name:       station.Name,
			LocationID: station.LocationID,
		},
		ProtocolVersion: ProtocolVersion,
		ServerBuild:     h.build,
	}); err != nil {
		return err
	}

	err := h.readLoop(ctx, ws, connection)
	cancel()
	<-done
	return err
}

func (h *Hub) readLoop(
	ctx context.Context,
	ws *websocket.Conn,
	connection *controlConnection,
) error {
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		var message clientMessage
		if err := json.Unmarshal(data, &message); err != nil {
			_ = ws.Close(websocket.StatusPolicyViolation, "invalid control message")
			return fmt.Errorf("decode station control message: %w", err)
		}
		if message.Type != messagePresence {
			_ = ws.Close(websocket.StatusPolicyViolation, "unexpected control message")
			return fmt.Errorf("unexpected station control message %q", message.Type)
		}
		if err := h.observe(ctx, connection); err != nil {
			_ = ws.Close(websocket.StatusPolicyViolation, "station authorization changed")
			return err
		}
	}
}

func (h *Hub) writeLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	connection *controlConnection,
) {
	defer cancel()
	ticker := time.NewTicker(presenceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-connection.send:
			if err := connection.ws.Write(ctx, websocket.MessageText, message); err != nil {
				return
			}
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, presenceTimeout)
			err := h.observe(pingCtx, connection)
			if err == nil {
				err = connection.ws.Ping(pingCtx)
			}
			pingCancel()
			if err != nil {
				if !errors.Is(err, ErrUnauthorized) && ctx.Err() == nil {
					h.logger.WarnContext(ctx, "station presence failed",
						"station_id", connection.stationID,
						"err", err,
					)
				}
				return
			}
		}
	}
}

func (h *Hub) observe(ctx context.Context, connection *controlConnection) error {
	protocolVersion := ProtocolVersion
	return h.store.observeClient(
		ctx,
		connection.stationID,
		connection.secret,
		&protocolVersion,
		connection.appBuild,
	)
}

func (h *Hub) register(connection *controlConnection) bool {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return false
	}
	previous := h.conns[connection.stationID]
	h.conns[connection.stationID] = connection
	h.mu.Unlock()

	if previous != nil {
		_ = previous.ws.Close(websocket.StatusPolicyViolation, "replaced by a new connection")
	}
	return true
}

func (h *Hub) unregister(connection *controlConnection) {
	h.mu.Lock()
	if h.conns[connection.stationID] == connection {
		delete(h.conns, connection.stationID)
	}
	h.mu.Unlock()
}

func (h *Hub) configurationChanged(stationID int64, locationID int64) {
	h.mu.Lock()
	connection := h.conns[stationID]
	if connection != nil {
		connection.locationID = locationID
	}
	h.mu.Unlock()
	if connection != nil {
		h.enqueue(connection, mustMarshal(serverMessage{Type: messageConfigurationChanged}))
	}
}

func (h *Hub) configurationChangedForLocation(locationID int64) {
	h.mu.Lock()
	connections := make([]*controlConnection, 0)
	for _, connection := range h.conns {
		if connection.locationID == locationID {
			connections = append(connections, connection)
		}
	}
	h.mu.Unlock()
	message := mustMarshal(serverMessage{Type: messageConfigurationChanged})
	for _, connection := range connections {
		h.enqueue(connection, message)
	}
}

func (h *Hub) disconnect(stationID int64, reason string) {
	h.mu.Lock()
	connection := h.conns[stationID]
	if connection != nil {
		delete(h.conns, stationID)
	}
	h.mu.Unlock()
	if connection != nil {
		_ = connection.ws.Close(websocket.StatusPolicyViolation, reason)
	}
}

func (h *Hub) enqueueJSON(connection *controlConnection, value any) error {
	message, err := json.Marshal(value)
	if err != nil {
		return err
	}
	h.enqueue(connection, message)
	return nil
}

func (h *Hub) enqueue(connection *controlConnection, message []byte) {
	select {
	case connection.send <- message:
	default:
		_ = connection.ws.Close(websocket.StatusPolicyViolation, "station fell behind")
	}
}

func mustMarshal(value any) []byte {
	message, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return message
}
