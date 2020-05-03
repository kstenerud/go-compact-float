package compact_float

import (
	"fmt"
	"testing"

	"github.com/cockroachdb/apd"
)

func demonstrateEncodeDecode() {
	originalValue, _, err := apd.NewFromString("-9.4452837206285466345998345667683453466347345e-5000")
	if err != nil {
		// TODO: Handle error
	}

	buffer := make([]byte, 50)
	bytesEncoded, ok := Encode(originalValue, buffer)
	if !ok {
		// TODO: There wasn't enough room to encode
	}
	buffer = buffer[:bytesEncoded]
	fmt.Printf("%v encoded = %v\n", originalValue, buffer)

	decodedValue, bytesDecoded, err := Decode(buffer)
	if err != nil {
		// TODO: Check if is compact_float.ErrorIncomplete or something else
	}
	fmt.Printf("%v decoded (%v bytes) = %v\n", buffer, bytesDecoded, decodedValue)

	// Prints:
	// -9.4452837206285466345998345667683453466347345E-5000 encoded = [207 157 1 209 142 162 230 131 138 191 193 187 225 243 223 252 238 172 229 254 225 143 226 67]
	// [207 157 1 209 142 162 230 131 138 191 193 187 225 243 223 252 238 172 229 254 225 143 226 67] decoded (24 bytes) = -9.4452837206285466345998345667683453466347345E-5000
}

func demonstrateEncodeDecodeFloat64() {
	originalValue := 0.1473445219134543
	significantDigits := 6
	buffer := make([]byte, 15)
	bytesEncoded, ok := EncodeFloat64(originalValue, significantDigits, buffer)
	if !ok {
		// TODO: There wasn't enough room to encode
	}
	buffer = buffer[:bytesEncoded]
	fmt.Printf("%v encoded using %d significant digits = %v\n", originalValue, significantDigits, buffer)

	decodedValue, bytesDecoded, err := Decode(buffer)
	if err != nil {
		// TODO: Check if is compact_float.ErrorIncomplete or something else
	}
	fmt.Printf("%v decoded (%v bytes) = %v\n", buffer, bytesDecoded, decodedValue)

	// Prints:
	// 0.1473445219134543 encoded using 6 significant digits = [26 145 255 8]
	// [26 145 255 8] decoded (4 bytes) = 0.147345
}

func TestReadmeExamples(t *testing.T) {
	demonstrateEncodeDecode()
	demonstrateEncodeDecodeFloat64()
}
