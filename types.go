package bytes

// A Buffer is a basic Provisioner.
type Buffer []byte

func (x *Buffer) Provision(n int) ([]byte, error) {
	if len(*x) < n {
		(*x) = make([]byte, n)
	}
	return (*x)[:n], nil
}
