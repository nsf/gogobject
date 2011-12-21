package main

import "io"
import "bufio"

type CommentSkipper struct {
	r *bufio.Reader
}

func NewCommentSkipper(r io.Reader) *CommentSkipper {
	return &CommentSkipper{ bufio.NewReader(r) }
}

// advance to str and consume it or return error if it's not possible
func (cs *CommentSkipper) advanceTo(str string) error {
	if len(str) == 0 {
		panic("zero-length string is not acceptable")
	}

	cur := 0
	for {
		b, err := cs.r.ReadByte()
		if err != nil {
			return err
		}

		for {
			// check if we have match with cur
			if str[cur] != b {
				break
			}

			// got match, see if there are other
			// symbols to match with and continue if so
			if len(str)-1 > cur {
				cur++
				b, err = cs.r.ReadByte()
				if err != nil {
					return err
				}

				continue
			}

			return nil
		}
	}

	panic("unreachable")
	return nil
}

// advance to str, consume it, read and return the next byte if possible
func (cs *CommentSkipper) advanceToAndReadByte(str string) (byte, error) {
	err := cs.advanceTo(str)
	if err != nil {
		return 0, err
	}

	b, err := cs.r.ReadByte()
	if err != nil {
		return 0, err
	}

	return b, nil
}

func (cs *CommentSkipper) Read(data []byte) (int, error) {
	read := 0
	for {
		// check if we're done here
		if read == len(data) {
			return read, nil
		}

		b, err := cs.r.ReadByte()
		if err != nil {
			return read, err
		}

		// skip possible comments
		if b == '/' {
			b, err = cs.r.ReadByte()
			if err != nil {
				return read, err
			}

			switch b {
			case '/':
				// C++ comment
				err = cs.advanceTo("\n")
				if err != nil {
					return read, err
				}
				b = '\n'
			case '*':
				// C comment
				b, err = cs.advanceToAndReadByte("*/")
				if err != nil {
					return read, err
				}
			default:
				err = cs.r.UnreadByte()
				if err != nil {
					panic("shouldn't ever happen")
				}
				b = '/'
			}
		}

		data[read] = b
		read++
	}

	panic("unreachable")
	return 0, nil
}
