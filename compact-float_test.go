package compact_float

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/cockroachdb/apd"
	"github.com/kstenerud/go-describe"
)

func assertEncodeDecode(t *testing.T, strValue string, expectedEncoded []byte) {
	sourceValue, _, err := apd.NewFromString(strValue)
	if err != nil {
		t.Error(err)
		return
	}
	actualEncoded := make([]byte, 1000)
	bytesEncoded, ok := Encode(sourceValue, actualEncoded)
	if !ok {
		t.Errorf("Value %v: could not encode into %v bytes", sourceValue, len(actualEncoded))
		return
	}
	if bytesEncoded != len(expectedEncoded) {
		t.Errorf("Value %v: Expected to encode %v bytes but encoded %v", sourceValue, len(expectedEncoded), bytesEncoded)
		return
	}
	actualEncoded = actualEncoded[:bytesEncoded]
	if !bytes.Equal(expectedEncoded, actualEncoded) {
		t.Errorf("Value %v: Expected encoded %v but got %v", sourceValue, describe.D(expectedEncoded), describe.D(actualEncoded))
		return
	}
	decimalValue, bytesDecoded, err := Decode(expectedEncoded)
	if err != nil {
		t.Error(err)
		return
	}
	if bytesDecoded != len(actualEncoded) {
		t.Errorf("Value %v: Expected to decode %v bytes but decoded %v", sourceValue, len(expectedEncoded), bytesDecoded)
		return
	}
	if decimalValue.Cmp(sourceValue) != 0 {
		t.Errorf("Expected %v but got %v", sourceValue, decimalValue)
	}
}

func assertEncodeDecodeFloat64(t *testing.T, sourceValue float64, significantDigits int, expectedEncoded []byte) float64 {
	actualEncoded := make([]byte, 15)
	bytesEncoded, ok := EncodeFloat64(sourceValue, significantDigits, actualEncoded)
	if !ok {
		t.Errorf("Value %v: could not encode into %v bytes", sourceValue, len(actualEncoded))
		return 0
	}
	if bytesEncoded != len(expectedEncoded) {
		t.Errorf("Value %v: Expected to encode %v bytes but encoded %v", sourceValue, len(expectedEncoded), bytesEncoded)
		return 0
	}
	actualEncoded = actualEncoded[:bytesEncoded]
	if !bytes.Equal(expectedEncoded, actualEncoded) {
		t.Errorf("Value %v: Expected encoded %v but got %v", sourceValue, describe.D(expectedEncoded), describe.D(actualEncoded))
		return 0
	}
	decimalValue, bytesDecoded, err := Decode(expectedEncoded)
	if err != nil {
		t.Error(err)
		return 0
	}
	if bytesDecoded != len(actualEncoded) {
		t.Errorf("Value %v: Expected to decode %v bytes but decoded %v", sourceValue, len(expectedEncoded), bytesDecoded)
		return 0
	}
	actualValue, err := decimalValue.Float64()
	if err != nil {
		t.Error(err)
		return 0
	}
	return actualValue
}

func assertFloat64(t *testing.T, sourceValue float64, significantDigits int, expectedValue float64, expectedEncoded []byte) {
	actualValue := assertEncodeDecodeFloat64(t, sourceValue, significantDigits, expectedEncoded)
	actualAsString := fmt.Sprintf("%v", actualValue)
	expectedAsString := fmt.Sprintf("%v", expectedValue)
	if actualAsString != expectedAsString {
		t.Errorf("Value %v: Expected decoded value %v but got %v", sourceValue, expectedValue, actualValue)
	}
}

func assertNan(t *testing.T, sourceValue float64, significantDigits int, expectedEncoded []byte) {
	actualValue := assertEncodeDecodeFloat64(t, sourceValue, significantDigits, expectedEncoded)
	if !math.IsNaN(actualValue) {
		t.Errorf("Value %v: Expected NaN but got %v", sourceValue, actualValue)
	}
}

func TestZeroF64(t *testing.T) {
	assertFloat64(t, 0, 0, 0, []byte{0x02})
}

func TestDecimal(t *testing.T) {
	assertEncodeDecode(t, "0", []byte{0x02})
	assertEncodeDecode(t, "-0", []byte{0x03})
	assertEncodeDecode(t, "inf", []byte{0x82, 0x00})
	assertEncodeDecode(t, "-inf", []byte{0x83, 0x00})
	assertEncodeDecode(t, "nan", []byte{0x80, 0x00})
	assertEncodeDecode(t, "snan", []byte{0x81, 0x00})

	assertEncodeDecode(t, "1", []byte{0x00, 0x01})
	assertEncodeDecode(t, "1.5", []byte{0x06, 0x0f})
	assertEncodeDecode(t, "-1.2", []byte{0x07, 0x0c})
	assertEncodeDecode(t, "9.445283e+5000", []byte{0x88, 0x9c, 0x01, 0xa3, 0xbf, 0xc0, 0x04})

	assertEncodeDecode(t, "-9.4452837206285466345998345667683453466347345e-5000",
		[]byte{0xcf, 0x9d, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
	assertEncodeDecode(t, "9.4452837206285466345998345667683453466347345e-5000",
		[]byte{0xce, 0x9d, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
	assertEncodeDecode(t, "-9.4452837206285466345998345667683453466347345e+5000",
		[]byte{0xf5, 0x9a, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
	assertEncodeDecode(t, "9.4452837206285466345998345667683453466347345e+5000",
		[]byte{0xf4, 0x9a, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
}

func TestNegZero(t *testing.T) {
	negZero := math.Copysign(0, -1)
	assertFloat64(t, negZero, 0, negZero, []byte{0x03})
}

func TestInfinity(t *testing.T) {
	assertFloat64(t, math.Inf(1), 0, math.Inf(1), []byte{0x82, 0x00})
}

func TestNegativeInfinity(t *testing.T) {
	assertFloat64(t, math.Inf(-1), 0, math.Inf(-1), []byte{0x83, 0x00})
}

func TestQuietNan(t *testing.T) {
	assertNan(t, math.NaN(), 0, []byte{0x80, 0x00})
}

func Test1_0(t *testing.T) {
	assertFloat64(t, 1.0, 0, 1.0, []byte{0x00, 0x01})
}

func Test1_5(t *testing.T) {
	assertFloat64(t, 1.5, 0, 1.5, []byte{0x06, 0x0f})
}

func Test1_2(t *testing.T) {
	assertFloat64(t, 1.2, 0, 1.2, []byte{0x06, 0x0c})
}

func Test1_25(t *testing.T) {
	assertFloat64(t, 1.25, 0, 1.25, []byte{0x0a, 0x7d})
}

func Test8_8419305(t *testing.T) {
	assertFloat64(t, 8.8419305, 0, 8.8419305, []byte{0x1e, 0xe9, 0xd7, 0x94, 0x2a})
}

func Test1999999999999999(t *testing.T) {
	assertFloat64(t, 1999999999999999.0, 0, 1999999999999999.0, []byte{0x00, 0xff, 0xff, 0xb3, 0xcc, 0xd4, 0xdf, 0xc6, 0x03})
}

func Test9_3942e100(t *testing.T) {
	assertFloat64(t, 9.3942e100, 0, 9.3942e100, []byte{0x80, 0x03, 0xf6, 0xdd, 0x05})
}

func Test4_192745343en122(t *testing.T) {
	assertFloat64(t, 4.192745343e-122, 0, 4.192745343e-122, []byte{0x8e, 0x04, 0xff, 0xee, 0xa0, 0xcf, 0x0f})
}

func Test0_2Round4(t *testing.T) {
	assertFloat64(t, 0.2, 4, 0.2, []byte{0x06, 0x02})
}

func Test0_5935555Round4(t *testing.T) {
	assertFloat64(t, 0.5935555, 4, 0.5936, []byte{0x12, 0xb0, 0x2e})
}

func Test0_1473445219134543Round6(t *testing.T) {
	assertFloat64(t, 14.73445219134543, 6, 14.7345, []byte{0x12, 0x91, 0xff, 0x08})
}
