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
	"fmt"
	"math/big"
	"testing"

	"github.com/cockroachdb/apd/v2"
)

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

func assertConvertToString(t *testing.T, value DFloat, expected string) {
	actual := fmt.Sprint(value)
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func assertConvertToUint(t *testing.T, value DFloat, expected uint64) {
	actual, err := value.Uint()
	if err != nil {
		t.Error(err)
		return
	}
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func assertConvertToUintFails(t *testing.T, value DFloat) {
	_, err := value.Uint()
	if err == nil {
		t.Errorf("Expected uint conversion to fail")
	}
}

func assertConvertToInt(t *testing.T, value DFloat, expected int64) {
	actual, err := value.Int()
	if err != nil {
		t.Error(err)
		return
	}
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func assertConvertToIntFails(t *testing.T, value DFloat) {
	_, err := value.Int()
	if err == nil {
		t.Errorf("Expected int conversion to fail")
	}
}

func assertConvertToFloat(t *testing.T, value DFloat, expected float64) {
	actual := value.Float()
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func assertConvertToBigInt(t *testing.T, value DFloat, expected *big.Int) {
	actual, err := value.BigInt()
	if err != nil {
		t.Error(err)
		return
	}
	if actual.Cmp(expected) != 0 {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func assertConvertToBigIntFails(t *testing.T, value DFloat) {
	_, err := value.BigInt()
	if err == nil {
		t.Errorf("Expected big.Int conversion to fail")
	}
}

func assertConvertToBigFloat(t *testing.T, value DFloat, expected *apd.Decimal) {
	actual := value.APD()
	if actual.Cmp(expected) != 0 {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

// ============================================================================

func TestZero(t *testing.T) {
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

func TestNZero(t *testing.T) {
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

func TestInf(t *testing.T) {
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

func TestNInf(t *testing.T) {
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

func TestQNan(t *testing.T) {
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

func TestSNan(t *testing.T) {
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

func TestText(t *testing.T) {
	assertTextFormat(t, "1.0", 'e', "1e+0")
	assertTextFormat(t, "1.0", 'E', "1E+0")
	assertTextFormat(t, "1.0", 'f', "1")
	assertTextFormat(t, "1.0", 'g', "1")
	assertTextFormat(t, "1.0", 'G', "1")

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

func TestConstructor(t *testing.T) {
	assertConvertToString(t, DFloatValue(100, 1), "1e+100")
	assertConvertToString(t, DFloatValue(100, 12), "1.2e+101")
}

func TestConvertFromUInt(t *testing.T) {
	assertConvertToString(t, DFloatFromUInt(uint64(9223372036854775807)), "9223372036854775807")
	assertConvertToString(t, DFloatFromUInt(uint64(9223372036854775808)), "9.22337203685477581e+18")
	assertConvertToString(t, DFloatFromUInt(uint64(9223372036854775815)), "9.22337203685477582e+18")
	assertConvertToString(t, DFloatFromUInt(uint64(9223372036854775825)), "9.22337203685477582e+18")
}

func TestConvertFromFloat(t *testing.T) {
	assertConvertToString(t, DFloatFromFloat64(1.594365, 6), "1.59436")
	assertConvertToString(t, DFloatFromFloat64(7.94812e+100, 3), "7.95e+100")
}

func TestConvertFromBigInt(t *testing.T) {
	assertConvertToString(t, DFloatFromBigInt(new(big.Int).Exp(big.NewInt(1000), big.NewInt(1000), nil)), "1e+3000")
}

func TestConvertFromBigDecimalFloat(t *testing.T) {
	bdf, _, err := apd.NewFromString("1.49634e+100")
	if err != nil {
		t.Error(err)
	}
	assertConvertToString(t, DFloatFromAPD(bdf), "1.49634e+100")
}

func TestConvertToUint(t *testing.T) {
	assertConvertToUint(t, DFloatValue(5, 60340534), uint64(6034053400000))
	assertConvertToUint(t, DFloatValue(1, 1844674407370955161), uint64(18446744073709551610))
	assertConvertToUintFails(t, DFloatValue(2, 1844674407370955161))
	assertConvertToUintFails(t, DFloatValue(-1, 1844674407370955161))
}

func TestConvertToInt(t *testing.T) {
	assertConvertToInt(t, DFloatValue(1, 1234), int64(12340))
	assertConvertToInt(t, DFloatValue(0, 0x7fffffffffffffff), int64(0x7fffffffffffffff))
	assertConvertToInt(t, DFloatValue(18, 1), int64(1000000000000000000))
	assertConvertToIntFails(t, DFloatValue(19, 1))
	assertConvertToIntFails(t, DFloatValue(-1, 1))
}

func TestConvertToFloat(t *testing.T) {
	assertConvertToFloat(t, DFloatValue(1, 1), 10.0)
	assertConvertToFloat(t, DFloatValue(97, 5053), 5.053e+100)
}

func TestConvertToBigInt(t *testing.T) {
	assertConvertToBigInt(t, DFloatValue(1, 1234), big.NewInt(12340))
	assertConvertToBigIntFails(t, DFloatValue(-1, 1))
	assertConvertToBigInt(t, DFloatValue(100, 1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil))
}

func TestConvertToBigFloat(t *testing.T) {
	assertConvertToBigFloat(t, DFloatValue(1, 1), apd.NewWithBigInt(big.NewInt(10), 0))
	assertConvertToBigFloat(t, DFloatValue(100, 105833), apd.NewWithBigInt(big.NewInt(105833), 100))
}
