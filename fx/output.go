package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
)

type Output [][]*output

type output struct {
	Name string
	Type string
	Data []byte
}

func (o Output) Encode(compress ...bool) []byte {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	var gz *gzip.Writer
	if len(compress) == 0 || compress[0] {
		gz, _ = gzip.NewWriterLevel(&buf, gzip.BestCompression)
		w = gz
	}
	writeUvarint(w, uint64(len(o)))
	for _, entries := range o {
		encodeEntries(w, entries)
	}
	if gz != nil {
		gz.Close()
	}
	return buf.Bytes()
}

func encodeEntries(w io.Writer, entries []*output) {
	writeUvarint(w, uint64(len(entries)))
	for _, entry := range entries {
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
