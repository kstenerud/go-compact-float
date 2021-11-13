// Copyright 2019 Karl Stenerud
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

package compact_float

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/cockroachdb/apd/v2"
	"github.com/kstenerud/go-describe"
)

func assertCodecAPD(t *testing.T, sourceValue *apd.Decimal, expectedEncoded []byte) {
	actualEncoded := &bytes.Buffer{}
	bytesEncoded, err := EncodeBig(sourceValue, actualEncoded)
	if err != nil {
		t.Errorf("Value %v: Error encoding: %v", sourceValue, err)
		return
	}
	if bytesEncoded != len(expectedEncoded) {
		t.Errorf("Value %v: Expected to encode %v bytes but encoded %v", sourceValue, len(expectedEncoded), bytesEncoded)
		return
	}
	if !bytes.Equal(expectedEncoded, actualEncoded.Bytes()) {
		t.Errorf("Value %v: Expected encoded %v but got %v", sourceValue, describe.D(expectedEncoded), describe.D(actualEncoded.Bytes()))
		return
	}
	var value DFloat
	var bigValue *apd.Decimal
	var bytesDecoded int
	for i := 0; i < 2; i++ {
		oversizeEncoded := bytes.NewBuffer(expectedEncoded)
		for j := 0; j < i; j++ {
			oversizeEncoded.WriteByte(0)
		}
		value, bigValue, bytesDecoded, err = Decode(oversizeEncoded)
		if err != nil {
			t.Errorf("Value %v: %v", sourceValue, err)
			return
		}
		if bytesDecoded != len(expectedEncoded) {
			t.Errorf("Value %v: Expected to decode %v bytes but decoded %v", sourceValue, len(expectedEncoded), bytesDecoded)
			return
		}
		if bigValue != nil {
			if bigValue.Cmp(sourceValue) != 0 {
				t.Errorf("Expected decoded big %v but got %v", sourceValue, bigValue)
			}
			return
		}
	}

	expectedValue, err := DFloatFromAPD(sourceValue)
	if err != nil {
		t.Errorf("Unexpected error converting from apd.Decimal %v to dfloat: %v", sourceValue, err)
		return
	}
	if value != expectedValue {
		t.Errorf("Value %v: Expected decoded dfloat %v but got %v", sourceValue, expectedValue, value)
		return
	}
}

func assertCodecDecimal(t *testing.T, expectedValue DFloat, expectedEncoded []byte) {
	actualEncoded := &bytes.Buffer{}
	bytesEncoded, err := Encode(expectedValue, actualEncoded)
	if err != nil {
		t.Errorf("Value %v: Error encoding: %v", expectedValue, err)
		return
	}
	if bytesEncoded != len(expectedEncoded) {
		t.Errorf("Value %v: Expected to encode %v bytes but encoded %v", expectedValue, len(expectedEncoded), bytesEncoded)
		return
	}
	if !bytes.Equal(expectedEncoded, actualEncoded.Bytes()) {
		t.Errorf("Value %v: Expected encoded %v but got %v", expectedValue, describe.D(expectedEncoded), describe.D(actualEncoded.Bytes()))
		return
	}
	actualValue, _, bytesDecoded, err := Decode(bytes.NewBuffer(expectedEncoded))
	if err != nil {
		t.Errorf("Value %v: %v", expectedValue, err)
		return
	}
	if bytesDecoded != len(expectedEncoded) {
		t.Errorf("Value %v: Expected to decode %v bytes but decoded %v", expectedValue, len(expectedEncoded), bytesDecoded)
		return
	}
	if actualValue != expectedValue {
		t.Errorf("Expected %v but got %v", expectedValue, actualValue)
		return
	}
	assertCodecAPD(t, expectedValue.APD(), expectedEncoded)
}

func assertAPD(t *testing.T, strValue string, expectedEncoded []byte) {
	sourceValue, _, err := apd.NewFromString(strValue)
	if err != nil {
		t.Errorf("Unexpected error converting string %v to apd.Decimal: %v", strValue, err)
		return
	}
	assertCodecAPD(t, sourceValue, expectedEncoded)
}

func assertDecimal(t *testing.T, strValue string, expectedEncoded []byte, expectedErr error) {
	expectedValue := assertDFloatFromString(t, strValue, expectedErr)
	assertCodecDecimal(t, expectedValue, expectedEncoded)
	fromAPD, err := DFloatFromAPD(expectedValue.APD())
	if err != nil {
		t.Errorf("Unexpected error converting from apd.Decimal %v to dfloat: %v", expectedValue.APD(), err)
		return
	}
	if fromAPD != expectedValue {
		t.Errorf("Expected conversion from APD to be %v but got %v", expectedValue, fromAPD)
	}
}

func assertFloat64(t *testing.T, sourceValue float64, significantDigits int, expectedValue float64, expectedEncoded []byte, expectedErr error) {
	sourceDecimal := assertDFloatFromFloat64(t, sourceValue, significantDigits, expectedErr)
	assertCodecDecimal(t, sourceDecimal, expectedEncoded)
	actualValue := sourceDecimal.Float()

	if math.IsNaN(expectedValue) {
		if !math.IsNaN(actualValue) {
			t.Errorf("Value %v, digits %v: Expected %v but got %v", sourceValue, significantDigits, expectedValue, actualValue)
			return
		}
		expectedQuietBit := math.Float64bits(expectedValue) & quietBit
		actualQuietBit := math.Float64bits(actualValue) & quietBit
		if expectedQuietBit != actualQuietBit {
			t.Errorf("Value %v, digits %v: Expected quiet bit %x but got %x", sourceValue, significantDigits, expectedQuietBit, actualQuietBit)
			return
		}
	}

	expectedString := fmt.Sprintf("%g", expectedValue)
	actualString := fmt.Sprintf("%g", actualValue)
	if actualString != expectedString {
		t.Errorf("Value %v, digits %v: Expected string value %v but got %v", sourceValue, significantDigits, expectedString, actualString)
		return
	}
}

// ============================================================================
// ============================================================================

func TestAPD(t *testing.T) {
	assertAPD(t, "0", []byte{0x02})
	assertAPD(t, "-0", []byte{0x03})
	assertAPD(t, "inf", []byte{0x82, 0x00})
	assertAPD(t, "-inf", []byte{0x83, 0x00})
	assertAPD(t, "nan", []byte{0x80, 0x00})
	assertAPD(t, "snan", []byte{0x81, 0x00})

	assertAPD(t, "1", []byte{0x00, 0x01})
	assertAPD(t, "1.5", []byte{0x06, 0x0f})
	assertAPD(t, "-1.2", []byte{0x07, 0x0c})
	assertAPD(t, "9.445283e+5000", []byte{0x88, 0x9c, 0x01, 0xa3, 0xbf, 0xc0, 0x04})

	assertAPD(t, "-9.4452837206285466345998345667683453466347345e-5000",
		[]byte{0xcf, 0x9d, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
	assertAPD(t, "9.4452837206285466345998345667683453466347345e-5000",
		[]byte{0xce, 0x9d, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
	assertAPD(t, "-9.4452837206285466345998345667683453466347345e+5000",
		[]byte{0xf5, 0x9a, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
	assertAPD(t, "9.4452837206285466345998345667683453466347345e+5000",
		[]byte{0xf4, 0x9a, 0x01, 0xd1, 0x8e, 0xa2, 0xe6, 0x83, 0x8a, 0xbf, 0xc1, 0xbb,
			0xe1, 0xf3, 0xdf, 0xfc, 0xee, 0xac, 0xe5, 0xfe, 0xe1, 0x8f, 0xe2, 0x43})
}

func TestDecimal(t *testing.T) {
	assertDecimal(t, "0", []byte{0x02}, nil)
	assertDecimal(t, "-0", []byte{0x03}, nil)
	assertDecimal(t, "inf", []byte{0x82, 0x00}, nil)
	assertDecimal(t, "-inf", []byte{0x83, 0x00}, nil)
	assertDecimal(t, "nan", []byte{0x80, 0x00}, nil)
	assertDecimal(t, "snan", []byte{0x81, 0x00}, nil)

	assertDecimal(t, "1", []byte{0x00, 0x01}, nil)
	assertDecimal(t, "1.5", []byte{0x06, 0x0f}, nil)
	assertDecimal(t, "-1.2", []byte{0x07, 0x0c}, nil)
	assertDecimal(t, "9.445283e+5000", []byte{0x88, 0x9c, 0x01, 0xa3, 0xbf, 0xc0, 0x04}, nil)

	// 0x7fffffffffffffff
	assertDecimal(t, "9223372036854775807", []byte{0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, nil)
	// 0x8000000000000000, rounded
	assertDecimal(t, "9223372036854775808", []byte{0x04, 0xcd, 0x99, 0xb3, 0xe6, 0xcc, 0x99, 0xb3, 0xe6, 0x0c}, RoundingError())
	assertDecimal(t, "9223372036854775809", []byte{0x04, 0xcd, 0x99, 0xb3, 0xe6, 0xcc, 0x99, 0xb3, 0xe6, 0x0c}, RoundingError())
	assertDecimal(t, "9223372036854775815", []byte{0x04, 0xce, 0x99, 0xb3, 0xe6, 0xcc, 0x99, 0xb3, 0xe6, 0x0c}, RoundingError())
}

func TestSpecialF64(t *testing.T) {
	qnan := math.NaN()
	qnan = math.Float64frombits(math.Float64bits(qnan) | quietBit)
	if !math.IsNaN(qnan) {
		t.Errorf("Expected nan but got %v", qnan)
		return
	}
	snan := math.NaN()
	snan = math.Float64frombits(math.Float64bits(snan) & ^uint64(quietBit))
	if !math.IsNaN(snan) {
		t.Errorf("Expected nan but got %v", snan)
		return
	}
	nzero := 0.0
	nzero = -nzero
	inf := math.Inf(1)
	ninf := math.Inf(-1)

	assertFloat64(t, 0, 0, 0, []byte{0x02}, nil)
	assertFloat64(t, nzero, 0, nzero, []byte{0x03}, nil)
	assertFloat64(t, qnan, 0, qnan, []byte{0x80, 0x00}, nil)
	assertFloat64(t, snan, 0, snan, []byte{0x81, 0x00}, nil)
	assertFloat64(t, inf, 0, inf, []byte{0x82, 0x00}, nil)
	assertFloat64(t, ninf, 0, ninf, []byte{0x83, 0x00}, nil)
}

func Test1_0(t *testing.T) {
	assertFloat64(t, 1.0, 0, 1.0, []byte{0x00, 0x01}, nil)
}

func Test1_5(t *testing.T) {
	assertFloat64(t, 1.5, 0, 1.5, []byte{0x06, 0x0f}, nil)
}

func Test1_2(t *testing.T) {
	assertFloat64(t, 1.2, 0, 1.2, []byte{0x06, 0x0c}, nil)
}

func Test1_25(t *testing.T) {
	assertFloat64(t, 1.25, 0, 1.25, []byte{0x0a, 0x7d}, nil)
}

func Test8_8419305(t *testing.T) {
	assertFloat64(t, 8.8419305, 0, 8.8419305, []byte{0x1e, 0xe9, 0xd7, 0x94, 0x2a}, nil)
}

func Test1999999999999999(t *testing.T) {
	assertFloat64(t, 1999999999999999.0, 0, 1999999999999999.0, []byte{0x00, 0xff, 0xff, 0xb3, 0xcc, 0xd4, 0xdf, 0xc6, 0x03}, nil)
}

func Test9_3942e100(t *testing.T) {
	assertFloat64(t, 9.3942e100, 0, 9.3942e100, []byte{0x80, 0x03, 0xf6, 0xdd, 0x05}, nil)
}

func Test4_192745343en122(t *testing.T) {
	assertFloat64(t, 4.192745343e-122, 0, 4.192745343e-122, []byte{0x8e, 0x04, 0xff, 0xee, 0xa0, 0xcf, 0x0f}, nil)
}

func Test0_2Round4(t *testing.T) {
	assertFloat64(t, 0.2, 4, 0.2, []byte{0x06, 0x02}, nil)
}

func Test0_5935555Round4(t *testing.T) {
	assertFloat64(t, 0.5935555, 4, 0.5936, []byte{0x12, 0xb0, 0x2e}, RoundingError())
}

func Test0_1473445219134543Round6(t *testing.T) {
	assertFloat64(t, 14.73445219134543, 6, 14.7345, []byte{0x12, 0x91, 0xff, 0x08}, RoundingError())
}
