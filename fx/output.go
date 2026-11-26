package fx

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"slices"
)

type One struct {
	Meta map[string]string
	Data []byte
}

type N []*One

// Encode serializes the payload, optionally gzip-compressing it.
func (entries N) Encode(compress ...bool) []byte {
	var data []byte
	var keys []string
	for _, entry := range entries {
		data = binary.AppendUvarint(data, uint64(len(entry.Meta)))
		keys = keys[:0]
		for key := range entry.Meta {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			for _, text := range []string{key, entry.Meta[key]} {
				data = binary.AppendUvarint(data, uint64(len(text)))
				data = append(data, text...)
			}
		}
		data = binary.AppendUvarint(data, uint64(len(entry.Data)))
		data = append(data, entry.Data...)
	}
	if len(compress) > 0 && compress[0] {
		return zip(data)
	}
	return data
}

func zip(data []byte) []byte {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	if _, err := writer.Write(data); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// Decode reconstructs a payload from its binary encoding.
func Decode(data []byte) (N, error) {
	field := func() ([]byte, error) {
		length, n := binary.Uvarint(data)
		if n <= 0 {
			return nil, fmt.Errorf("decode: invalid field length varint")
		}
		data = data[n:]
		if length > uint64(len(data)) {
			return nil, fmt.Errorf("decode: field length %d exceeds remaining data", length)
		}
		value := data[:length:length]
		data = data[length:]
		return value, nil
	}

	entries := N{}
	for len(data) > 0 {
		count, n := binary.Uvarint(data)
		if n <= 0 {
			return nil, fmt.Errorf("decode: invalid metadata count varint")
		}
		data = data[n:]
		if count > uint64(len(data)/2) {
			return nil, fmt.Errorf("decode: metadata count %d exceeds remaining data", count)
		}

		meta := make(map[string]string, int(count))
		for range count {
			key, err := field()
			if err != nil {
				return nil, err
			}
			value, err := field()
			if err != nil {
				return nil, err
			}
			name := string(key)
			if _, exists := meta[name]; exists {
				return nil, fmt.Errorf("decode: duplicate metadata key %q", name)
			}
			meta[name] = string(value)
		}
		value, err := field()
		if err != nil {
			return nil, err
		}
		entries = append(entries, &One{
			Meta: meta,
			Data: bytes.Clone(value),
		})
	}
	return entries, nil
}
