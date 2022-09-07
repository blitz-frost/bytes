package msg

import (
	"github.com/blitz-frost/bytes"
	"github.com/blitz-frost/io/msg"
)

type exchangeReader struct {
	reader
	writerGiver
}

func makeExchangeReader(er msg.ExchangeReader, p bytes.Provisioner) exchangeReader {
	return exchangeReader{
		reader: reader{
			Reader: er,
			p:      p,
		},
		writerGiver: writerGiver{
			wg: er,
		},
	}
}

type exchangeReaderChainer struct {
	ert ExchangeReaderTaker
	p   bytes.Provisioner
}

func (x *exchangeReaderChainer) ReaderChain(ert ExchangeReaderTaker) error {
	x.ert = ert
	return nil
}

func (x *exchangeReaderChainer) ReaderTake(er msg.ExchangeReader) error {
	return x.ert.ReaderTake(makeExchangeReader(er, x.p))
}

type exchangeWriter struct {
	writer
	readerGiver
}

func makeExchangeWriter(ew msg.ExchangeWriter, p bytes.Provisioner) exchangeWriter {
	return exchangeWriter{
		writer: writer{ew},
		readerGiver: readerGiver{
			rg: ew,
			p:  p,
		},
	}
}

type exchangeWriterGiver struct {
	ewg msg.ExchangeWriterGiver
	p   bytes.Provisioner
}

func (x exchangeWriterGiver) Writer() (ExchangeWriter, error) {
	w, err := x.ewg.Writer()
	if err != nil {
		return nil, err
	}
	return makeExchangeWriter(w, x.p), nil
}

type ioReader struct {
	Reader
}

func (x ioReader) Read(b []byte) (int, error) {
	s, err := x.View(len(b))
	copy(b, s)
	return len(s), err
}

type ioWriter struct {
	Writer
}

func (x ioWriter) Write(b []byte) (int, error) {
	return x.Use(b)
}

type reader struct {
	msg.Reader
	p bytes.Provisioner
}

func (x reader) View(n int) ([]byte, error) {
	b, err := x.p.Provision(n)
	if err != nil {
		return nil, err
	}
	m, err := x.Reader.Read(b)
	return b[:m], err
}

type readerChainer struct {
	rt ReaderTaker
	p  bytes.Provisioner
}

func (x *readerChainer) ReaderChain(rt ReaderTaker) error {
	x.rt = rt
	return nil
}

func (x *readerChainer) ReaderTake(r msg.Reader) error {
	return x.rt.ReaderTake(reader{
		Reader: r,
		p:      x.p,
	})
}

type readerGiver struct {
	rg msg.ReaderGiver
	p  bytes.Provisioner
}

func (x readerGiver) Reader() (Reader, error) {
	r, err := x.rg.Reader()
	if err != nil {
		return nil, err
	}
	return reader{
		Reader: r,
		p:      x.p,
	}, nil
}

type writer struct {
	msg.Writer
}

func (x writer) Use(b []byte) (int, error) {
	return x.Writer.Write(b)
}

type writerGiver struct {
	wg msg.WriterGiver
}

func (x writerGiver) Writer() (Writer, error) {
	w, err := x.wg.Writer()
	if err != nil {
		return nil, err
	}
	return writer{w}, nil
}

func ExchangeReaderOf(er msg.ExchangeReader, p bytes.Provisioner) ExchangeReader {
	return makeExchangeReader(er, p)
}

// The returned value is also a [msg.ReaderTaker].
func ExchangeReaderChainerOf(erc msg.ExchangeReaderChainer, p bytes.Provisioner) (ExchangeReaderChainer, error) {
	x := &exchangeReaderChainer{
		p: p,
	}
	return x, erc.ReaderChain(x)
}

func ExchangeWriterOf(ew msg.ExchangeWriter, p bytes.Provisioner) ExchangeWriter {
	return makeExchangeWriter(ew, p)
}

func ExchangeWriterGiverOf(ewg msg.ExchangeWriterGiver, p bytes.Provisioner) ExchangeWriterGiver {
	return exchangeWriterGiver{
		ewg: ewg,
		p:   p,
	}
}

func IoReaderOf(r Reader) msg.Reader {
	return ioReader{r}
}

func IoWriterOf(w Writer) msg.Writer {
	return ioWriter{w}
}

func ReaderOf(r msg.Reader, p bytes.Provisioner) Reader {
	return reader{
		Reader: r,
		p:      p,
	}
}

// The returned value is also a [msg/ReaderTaker].
func ReaderChainerOf(rc msg.ReaderChainer, p bytes.Provisioner) (ReaderChainer, error) {
	x := &readerChainer{
		p: p,
	}
	return x, rc.ReaderChain(x)
}

func ReaderGiverOf(rg msg.ReaderGiver, p bytes.Provisioner) ReaderGiver {
	return readerGiver{
		rg: rg,
		p:  p,
	}
}

func WriterOf(w msg.Writer) Writer {
	return writer{w}
}

func WriterGiverOf(wg msg.WriterGiver) WriterGiver {
	return writerGiver{wg}
}
