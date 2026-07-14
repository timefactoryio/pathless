package one

import (
	"encoding/binary"

	"github.com/timefactoryio/pathless/fx"
)

// Encode is a route's over-the-wire form: one Value framed as
//
//	[1B typeLen][type][payload]
//
// payload is a leaf's Data, or a bundle's children as a nested sequence. The
// payload's length is implicit — it's the rest of the buffer — since a route
// is exactly one value. The type travels in-band, so the transport needs no
// Content-Type and the client reconstructs the Value from the bytes alone.
// Encode is the sole codec — the client mirrors it in reverse; there is no
// server-side decode. It is deliberately separate from Value.Save's gob
// persistence (in fx), so changing this format never invalidates anything
// already saved.
func Encode(v *fx.Value) []byte {
	p := payload(v)
	out := make([]byte, 1+len(v.Type)+len(p))
	out[0] = byte(len(v.Type))
	n := 1 + copy(out[1:], v.Type)
	copy(out[n:], p)
	return out
}

// payload is a Value's body: a leaf's Data, or a bundle's Children packed as a
// sequence.
func payload(v *fx.Value) []byte {
	if len(v.Children) > 0 {
		return sequence(v.Children)
	}
	return v.Data
}

// typeTable collects the distinct types across values in first-seen order and
// encodes the wire type table — [1B typeCount][typeCount x ([1B typeLen]
// [type])] — the small dictionary each value's type is referenced by. It
// returns the encoded table, the type->id lookup, and whether every value
// shares one type (single), in which case callers omit the per-entry id byte.
func typeTable(values []*fx.Value) (table []byte, typeID map[string]byte, single bool) {
	typeID = make(map[string]byte, 4)
	var types []string
	for _, v := range values {
		if _, ok := typeID[v.Type]; !ok {
			typeID[v.Type] = byte(len(types))
			types = append(types, v.Type)
		}
	}
	size := 1
	for _, t := range types {
		size += 1 + len(t)
	}
	table = make([]byte, size)
	table[0] = byte(len(types))
	pos := 1
	for _, t := range types {
		table[pos] = byte(len(t))
		pos++
		pos += copy(table[pos:], t)
	}
	return table, typeID, len(types) == 1
}

// sequence packs many Values into one blob the client decodes into an array —
// a bundle's children. The type table (typeTable) is written once, up front;
// each entry then costs a 4-byte length ahead of its data, plus a 1-byte type
// id — unless every value shares one type (a frame pool, an image directory),
// in which case the id is implied by the table and omitted. A child that is
// itself a bundle carries its own children under the "application/x-bundle"
// type, decoded on the client through Value.children. Names are not encoded —
// order is the contract. Layout:
//
//	[type table]
//	[1B entryCount]
//	[1B typeID]?[4B len(data)][data]...  (repeated entryCount times; typeID
//	omitted when single)
func sequence(values []*fx.Value) []byte {
	table, typeID, single := typeTable(values)

	datas := make([][]byte, len(values))
	size := len(table) + 1 // table + entryCount
	for i, v := range values {
		datas[i] = payload(v)
		if !single {
			size++
		}
		size += 4 + len(datas[i])
	}

	out := make([]byte, size)
	pos := copy(out, table)
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
