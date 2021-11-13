package compact_float

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/cockroachdb/apd/v2"
)

func demonstrateEncodeDecodeDFloat() {
	originalValue := DFloat{
		Exponent:    100,
		Coefficient: 863994506,
	}

	buffer := &bytes.Buffer{}
	_, err := Encode(originalValue, buffer)
	if err != nil {
		// TODO: Handle error
	}
	fmt.Printf("%v encoded = %v\n", originalValue, buffer.Bytes())

	fmt.Printf("%v decoded ", buffer.Bytes())
	value, _, bytesDecoded, err := Decode(buffer)
	if err != nil {
		// TODO: Handle error
	}
	fmt.Printf("(%v bytes) = %v\n", bytesDecoded, value)

	// Prints:
	// 8.63994506e+108 encoded = [144 3 138 133 254 155 3]
	// [144 3 138 133 254 155 3] decoded (7 bytes) = 8.63994506e+108
}

func demonstrateEncodeDecodeBig() {
	// This value is too big to fit into a float64 or DFloat.
	originalValue, _, err := apd.NewFromString("-9.4452837206285466345998345667683453466347345e-5000")
	if err != nil {
		// TODO: Handle error
	}

	buffer := &bytes.Buffer{}
	if _, err = EncodeBig(originalValue, buffer); err != nil {
		// TODO: Handle error
	}
	fmt.Printf("%v encoded = %v\n", originalValue, buffer.Bytes())

	fmt.Printf("%v decoded ", buffer.Bytes())
	_, bigValue, bytesDecoded, err := Decode(buffer)
	if err != nil {
		// TODO: Handle error
	}
	fmt.Printf("(%v bytes) = %v\n", bytesDecoded, bigValue)

	// Prints:
	// -9.4452837206285466345998345667683453466347345E-5000 encoded = [207 157 1 209 142 162 230 131 138 191 193 187 225 243 223 252 238 172 229 254 225 143 226 67]
	// [207 157 1 209 142 162 230 131 138 191 193 187 225 243 223 252 238 172 229 254 225 143 226 67] decoded (24 bytes) = -9.4452837206285466345998345667683453466347345E-5000
}

func demonstrateEncodeDecodeFloat64() {
	originalValue := 0.1473445219134543
	significantDigits := 6
	buffer := &bytes.Buffer{}
	value, err := DFloatFromFloat64(originalValue, significantDigits)
	if err != nil {
		// TODO: Handle error (which might be RoundingError, in which case value is the rounded result)
	}
	_, err = Encode(value, buffer)
	if err != nil {
		// TODO: Handle error
	}
	fmt.Printf("%v encoded using %d significant digits = %v\n", originalValue, significantDigits, buffer.Bytes())

	fmt.Printf("%v decoded ", buffer.Bytes())
	decodedValue, _, bytesDecoded, err := Decode(buffer)
	if err != nil {
		// TODO: Handle error
	}
	fmt.Printf("(%v bytes) = %v\n", bytesDecoded, decodedValue.Float())

	// Prints:
	// 0.1473445219134543 encoded using 6 significant digits = [26 145 255 8]
	// [26 145 255 8] decoded (4 bytes) = 0.147345
}

func TestReadmeExamples(t *testing.T) {
	demonstrateEncodeDecodeDFloat()
	demonstrateEncodeDecodeBig()
	demonstrateEncodeDecodeFloat64()
}
