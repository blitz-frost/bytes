package bytes

import (
	"unsafe"

	"github.com/blitz-frost/errors"
	"github.com/blitz-frost/io"
	"github.com/blitz-frost/io/msg"
)

// A Reader attaches a simple io.Reader interface to a byte slice.
type Reader []byte

func (x *Reader) Read(b []byte) *io.Error {
	n := copy(b, *x)

	var err *io.Error
	if n < len(b) {
		err = &io.Error{n, io.EOF}
	}
	*x = (*x)[n:]

	return err
}

func (x *Reader) WriteTo(w msg.Writer) (int64, error) {
	if err := w(*x); err != nil {
		return int64(err.N), err.Err
	}
	return int64(len(*x)), nil
}

// A Writer attaches a simple io.Writer interface to a byte slice.
// It does not overflow its initial length.
type Writer []byte

func (x *Writer) ReadFrom(r msg.Reader) (int64, error) {
	var o int64

	var err error
	if ioErr := r(*x); ioErr != nil {
		o = int64(ioErr.N)
		err = ioErr
	} else {
		o = int64(len(*x))
	}

	*x = (*x)[o:]

	return o, err
}

func (x *Writer) Write(b []byte) *io.Error {
	n := copy(*x, b)

	var err *io.Error
	if n < len(b) {
		err = &io.Error{n, errors.Simple("buffer overflow")}
	}
	*x = (*x)[n:]

	return err
}

const messageReserve = 8 // always reserve space for a uint64 value at the beginning of a message

// MessageReader is a specialized io.Reader through a Messenger.
// It will repeatedly pull messages as needed to satisfy the current read.
//
// The underlying Message.Data must be at least 9 bytes long.
//
// The sending side must use a MessageWriter.
type MessageReader struct {
	Src Messenger

	msg Message
	b   []byte // data left in current message
	eof bool   // message end has been encountered
	err error  // an error has been encountered after a full Read
}

func (x *MessageReader) Close() error {
	if x.err != nil {
		// already in an abnormal state
		return x.err
	}

	if len(x.b) == 0 {
		if x.eof {
			// previous message was closed at EOF, nothing to do
			return nil
		}

		// not done, get next message
		if err := x.next(); err != nil {
			return err
		}
	}

	// close messages until EOF
	for {
		if err := x.msg.Close(); err != nil {
			return err
		}

		if x.eof {
			break
		}

		if err := x.next(); err != nil {
			return err
		}
	}

	return nil
}

func (x *MessageReader) Read(b []byte) *io.Error {
	if x.err != nil {
		return &io.Error{0, x.err}
	}
	if len(x.b) == 0 {
		// no current message
		if x.eof {
			return &io.Error{0, io.EOF}
		}
		if x.err = x.next(); x.err != nil {
			return &io.Error{0, x.err}
		}
	}

	var o int
	for len(b) > len(x.b) {
		// not enough available data

		// copy what we can
		n := x.copy(b)
		o += n
		x.err = x.msg.Close()
		if x.eof {
			return &io.Error{o, io.EOF}
		}
		if x.err != nil {
			return &io.Error{o, x.err}
		}

		// get next message
		if x.err = x.next(); x.err != nil {
			return &io.Error{o, x.err}
		}

		// update
		b = b[n:]
	}

	// have enough data
	o += x.copy(b)
	if len(x.b) == 0 {
		// current message exhausted; release underlying message ASAP
		x.err = x.msg.Close()
	}

	return nil
}

// WriteTo writes the rest of the message to w. Does not close the MessageReader.
func (x *MessageReader) WriteTo(w msg.Writer) (int64, error) {
	if x.err != nil {
		return 0, x.err
	}
	if len(x.b) == 0 {
		if x.eof {
			return 0, nil
		}

		if x.err = x.next(); x.err != nil {
			return 0, x.err
		}

		// it's possible to immediately encounter an empty EOF; avoid empty write
		if len(x.b) == 0 {
			x.err = x.msg.Close()
			return 0, nil
		}
	}

	var o int64
	for {
		if err := w(x.b); err != nil {
			// not an own error, so we don't keep it
			x.b = x.b[err.N:]
			o += int64(err.N)
			return o, err.Err
		}

		o += int64(len(x.b))
		x.b = x.b[len(x.b):]

		x.err = x.msg.Close()
		if x.eof {
			return o, nil
		}
		if x.err != nil {
			return o, x.err
		}

		if x.err = x.next(); x.err != nil {
			return o, x.err
		}

		// it's possible to immediately encounter an empty EOF; avoid empty write
		if len(x.b) == 0 {
			x.err = x.msg.Close()
			return o, nil
		}
	}
}

func (x *MessageReader) copy(b []byte) int {
	o := copy(b, x.b)
	x.b = x.b[o:]
	return o
}

func (x *MessageReader) next() error {
	var err error
	if x.msg, err = x.Src(); err != nil {
		return err
	}

	l := *(*uint64)(unsafe.Pointer(&x.msg.Data[0]))
	n := messageReserve + l
	x.b = x.msg.Data[messageReserve:n]

	if n < uint64(len(x.msg.Data)) {
		// short message indicates EOF
		x.eof = true
	}

	return err
}

// MessageWriter forms a specialized Writer through an Messenger.
// It will repeatedly fill up and send messages as needed, until closed.
//
// The underlying Message.Data must be at last 9 bytes long.
//
// The receiving side must use a MessageReader.
type MessageWriter struct {
	Dst Messenger

	msg Message
	b   []byte // slice of msg.Data that is still available for writing
}

func (x *MessageWriter) Close() error {
	if len(x.b) == 0 {
		// previous message was full, must send an empty one to mark end
		if err := x.next(); err != nil {
			return err
		}
	}

	return x.send()
}

// ReadFrom reads from r until EOF.
//
// Does not close the MessageWriter.
func (x *MessageWriter) ReadFrom(r msg.Reader) (int64, error) {
	if len(x.b) == 0 {
		if err := x.next(); err != nil {
			return 0, err
		}
	}

	var o int64
	for {
		err := r(x.b)
		if err != nil {
			o += int64(err.N)
			x.b = x.b[err.N:]
			if err.Err == io.EOF {
				// reading to EOF is the normal end
				break
			} else {
				return o, err.Err
			}
		}

		o += int64(len(x.b))
		x.b = x.b[len(x.b):]

		if err := x.send(); err != nil {
			return o, err
		}

		if err := x.next(); err != nil {
			return o, err
		}
	}

	var err error
	if len(x.b) == 0 {
		err = x.send()
	}

	return o, err
}

func (x *MessageWriter) Write(b []byte) *io.Error {
	if len(x.b) == 0 {
		// no current message
		if err := x.next(); err != nil {
			return &io.Error{0, err}
		}
	}

	var o int // total sent bytes
	for len(b) > len(x.b) {
		// data doesn't fit in current message

		// copy what we can
		n := x.copy(b)

		// send full message
		if err := x.send(); err != nil {
			return &io.Error{o, err}
		}
		o += n

		// load next message
		if err := x.next(); err != nil {
			return &io.Error{o, err}
		}

		// update
		b = b[n:]
	}

	// data fits in current message
	x.copy(b)

	if len(x.b) == 0 {
		// message full, send it
		// this makes it available to the read side ASAP
		if err := x.send(); err != nil {
			return &io.Error{o, err}
		}
	}

	return nil
}

// copy copies as much as it can from b into what is still available of the message data.
// Returns the number of bytes copied.
func (x *MessageWriter) copy(b []byte) int {
	o := copy(x.b, b)
	x.b = x.b[o:]
	return o
}

// lengthWrite writes the current message length into the reserved space.
func (x *MessageWriter) lengthWrite() {
	l := uint64(len(x.msg.Data) - messageReserve - len(x.b))
	b := unsafe.Slice((*byte)(unsafe.Pointer(&l)), messageReserve)
	copy(x.msg.Data, b)
}

func (x *MessageWriter) next() error {
	var err error
	if x.msg, err = x.Dst(); err != nil {
		return err
	}

	x.b = x.msg.Data[messageReserve:] // reserve message length space

	return nil
}

// send sends the current message.
func (x *MessageWriter) send() error {
	x.lengthWrite()

	return x.msg.Close()
}

func MessageInlet(src Messenger) msg.Inlet {
	return func() (msg.Input, error) {
		r := &MessageReader{Src: src}

		return msg.Input{
			Read:    r.Read,
			Close:   r.Close,
			WriteTo: r.WriteTo,
		}, nil
	}
}

func MessageOutlet(dst Messenger) msg.Outlet {
	return func() (msg.Output, error) {
		w := &MessageWriter{Dst: dst}

		return msg.Output{
			Write:    w.Write,
			Close:    w.Close,
			ReadFrom: w.ReadFrom,
		}, nil
	}
}
