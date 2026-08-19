package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
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

func decode(data []byte) ([]*One, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	wire, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	d := decoder{bytes.NewReader(wire)}
	ones, err := d.readOnes(0)
	if err != nil {
		return nil, err
	}
	if d.Len() != 0 {
		return nil, fmt.Errorf("decode: %d trailing bytes", d.Len())
	}
	return ones, nil
}

type decoder struct {
	*bytes.Reader
}

func (d decoder) readOnes(depth int) ([]*One, error) {
	if depth > 1000 {
		return nil, fmt.Errorf("decode: nesting too deep")
	}
	count, err := binary.ReadUvarint(d)
	if err != nil {
		return nil, err
	}
	if count > uint64(d.Len()/4) {
		return nil, fmt.Errorf("decode: invalid entry count %d", count)
	}

	ones := make([]*One, 0, count)
	for range count {
		name, err := d.readField()
		if err != nil {
			return nil, err
		}
		typ, err := d.readField()
		if err != nil {
			return nil, err
		}
		data, err := d.readField()
		if err != nil {
			return nil, err
		}
		children, err := d.readOnes(depth + 1)
		if err != nil {
			return nil, err
		}
		ones = append(ones, &One{
			Name: string(name),
			Type: string(typ),
			Data: data,
			Ones: children,
		})
	}
	return ones, nil
}

func (d decoder) readField() ([]byte, error) {
	length, err := binary.ReadUvarint(d)
	if err != nil {
		return nil, err
	}
	if length > uint64(d.Len()) {
		return nil, fmt.Errorf("decode: field length %d exceeds remaining data", length)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(d, data); err != nil {
		return nil, err
	}
	return data, nil
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
