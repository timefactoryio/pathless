package one

import (
	"encoding/binary"
	"encoding/gob"
	"os"
	"path/filepath"

	"github.com/timefactoryio/pathless/fx"
)

// Encode writes the wire format the client decodes: distinct types are
// written once, up front, as a small dictionary; each entry then costs just
// a 1-byte type id and a 4-byte length ahead of its data — no shared table
// beyond that, no repeated type strings. A directory Value carries no bytes
// of its own; its wire payload is Encode(Children), so the client decodes it
// the same way it decodes the top-level response. This is purely the
// over-the-network format — Save uses gob instead, so changing this format
// never requires re-saving anything already persisted. Layout:
//
//	[1B typeCount][typeCount x ([1B typeLen][type])]
//	[1B typeID][4B len(data)][data]...  (repeated, one per value)
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

	size := 1
	for _, t := range types {
		size += 1 + len(t)
	}
	for _, d := range datas {
		size += 1 + 4 + len(d)
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
	for i, v := range values {
		out[pos] = typeID[v.Type]
		pos++
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
// caller).
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

	var values []*fx.Value
	for pos < len(buf) {
		typ := types[buf[pos]]
		pos++
		n := int(binary.BigEndian.Uint32(buf[pos:]))
		pos += 4
		values = append(values, &fx.Value{Type: typ, Data: buf[pos : pos+n]})
		pos += n
	}
	return values
}

// Save persists v (the full Value tree — Name, Type, Data, Children — not
// the wire format) to s3/key using gob, so changes to Encode/Decode never
// require re-saving anything already in S3. Encode is only ever applied
// fresh, at serve time.
func Save(key string, v *fx.Value) error {
	if err := os.MkdirAll("s3", 0755); err != nil {
		return err
	}
	data, err := os.Create(filepath.Join("s3", key))
	if err != nil {
		return err
	}
	defer data.Close()
	return gob.NewEncoder(data).Encode(v)
}
