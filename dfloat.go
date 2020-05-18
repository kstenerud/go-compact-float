package compact_float

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/cockroachdb/apd"
)

const ExpSpecial = int32(-0x80000000)
const (
	CoeffNegativeZero     = 0
	CoeffInfinity         = 1
	CoeffNegativeInfinity = 5
	CoeffNan              = 2
	CoeffSignalingNan     = 6
)

// DFloat represents a decimal floating point value. It supports coefficient
// values within the range of int64, and exponent values from -0x7fffffff to
// 0x7fffffff. The exponent value -0x80000000 is used to indicate special
// values, and is not allowed as an actual exponent value.
type DFloat struct {
	Exponent    int32
	Coefficient int64
}

func DFloatValue(exponent int32, coefficient int64) DFloat {
	return DFloat{
		Exponent:    exponent,
		Coefficient: coefficient,
	}
}

func Zero() DFloat {
	return dfloatZero.Clone()
}

func NegativeZero() DFloat {
	return dfloatNegativeZero.Clone()
}

func Infinity() DFloat {
	return dfloatInfinity.Clone()
}

func NegativeInfinity() DFloat {
	return dfloatNegativeInfinity.Clone()
}

func QuietNaN() DFloat {
	return dfloatNaN.Clone()
}

func SignalingNaN() DFloat {
	return dfloatSignalingNaN.Clone()
}

func (this DFloat) Clone() DFloat {
	return DFloatValue(this.Exponent, this.Coefficient)
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
// This is a direct bridge to *apd.Decimal.Text()
func (this DFloat) Text(format byte) string {
	return this.APD().Text(format)
}

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

	coefficient := this.Coefficient
	if coefficient < 0 {
		coefficient = -coefficient
	}

	apdForm := apd.Decimal{
		Negative: this.Coefficient < 0,
		Exponent: this.Exponent,
		Coeff:    *big.NewInt(coefficient),
	}

	result, _ := strconv.ParseFloat(apdForm.Text('g'), 64)
	return result
}

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

// Convert an iee754 binary floating point value to DFloat, with the specified
// number of significant digits. Rounding is half-to-even, meaning it rounds
// towards an even number when exactly halfway.
// If significantDigits is less than 1, no rounding takes place.
func DFloatFromFloat64(value float64, significantDigits int) DFloat {
	if math.Float64bits(value) == math.Float64bits(0) {
		return dfloatZero.Clone()
	} else if value == math.Copysign(0, -1) {
		return dfloatNegativeZero.Clone()
	} else if math.IsInf(value, 1) {
		return dfloatInfinity.Clone()
	} else if math.IsInf(value, -1) {
		return dfloatNegativeInfinity.Clone()
	} else if math.IsNaN(value) {
		bits := math.Float64bits(value)
		if bits&quietBit != 0 {
			return dfloatNaN.Clone()
		}
		return dfloatSignalingNaN.Clone()
	}

	asString := strconv.FormatFloat(value, 'g', -1, 64)
	d, err := decodeFromString(asString, significantDigits)
	if err != nil {
		panic(fmt.Errorf("BUG: error decoding stringified float64: %v", err))
	}
	return d
}

// Convert an unsigned int to DFloat. If the value is too big to fit, its lowest
// significant digit will be rounded (half-to-even).
func DFloatFromUInt(value uint64) DFloat {
	if value <= 0x7fffffffffffffff {
		return DFloat{
			Coefficient: int64(value),
		}
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
	return DFloat{
		Exponent:    1,
		Coefficient: int64(value),
	}
}

// Convert a big.Int to DFloat. If the value is too big to fit, its lower
// significant digits will be rounded (half-to-even).
func DFloatFromBigInt(value *big.Int) DFloat {
	if value.IsInt64() {
		return DFloat{
			Coefficient: value.Int64(),
		}
	}

	bi := *value

	for !bi.IsInt64() {

	}

	if is32Bit() {
	} else {
		// TODO: Need the last truncated digit to implement round-half-to-even
		magnitude := big.NewInt(100)
		for len(bi.Bits()) > 1 {
			bi.Div(&bi, magnitude)
		}
		if bi.Bits()[0] > 0x7fffffffffffffff {
			bi.Div(&bi, big.NewInt(10))
		}

	}
	return DFloat{}
}

// Convert an apd.Decimal to DFloat. If the value is too big to fit, its lower
// significant digits will be rounded (half-to-even).
func DFloatFromAPD(value *apd.Decimal) (result DFloat, err error) {
	if value.IsZero() {
		if value.Negative {
			return dfloatNegativeZero.Clone(), nil
		}
		return dfloatZero.Clone(), nil
	}
	switch value.Form {
	case apd.Infinite:
		if value.Negative {
			return dfloatNegativeInfinity.Clone(), nil
		}
		return dfloatInfinity.Clone(), nil
	case apd.NaN:
		return dfloatNaN.Clone(), nil
	case apd.NaNSignaling:
		return dfloatSignalingNaN.Clone(), nil
	}

	if value.Coeff.IsInt64() {
		result = DFloat{
			Exponent:    value.Exponent,
			Coefficient: value.Coeff.Int64(),
		}
		if value.Negative {
			result.Coefficient = -result.Coefficient
		}
		return
	}

	return DFloatFromString(value.String())
}

// Convert a string float representation to DFloat. If the value is too big to
// fit, its lower significant digits will be rounded (half-to-even).
func DFloatFromString(str string) (result DFloat, err error) {
	return decodeFromString(str, 0)
}

var digitsMax = []uint64{
	0, 9, 99, 999, 9999, 99999, 999999, 9999999, 99999999, 999999999,
	9999999999, 99999999999, 999999999999, 9999999999999, 99999999999999,
	999999999999999, 9999999999999999, 99999999999999999, 999999999999999999,
	9999999999999999999,
}

func decodeFromString(value string, significantDigits int) (result DFloat, err error) {
	if len(value) < 1 {
		return DFloat{}, nil
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

	if value[0] >= 'A' {
		value := strings.ToLower(value)
		switch value {
		case "inf", "infinity":
			if significandSign < 0 {
				result = dfloatNegativeInfinity.Clone()
			} else {
				result = dfloatInfinity.Clone()
			}
			return
		case "nan":
			if significandSign < 0 {
				err = fmt.Errorf("NaN cannot be negative")
				return
			}
			result = dfloatNaN.Clone()
			return
		case "snan":
			if significandSign < 0 {
				err = fmt.Errorf("NaN cannot be negative")
				return
			}
			result = dfloatSignalingNaN.Clone()
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
		return DFloat{}, err
	}

	if rounded > 5 || (rounded == 5 && significand&1 == 1) {
		significand++
	}

	exponent += int64(cutoffDigitCount)
	exponent -= int64(fractionalDigitCount)

	if significand == 0 && significandSign < 0 {
		return DFloat{
			Coefficient: CoeffNegativeZero,
			Exponent:    ExpSpecial,
		}, nil
	}

	return DFloat{
		Coefficient: int64(significand) * significandSign,
		Exponent:    int32(exponent),
	}, nil
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
