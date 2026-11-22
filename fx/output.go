package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"slices"
)

func (f *fx) Zip(collections ...N) []byte {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	if _, err := writer.Write(f.Encode(collections...)); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
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
