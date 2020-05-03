package compact_float

import (
	"fmt"
	"math"
	"math/big"

	"github.com/cockroachdb/apd"
	"github.com/kstenerud/go-uleb128"
)

var ErrorIncomplete = fmt.Errorf("Compact float value is incomplete")

func Encode(value *apd.Decimal, dst []byte) (bytesEncoded int, ok bool) {
	if value.IsZero() {
		if value.Negative {
			return encodeNegativeZero(dst)
		}
		return encodeZero(dst)
	}
	switch value.Form {
	case apd.Infinite:
		if value.Negative {
			return encodeNegativeInfinity(dst)
		}
		return encodeInfinity(dst)
	case apd.NaN:
		return encodeQuietNan(dst)
	case apd.NaNSignaling:
		return encodeSignalingNan(dst)
	}

	exponent := value.Exponent
	exponentSign := 0
	if exponent < 0 {
		exponent = -exponent
		exponentSign = 1
	}
	significandSign := 0
	if value.Negative {
		significandSign = 1
	}
	exponentField := uint64(exponent)<<2 | uint64(exponentSign)<<1 | uint64(significandSign)
	bytesEncoded, ok = uleb128.EncodeUint64(exponentField, dst)
	if !ok {
		return
	}
	offset := bytesEncoded
	bytesEncoded, ok = uleb128.Encode(&value.Coeff, dst[offset:])
	if !ok {
		return
	}
	bytesEncoded += offset
	return
}

// Encode an iee754 binary floating point value, with the specified number of significant digits.
// Rounding is half-to-even, meaning it rounds towards an even number when exactly halfway.
// If significantDigits is less than 1, no rounding takes place.
func EncodeFloat64(value float64, significantDigits int, dst []byte) (bytesEncoded int, ok bool) {
	if math.Float64bits(value) == math.Float64bits(0) {
		return encodeZero(dst)
	} else if value == math.Copysign(0, -1) {
		return encodeNegativeZero(dst)
	} else if math.IsInf(value, 1) {
		return encodeInfinity(dst)
	} else if math.IsInf(value, -1) {
		return encodeNegativeInfinity(dst)
	} else if math.IsNaN(value) {
		bits := math.Float64bits(value)
		if bits&quietBit != 0 {
			return encodeQuietNan(dst)
		}
		return encodeSignalingNan(dst)
	}

	exponent, significand := extractFloat(value, significantDigits)

	exponentSign := (exponent >> 31) & 1
	significandSign := (significand >> 63) & 1

	exponentField := uint64(abs32(int32(exponent)))
	exponentField <<= 1
	exponentField |= uint64(exponentSign)
	exponentField <<= 1
	exponentField |= uint64(significandSign)

	significandField := uint64(abs64(significand))

	bytesEncoded, ok = uleb128.EncodeUint64(exponentField, dst)
	if !ok {
		return
	}
	offset := bytesEncoded
	bytesEncoded, ok = uleb128.EncodeUint64(significandField, dst[offset:])
	if !ok {
		return
	}
	bytesEncoded += offset
	return
}

// Decode a float
func Decode(src []byte) (value *apd.Decimal, bytesDecoded int, err error) {
	switch len(src) {
	case 0:
		err = ErrorIncomplete
		return
	case 1:
		switch src[0] {
		case 2:
			value = &apd.Decimal{}
			bytesDecoded = 1
			return
		case 3:
			value = &apd.Decimal{Negative: true}
			bytesDecoded = 1
			return
		}
	case 2:
		if src[1] != 0 {
			break
		}
		switch src[0] {
		case 0x80:
			value = &apd.Decimal{Form: apd.NaN}
			bytesDecoded = 2
			return
		case 0x81:
			value = &apd.Decimal{Form: apd.NaNSignaling}
			bytesDecoded = 2
			return
		case 0x82:
			value = &apd.Decimal{Form: apd.Infinite}
			bytesDecoded = 2
			return
		case 0x83:
			value = &apd.Decimal{Form: apd.Infinite, Negative: true}
			bytesDecoded = 2
			return
		}
	}

	asUint, asBig, bytesDecoded, ok := uleb128.Decode(0, 0, src)
	if !ok {
		err = ErrorIncomplete
	}
	ok = false
	if asBig != nil {
		err = fmt.Errorf("Exponent %v is too big", asBig)
		return
	}
	// apd stores the exponent in a signed 32-bit int
	maxEncodedExponent := uint64(0x1ffffffff)
	if asUint > maxEncodedExponent {
		err = fmt.Errorf("Exponent %v is too big", asUint)
		return
	}
	negativeFlag := asUint&1 != 0
	exponent := int32(asUint >> 2)
	if asUint&2 != 0 {
		exponent = -exponent
	}

	offset := bytesDecoded
	asUint, asBig, bytesDecoded, ok = uleb128.Decode(0, 0, src[offset:])
	if !ok {
		err = ErrorIncomplete
		return
	}
	ok = false
	bytesDecoded += offset

	if asBig != nil {
		value = apd.NewWithBigInt(asBig, exponent)
		value.Negative = negativeFlag
		return
	}

	if asUint&0x8000000000000000 != 0 {
		if is32Bit() {
			value = &apd.Decimal{
				Negative: negativeFlag,
				Exponent: exponent,
			}
			value.Coeff.SetBits([]big.Word{big.Word(asUint), big.Word(asUint >> 32)})
		} else {
			value = &apd.Decimal{
				Negative: negativeFlag,
				Exponent: exponent,
			}
			value.Coeff.SetBits([]big.Word{big.Word(asUint)})
		}
		return
	}

	coeff := int64(asUint)
	if negativeFlag {
		coeff = -coeff
	}
	value = apd.New(coeff, exponent)
	return
}

// Maximum byte length that this library can encode (64-bit float)
const MaxEncodeLength = 10

const quietBit = 1 << 51

var digitsMax = [...]uint64{
	0,
	9,
	99,
	999,
	9999,
	99999,
	999999,
	9999999,
	99999999,
	999999999,
	9999999999,
	99999999999,
	999999999999,
	9999999999999,
	99999999999999,
	999999999999999,
	9999999999999999,
	99999999999999999,
	999999999999999999,
	9999999999999999999, // 19 digits
	// Max digits for uint64 is 20
}

func countDigits(value uint64) int {
	// This is MUCH faster than the string method, and 4x faster than int(math.Log10(float64(value))) + 1
	// Subdividing any further yields no performance gains.
	if value <= digitsMax[10] {
		for i := 1; i < 10; i++ {
			if value <= digitsMax[i] {
				return i
			}
		}
		return 10
	}

	for i := 11; i < 20; i++ {
		if value <= digitsMax[i] {
			return i
		}
	}
	return 20
}

func abs32(value int32) int32 {
	mask := value >> 31
	return (value + mask) ^ mask
}

func abs64(value int64) int64 {
	mask := value >> 63
	return (value + mask) ^ mask
}

func encodeSpecialValue(dst []byte, value byte) (bytesEncoded int, ok bool) {
	if len(dst) < 1 {
		return 1, false
	}
	dst[0] = value
	return 1, true
}

func encodeExtendedSpecialValue(dst []byte, value byte) (bytesEncoded int, ok bool) {
	if len(dst) < 2 {
		return 2, false
	}
	dst[0] = 0x80 | value
	dst[1] = 0
	return 2, true
}

func encodeQuietNan(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 0)
}

func encodeSignalingNan(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 1)
}

func encodeInfinity(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 2)
}

func encodeNegativeInfinity(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 3)
}

func encodeZero(dst []byte) (bytesEncoded int, ok bool) {
	return encodeSpecialValue(dst, 2)
}

func encodeNegativeZero(dst []byte) (bytesEncoded int, ok bool) {
	return encodeSpecialValue(dst, 3)
}

func extractFloat(value float64, significantDigits int) (exponent int, significand int64) {
	// Assuming no inf or nan
	stringRep := fmt.Sprintf("%v", value)
	// (-)d+(.d+)(e[+-]d+)
	encounteredDot := false
	encounteredExp := false
	isRounding := false
	significandSign := int64(1)
	exponentFromString := 0
	exponentSign := 1
	startIndex := 0

	if stringRep[0] == '-' {
		significandSign = -1
		startIndex++
	}

	digitCount := 0
	rounded := int64(0)
	roundedDivider := 1
	lastSigDigit := 0
	for i := startIndex; i < len(stringRep); i++ {
		ch := stringRep[i]
		switch ch {
		case '.':
			encounteredDot = true
			continue
		case 'e':
			encounteredExp = true
			continue
		case '-':
			exponentSign = -1
			continue
		case '+':
			exponentSign = 1
			continue
		}

		nextDigit := int(ch - '0')

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
		significand = significand*10 + int64(nextDigit)
		if encounteredDot {
			exponent--
		}
	}
	exponent += exponentFromString * exponentSign
	significand = significand * significandSign
	fractional := float64(rounded) / float64(roundedDivider)
	if fractional != 0 {
		if fractional > 0.5 {
			significand++
		} else if fractional < 0.5 {
			significand--
		} else if lastSigDigit&1 == 1 {
			significand++
		} else {
			significand--
		}
	}

	return exponent, significand
}

func is32Bit() bool {
	return ^uint(0) == 0xffffffff
}
