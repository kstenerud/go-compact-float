package compact_float

import (
	"fmt"
	"math/big"

	"github.com/cockroachdb/apd"
	"github.com/kstenerud/go-uleb128"
)

var ErrorIncomplete = fmt.Errorf("Compact float value is incomplete")

func Encode(value DFloat, dst []byte) (bytesEncoded int, ok bool) {
	if value.IsZero() {
		if value.IsNegativeZero() {
			return encodeNegativeZero(dst)
		}
		return encodeZero(dst)
	}
	if value.IsSpecial() {
		switch value.Coefficient {
		case CoeffInfinity:
			return encodeInfinity(dst)
		case CoeffNegativeInfinity:
			return encodeNegativeInfinity(dst)
		case CoeffNan:
			return encodeQuietNan(dst)
		case CoeffSignalingNan:
			return encodeSignalingNan(dst)
		default:
			panic(fmt.Errorf("%v: Illegal special coefficient", value.Coefficient))
		}
	}

	exponent := value.Exponent
	exponentSign := 0
	if exponent < 0 {
		exponent = -exponent
		exponentSign = 2
	}
	coefficient := value.Coefficient
	coefficientSign := 0
	if coefficient < 0 {
		coefficient = -coefficient
		coefficientSign = 1
	}
	exponentField := uint64(exponent)<<2 | uint64(exponentSign) | uint64(coefficientSign)
	bytesEncoded, ok = uleb128.EncodeUint64(exponentField, dst)
	if !ok {
		return
	}
	offset := bytesEncoded
	bytesEncoded, ok = uleb128.EncodeUint64(uint64(coefficient), dst[offset:])
	if !ok {
		return
	}
	bytesEncoded += offset
	return
}

func EncodeBig(value *apd.Decimal, dst []byte) (bytesEncoded int, ok bool) {
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

// Decode a float.
// bigValue will be nil unless the decoded value is too big to fit into a DFloat.
func Decode(src []byte) (value DFloat, bigValue *apd.Decimal, bytesDecoded int, err error) {
	switch len(src) {
	case 0:
		err = ErrorIncomplete
		return
	case 1:
		switch src[0] {
		case 2:
			value = dfloatZero.Clone()
			bytesDecoded = 1
			return
		case 3:
			value = dfloatNegativeZero.Clone()
			bytesDecoded = 1
			return
		default:
			err = ErrorIncomplete
			return
		}
	case 2:
		if src[1] != 0 {
			break
		}
		switch src[0] {
		case 0x80:
			value = dfloatNaN.Clone()
			bytesDecoded = 2
			return
		case 0x81:
			value = dfloatSignalingNaN.Clone()
			bytesDecoded = 2
			return
		case 0x82:
			value = dfloatInfinity.Clone()
			bytesDecoded = 2
			return
		case 0x83:
			value = dfloatNegativeInfinity.Clone()
			bytesDecoded = 2
			return
		default:
			err = ErrorIncomplete
			return
		}
	}

	asUint, asBig, bytesDecoded, ok := uleb128.Decode(0, 0, src)
	if !ok {
		err = ErrorIncomplete
		return
	}
	ok = false
	if asBig != nil {
		err = fmt.Errorf("Exponent %v is too big", asBig)
		return
	}
	maxEncodedExponent := uint64(0x1ffffffff)
	if asUint > maxEncodedExponent {
		err = fmt.Errorf("Exponent %v is too big", asUint)
		return
	}

	negMult := []int{1, -1}
	coeffMult := int64(negMult[asUint&1])
	expMult := int32(negMult[(asUint>>1)&1])

	exponent := int32(asUint>>2) * expMult

	offset := bytesDecoded
	asUint, asBig, bytesDecoded, ok = uleb128.Decode(0, 0, src[offset:])
	if !ok {
		err = ErrorIncomplete
		return
	}
	ok = false
	bytesDecoded += offset

	if asBig != nil {
		bigValue = apd.NewWithBigInt(asBig, exponent)
		bigValue.Negative = coeffMult < 0
		return
	}

	if asUint&0x8000000000000000 != 0 {
		if is32Bit() {
			bigValue = &apd.Decimal{
				Negative: coeffMult < 0,
				Exponent: exponent,
			}
			bigValue.Coeff.SetBits([]big.Word{big.Word(asUint), big.Word(asUint >> 32)})
		} else {
			bigValue = &apd.Decimal{
				Negative: coeffMult < 0,
				Exponent: exponent,
			}
			bigValue.Coeff.SetBits([]big.Word{big.Word(asUint)})
		}
		return
	}

	coefficient := int64(asUint) * coeffMult
	value = DFloat{
		Exponent:    exponent,
		Coefficient: coefficient,
	}
	return
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

func is32Bit() bool {
	return ^uint(0) == 0xffffffff
}
