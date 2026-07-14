package one

import (
	"encoding/binary"

	"github.com/timefactoryio/pathless/fx"
)

// Encode writes the wire format the client decodes: distinct types are
// written once, up front, as a small dictionary; each entry then costs a
// 4-byte length ahead of its data, plus a 1-byte type id — unless every
// value shares one type, in which case the id is implied by the type table
// and omitted entirely. A directory Value carries no bytes of its own; its
// wire payload is Encode(Children), tagged with the "application/x-bundle"
// type so the client's decode recognizes it and recurses automatically,
// rather than handing back an entry the caller must decode a second time.
// This is purely the over-the-network format, deliberately separate from
// Value.Save's gob persistence (in fx), so changing this format never
// invalidates anything already persisted. Layout:
//
//	[1B typeCount][typeCount x ([1B typeLen][type])]
//	[1B entryCount]
//	[1B typeID]?[4B len(data)][data]...  (repeated entryCount times; typeID
//	omitted when typeCount == 1)
//
// Names are not encoded — order is the contract.
func Encode(values ...*fx.Value) []byte {
	datas := make([][]byte, len(values))
	typeID := make(map[string]byte, 4)
	var types []string
	for i, v := range values {
		datas[i] = payload(v)
		if _, ok := typeID[v.Type]; !ok {
			typeID[v.Type] = byte(len(types))
			types = append(types, v.Type)
		}
	}
	single := len(types) == 1

	size := 2 // typeCount + entryCount
	for _, t := range types {
		size += 1 + len(t)
	}
	for _, d := range datas {
		if !single {
			size++
		}
		size += 4 + len(d)
	}

	out := make([]byte, size)
	pos := 0
	out[pos] = byte(len(types))
	pos++
	for _, t := range types {
		out[pos] = byte(len(t))
		pos++
		pos += copy(out[pos:], t)
	}
	out[pos] = byte(len(values))
	pos++
	for i, v := range values {
		if !single {
			out[pos] = typeID[v.Type]
			pos++
		}
		binary.BigEndian.PutUint32(out[pos:], uint32(len(datas[i])))
		pos += 4
		pos += copy(out[pos:], datas[i])
	}
	return out
}

// payload is a Value's wire bytes: a leaf's Data, or a directory's Children
// encoded as their own nested blob.
func payload(v *fx.Value) []byte {
	if len(v.Children) > 0 {
		return Encode(v.Children...)
	}
	return v.Data
}

// Decode is the inverse of Encode for one level (leaves come back with Data;
// a directory entry's Data is itself an encoded blob, decoded again by the
// caller). The entry count in the header lets the result be allocated at
// its exact size up front.
func Decode(buf []byte) []*fx.Value {
	if len(buf) == 0 {
		return nil
	}
	pos := 0
	types := make([]string, buf[pos])
	pos++
	for i := range types {
		tl := int(buf[pos])
		pos++
		types[i] = string(buf[pos : pos+tl])
		pos += tl
	}

	n := int(buf[pos])
	pos++
	single := len(types) == 1

	values := make([]*fx.Value, n)
	for i := 0; i < n; i++ {
		typ := types[0]
		if !single {
			typ = types[buf[pos]]
			pos++
		}
		dl := int(binary.BigEndian.Uint32(buf[pos:]))
		pos += 4
		values[i] = &fx.Value{Type: typ, Data: buf[pos : pos+dl]}
		pos += dl
	}
	return values
}
