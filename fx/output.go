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
	for _, ones := range collections {
		prefix := uvarintSize(uint64(len(ones)))
		if prefix > maxInt-size {
			panic("encode: encoded size overflows int")
		}
		size += prefix
		for _, one := range ones {
			for _, length := range [...]int{len(one.Name), len(one.Type), len(one.Data)} {
				prefix := uvarintSize(uint64(length))
				if length > maxInt-size-prefix {
					panic("encode: encoded size overflows int")
				}
				size += prefix + length
			}
		}
	}

	data := make([]byte, 0, size)
	for _, ones := range collections {
		data = binary.AppendUvarint(data, uint64(len(ones)))
		for _, one := range ones {
			data = binary.AppendUvarint(data, uint64(len(one.Name)))
			data = append(data, one.Name...)
			data = binary.AppendUvarint(data, uint64(len(one.Type)))
			data = append(data, one.Type...)
			data = binary.AppendUvarint(data, uint64(len(one.Data)))
			data = append(data, one.Data...)
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
