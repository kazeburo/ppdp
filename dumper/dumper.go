package dumper

import (
	"bytes"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

var flushDumperInterval time.Duration = 300
var mysqlPing = "01 00 00 00 0e"

// Dumper dumper struct
type Dumper struct {
	direction uint
	logger    *zap.Logger
	buf       *bytes.Buffer
	mu        sync.Mutex
	closer    func()
	dumpPing  bool
}

// New new handler
func New(direction uint, dumpPing bool, logger *zap.Logger) *Dumper {

	ticker := time.NewTicker(flushDumperInterval * time.Millisecond)
	ch := make(chan struct{})
	closer := func() {
		ticker.Stop()
		close(ch)
	}

	d := &Dumper{
		direction: direction,
		logger:    logger,
		buf:       new(bytes.Buffer),
		closer:    closer,
		dumpPing:  dumpPing,
	}

	go func() {
		for {
			select {
			case <-ch:
				return
			case _ = <-ticker.C:
				d.Flush()
			}
		}
	}()
	return d
}

// Write to dump
func (d *Dumper) Write(p []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.buf.Write(p)
	return len(p), nil
}

// Flush  flush buffer
func (d *Dumper) Flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.buf.Len() == 0 {
		return
	}
	hexdump := strings.Split(hex.Dump(d.buf.Bytes()), "\n")
	d.buf.Truncate(0)
	byteStrings := []string{}
	asciis := []string{}
	for _, hd := range hexdump {
		if hd == "" {
			continue
		}
		byteString := strings.TrimRight(strings.Replace(hd[10:58], "  ", " ", 1), " ")
		if byteString == mysqlPing && d.dumpPing == false {
			continue
		}
		byteStrings = append(byteStrings, byteString)
		asciis = append(asciis, hd[61:len(hd)-1])
	}
	if len(byteStrings) == 0 {
		return
	}
	d.logger.Info("dump",
		zap.Uint("direction", d.direction),
		zap.String("hex", strings.Join(byteStrings, " ")),
		zap.String("ascii", strings.Join(asciis, "")),
	)
}

// Stop : flush and stop flusher
func (d *Dumper) Stop() {
	d.Flush()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closer != nil {
		d.closer()
		d.closer = nil
	}
}
