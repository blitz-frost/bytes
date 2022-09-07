package msg

type Void struct{}

func (x Void) Close() error {
	return nil
}

func (x Void) ReaderTake(r Reader) error {
	return r.Close()
}

func (x Void) Use(b []byte) (int, error) {
	return len(b), nil
}
