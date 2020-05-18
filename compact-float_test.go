package compact_float

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/cockroachdb/apd"
	"github.com/kstenerud/go-describe"
)

func testAPD(t *testing.T, sourceValue *apd.Decimal, expectedEncoded []byte) {
	actualEncoded := make([]byte, MaxEncodeLengthBig(sourceValue))
	bytesEncoded, ok := EncodeBig(sourceValue, actualEncoded)
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
	value, bigValue, bytesDecoded, err := Decode(expectedEncoded)
	if err != nil {
		t.Errorf("Value %v: %v", sourceValue, err)
		return
	}
	if bytesDecoded != len(actualEncoded) {
		t.Errorf("Value %v: Expected to decode %v bytes but decoded %v", sourceValue, len(expectedEncoded), bytesDecoded)
		return
	}
	if bigValue != nil {
		if bigValue.Cmp(sourceValue) != 0 {
			t.Errorf("Expected decoded big %v but got %v", sourceValue, bigValue)
		}
		return
	}

	expectedValue, err := DFloatFromAPD(sourceValue)
	if err != nil {
		t.Error(err)
		return
	}
	if value != expectedValue {
		t.Errorf("Value %v: Expected decoded dfloat %v but got %v", sourceValue, expectedValue, value)
		return
	}
}

func testDecimal(t *testing.T, expectedValue DFloat, expectedEncoded []byte) DFloat {
	actualEncoded := make([]byte, MaxEncodeLength())
	bytesEncoded, ok := Encode(expectedValue, actualEncoded)
	if !ok {
		t.Errorf("Value %v: could not encode into %v bytes", expectedValue, len(actualEncoded))
		return dfloatZero
	}
	if bytesEncoded != len(expectedEncoded) {
		t.Errorf("Value %v: Expected to encode %v bytes but encoded %v", expectedValue, len(expectedEncoded), bytesEncoded)
		return dfloatZero
	}
	actualEncoded = actualEncoded[:bytesEncoded]
	if !bytes.Equal(expectedEncoded, actualEncoded) {
		t.Errorf("Value %v: Expected encoded %v but got %v", expectedValue, describe.D(expectedEncoded), describe.D(actualEncoded))
		return dfloatZero
	}
	actualValue, _, bytesDecoded, err := Decode(expectedEncoded)
	if err != nil {
		t.Errorf("Value %v: %v", expectedValue, err)
		return dfloatZero
	}
	if bytesDecoded != len(actualEncoded) {
		t.Errorf("Value %v: Expected to decode %v bytes but decoded %v", expectedValue, len(expectedEncoded), bytesDecoded)
		return dfloatZero
	}
	if actualValue != expectedValue {
		t.Errorf("Expected %v but got %v", expectedValue, actualValue)
		return dfloatZero
	}
	testAPD(t, expectedValue.APD(), expectedEncoded)
	return actualValue
}

func assertAPD(t *testing.T, strValue string, expectedEncoded []byte) {
	sourceValue, _, err := apd.NewFromString(strValue)
	if err != nil {
		t.Error(err)
		return
	}
	testAPD(t, sourceValue, expectedEncoded)
}

func assertDecimal(t *testing.T, strValue string, expectedEncoded []byte) {
	expectedValue, err := DFloatFromString(strValue)
	if err != nil {
		t.Error(err)
		return
	}
	testDecimal(t, expectedValue, expectedEncoded)
}

func assertFloat64(t *testing.T, sourceValue float64, significantDigits int, expectedValue float64, expectedEncoded []byte) {
	sourceDecimal := DFloatFromFloat64(sourceValue, significantDigits)
	actualDecimal := testDecimal(t, sourceDecimal, expectedEncoded)
	actualValue := actualDecimal.Float()

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

func assertSpecialValue(t *testing.T, expectedByteCount int,
	f func(dst []byte) (bytesEncoded int, ok bool),
	tst func(DFloat)) {
	actualEncoded := make([]byte, expectedByteCount, expectedByteCount)
	bytesEncoded, ok := f(actualEncoded)
	if !ok {
		t.Errorf("Could not encode into %v bytes", len(actualEncoded))
		return
	}
	if bytesEncoded != expectedByteCount {
		t.Errorf("Expected to encode into %v bytes but used %v", expectedByteCount, bytesEncoded)
		return
	}
	decimalValue, _, bytesDecoded, err := Decode(actualEncoded)
	if err != nil {
		t.Error(err)
		return
	}
	if bytesDecoded != expectedByteCount {
		t.Errorf("Expected to decode %v bytes but decoded %v", expectedByteCount, bytesDecoded)
		return
	}
	tst(decimalValue)
}

// ============================================================================
// ============================================================================

func TestZeroTests(t *testing.T) {
	v := Zero()
	if !v.IsZero() {
		t.Errorf("%v should be zero", v)
	}
	if v.IsNegativeZero() {
		t.Errorf("%v should not be -0", v)
	}
	if v.IsInfinity() {
		t.Errorf("%v should not be inf", v)
	}
	if v.IsNegativeInfinity() {
		t.Errorf("%v should not be -inf", v)
	}
	if v.IsNan() {
		t.Errorf("%v should not be NaN", v)
	}
	if v.IsSignalingNan() {
		t.Errorf("%v should not be signaling NaN", v)
	}
}

func TestNZeroTests(t *testing.T) {
	v := NegativeZero()
	if !v.IsZero() {
		t.Errorf("%v should be zero", v)
	}
	if !v.IsNegativeZero() {
		t.Errorf("%v should be -0", v)
	}
	if v.IsInfinity() {
		t.Errorf("%v should not be inf", v)
	}
	if v.IsNegativeInfinity() {
		t.Errorf("%v should not be -inf", v)
	}
	if v.IsNan() {
		t.Errorf("%v should not be NaN", v)
	}
	if v.IsSignalingNan() {
		t.Errorf("%v should not be signaling NaN", v)
	}
}

func TestInfTests(t *testing.T) {
	v := Infinity()
	if v.IsZero() {
		t.Errorf("%v should not be zero", v)
	}
	if v.IsNegativeZero() {
		t.Errorf("%v should not be -0", v)
	}
	if !v.IsInfinity() {
		t.Errorf("%v should be inf", v)
	}
	if v.IsNegativeInfinity() {
		t.Errorf("%v should not be -inf", v)
	}
	if v.IsNan() {
		t.Errorf("%v should not be NaN", v)
	}
	if v.IsSignalingNan() {
		t.Errorf("%v should not be signaling NaN", v)
	}
}

func TestNInfTests(t *testing.T) {
	v := NegativeInfinity()
	if v.IsZero() {
		t.Errorf("%v should not be zero", v)
	}
	if v.IsNegativeZero() {
		t.Errorf("%v should not be -0", v)
	}
	if !v.IsInfinity() {
		t.Errorf("%v should be inf", v)
	}
	if !v.IsNegativeInfinity() {
		t.Errorf("%v should be -inf", v)
	}
	if v.IsNan() {
		t.Errorf("%v should not be NaN", v)
	}
	if v.IsSignalingNan() {
		t.Errorf("%v should not be signaling NaN", v)
	}
}

func TestQNanTests(t *testing.T) {
	v := QuietNaN()
	if v.IsZero() {
		t.Errorf("%v should not be zero", v)
	}
	if v.IsNegativeZero() {
		t.Errorf("%v should not be -0", v)
	}
	if v.IsInfinity() {
		t.Errorf("%v should not be inf", v)
	}
	if v.IsNegativeInfinity() {
		t.Errorf("%v should not be -inf", v)
	}
	if !v.IsNan() {
		t.Errorf("%v should be NaN", v)
	}
	if v.IsSignalingNan() {
		t.Errorf("%v should not be signaling NaN", v)
	}
}

func TestSNanTests(t *testing.T) {
	v := SignalingNaN()
	if v.IsZero() {
		t.Errorf("%v should not be zero", v)
	}
	if v.IsNegativeZero() {
		t.Errorf("%v should not be -0", v)
	}
	if v.IsInfinity() {
		t.Errorf("%v should not be inf", v)
	}
	if v.IsNegativeInfinity() {
		t.Errorf("%v should not be -inf", v)
	}
	if !v.IsNan() {
		t.Errorf("%v should be NaN", v)
	}
	if !v.IsSignalingNan() {
		t.Errorf("%v should be signaling NaN", v)
	}
}

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
	assertDecimal(t, "0", []byte{0x02})
	assertDecimal(t, "-0", []byte{0x03})
	assertDecimal(t, "inf", []byte{0x82, 0x00})
	assertDecimal(t, "-inf", []byte{0x83, 0x00})
	assertDecimal(t, "nan", []byte{0x80, 0x00})
	assertDecimal(t, "snan", []byte{0x81, 0x00})

	assertDecimal(t, "1", []byte{0x00, 0x01})
	assertDecimal(t, "1.5", []byte{0x06, 0x0f})
	assertDecimal(t, "-1.2", []byte{0x07, 0x0c})
	assertDecimal(t, "9.445283e+5000", []byte{0x88, 0x9c, 0x01, 0xa3, 0xbf, 0xc0, 0x04})
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

	assertFloat64(t, 0, 0, 0, []byte{0x02})
	assertFloat64(t, nzero, 0, nzero, []byte{0x03})
	assertFloat64(t, qnan, 0, qnan, []byte{0x80, 0x00})
	assertFloat64(t, snan, 0, snan, []byte{0x81, 0x00})
	assertFloat64(t, inf, 0, inf, []byte{0x82, 0x00})
	assertFloat64(t, ninf, 0, ninf, []byte{0x83, 0x00})
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

func assertTextFormat(t *testing.T, value string, format byte, expected string) {
	dvalue, err := DFloatFromString(value)
	if err != nil {
		t.Error(err)
		return
	}
	actual := dvalue.Text(format)
	if actual != expected {
		t.Errorf("Value %v, format %c: Expected %v but got %v", dvalue, format, expected, actual)
	}
}

func TestText(t *testing.T) {
	assertTextFormat(t, "1.0", 'e', "1.0e+0")
	assertTextFormat(t, "1.0", 'E', "1.0E+0")
	assertTextFormat(t, "1.0", 'f', "1.0")
	assertTextFormat(t, "1.0", 'g', "1.0")
	assertTextFormat(t, "1.0", 'G', "1.0")

	assertTextFormat(t, "1.2345678901234", 'e', "1.2345678901234e+0")
	assertTextFormat(t, "1.2345678901234", 'E', "1.2345678901234E+0")
	assertTextFormat(t, "1.2345678901234", 'f', "1.2345678901234")
	assertTextFormat(t, "1.2345678901234", 'g', "1.2345678901234")
	assertTextFormat(t, "1.2345678901234", 'G', "1.2345678901234")

	assertTextFormat(t, "123.45678901234", 'e', "1.2345678901234e+2")
	assertTextFormat(t, "123.45678901234", 'E', "1.2345678901234E+2")
	assertTextFormat(t, "123.45678901234", 'f', "123.45678901234")
	assertTextFormat(t, "123.45678901234", 'g', "123.45678901234")
	assertTextFormat(t, "123.45678901234", 'G', "123.45678901234")

	assertTextFormat(t, "1.2345678901234e+100", 'e', "1.2345678901234e+100")
	assertTextFormat(t, "1.2345678901234e+100", 'E', "1.2345678901234E+100")
	assertTextFormat(t, "1.2345678901234e+100", 'g', "1.2345678901234e+100")
	assertTextFormat(t, "1.2345678901234e+100", 'G', "1.2345678901234E+100")
}

func TestUIntNoTruncate(t *testing.T) {
	v := uint64(9223372036854775807)
	df := DFloatFromUInt(v)
	expected := fmt.Sprint(v)
	actual := fmt.Sprint(df)
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestUIntTruncate(t *testing.T) {
	v := uint64(9223372036854775808)
	df := DFloatFromUInt(v)
	expected := "9.22337203685477581e+18"
	actual := fmt.Sprint(df)
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestUIntTruncateRoundUp(t *testing.T) {
	v := uint64(9223372036854775815)
	df := DFloatFromUInt(v)
	expected := "9.22337203685477582e+18"
	actual := fmt.Sprint(df)
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestUIntTruncateRoundDown(t *testing.T) {
	v := uint64(9223372036854775825)
	df := DFloatFromUInt(v)
	expected := "9.22337203685477582e+18"
	actual := fmt.Sprint(df)
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}
