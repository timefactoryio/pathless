package one

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"net/http"

	"github.com/timefactoryio/pathless/fx"
)

// One is the HTTP layer. It takes zero's compiled assets and fx's processed
// Values, encodes them into the client wire format, gzips everything once at
// startup, and serves it from memory: the shell on :1000, the wire gateway
// on :1001.
type One struct {
	origin   string
	shell    []byte
	pathless *http.ServeMux
	circuit  *http.ServeMux
}

// NewOne registers the wire endpoints served from :1001. Every route — the
// root and each registered route alike — is exactly one *fx.Output, served by
// serve as Encode(v): the value's type travels in-band in the wire table, so
// the response needs no meaningful Content-Type. The root is just a bundle
// whose children are the universe payload and the frame and panel pools, so
// it needs no special handling. origin is the CORS allow-origin; shell and
// universe are zero's asset bytes; f supplies the frame/panel pools and route
// map.
func NewOne(origin string, shell, universe []byte, f *fx.Fx) *One {
	o := &One{
		origin:   origin,
		shell:    zip(shell),
		pathless: http.NewServeMux(),
		circuit:  http.NewServeMux(),
	}
	o.pathless.HandleFunc("/", o.handlePathless)

	o.serve("/", &fx.Output{Type: "application/x-bundle", Inputs: []*fx.Output{
		{Type: "text/html", Data: universe},
		f.Frames,
		f.Panels,
	}})
	for key, v := range f.Routes {
		o.serve("/"+key, v)
	}
	return o
}

func (o *One) handlePathless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" || r.URL.RawQuery != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(o.shell)
}

// serve registers one wire endpoint: Encode(v) — the value and its type,
// self-describing — gzipped once at startup and written from memory. The
// response carries no meaningful Content-Type; the bytes are the whole
// contract, and the client decodes them into a Output.
func (o *One) serve(path string, v *fx.Output) {
	data := zip(Encode(v))
	o.circuit.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	})
}

func (o *One) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", o.origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (o *One) Serve() {
	go http.ListenAndServe(":1001", o.cors(o.circuit))
	http.ListenAndServe(":1000", o.pathless)
}

// zip gzip-compresses data at maximum compression. Bundles are compressed
// once at build time and served directly with Content-Encoding: gzip.
func zip(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

// Encode is a route's over-the-wire form: one Output framed as
//
//	[1B typeLen][type][payload]
//
// payload is a leaf's Data, or a bundle's outputs as a nested sequence. The
// payload's length is implicit — it's the rest of the buffer — since a route
// is exactly one value. The type travels in-band, so the transport needs no
// Content-Type and the client reconstructs the Output from the bytes alone.
// Encode is the sole codec — the client mirrors it in reverse; there is no
// server-side decode. It is deliberately separate from Output.Save's gob
// persistence (in fx), so changing this format never invalidates anything
// already saved.
func Encode(v *fx.Output) []byte {
	p := payload(v)
	out := make([]byte, 1+len(v.Type)+len(p))
	out[0] = byte(len(v.Type))
	n := 1 + copy(out[1:], v.Type)
	copy(out[n:], p)
	return out
}

// payload is a Output's body: a leaf's Data, or a bundle's Inputs packed as a
// sequence.
func payload(v *fx.Output) []byte {
	if len(v.Inputs) > 0 {
		return sequence(v.Inputs)
	}
	return v.Data
}

// typeTable collects the distinct types across values in first-seen order and
// encodes the wire type table — [1B typeCount][typeCount x ([1B typeLen]
// [type])] — the small dictionary each value's type is referenced by. It
// returns the encoded table, the type->id lookup, and whether every value
// shares one type (single), in which case callers omit the per-entry id byte.
func typeTable(values []*fx.Output) (table []byte, typeID map[string]byte, single bool) {
	typeID = make(map[string]byte, 4)
	var types []string
	for _, v := range values {
		if _, ok := typeID[v.Type]; !ok {
			typeID[v.Type] = byte(len(types))
			types = append(types, v.Type)
		}
	}
	size := 1
	for _, t := range types {
		size += 1 + len(t)
	}
	table = make([]byte, size)
	table[0] = byte(len(types))
	pos := 1
	for _, t := range types {
		table[pos] = byte(len(t))
		pos++
		pos += copy(table[pos:], t)
	}
	return table, typeID, len(types) == 1
}

// sequence packs many Outputs into one blob the client decodes into an array —
// a bundle's outputs. The type table (typeTable) is written once, up front;
// each entry then costs a 4-byte length ahead of its data, plus a 1-byte type
// id — unless every value shares one type (a frame pool, an image directory),
// in which case the id is implied by the table and omitted. A child that is
// itself a bundle carries its own payload under the "application/x-bundle"
// type, decoded on the client through Output.children. Names are not encoded —
// order is the contract. Layout:
//
//	[type table]
//	[1B entryCount]
//	[1B typeID]?[4B len(data)][data]...  (repeated entryCount times; typeID
//	omitted when single)
func sequence(values []*fx.Output) []byte {
	table, typeID, single := typeTable(values)

	datas := make([][]byte, len(values))
	size := len(table) + 1 // table + entryCount
	for i, v := range values {
		datas[i] = payload(v)
		if !single {
			size++
		}
		size += 4 + len(datas[i])
	}

	out := make([]byte, size)
	pos := copy(out, table)
	out[pos] = byte(len(values))
	pos++
	for i, v := range values {
		if !single {
			out[pos] = typeID[v.Type]
			pos++
		}
		binary.BigEndian.PutUint32(out[pos:], uint32(len(datas[i])))
		pos += 4
		pos += copy(out[pos:], datas[i])
	}
	return out
}
