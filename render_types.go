package gospa

import (
	"encoding/binary"
	"math"
	"time"
)

// ssgEntry holds a cached HTML page and when it was generated.
type ssgEntry struct {
	html      []byte
	createdAt time.Time
}

// pprEntry holds a cached static shell for PPR pages.
type pprEntry struct {
	html      []byte
	createdAt time.Time
}

// encodeSsgEntry encodes an SSG entry into bytes.
func encodeSsgEntry(entry ssgEntry) []byte {
	buf := make([]byte, 8+len(entry.html))
	binary.LittleEndian.PutUint64(buf[0:8], uint64(entry.createdAt.UnixNano()))
	copy(buf[8:], entry.html)
	return buf
}

// decodeSsgEntry decodes bytes into an SSG entry.
func decodeSsgEntry(data []byte) (ssgEntry, bool) {
	if len(data) < 8 {
		return ssgEntry{}, false
	}
	createdAtNano := binary.LittleEndian.Uint64(data[0:8])
	if createdAtNano > uint64(math.MaxInt64) {
		return ssgEntry{}, false
	}
	return ssgEntry{
		html:      data[8:],
		createdAt: time.Unix(0, int64(createdAtNano)),
	}, true
}
