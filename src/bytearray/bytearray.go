package bytearray

import (
	"bytes"
	"errors"
	"fmt"
)

type ByteArray []byte

func charToValue(b byte) (byte, error) {
	if b >= '0' && b <= '9' {
		return b - '0', nil
	} else if b >= 'a' && b <= 'f' {
		return b - 'a' + 10, nil
	} else if b >= 'A' && b <= 'F' {
		return b - 'A' + 10, nil
	}
	return 0, fmt.Errorf("%q is not a valid hex character", b)
}

func (n *ByteArray) Set(value string) error {
	l := len(value)
	if l%2 != 0 {
		return errors.New("must specify an even number of (hex) characters to specify a byte sequence")
	}
	*n = make([]byte, 0, l/2)
	for i := 0; i < l; i += 2 {
		var err error
		var v1, v2 byte
		b1 := value[i]
		b2 := value[i+1]
		if v1, err = charToValue(b1); err != nil {
			return err
		}
		if v2, err = charToValue(b2); err != nil {
			return err
		}

		*n = append(*n, v1*16+v2)
	}

	return nil
}

func (n *ByteArray) String() string {
	var buf bytes.Buffer
	for _, b := range *n {
		buf.WriteString(fmt.Sprintf("%02X", b))
	}
	return buf.String()
}
