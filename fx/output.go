package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
)

type Value interface {
	encode(io.Writer)
}

type Payload []Value

type Data []byte

type Entries []*Entry

type Entry struct {
	Name string
	Type string
	Data Data
}

const (
	dataValue byte = iota
	entriesValue
)

func (p Payload) Encode(compress ...bool) []byte {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	var gz *gzip.Writer
	if len(compress) == 0 || compress[0] {
		gz, _ = gzip.NewWriterLevel(&buf, gzip.BestCompression)
		w = gz
	}
	writeUvarint(w, uint64(len(p)))
	for _, value := range p {
		value.encode(w)
	}
	if gz != nil {
		gz.Close()
	}
	return buf.Bytes()
}

func (d Data) encode(w io.Writer) {
	w.Write([]byte{dataValue})
	writeField(w, d)
}

func (e Entries) encode(w io.Writer) {
	w.Write([]byte{entriesValue})
	writeUvarint(w, uint64(len(e)))
	for _, entry := range e {
		writeField(w, []byte(entry.Name))
		writeField(w, []byte(entry.Type))
		writeField(w, entry.Data)
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
