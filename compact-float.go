package compact_float

import (
	"fmt"
	"math"

	"github.com/kstenerud/go-vlq"
)

// func requireBytes(byteCount int, buffer []byte) error {
// 	if len(buffer) < byteCount {
// 		return fmt.Errorf("%v free bytes required to encode, but only %v available", byteCount, len(buffer))
// 	}
// 	return nil
// }

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
	dst[0] = 0x80
	dst[1] = value
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

// Encode an iee754 binary floating point value, with the specified number of significant digits.
// Rounding is half-to-even, meaning it rounds towards an even number when exactly halfway.
// If significantDigits is <1 or >15, no rounding takes place.
func Encode(value float64, significantDigits int, dst []byte) (bytesEncoded int, ok bool) {

	if math.Float64bits(value) == math.Float64bits(0) {
		return encodeZero(dst)
	} else if value == math.Copysign(0, -1) {
		return encodeNegativeZero(dst)
	} else if math.IsInf(value, 1) {
		return encodeInfinity(dst)
	} else if math.IsInf(value, -1) {
		return encodeNegativeInfinity(dst)
	} else if math.IsNaN(value) {
		return encodeQuietNan(dst)
	}
	// TODO: Signaling NaN

	exponent, significand := extractFloat(value, significantDigits)

	exponentSign := (exponent >> 31) & 1
	significandSign := (significand >> 63) & 1

	exponentVlq := vlq.Rvlq(abs32(int32(exponent)))
	exponentVlq <<= 1
	exponentVlq |= vlq.Rvlq(exponentSign)
	exponentVlq <<= 1
	exponentVlq |= vlq.Rvlq(significandSign)

	significandVlq := vlq.Rvlq(abs64(significand))

	bytesEncoded, ok = exponentVlq.EncodeTo(dst)
	if !ok {
		return bytesEncoded, ok
	}
	offset := bytesEncoded

	bytesEncoded, ok = significandVlq.EncodeTo(dst[offset:])
	if !ok {
		return offset + bytesEncoded, ok
	}

	return offset + bytesEncoded, true
}

// Decode a float
func Decode(src []byte) (value float64, bytesDecoded int, ok bool) {
	var exponentVlq vlq.Rvlq
	var significand vlq.Rvlq
	var isComplete bool
	exponentVlq, bytesDecoded, isComplete = vlq.DecodeRvlqFrom(src)
	if !isComplete {
		return value, bytesDecoded, isComplete
	}

	if vlq.IsExtended(src) {
		switch exponentVlq {
		case 0:
			return math.NaN(), bytesDecoded, true
		case 1:
			// TODO: Signaling nan
			return math.NaN(), bytesDecoded, true
		case 2:
			return math.Inf(1), bytesDecoded, true
		case 3:
			return math.Inf(-1), bytesDecoded, true
		}
	}

	if exponentVlq == 2 {
		return 0, bytesDecoded, true
	}
	if exponentVlq == 3 {
		return math.Copysign(0, -1), bytesDecoded, true
	}

	offset := bytesDecoded
	significand, bytesDecoded, isComplete = vlq.DecodeRvlqFrom(src[offset:])
	if !isComplete {
		return value, offset + bytesDecoded, isComplete
	}

	significandSign := exponentVlq & 1
	exponentVlq >>= 1
	exponentSign := exponentVlq & 1
	exponentVlq >>= 1
	exponent := int32(exponentVlq)

	if exponentSign == 1 {
		exponent = -exponent
	}
	if significandSign == 1 {
		significand = -significand
	}

	floatString := fmt.Sprintf("%de%d", significand, exponent)
	_, err := fmt.Sscanf(floatString, "%f", &value)
	if err != nil {
		panic(fmt.Errorf("BUG: Failed to convert float string [%v]: %v", floatString, err))
	}

	return value, offset + bytesDecoded, true
}
