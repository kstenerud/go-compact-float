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
	"io"
	"math/big"

	"github.com/cockroachdb/apd/v2"
	"github.com/kstenerud/go-uleb128"
)

var ErrorIncomplete = fmt.Errorf("Compact float value is incomplete")

// Maximum number of bytes required to encode a DFloat.
func MaxEncodeLength() int {
	// (64 bits / 7) + (33 bits / 7)
	return 10 + 5
}

// Maximum number of bytes required to encode a particular apd.Decimal.
// This is an estimate; it may be smaller, but never bigger.
func MaxEncodeLengthBig(value *apd.Decimal) int {
	if is32Bit() {
		return len(value.Coeff.Bits())*32/7 + 1 + 5
	}
	return len(value.Coeff.Bits())*64/7 + 1 + 5
}

// Encodes a DFloat to a writer.
func Encode(value DFloat, writer io.Writer) (bytesEncoded int, err error) {
	buffer := make([]byte, MaxEncodeLength())
	bytesEncoded = EncodeToBytes(value, buffer)
	return writer.Write(buffer[:bytesEncoded])
}

// Encodes a DFloat to a byte buffer.
// Assumes the buffer is big enough (see MaxEncodeLength()).
func EncodeToBytes(value DFloat, buffer []byte) (bytesEncoded int) {
	if value.IsZero() {
		if value.IsNegativeZero() {
			return EncodeNegativeZero(buffer)
		}
		return EncodeZero(buffer)
	}
	if value.IsSpecial() {
		switch value.Coefficient {
		case CoeffInfinity:
			return EncodeInfinity(buffer)
		case CoeffNegativeInfinity:
			return EncodeNegativeInfinity(buffer)
		case CoeffNan:
			return EncodeQuietNan(buffer)
		case CoeffSignalingNan:
			return EncodeSignalingNan(buffer)
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
	bytesEncoded = uleb128.EncodeUint64ToBytes(exponentField, buffer)
	bytesEncoded += uleb128.EncodeUint64ToBytes(uint64(coefficient), buffer[bytesEncoded:])
	return
}

// Encodes an apd.Decimal to a writer.
func EncodeBig(value *apd.Decimal, writer io.Writer) (bytesEncoded int, err error) {
	buffer := make([]byte, MaxEncodeLengthBig(value))
	bytesEncoded = EncodeBigToBytes(value, buffer)
	return writer.Write(buffer[:bytesEncoded])
}

// Encodes an apt.Decimal to a buffer.
// Assumes the buffer is big enough (see MaxEncodeLengthBig()).
func EncodeBigToBytes(value *apd.Decimal, buffer []byte) (bytesEncoded int) {
	if value.IsZero() {
		if value.Negative {
			return EncodeNegativeZero(buffer)
		}
		return EncodeZero(buffer)
	}
	switch value.Form {
	case apd.Infinite:
		if value.Negative {
			return EncodeNegativeInfinity(buffer)
		}
		return EncodeInfinity(buffer)
	case apd.NaN:
		return EncodeQuietNan(buffer)
	case apd.NaNSignaling:
		return EncodeSignalingNan(buffer)
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
	bytesEncoded = uleb128.EncodeUint64ToBytes(exponentField, buffer)
	bytesEncoded += uleb128.EncodeToBytes(&value.Coeff, buffer[bytesEncoded:])
	return
}

// Encodes a quiet NaN, using 2 bytes.
func EncodeQuietNan(buffer []byte) (bytesEncoded int) {
	return encodeExtendedSpecialValue(0, buffer)
}

// Encodes a signaling NaN, using 2 bytes.
func EncodeSignalingNan(buffer []byte) (bytesEncoded int) {
	return encodeExtendedSpecialValue(1, buffer)
}

// Encodes positive infinity, using 2 bytes.
func EncodeInfinity(buffer []byte) (bytesEncoded int) {
	return encodeExtendedSpecialValue(2, buffer)
}

// Encodes negative infinity, using 2 bytes.
func EncodeNegativeInfinity(buffer []byte) (bytesEncoded int) {
	return encodeExtendedSpecialValue(3, buffer)
}

// Encodes positive zero, using 1 byte.
func EncodeZero(buffer []byte) (bytesEncoded int) {
	return encodeSpecialValue(2, buffer)
}

// Encodes negative zero, using 1 byte.
func EncodeNegativeZero(buffer []byte) (bytesEncoded int) {
	return encodeSpecialValue(3, buffer)
}

// Decode a float.
// bigValue will be nil unless the decoded value is too big to fit into a DFloat.
func Decode(reader io.Reader) (value DFloat, bigValue *apd.Decimal, bytesDecoded int, err error) {
	buffer := []byte{0}
	return DecodeWithByteBuffer(reader, buffer)
}

// Decode a float using the supplied single-byte buffer.
// bigValue will be nil unless the decoded value is too big to fit into a DFloat.
func DecodeWithByteBuffer(reader io.Reader, buffer []byte) (value DFloat, bigValue *apd.Decimal, bytesDecoded int, err error) {
	asUint, asBig, bytesDecoded, err := uleb128.DecodeWithByteBuffer(reader, buffer)
	if err != nil {
		return
	}
	if asBig != nil {
		err = fmt.Errorf("Exponent %v is too big", asBig)
		return
	}

	switch bytesDecoded {
	case 1:
		switch asUint {
		case 2:
			value = dfloatZero
			return
		case 3:
			value = dfloatNegativeZero
			return
		}
	case 2:
		switch asUint {
		case 0:
			value = dfloatNaN
			return
		case 1:
			value = dfloatSignalingNaN
			return
		case 2:
			value = dfloatInfinity
			return
		case 3:
			value = dfloatNegativeInfinity
			return
		}
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
	if asUint, asBig, bytesDecoded, err = uleb128.DecodeWithByteBuffer(reader, buffer); err != nil {
		return
	}
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

func encodeSpecialValue(value byte, buffer []byte) (bytesEncoded int) {
	buffer[0] = value
	return 1
}

func encodeExtendedSpecialValue(value byte, buffer []byte) (bytesEncoded int) {
	buffer[0] = value | 0x80
	buffer[1] = 0
	return 2
}

func is32Bit() bool {
	return ^uint(0) == 0xffffffff
}
