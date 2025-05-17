// Package bytes defines byte transfer interfaces and utilities.
// It is meant to complement the io package, which is better suited for universal process border traversal, while this package focuses on efficient in-process, in-memory transfers.
package bytes

import (
	"github.com/blitz-frost/msg"
)

type Giver interface {
	Bytes() ([]byte, error)
}

type Messenger func() (Message, error)

type Message struct {
	Data  []byte
	Close msg.Closer
}

type Taker interface {
	BytesTake([]byte) error
}
