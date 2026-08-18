package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
)

type One struct {
	Name string
	Type string
	Data []byte
	Ones []*One
}

type encoder struct {
	io.Writer
}

func encode(ones []*One) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	encoder{w}.writeOnes(ones)
	w.Close()
	return buf.Bytes()
}

func (e encoder) writeOnes(ones []*One) {
	e.writeUvarint(uint64(len(ones)))
	for _, one := range ones {
		e.writeField([]byte(one.Name))
		e.writeField([]byte(one.Type))
		e.writeField(one.Data)
		e.writeOnes(one.Ones)
	}
}

func (e encoder) writeField(data []byte) {
	e.writeUvarint(uint64(len(data)))
	e.Write(data)
}

func (e encoder) writeUvarint(v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], v)
	e.Write(tmp[:n])
}
