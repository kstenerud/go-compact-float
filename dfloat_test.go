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

func assertConvertFromString(t *testing.T, str string, expected string, expectedErr error) {
	value, err := DFloatFromString(str)
	if err != expectedErr {
		t.Errorf("Expected conversion of string %v to cause error %v but got %v (produced value %v)", str, expectedErr, err, value)
		return
	}
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

func assertDFloatFromString(t *testing.T, value string, expectedErr error) DFloat {
	result, err := DFloatFromString(value)
	if err != expectedErr {
		t.Errorf("Expected conversion of %v to produce error %v but got %v", value, expectedErr, err)
	}
	return result
}

func assertDFloatFromUint(t *testing.T, value uint64, expectedErr error) DFloat {
	result, err := DFloatFromUInt(value)
	if err != expectedErr {
		t.Errorf("Expected conversion of %v to produce error %v but got %v", value, expectedErr, err)
	}
	return result
}

func assertDFloatFromBigInt(t *testing.T, value *big.Int, expectedErr error) DFloat {
	result, err := DFloatFromBigInt(value)
	if err != expectedErr {
		t.Errorf("Expected conversion of %v to produce error %v but got %v", value, expectedErr, err)
	}
	return result
}

func assertDFloatFromFloat64(t *testing.T, value float64, sigDigits int, expectedErr error) DFloat {
	result, err := DFloatFromFloat64(value, sigDigits)
	if err != expectedErr {
		t.Errorf("Expected conversion of %v with %v significant digits to produce error %v but got %v", value, sigDigits, expectedErr, err)
	}
	return result
}

func assertDFloatFromBigFloat(t *testing.T, value *big.Float, expectedErr error) DFloat {
	result, err := DFloatFromBigFloat(value)
	if err != expectedErr {
		t.Errorf("Expected conversion of %v to produce error %v but got %v", value, expectedErr, err)
	}
	return result
}

func assertDFloatFromAPD(t *testing.T, value *apd.Decimal, expectedErr error) DFloat {
	result, err := DFloatFromAPD(value)
	if err != expectedErr {
		t.Errorf("Expected conversion of %v to produce error %v but got %v", value, expectedErr, err)
	}
	return result
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

	assertTextFormat(t, "-0", 'e', "-0")
	assertTextFormat(t, "-0", 'E', "-0")
	assertTextFormat(t, "-0", 'f', "-0")
	assertTextFormat(t, "-0", 'g', "-0")
	assertTextFormat(t, "-0", 'G', "-0")

	assertTextFormat(t, "-0.0", 'e', "-0")
	assertTextFormat(t, "-0.0", 'E', "-0")
	assertTextFormat(t, "-0.0", 'f', "-0")
	assertTextFormat(t, "-0.0", 'g', "-0")
	assertTextFormat(t, "-0.0", 'G', "-0")

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
	assertConvertToString(t, assertDFloatFromUint(t, uint64(9223372036854775807), nil), "9223372036854775807")
	assertConvertToString(t, assertDFloatFromUint(t, uint64(9223372036854775808), RoundingError()), "9.22337203685477581e+18")
	assertConvertToString(t, assertDFloatFromUint(t, uint64(9223372036854775815), RoundingError()), "9.22337203685477582e+18")
	assertConvertToString(t, assertDFloatFromUint(t, uint64(9223372036854775825), RoundingError()), "9.22337203685477582e+18")
}

func TestConvertFromFloat(t *testing.T) {
	assertConvertToString(t, assertDFloatFromFloat64(t, 1.594365, 6, RoundingError()), "1.59436")
	assertConvertToString(t, assertDFloatFromFloat64(t, 7.94812e+100, 3, RoundingError()), "7.95e+100")
}

func TestConvertFromBigInt(t *testing.T) {
	assertConvertToString(t, assertDFloatFromBigInt(t, new(big.Int).Exp(big.NewInt(1000), big.NewInt(1000), nil), RoundingError()), "1e+3000")
}

func TestConvertFromBigFloat(t *testing.T) {
	v := big.NewFloat(123456789012345)
	v.SetPrec(100) // 30 digits = 10 sets of 3 at 10 bits per set of 3
	v = v.Add(v, big.NewFloat(0.678901234567890))
	assertConvertToString(t, assertDFloatFromBigFloat(t, v, RoundingError()), "123456789012345.6789")
}

func TestConvertFromBigDecimalFloat(t *testing.T) {
	bdf, _, err := apd.NewFromString("1.49634e+100")
	if err != nil {
		t.Error(err)
	}
	assertConvertToString(t, assertDFloatFromAPD(t, bdf, nil), "1.49634e+100")
}

func TestConvertFromString(t *testing.T) {
	assertConvertFromString(t, "1e+3000", "1e+3000", nil)
	assertConvertFromString(t, "1.23456789123456789123456789e+100", "1.234567891234567891e+100", RoundingError())
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

func TestConvertToBigFloat(t *testing.T) {
	str := "123456789012345.6789"
	expected, _, err := big.ParseFloat(str, 10, 63, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	df, err := DFloatFromString(str)
	if err != nil {
		panic(err)
	}
	actual := df.BigFloat()
	if actual.Cmp(expected) != 0 {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestConvertToBigInt(t *testing.T) {
	assertConvertToBigInt(t, DFloatValue(1, 1234), big.NewInt(12340))
	assertConvertToBigIntFails(t, DFloatValue(-1, 1))
	assertConvertToBigInt(t, DFloatValue(100, 1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil))
}

func TestConvertToBigDecimalFloat(t *testing.T) {
	assertConvertToBigFloat(t, DFloatValue(1, 1), apd.NewWithBigInt(big.NewInt(10), 0))
	assertConvertToBigFloat(t, DFloatValue(100, 105833), apd.NewWithBigInt(big.NewInt(105833), 100))
}
