package fx

import (
	"net/http"

	"github.com/timefactoryio/pathless/fx/core"
	"github.com/timefactoryio/pathless/fx/templates"
	"github.com/timefactoryio/pathless/zero"
)

type Fx struct {
	*zero.Zero
	Forge
	*templates.Templates
	*core.Core
	Hello []byte
}

func NewFx(z *zero.Zero) *Fx {
	return &Fx{
		Forge:     NewForge().(*forge),
		Zero:      z,
		Templates: templates.Init(),
	}
}

// func (fx *Fx) BuildHello() {
// 	var chunks [][]byte
// 	chunks = append(chunks, []byte(fx.Keyboard))
// 	for _, frame := range fx.Frames() {
// 		if frame != nil {
// 			chunks = append(chunks, []byte(*frame))
// 		}
// 	}

// 	n := len(chunks)
// 	total := 4 + n*8
// 	for _, c := range chunks {
// 		total += len(c)
// 	}

// 	buf := make([]byte, total)
// 	binary.BigEndian.PutUint32(buf, uint32(n))
// 	off := uint32(4 + n*8)
// 	for i, c := range chunks {
// 		binary.BigEndian.PutUint32(buf[4+i*8:], off)
// 		binary.BigEndian.PutUint32(buf[8+i*8:], uint32(len(c)))
// 		off += uint32(len(c))
// 	}
// 	pos := 4 + n*8
// 	for _, c := range chunks {
// 		copy(buf[pos:], c)
// 		pos += len(c)
// 	}

// 	fx.Hello = fx.Compress(buf)
// }

func (fx *Fx) HandleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Encoding", "gzip")
	w.Write(fx.Hello)
}
