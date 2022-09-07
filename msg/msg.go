package msg

import (
	"github.com/blitz-frost/bytes"
	"github.com/blitz-frost/msg"
)

type ExchangeReader interface {
	Reader
	WriterGiver
}

type ExchangeReaderChainer = msg.ReaderChainer[ExchangeReader]

type ExchangeReaderTaker = msg.ReaderTaker[ExchangeReader]

type ExchangeWriter interface {
	Writer
	ReaderGiver
}

type ExchangeWriterGiver = msg.WriterGiver[ExchangeWriter]

type Reader = bytes.ViewCloser

type ReaderChainer = msg.ReaderChainer[Reader]

type ReaderGiver = msg.ReaderGiver[Reader]

type ReaderTaker = msg.ReaderTaker[Reader]

type Writer = bytes.UseCloser

type WriterGiver = msg.WriterGiver[Writer]
