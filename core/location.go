package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
)

// Location is the struct that describes a point in space and time.
// Size: 32 bytes
type Location struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Alt       float32 `json:"alt"`
	Heading   float32 `json:"heading"`
	Timestamp int64 `json:"timestamp"`
}

// MakeLocation creates a new location in time and space
func MakeLocation(lat float64, lng float64, alt float32, heading float32, t int64) Location {
	loc := Location{lat, lng, alt, heading, t}
	return loc
}

func (l *Location) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}

// Serialize does ..
func Serialize(location *Location, buf *bytes.Buffer) {
	binary.Write(buf, binary.BigEndian, location.Lat)
	binary.Write(buf, binary.BigEndian, location.Lng)
	binary.Write(buf, binary.BigEndian, location.Alt)
	binary.Write(buf, binary.BigEndian, location.Heading)
	binary.Write(buf, binary.BigEndian, location.Timestamp)
}

// Deserialize ..
func Deserialize(location *Location, buf *bytes.Buffer) {
	binary.Read(buf, binary.BigEndian, location)
}

// ToJSONString returns the location as a JSON string
func ToJSONString(location *Location) string {
	s, err := json.Marshal(location)
	if err != nil {
		return err.Error()
	}
	return string(s)
}
