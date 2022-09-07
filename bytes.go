// Package bytes defines byte transfer interfaces and utilities.
// It is meant to complement the io package, which is better suited for process border traversal, while this package focuses on efficient in-process transfers.
package bytes

import (
	"io"
)

var EOF = io.EOF

type Closer = io.Closer

type Giver interface {
	Bytes() ([]byte, error)
}

// A Provisioner provides usable memory.
type Provisioner interface {
	Provision(int) ([]byte, error)
}

type Taker interface {
	BytesTake([]byte) error
}

// A User makes use of provided binary data.
// The returned int specifies how much of the input slice has been used. A value lower than the input slice length is valid, and should not automatically be considered an error.
// No default assumptions should be made about ownership or mutability of the binary data, but implementations should not tamper with bytes beyond the returned index.
type User interface {
	Use([]byte) (int, error)
}

type UseCloser interface {
	User
	Closer
}

// Viewer is similar to a io.Reader, but allows direct memory transfer.
// No default assumptions should be made about ownership or mutability of the returned value.
type Viewer interface {
	View(int) ([]byte, error)
}

type ViewCloser interface {
	Viewer
	Closer
}
