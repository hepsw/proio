package eicio

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
)

type Event struct {
	Header  *EventHeader
	payload []byte
}

func NewEvent() *Event {
	return &Event{Header: &EventHeader{}}
}

func (evt *Event) String() string {
	buffer := &bytes.Buffer{}

	fmt.Fprint(buffer, "Event header...\n", proto.MarshalTextString(evt.Header), "\n")
	for _, collHdr := range evt.Header.Collection {
		coll := evt.GetCollection(collHdr.Name)
		fmt.Fprint(buffer, collHdr.Name, " collection\n", proto.MarshalTextString(coll), "\n")
	}

	return string(buffer.Bytes())
}

type Message interface {
	proto.Message

	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

func (evt *Event) AddCollection(collection Message, name string) {
	collHdr := &EventHeader_CollectionHeader{}

	switch collection.(type) {
	case *MCParticleCollection:
		collHdr.Type = EventHeader_CollectionHeader_MCParticle
	case *SimTrackerHitCollection:
		collHdr.Type = EventHeader_CollectionHeader_SimTrackerHit
	}

	collHdr.Name = name

	collBuf, err := collection.Marshal()
	if err != nil {
		return
	}
	collHdr.PayloadSize = uint32(len(collBuf))

	if evt.Header == nil {
		evt.Header = &EventHeader{}
	}
	evt.Header.Collection = append(evt.Header.Collection, collHdr)
	evt.payload = append(evt.payload, collBuf...)
}

func (evt *Event) GetCollection(name string) Message {
	offset := uint32(0)
	size := uint32(0)
	collType := EventHeader_CollectionHeader_NONE
	for _, coll := range evt.Header.Collection {
		if coll.Name == name {
			collType = coll.Type
			size = coll.PayloadSize
			break
		}
		offset += coll.PayloadSize
	}
	if collType == EventHeader_CollectionHeader_NONE {
		return nil
	}

	var coll Message
	switch collType {
	case EventHeader_CollectionHeader_MCParticle:
		coll = &MCParticleCollection{}
	case EventHeader_CollectionHeader_SimTrackerHit:
		coll = &SimTrackerHitCollection{}
	}

	if err := coll.Unmarshal(evt.payload[offset : offset+size]); err != nil {
		panic(err)
	}

	return coll
}

func (evt *Event) getPayload() []byte {
	return evt.payload
}

func (evt *Event) setPayload(payload []byte) {
	evt.payload = payload
}
