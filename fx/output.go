package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
)

type Output struct {
	Name string
	Type string
	Zero []byte
	One  []*Output
}

type Payload struct {
	*Manifest
	*Entries
}

type Manifest struct {
	Name       string
	Dictionary map[string]string
}

type Entries []*Entry

type Entry struct {
	Name string
	Type string
	Data []byte
}

// Encode serializes the Output tree (Name, Type, Zero, and all descendants) into a single
// binary blob for the client to decode; the server only ever writes this, so fields are
// varint length-prefixed to keep the payload as small as possible. The result is
// gzip-compressed unless compress is explicitly false.
func (o *Output) Encode(compress ...bool) []byte {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	var gz *gzip.Writer
	if len(compress) == 0 || compress[0] {
		gz, _ = gzip.NewWriterLevel(&buf, gzip.BestCompression)
		w = gz
	}
	writeField(w, []byte(o.Name)) // root has no parent dictionary to resolve it from
	writeField(w, []byte(o.Type))
	o.encode(w)
	if gz != nil {
		gz.Close()
	}
	return buf.Bytes()
}

// encode writes either o's Zero content (leaf) or dictionaries of o.One's distinct
// Name and Type values, then each child — prefixed by a dictionary index only when
// that dictionary holds more than one entry.
func (o *Output) encode(w io.Writer) {
	if o.Type != "" {
		writeField(w, o.Zero)
		writeUvarint(w, 0)
		return
	}

	names, nameIndex := dictionary(o.One, func(c *Output) string { return c.Name })
	types, typeIndex := dictionary(o.One, func(c *Output) string { return c.Type })
	writeDictionary(w, names)
	writeDictionary(w, types)

	writeUvarint(w, uint64(len(o.One)))
	for _, one := range o.One {
		if len(names) > 1 {
			writeUvarint(w, uint64(nameIndex[one.Name]))
		}
		if len(types) > 1 {
			writeUvarint(w, uint64(typeIndex[one.Type]))
		}
		one.encode(w)
	}
}

// dictionary collects o.One's distinct values of key, in first-seen order.
func dictionary(one []*Output, key func(*Output) string) ([]string, map[string]int) {
	var values []string
	index := make(map[string]int, len(one))
	for _, o := range one {
		v := key(o)
		if _, ok := index[v]; !ok {
			index[v] = len(values)
			values = append(values, v)
		}
	}
	return values, index
}

func writeDictionary(w io.Writer, values []string) {
	writeUvarint(w, uint64(len(values)))
	for _, v := range values {
		writeField(w, []byte(v))
	}
}

// writeField writes data preceded by its length as a varint.
func writeField(w io.Writer, data []byte) {
	writeUvarint(w, uint64(len(data)))
	w.Write(data)
}

func writeUvarint(w io.Writer, v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], v)
	w.Write(tmp[:n])
}
