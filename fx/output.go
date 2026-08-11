package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
)

type Output struct {
	Name string
	Type string
	Zero []byte
	One  []*Output
}

// Encode serializes the Output tree (Name, Type, Zero, and all descendants) into a single
// binary blob for the client to decode; the server only ever writes this, so fields are
// varint length-prefixed to keep the payload as small as possible. If compress is true,
// the result is gzip-compressed.
func (o *Output) Encode(compress bool) []byte {
	var buf bytes.Buffer
	writeField(&buf, []byte(o.Type)) // root has no siblings to dictionary-reference against
	o.encode(&buf)
	data := buf.Bytes()
	if compress {
		return zip(data)
	}
	return data
}

func zip(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

// encode writes o's Name, then either its Zero content (leaf) or a dictionary of its
// children's distinct Type values (parent), then each child referencing that dictionary
// by index instead of repeating its Type string.
func (o *Output) encode(buf *bytes.Buffer) {
	writeField(buf, []byte(o.Name))

	var index map[string]int
	if len(o.One) == 0 {
		writeField(buf, o.Zero)
	} else {
		var types []string
		index = make(map[string]int, len(o.One))
		for _, child := range o.One {
			if _, ok := index[child.Type]; !ok {
				index[child.Type] = len(types)
				types = append(types, child.Type)
			}
		}
		writeUvarint(buf, uint64(len(types)))
		for _, t := range types {
			writeField(buf, []byte(t))
		}
	}

	writeUvarint(buf, uint64(len(o.One)))
	for _, child := range o.One {
		writeUvarint(buf, uint64(index[child.Type]))
		child.encode(buf)
	}
}

// writeField writes data preceded by its length as a varint.
func writeField(buf *bytes.Buffer, data []byte) {
	writeUvarint(buf, uint64(len(data)))
	buf.Write(data)
}

func writeUvarint(buf *bytes.Buffer, v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], v)
	buf.Write(tmp[:n])
}
