package compact_float

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

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
	return this.APD().String()
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
	return decodeFromString(asString, significantDigits)
}

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

	if !value.Coeff.IsInt64() {
		err = fmt.Errorf("%v cannot fit into a DFloat", value)
		return
	}

	result = DFloat{
		Exponent:    value.Exponent,
		Coefficient: value.Coeff.Int64(),
	}
	if value.Negative {
		result.Coefficient = -result.Coefficient
	}
	return
}

func DFloatFromString(str string) (result DFloat, err error) {
	var stackAlloced apd.Decimal
	value, _, err := apd.BaseContext.SetString(&stackAlloced, str)
	if err != nil {
		return
	}
	result, err = DFloatFromAPD(value)
	return
}

func decodeFromString(value string, significantDigits int) DFloat {
	// No inf or nan. Format: (-)d+(.d+)(e[+-]d+)
	encounteredDot := false
	encounteredExp := false
	isRounding := false
	significandSign := int64(1)
	exponentFromString := int32(0)
	exponentSign := int32(1)
	startIndex := 0

	if value[0] == '-' {
		significandSign = -1
		startIndex++
	}

	exponent := int32(0)
	coefficient := int64(0)
	digitCount := 0
	rounded := int64(0)
	roundedDivider := 1
	lastSigDigit := int32(0)
	for i := startIndex; i < len(value); i++ {
		ch := value[i]
		switch ch {
		case '.':
			encounteredDot = true
			continue
		case 'e', 'E':
			encounteredExp = true
			continue
		case '-':
			exponentSign = -1
			continue
		case '+':
			exponentSign = 1
			continue
		}

		nextDigit := int32(ch - '0')

		if encounteredExp {
			exponentFromString = exponentFromString*10 + nextDigit
			continue
		}
		if isRounding {
			rounded = rounded*10 + int64(nextDigit)
			roundedDivider = roundedDivider * 10
			continue
		}

		if digitCount > 1 || nextDigit > 0 {
			digitCount++
		}
		if significantDigits > 0 && digitCount >= significantDigits {
			lastSigDigit = nextDigit
			isRounding = true
		}
		coefficient = coefficient*10 + int64(nextDigit)
		if encounteredDot {
			exponent--
		}
	}
	exponent += exponentFromString * exponentSign
	coefficient = coefficient * significandSign
	fractional := float64(rounded) / float64(roundedDivider)
	if fractional != 0 {
		if fractional > 0.5 {
			coefficient++
		} else if fractional < 0.5 {
			coefficient--
		} else if lastSigDigit&1 == 1 {
			coefficient++
		} else {
			coefficient--
		}
	}

	return DFloat{
		Exponent:    exponent,
		Coefficient: coefficient,
	}
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
