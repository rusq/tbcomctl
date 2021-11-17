package registry

import (
	"sync"
	"time"

	"github.com/google/uuid"
	tb "gopkg.in/tucnak/telebot.v3"
)

const (
	Unknown = "[unknown]"
	nothing = 0
)

// Memory holds the state of the user interaction in-memory.
type Memory struct {
	cache      map[string]map[int]uuid.UUID // requests cache, maps message ID to request.
	waitMsgID  map[string]int               // await maps userID to the messageID and indicates that we're waiting for user to reply.
	values     map[string]string            // values entered, maps userID to the value
	messageIDs map[string]int               // messages sent, maps userID to the message_id
	mu         sync.RWMutex
}

// NewMemRegistry initialises new in-memory message and user registry.
func NewMemRegistry() *Memory {
	return &Memory{
		cache:      make(map[string]map[int]uuid.UUID),
		waitMsgID:  make(map[string]int),
		values:     make(map[string]string),
		messageIDs: make(map[string]int),
	}
}

// Register inserts the message into cache assigning it a random request id.
func (reg *Memory) Register(r tb.Recipient, msgID int) uuid.UUID {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	reg.requestEnsure(r)

	reqID := uuid.Must(uuid.NewUUID())
	reg.cache[r.Recipient()][msgID] = reqID
	reg.messageIDs[r.Recipient()] = msgID
	return reqID
}

// requestEnsure ensures that request cache is initialised.
func (reg *Memory) requestEnsure(r tb.Recipient) {
	if reg.cache == nil {
		reg.cache = make(map[string]map[int]uuid.UUID)
	}
	if reg.cache[r.Recipient()] == nil {
		reg.cache[r.Recipient()] = make(map[int]uuid.UUID)
	}
}

// RequestFor returns a request id for message ID and a bool. Bool will be true if
// message is registered and false otherwise.
func (reg *Memory) RequestFor(r tb.Recipient, msgID int) (uuid.UUID, bool) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	reg.requestEnsure(r)

	reqID, ok := reg.cache[r.Recipient()][msgID]
	return reqID, ok
}

// Unregister removes the request from cache.
func (reg *Memory) Unregister(r tb.Recipient, msgID int) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	delete(reg.cache[r.Recipient()], msgID)
}

// RequestInfo returns a request ID (or <unknown>) and a time of the request (or
// zero time) by calling parsing functions of the UUID instance.
func (reg *Memory) RequestInfo(r tb.Recipient, msgID int) (string, time.Time) {
	reqID, ok := reg.RequestFor(r, msgID)
	if !ok {
		return Unknown, time.Time{}
	}
	return reqID.String(), time.Unix(reqID.Time().UnixTime())
}

// Value returns the Controller Value for the recipient.
func (reg *Memory) Value(recipient string) (string, bool) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	if reg.values == nil {
		reg.values = make(map[string]string)
	}
	v, ok := reg.values[recipient]
	return v, ok
}

// SetValue sets the Controller value.
func (reg *Memory) SetValue(recipient string, value string) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if reg.values == nil {
		reg.values = make(map[string]string)
	}
	reg.values[recipient] = value
}

// OutgoingID returns the controller's outgoing message ID for the user.
func (reg *Memory) OutgoingID(recipient string) (int, bool) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	id, ok := reg.messageIDs[recipient]
	return id, ok
}

//
// waiting function
//

// Wait places the outbound message ID to the waiting list.  MessageID in
// outbound waiting list means that we expect the user to respond.
func (reg *Memory) Wait(r tb.Recipient, outboundID int) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if reg.waitMsgID == nil {
		reg.waitMsgID = make(map[string]int)
	}
	reg.waitMsgID[r.Recipient()] = outboundID
}

// StopWait removes the recipient from the wait list.
func (reg *Memory) StopWait(r tb.Recipient) int {
	outboundID := reg.waitMsgID[r.Recipient()]
	reg.waitMsgID[r.Recipient()] = nothing
	return outboundID
}

// WaitMsgID returns the ID of the message that was sent to the recipient, for
// which the control awaits the user response.
func (reg *Memory) WaitMsgID(r tb.Recipient) int {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	return reg.waitMsgID[r.Recipient()]
}

// IsWaiting returns true if we yet expecting to hear from the recipient.
func (reg *Memory) IsWaiting(r tb.Recipient) bool {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	return reg.waitMsgID[r.Recipient()] != nothing
}
