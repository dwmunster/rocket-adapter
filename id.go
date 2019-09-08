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

func NewRandID() *randID {
	s := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &randID{s}
}
