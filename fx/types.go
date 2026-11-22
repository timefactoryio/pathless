package fx

import (
	"encoding/binary"
	"math/bits"
)

type One struct {
	Name string
	Type string
	Data []byte
}

type OneV2 struct {
	Map  map[string]any
	Data []byte
}

type N []n
type n []*One

func (entries N) Encode() []byte {
	var data []byte
	data = binary.AppendUvarint(data, uint64(len(entries)))
	for _, entry := range entries {
		data = binary.AppendUvarint(data, uint64(len(entry.Name)))
		data = append(data, entry.Name...)
		data = binary.AppendUvarint(data, uint64(len(entry.Type)))
		data = append(data, entry.Type...)
		data = binary.AppendUvarint(data, uint64(len(entry.Data)))
		data = append(data, entry.Data...)
	}
	return data
}

// EncodeData encodes the entry count followed by length-prefixed data fields.
func (entries N) EncodeData() []byte {
	size := uvarintSize(uint64(len(entries)))
	maxInt := int(^uint(0) >> 1)
	for _, entry := range entries {
		length := len(entry.Data)
		lengthSize := uvarintSize(uint64(length))
		if length > maxInt-size-lengthSize {
			panic("encode data: encoded size overflows int")
		}
		size += lengthSize + length
	}

	data := make([]byte, 0, size)
	data = binary.AppendUvarint(data, uint64(len(entries)))
	for _, entry := range entries {
		data = binary.AppendUvarint(data, uint64(len(entry.Data)))
		data = append(data, entry.Data...)
	}
	return data
}

func uvarintSize(value uint64) int {
	return max(1, (bits.Len64(value)+6)/7)
}
