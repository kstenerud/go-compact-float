package compact_float

import (
	"fmt"
	"testing"
)

func demonstrateEncodeDecode() {
	originalValue := 0.1473445219134543
	significantDigits := 6
	buffer := make([]byte, 15)
	bytesEncoded, ok := Encode(originalValue, significantDigits, buffer)
	if !ok {
		// TODO: There wasn't enough room to encode
	}
	buffer = buffer[:bytesEncoded]
	fmt.Printf("%v encoded using %d significant digits = %v\n", originalValue, significantDigits, buffer)

	decodedValue, bytesDecoded, ok := Decode(buffer)
	if !ok {
		// TODO: The buffer has been truncated
	}
	fmt.Printf("%v decoded %v bytes = %v\n", buffer, bytesDecoded, decodedValue)

	// Prints:
	// 0.1473445219134543 encoded using 6 significant digits = [26 136 255 17]
	// [26 136 255 17] decoded 4 bytes = 0.147345
}

func TestReadmeExamples(t *testing.T) {
	demonstrateEncodeDecode()
}
