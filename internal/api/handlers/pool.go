package handlers

import (
	"bytes"
	"encoding/json"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() any {
		// Pre-allocate 1KB which covers most OIDC response sizes
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

// EncodeJSONPooled encodes the data to JSON using a pooled buffer to reduce GC pressure.
// It returns the byte slice and a release function that MUST be called after use.
func EncodeJSONPooled(data any) ([]byte, func(), error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	enc := json.NewEncoder(buf)
	if err := enc.Encode(data); err != nil {
		bufferPool.Put(buf)
		return nil, nil, err
	}

	release := func() {
		bufferPool.Put(buf)
	}

	return buf.Bytes(), release, nil
}
