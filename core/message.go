package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	"github.com/google/uuid"
)

// Version ...
const Version = 1

const MsgTypeInvalid = 0
const MsgTypeSyncRequest = 1
const MsgTypeSyncResponse = 2
const MsgTypeDataRequest = 3
const MsgTypeDataResponse = 4

const MsgUserAgentUnknown = 0

// Message interface
type Message interface {
	Read(c net.Conn) error
	Write(c net.Conn) (int, error)
}

// Header ...
type Header struct {
	Magic     byte
	Version   uint8
	MsgType   uint8
	UserAgent uint8
	UUID      uuid.UUID
	Length    uint32
}

func serialize(hdr *Header, buf *bytes.Buffer) {
	binary.Write(buf, binary.BigEndian, hdr.Magic)
	binary.Write(buf, binary.BigEndian, hdr.Version)
	binary.Write(buf, binary.BigEndian, hdr.MsgType)
	binary.Write(buf, binary.BigEndian, hdr.UserAgent)
	binary.Write(buf, binary.BigEndian, &hdr.UUID) // 16 bytes
	binary.Write(buf, binary.BigEndian, hdr.Length)
}

// makeHeader ...
func makeHeader(msgType uint8, userAgent uint8) Header {
	h := Header{
		Magic:     byte('e'),
		Version:   Version,
		MsgType:   msgType,
		UserAgent: userAgent,
		UUID:      uuid.UUID{},
		Length:    0,
	}
	return h
}

// SyncRequestMsg ...
type SyncRequestMsg struct {
	Hdr         Header    // 24 bytes
	ServiceUUID uuid.UUID // 16 bytes
	Location    Location  // 24 bytes
}

// Write a SyncRequestMsg to the connection 'c'. Return error on failure
func (m *SyncRequestMsg) Write(c net.Conn) (int, error) {
	buf := new(bytes.Buffer)
	m.Hdr.Length = 48                                   // payload only
	serialize(&m.Hdr, buf)                              // write the header
	binary.Write(buf, binary.BigEndian, &m.ServiceUUID) // 16 bytes
	Serialize(&m.Location, buf)                         // 32 bytes
	return c.Write(buf.Bytes())
}

// Read a SyncRequestMsg from the connection 'c'. Return error on failure.
func (m *SyncRequestMsg) Read(c net.Conn) error {
	err := ReadHeader(&m.Hdr, c)
	if err != nil || m.Hdr.MsgType != MsgTypeSyncRequest {
		return err
	}

	// Read 'Length' bytes from the connection
	msgBytes := make([]byte, m.Hdr.Length)
	_, err = c.Read(msgBytes)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(msgBytes)
	binary.Read(buf, binary.BigEndian, &m.ServiceUUID)
	Deserialize(&m.Location, buf)
	return nil
}

// MakeSyncRequestMsg ...
func MakeSyncRequestMsg(clientUUID uuid.UUID, serviceUUID uuid.UUID, loc Location) SyncRequestMsg {
	h := makeHeader(MsgTypeSyncRequest, 0)
	h.UUID = clientUUID
	m := SyncRequestMsg{Hdr: h, ServiceUUID: serviceUUID, Location: loc}
	return m
}

// SyncResponseMsg ...
type SyncResponseMsg struct {
	Hdr         Header    // 24 bytes
	ServiceUUID uuid.UUID // 16 bytes
	TokenId     TokenID   // 16 bytes
}

// Write the SyncResponseMsg to the connection 'c'
func (m *SyncResponseMsg) Write(c net.Conn) (int, error) {
	buf := new(bytes.Buffer)
	m.Hdr.Length = 32                                   // payload only
	serialize(&m.Hdr, buf)                              // write the header
	binary.Write(buf, binary.BigEndian, &m.ServiceUUID) // 16 bytes
	binary.Write(buf, binary.BigEndian, &m.TokenId)			// 16 bytes
	return c.Write(buf.Bytes())
}

// Read a SyncResponseMsg from the connection 'c'. Return error on failure.
func (m *SyncResponseMsg) Read(c net.Conn) error {
	err := ReadHeader(&m.Hdr, c)
	if err != nil || m.Hdr.MsgType != MsgTypeSyncResponse {
		return err
	}

	// Read 'Length' bytes from the connection
	msgBytes := make([]byte, m.Hdr.Length)
	_, err = c.Read(msgBytes)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(msgBytes)
	binary.Read(buf, binary.BigEndian, &m.ServiceUUID)
	binary.Read(buf, binary.BigEndian, &m.TokenId)

	return nil
}

// MakeSyncResponseMsg ...
func MakeSyncResponseMsg(clientUUID uuid.UUID, tokenId TokenID) SyncResponseMsg {
	h := makeHeader(MsgTypeSyncResponse, 0)
	h.UUID = clientUUID
	m := SyncResponseMsg{Hdr: h, TokenId: tokenId}
	return m
}

// DataRequestMsg ...
type DataRequestMsg struct {
	Hdr      Header   // 24 bytes
	Location Location // 24 bytes
}

// Write the DataRequestMsg to the connection 'c'.
func (m *DataRequestMsg) Write(c net.Conn) (int, error) {
	buf := new(bytes.Buffer)
	m.Hdr.Length = 32           // payload only
	serialize(&m.Hdr, buf)      // write the header
	Serialize(&m.Location, buf) // 32 bytes
	return c.Write(buf.Bytes())
}

// Read a DataRequestMsg from the connection 'c'. Return error on failure.
func (m *DataRequestMsg) Read(c net.Conn) error {
	err := ReadHeader(&m.Hdr, c)
	if err != nil || m.Hdr.MsgType != MsgTypeDataRequest {
		return err
	}

	// Read 'Length' bytes from the connection
	msgBytes := make([]byte, m.Hdr.Length)
	_, err = c.Read(msgBytes)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(msgBytes)
	Deserialize(&m.Location, buf)
	return nil
}

// MakeDataRequestMsg ...
func MakeDataRequestMsg(tokenId TokenID, loc Location) DataRequestMsg {
	h := makeHeader(MsgTypeDataRequest, 0)
	h.UUID = uuid.UUID(tokenId)
	m := DataRequestMsg{Hdr: h, Location: loc}
	return m
}

// DataResponseMsg ...
type DataResponseMsg struct {
	Hdr  Header
	Data *bytes.Buffer
}

// Write the DataResponseMsg to the connection 'c'.
func (m *DataResponseMsg) Write(c net.Conn) (int, error) {
	buf := new(bytes.Buffer)
	m.Hdr.Length = uint32(m.Data.Len()) // payload only
	serialize(&m.Hdr, buf)              // write the header
	buf.Write(m.Data.Bytes())           // write the data as is
	return c.Write(buf.Bytes())
}

// Read a DataRequestMsg from the connection 'c'. Return error on failure.
func (m *DataResponseMsg) Read(c net.Conn) error {
	err := ReadHeader(&m.Hdr, c)
	if err != nil || m.Hdr.MsgType != MsgTypeDataResponse {
		return err
	}

	// Read 'Length-sizeof(token)' bytes from the connection
	msgBytes := make([]byte, m.Hdr.Length)
	_, err = c.Read(msgBytes)
	if err != nil {
		return err
	}

	m.Data = bytes.NewBuffer(msgBytes)

	return nil
}

// MakeDataResponseMsg ...
func MakeDataResponseMsg(tokenId TokenID) DataResponseMsg {
	h := makeHeader(MsgTypeDataResponse, 0)
	h.UUID = uuid.UUID(tokenId)
	m := DataResponseMsg{Hdr: h}
	m.Data = new(bytes.Buffer)
	return m
}

// ReadHeader ...
func ReadHeader(hdr *Header, c net.Conn) error {

	hdrBytes := make([]byte, 24) // header is 24 bytes in length
	numRead, err := c.Read(hdrBytes)
	if numRead < 8 || err != nil {
		return err
	}

	buf := bytes.NewBuffer(hdrBytes)
	err = binary.Read(buf, binary.BigEndian, hdr)
	if hdr.Magic != 101 || hdr.Version != 1 {
		err = errors.New("Incorrect Magic bytes or incorrect message version")
	}

	return err
}
