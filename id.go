package rocket

import (
	"math/rand"
	"strconv"
	"time"
)

type randID struct {
	*rand.Rand
}

func (r *randID) ID() (id string) {
	id = strconv.FormatUint(r.Uint64(), 32)
	return
}

// NewRandID generates a new randID and initializes it with the current time.
func NewRandID() *randID {
	s := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &randID{s}
}
