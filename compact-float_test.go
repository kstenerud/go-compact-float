package compact_float

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"math"
	"testing"
)

func assertEncodeDecode(t *testing.T, sourceValue float64, significantDigits int, expectedEncoded []byte) float64 {
	actualEncoded := make([]byte, 15)
	bytesEncoded, ok := Encode(sourceValue, significantDigits, actualEncoded)
	if !ok {
		t.Errorf("Value %v: could not encode into %v bytes", sourceValue, len(actualEncoded))
	}
	if bytesEncoded != len(expectedEncoded) {
		t.Errorf("Value %v: Expected to encode %v bytes but encoded %v", sourceValue, len(expectedEncoded), bytesEncoded)
	}
	actualEncoded = actualEncoded[:bytesEncoded]
	if !bytes.Equal(expectedEncoded, actualEncoded) {
		t.Errorf("Value %v: Expected encoded:\n%v but got:\n%v", sourceValue, hex.Dump(expectedEncoded), hex.Dump(actualEncoded))
	}
	actualValue, bytesDecoded, ok := Decode(expectedEncoded)
	if !ok {
		t.Errorf("Failed to decode from buffer %v", expectedEncoded)
	}
	if bytesDecoded != len(actualEncoded) {
		t.Errorf("Value %v: Expected to decode %v bytes but decoded %v", sourceValue, len(expectedEncoded), bytesDecoded)
	}
	return actualValue
}

func assertCompactFloat(t *testing.T, sourceValue float64, significantDigits int, expectedValue float64, expectedEncoded []byte) {
	actualValue := assertEncodeDecode(t, sourceValue, significantDigits, expectedEncoded)
	actualAsString := fmt.Sprintf("%v", actualValue)
	expectedAsString := fmt.Sprintf("%v", expectedValue)
	if actualAsString != expectedAsString {
		t.Errorf("Value %v: Expected decoded value %v but got %v", sourceValue, expectedValue, actualValue)
	}
}

func assertNan(t *testing.T, sourceValue float64, significantDigits int, expectedEncoded []byte) {
	actualValue := assertEncodeDecode(t, sourceValue, significantDigits, expectedEncoded)
	if !math.IsNaN(actualValue) {
		t.Errorf("Value %v: Expected NaN but got %v", sourceValue, actualValue)
	}
}

func TestZero(t *testing.T) {
	assertCompactFloat(t, 0, 0, 0, []byte{0x02})
}

func TestNegZero(t *testing.T) {
	negZero := math.Copysign(0, -1)
	assertCompactFloat(t, negZero, 0, negZero, []byte{0x03})
}

func TestInfinity(t *testing.T) {
	assertCompactFloat(t, math.Inf(1), 0, math.Inf(1), []byte{0x80, 0x02})
}

func TestNegativeInfinity(t *testing.T) {
	assertCompactFloat(t, math.Inf(-1), 0, math.Inf(-1), []byte{0x80, 0x03})
}

func TestQuietNan(t *testing.T) {
	assertNan(t, math.NaN(), 0, []byte{0x80, 0x00})
}

func Test1_0(t *testing.T) {
	assertCompactFloat(t, 1.0, 0, 1.0, []byte{0x00, 0x01})
}

func Test1_5(t *testing.T) {
	assertCompactFloat(t, 1.5, 0, 1.5, []byte{0x06, 0x0f})
}

func Test1_2(t *testing.T) {
	assertCompactFloat(t, 1.2, 0, 1.2, []byte{0x06, 0x0c})
}

func Test1_25(t *testing.T) {
	assertCompactFloat(t, 1.25, 0, 1.25, []byte{0x0a, 0x7d})
}

func Test8_8419305(t *testing.T) {
	assertCompactFloat(t, 8.8419305, 0, 8.8419305, []byte{0x1e, 0xaa, 0x94, 0xd7, 0x69})
}

func Test1999999999999999(t *testing.T) {
	assertCompactFloat(t, 1999999999999999.0, 0, 1999999999999999.0, []byte{0x00, 0x83, 0xc6, 0xdf, 0xd4, 0xcc, 0xb3, 0xff, 0x7f})
}

func Test9_3942e100(t *testing.T) {
	assertCompactFloat(t, 9.3942e100, 0, 9.3942e100, []byte{0x83, 0x00, 0x85, 0xdd, 0x76})
}

func Test4_192745343en122(t *testing.T) {
	assertCompactFloat(t, 4.192745343e-122, 0, 4.192745343e-122, []byte{0x84, 0x0e, 0x8f, 0xcf, 0xa0, 0xee, 0x7f})
}

func Test0_2Round4(t *testing.T) {
	assertCompactFloat(t, 0.2, 4, 0.2, []byte{0x06, 0x02})
}

func Test0_5935555Round4(t *testing.T) {
	assertCompactFloat(t, 0.5935555, 4, 0.5936, []byte{0x12, 0xae, 0x30})
}

func Test0_1473445219134543Round6(t *testing.T) {
	assertCompactFloat(t, 14.73445219134543, 6, 14.7345, []byte{0x12, 0x88, 0xff, 0x11})
}
