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

	"github.com/cockroachdb/apd/v2"
	"github.com/kstenerud/go-uleb128"
)

var ErrorIncomplete = fmt.Errorf("Compact float value is incomplete")

// Maximum number of bytes a DFloat (or float64) can occupy while encoded
func MaxEncodeLength() int {
	// (64 bits / 7) + (33 bits / 7)
	return 10 + 5
}

// Maximum number of bytes a particular apd.Decimal can occupy while encoded.
// This is an estimate; it may be smaller, but never bigger.
func MaxEncodeLengthBig(value *apd.Decimal) int {
	if is32Bit() {
		return len(value.Coeff.Bits())*32/7 + 1 + 5
	}
	return len(value.Coeff.Bits())*64/7 + 1 + 5
}

func Encode(value DFloat, dst []byte) (bytesEncoded int, ok bool) {
	if value.IsZero() {
		if value.IsNegativeZero() {
			return EncodeNegativeZero(dst)
		}
		return EncodeZero(dst)
	}
	if value.IsSpecial() {
		switch value.Coefficient {
		case CoeffInfinity:
			return EncodeInfinity(dst)
		case CoeffNegativeInfinity:
			return EncodeNegativeInfinity(dst)
		case CoeffNan:
			return EncodeQuietNan(dst)
		case CoeffSignalingNan:
			return EncodeSignalingNan(dst)
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
			return EncodeNegativeZero(dst)
		}
		return EncodeZero(dst)
	}
	switch value.Form {
	case apd.Infinite:
		if value.Negative {
			return EncodeNegativeInfinity(dst)
		}
		return EncodeInfinity(dst)
	case apd.NaN:
		return EncodeQuietNan(dst)
	case apd.NaNSignaling:
		return EncodeSignalingNan(dst)
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

func EncodeQuietNan(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 0)
}

func EncodeSignalingNan(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 1)
}

func EncodeInfinity(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 2)
}

func EncodeNegativeInfinity(dst []byte) (bytesEncoded int, ok bool) {
	return encodeExtendedSpecialValue(dst, 3)
}

func EncodeZero(dst []byte) (bytesEncoded int, ok bool) {
	return encodeSpecialValue(dst, 2)
}

func EncodeNegativeZero(dst []byte) (bytesEncoded int, ok bool) {
	return encodeSpecialValue(dst, 3)
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

func is32Bit() bool {
	return ^uint(0) == 0xffffffff
}
