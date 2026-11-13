package fx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
	"slices"
)

func (f *fx) Encode(collections ...N) []byte {
	size := 0
	maxInt := int(^uint(0) >> 1)
	for _, entries := range collections {
		countSize := uvarintSize(uint64(len(entries)))
		if countSize > maxInt-size {
			panic("encode: encoded size overflows int")
		}
		size += countSize
		for _, entry := range entries {
			for _, length := range [...]int{len(entry.Name), len(entry.Type), len(entry.Data)} {
				lengthSize := uvarintSize(uint64(length))
				if length > maxInt-size-lengthSize {
					panic("encode: encoded size overflows int")
				}
				size += lengthSize + length
			}
		}
	}

	data := make([]byte, 0, size)
	for _, entries := range collections {
		data = binary.AppendUvarint(data, uint64(len(entries)))
		for _, entry := range entries {
			data = binary.AppendUvarint(data, uint64(len(entry.Name)))
			data = append(data, entry.Name...)
			data = binary.AppendUvarint(data, uint64(len(entry.Type)))
			data = append(data, entry.Type...)
			data = binary.AppendUvarint(data, uint64(len(entry.Data)))
			data = append(data, entry.Data...)
		}
	}
	return data
}

func uvarintSize(value uint64) int {
	return max(1, (bits.Len64(value)+6)/7)
}

func (f *fx) Decode(data []byte) (N, error) {
	ones := N{}
	for len(data) > 0 {
		count, n := binary.Uvarint(data)
		if n <= 0 {
			return nil, fmt.Errorf("decode: invalid entry count varint")
		}
		data = data[n:]
		if count > uint64(len(data)/3) {
			return nil, fmt.Errorf("decode: invalid entry count %d", count)
		}

		ones = slices.Grow(ones, int(count))
		for range count {
			var fields [3][]byte
			for i := range fields {
				length, n := binary.Uvarint(data)
				if n <= 0 {
					return nil, fmt.Errorf("decode: invalid field length varint")
				}
				data = data[n:]
				if length > uint64(len(data)) {
					return nil, fmt.Errorf("decode: field length %d exceeds remaining data", length)
				}
				fields[i] = data[:length:length]
				data = data[length:]
			}
			ones = append(ones, &One{
				Name: string(fields[0]),
				Type: string(fields[1]),
				Data: bytes.Clone(fields[2]),
			})
		}
	}
	return ones, nil
}
