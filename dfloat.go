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
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/cockroachdb/apd/v2"
)

// An exponent value of ExpSpecial indicates that this is a special value.
// The coefficient will be a special code that determines what special value
// is represented (CoeffInfinity, CoeffNan, etc).
const ExpSpecial = int32(-0x80000000)
const (
	CoeffNegativeZero     = 0
	CoeffInfinity         = 1
	CoeffNegativeInfinity = 5
	CoeffNan              = 2
	CoeffSignalingNan     = 6
)

// DFloat represents a decimal floating point value in 96 bits.
// It supports coefficient values within the range of int64, and exponent
// values from -0x7fffffff to 0x7fffffff. The exponent -0x80000000 (ExpSpecial)
// is used to indicate special values, and is not allowed as an actual exponent
// value.
type DFloat struct {
	Exponent    int32
	Coefficient int64
}

func DFloatValue(exponent int32, coefficient int64) DFloat {
	return DFloat{
		Exponent:    exponent,
		Coefficient: coefficient,
	}.minimized()
}

// Convert an iee754 binary floating point value to DFloat, with the specified
// number of significant digits. Rounding is half-to-even, meaning it rounds
// towards an even number when exactly halfway.
// If significantDigits is less than 1, no rounding takes place.
func DFloatFromFloat64(value float64, significantDigits int) DFloat {
	if math.Float64bits(value) == math.Float64bits(0) {
		return dfloatZero
	} else if value == math.Copysign(0, -1) {
		return dfloatNegativeZero
	} else if math.IsInf(value, 1) {
		return dfloatInfinity
	} else if math.IsInf(value, -1) {
		return dfloatNegativeInfinity
	} else if math.IsNaN(value) {
		bits := math.Float64bits(value)
		if bits&quietBit != 0 {
			return dfloatNaN
		}
		return dfloatSignalingNaN
	}

	asString := strconv.FormatFloat(value, 'g', -1, 64)
	d, err := decodeFromString(asString, significantDigits)
	if err != nil {
		panic(fmt.Errorf("BUG: error decoding stringified float64 %g: %v", value, err))
	}
	return d
}

// Convert an unsigned int to DFloat. If the value is too big to fit, its lowest
// significant digit will be rounded (half-to-even).
func DFloatFromUInt(value uint64) DFloat {
	if value <= 0x7fffffffffffffff {
		return DFloatValue(0, int64(value))
	}

	remainder := value % 10
	value /= 10
	if remainder >= 5 {
		if remainder == 5 {
			if value&1 == 1 {
				value++
			}
		} else {
			value++
		}
	}
	return DFloatValue(1, int64(value))
}

// Convert a big.Int to DFloat. If the value is too big to fit, its lower
// significant digits will be rounded (half-to-even).
func DFloatFromBigInt(value *big.Int) DFloat {
	if value.IsInt64() {
		return DFloatValue(0, value.Int64())
	}

	if value.IsUint64() {
		return DFloatFromUInt(value.Uint64())
	}

	return DFloatFromAPD(apd.NewWithBigInt(value, 0))
}

var bitsToDigits = []int{0, 1, 1, 1, 1, 2, 2, 2, 3, 3}

func DFloatFromBigFloat(value *big.Float) DFloat {
	// Note: big.Float has no NaN representation
	if value.IsInf() {
		if value.Sign() < 0 {
			return dfloatNegativeInfinity
		}
		return dfloatInfinity
	}

	precisionBits := int(value.Prec())
	digits := (precisionBits/10)*3 + bitsToDigits[precisionBits%10]
	str := value.Text('g', digits)
	d, err := DFloatFromString(str)
	if err != nil {
		panic(fmt.Errorf("BUG: Could not parse \"%v\" from big.Float value", str))
	}
	return d
}

// Convert an apd.Decimal to DFloat. If the value is too big to fit, its lower
// significant digits will be rounded (half-to-even).
func DFloatFromAPD(value *apd.Decimal) DFloat {
	if value.IsZero() {
		if value.Negative {
			return dfloatNegativeZero
		}
		return dfloatZero
	}
	switch value.Form {
	case apd.Infinite:
		if value.Negative {
			return dfloatNegativeInfinity
		}
		return dfloatInfinity
	case apd.NaN:
		return dfloatNaN
	case apd.NaNSignaling:
		return dfloatSignalingNaN
	}

	if value.Coeff.IsInt64() {
		d := DFloat{
			Exponent:    value.Exponent,
			Coefficient: value.Coeff.Int64(),
		}.minimized()
		if value.Negative {
			d.Coefficient = -d.Coefficient
		}
		return d
	}

	str := value.Text('g')
	d, err := DFloatFromString(str)
	if err != nil {
		panic(fmt.Errorf("BUG: Could not parse \"%v\" from apd float value", str))
	}
	return d
}

// Convert a string float representation to DFloat. If the value is too big to
// fit, its lower significant digits will be rounded (half-to-even).
func DFloatFromString(str string) (DFloat, error) {
	return decodeFromString(str, 0)
}

func Zero() DFloat {
	return dfloatZero
}

func NegativeZero() DFloat {
	return dfloatNegativeZero
}

func Infinity() DFloat {
	return dfloatInfinity
}

func NegativeInfinity() DFloat {
	return dfloatNegativeInfinity
}

func QuietNaN() DFloat {
	return dfloatNaN
}

func SignalingNaN() DFloat {
	return dfloatSignalingNaN
}

func (this DFloat) IsSpecial() bool {
	return this.Exponent == ExpSpecial
}

// Returns true if the value is positive or negative zero
func (this DFloat) IsZero() bool {
	return this.Coefficient == 0
}

// Returns true if the value is positive or negative infinity
func (this DFloat) IsInfinity() bool {
	return this.IsSpecial() && this.Coefficient&CoeffInfinity != 0
}

// Returns true if the value is a quiet or signaling NaN
func (this DFloat) IsNan() bool {
	return this.IsSpecial() && this.Coefficient&CoeffNan != 0
}

func (this DFloat) IsNegativeZero() bool {
	return this == dfloatNegativeZero
}

func (this DFloat) IsNegativeInfinity() bool {
	return this == dfloatNegativeInfinity
}

func (this DFloat) IsSignalingNan() bool {
	return this == dfloatSignalingNaN
}

func (this DFloat) String() string {
	return this.Text('g')
}

// Text converts the floating-point number x to a string according
// to the given format. The format is one of:
//
//	'e'	-d.dddde±dd, decimal exponent, exponent digits
//	'E'	-d.ddddE±dd, decimal exponent, exponent digits
//	'f'	-ddddd.dddd, no exponent
//	'g'	like 'e' for large exponents, like 'f' otherwise
//	'G'	like 'E' for large exponents, like 'f' otherwise
//
// If format is a different character, Text returns a "%" followed by the
// unrecognized.Format character. The 'f' format has the possibility of
// displaying precision that is not present in the Decimal when it appends
// zeros. All other formats always show the exact precision of the Decimal.
//
// This method call is forwarded to *apd.Decimal.Text()
func (this DFloat) Text(format byte) string {
	return this.APD().Text(format)
}

// Returns the int64 representation of this value.
// Returns an error if the value cannot fit.
func (this DFloat) Int() (int64, error) {
	if this.Exponent < 0 {
		return 0, fmt.Errorf("%v cannot fit into int: Not a whole number", this)
	}
	if int(this.Exponent) >= len(exponentMultipliers)-1 {
		return 0, fmt.Errorf("%v cannot fit into int: Exponent too big", this)
	}
	expMult := int64(exponentMultipliers[this.Exponent])
	result := this.Coefficient * expMult
	if result < 0 || result/expMult != this.Coefficient {
		return 0, fmt.Errorf("%v cannot fit into uint: Value too big", this)
	}
	return result, nil
}

// Returns the uint64 representation of this value.
// Returns an error if the value cannot fit.
func (this DFloat) Uint() (uint64, error) {
	if this.Exponent < 0 {
		return 0, fmt.Errorf("%v cannot fit into uint: Not a whole number", this)
	}
	if int(this.Exponent) >= len(exponentMultipliers) {
		return 0, fmt.Errorf("%v cannot fit into uint: Exponent too big", this)
	}
	expMult := exponentMultipliers[this.Exponent]
	result := uint64(this.Coefficient) * expMult
	if result/expMult != uint64(this.Coefficient) {
		return 0, fmt.Errorf("%v cannot fit into uint: Value too big", this)
	}
	return result, nil
}

// Returns the big.Int representation of this value.
// Returns an error if the value is not a whole number.
func (this DFloat) BigInt() (*big.Int, error) {
	if this.Exponent < 0 {
		return nil, fmt.Errorf("%v cannot fit into big.Int: Not a whole number", this)
	}
	ten := big.NewInt(10)
	exp := big.NewInt(int64(this.Exponent))
	exp.Exp(ten, exp, nil)
	return exp.Mul(exp, big.NewInt(this.Coefficient)), nil
}

// Returns the float64 representation of this value. The result will be rounded
// according to strconv.ParseFloat() if it doesn't fit.
func (this DFloat) Float() float64 {
	switch this {
	case dfloatZero:
		return 0.0
	case dfloatNegativeZero:
		// Go doesn't handle literal -0.0 by design
		v := 0.0
		v = -v
		return v
	case dfloatInfinity:
		return math.Inf(1)
	case dfloatNegativeInfinity:
		return math.Inf(-1)
	case dfloatNaN:
		return math.Float64frombits(math.Float64bits(math.NaN()) | uint64(quietBit))
	case dfloatSignalingNaN:
		return math.Float64frombits(math.Float64bits(math.NaN()) & ^uint64(quietBit))
	}

	result, err := strconv.ParseFloat(this.String(), 64)
	if err != nil {
		panic(fmt.Errorf("BUG: error decoding stringified DFloat %v: %v", this, err))
	}
	return result
}

func (this DFloat) BigFloat() *big.Float {
	switch this {
	case dfloatZero:
		return big.NewFloat(0.0)
	case dfloatNegativeZero:
		// Go doesn't handle literal -0.0 by design
		v := 0.0
		v = -v
		return big.NewFloat(v)
	case dfloatInfinity:
		return big.NewFloat(math.Inf(1))
	case dfloatNegativeInfinity:
		return big.NewFloat(math.Inf(-1))
	case dfloatNaN:
		return big.NewFloat(math.Float64frombits(math.Float64bits(math.NaN()) | uint64(quietBit)))
	case dfloatSignalingNaN:
		return big.NewFloat(math.Float64frombits(math.Float64bits(math.NaN()) & ^uint64(quietBit)))
	}

	str := this.String()
	f, _, err := big.ParseFloat(str, 10, 63, big.ToNearestEven)
	if err != nil {
		panic(fmt.Errorf("BUG: error decoding stringified DFloat %v: %v", this, err))
	}
	return f
}

// Returns the apd.Decimal representation of this value. All DFloat values can
// be represented as apd.Decimal.
func (this DFloat) APD() *apd.Decimal {
	switch this {
	case dfloatNegativeZero:
		v := apd.New(0, 0)
		v.Negative = true
		return v
	case dfloatInfinity:
		v := apd.New(0, 0)
		v.Form = apd.Infinite
		return v
	case dfloatNegativeInfinity:
		v := apd.New(0, 0)
		v.Form = apd.Infinite
		v.Negative = true
		return v
	case dfloatNaN:
		v := apd.New(0, 0)
		v.Form = apd.NaN
		return v
	case dfloatSignalingNaN:
		v := apd.New(0, 0)
		v.Form = apd.NaNSignaling
		return v
	}
	return apd.New(this.Coefficient, this.Exponent)
}

func (this DFloat) minimized() (d DFloat) {
	d = this

	if d.Exponent == ExpSpecial {
		return
	}

	if d.Coefficient == 0 {
		d.Exponent = 0
		return
	}

	for {
		coeff := d.Coefficient / 10
		if coeff*10 != d.Coefficient {
			break
		}
		d.Coefficient = coeff
		d.Exponent++
	}

	return
}

var exponentMultipliers = []uint64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000,
	1000000000, 10000000000, 100000000000, 1000000000000, 10000000000000,
	100000000000000, 1000000000000000, 10000000000000000, 100000000000000000,
	1000000000000000000,  // Max for int64
	10000000000000000000, // Max for uint64
}

var digitsMax = []uint64{
	0, 9, 99, 999, 9999, 99999, 999999, 9999999, 99999999, 999999999,
	9999999999, 99999999999, 999999999999, 9999999999999, 99999999999999,
	999999999999999, 9999999999999999, 99999999999999999, 999999999999999999,
	9999999999999999999,
}

func decodeFromString(value string, significantDigits int) (result DFloat, err error) {
	if len(value) < 1 {
		return dfloatZero, nil
	}

	const significandCap = uint64(0x7fffffffffffffff)

	significandMax := uint64(0)
	significandMaxDigits := len(digitsMax) - 1
	if significantDigits <= 0 || significantDigits > significandMaxDigits {
		significandMax = uint64(0x7fffffffffffffff)
	} else {
		significandMax = digitsMax[significantDigits]
	}

	cutoffDigitCount := 0
	fractionalDigitCount := 0

	exponent := int64(0)
	significand := uint64(0)
	significandSign := int64(1)
	rounded := 0
	firstRounded := true

	if value[0] == '-' {
		significandSign = -1
		value = value[1:]
	}

	if value[0] > '9' {
		value := strings.ToLower(value)
		switch value {
		case "inf", "infinity":
			if significandSign < 0 {
				result = dfloatNegativeInfinity
			} else {
				result = dfloatInfinity
			}
			return
		case "nan":
			if significandSign < 0 {
				err = fmt.Errorf("NaN cannot be negative")
				return
			}
			result = dfloatNaN
			return
		case "snan":
			if significandSign < 0 {
				err = fmt.Errorf("NaN cannot be negative")
				return
			}
			result = dfloatSignalingNaN
			return
		default:
			err = fmt.Errorf("%v: Not a floating point value", value)
		}
	}

	decodeExponent := func(str string) error {
		const exponentCap = int64(0x7fffffff)
		exponentSign := int64(1)
		if str[0] == '-' {
			exponentSign = -1
			str = str[1:]
		} else if str[0] == '+' {
			str = str[1:]
		}

		for _, ch := range str {
			if ch < '0' || ch > '9' {
				return fmt.Errorf("%c: Unexpected character while decoding DFloat exponent", ch)
			}
			exponent = exponent*10 + int64(ch-'0')
			if exponent > exponentCap {
				return fmt.Errorf("Exponent overflow while decoding DFloat")
			}
		}
		exponent *= exponentSign
		return nil
	}

	decodeRoundedFractional := func(str string) error {
		for i, ch := range str {
			switch ch {
			case 'e', 'E':
				return decodeExponent(str[i+1:])
			}
			if ch < '0' || ch > '9' {
				return fmt.Errorf("%c: Unexpected character while decoding DFloat", ch)
			}
			if firstRounded || rounded == 5 {
				rounded = rounded + int(ch-'0')
				firstRounded = false
			}
		}
		return nil
	}

	decodeFractional := func(str string) error {
		for i, ch := range str {
			switch ch {
			case 'e', 'E':
				return decodeExponent(str[i+1:])
			}
			if ch < '0' || ch > '9' {
				return fmt.Errorf("%c: Unexpected character while decoding DFloat fractional", ch)
			}
			nextSignificand := significand*10 + uint64(ch-'0')
			if nextSignificand > significandMax {
				return decodeRoundedFractional(str[i:])
			}
			significand = nextSignificand
			fractionalDigitCount++
		}
		return nil
	}

	decodeRounded := func(str string) error {
		for i, ch := range str {
			switch ch {
			case '.':
				return decodeRoundedFractional(str[i+1:])
			case 'e', 'E':
				return decodeExponent(str[i+1:])
			}
			if ch < '0' || ch > '9' {
				return fmt.Errorf("%c: Unexpected character while decoding DFloat fractional", ch)
			}
			if firstRounded || rounded == 5 {
				rounded = rounded + int(ch-'0')
				firstRounded = false
			}
			cutoffDigitCount++
		}
		return nil
	}

	decodeSignificand := func(str string) error {
		for i, ch := range str {
			switch ch {
			case '.':
				return decodeFractional(str[i+1:])
			case 'e', 'E':
				return decodeExponent(str[i+1:])
			}
			if ch < '0' || ch > '9' {
				return fmt.Errorf("%c: Unexpected character while decoding DFloat significand", ch)
			}
			nextSignificand := significand*10 + uint64(ch-'0')
			if nextSignificand > significandMax {
				return decodeRounded(str[i:])
			}
			significand = nextSignificand
		}
		return nil
	}

	if err := decodeSignificand(value); err != nil {
		return dfloatZero, err
	}

	if rounded > 5 || (rounded == 5 && significand&1 == 1) {
		significand++
	}

	exponent += int64(cutoffDigitCount)
	exponent -= int64(fractionalDigitCount)

	if significand == 0 && significandSign < 0 {
		return dfloatNegativeZero, nil
	}

	result = DFloat{
		Coefficient: int64(significand) * significandSign,
		Exponent:    int32(exponent),
	}.minimized()

	return
}

const quietBit = 1 << 50

var (
	dfloatZero             = DFloat{0, 0}
	dfloatNegativeZero     = DFloat{ExpSpecial, 0}
	dfloatInfinity         = DFloat{ExpSpecial, CoeffInfinity}
	dfloatNegativeInfinity = DFloat{ExpSpecial, CoeffNegativeInfinity}
	dfloatNaN              = DFloat{ExpSpecial, CoeffNan}
	dfloatSignalingNaN     = DFloat{ExpSpecial, CoeffSignalingNan}
)
