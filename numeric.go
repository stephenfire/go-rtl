/*
 * Copyright 2020 Stephen Guo (stephen.fire@gmail.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rtl

import (
	"encoding/binary"
	"io"
	"math"
	"math/big"
	"reflect"
)

type numeric struct{}

func (numeric) writeUint(b []byte, i uint64) (int, error) {
	switch {
	case i < (1 << 8):
		if len(b) < 1 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i)
		return 1, nil
	case i < (1 << 16):
		if len(b) < 2 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i >> 8)
		b[1] = byte(i)
		return 2, nil
	case i < (1 << 24):
		if len(b) < 3 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i >> 16)
		b[1] = byte(i >> 8)
		b[2] = byte(i)
		return 3, nil
	case i < (1 << 32):
		if len(b) < 4 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i >> 24)
		b[1] = byte(i >> 16)
		b[2] = byte(i >> 8)
		b[3] = byte(i)
		return 4, nil
	case i < (1 << 40):
		if len(b) < 5 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i >> 32)
		b[1] = byte(i >> 24)
		b[2] = byte(i >> 16)
		b[3] = byte(i >> 8)
		b[4] = byte(i)
		return 5, nil
	case i < (1 << 48):
		if len(b) < 6 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i >> 40)
		b[1] = byte(i >> 32)
		b[2] = byte(i >> 24)
		b[3] = byte(i >> 16)
		b[4] = byte(i >> 8)
		b[5] = byte(i)
		return 6, nil
	case i < (1 << 56):
		if len(b) < 7 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i >> 48)
		b[1] = byte(i >> 40)
		b[2] = byte(i >> 32)
		b[3] = byte(i >> 24)
		b[4] = byte(i >> 16)
		b[5] = byte(i >> 8)
		b[6] = byte(i)
		return 7, nil
	default:
		if len(b) < 8 {
			return 0, ErrInsufficientLength
		}
		b[0] = byte(i >> 56)
		b[1] = byte(i >> 48)
		b[2] = byte(i >> 40)
		b[3] = byte(i >> 32)
		b[4] = byte(i >> 24)
		b[5] = byte(i >> 16)
		b[6] = byte(i >> 8)
		b[7] = byte(i)
		return 8, nil
	}
}

func (n numeric) NegIntToMinBytes(i int64) []byte {
	if i >= 0 {
		return nil
	}
	b := make([]byte, 8, 8)
	b[0] = byte(i >> 56)
	b[1] = byte(i >> 48)
	b[2] = byte(i >> 40)
	b[3] = byte(i >> 32)
	b[4] = byte(i >> 24)
	b[5] = byte(i >> 16)
	b[6] = byte(i >> 8)
	b[7] = byte(i)
	for i := 0; i < 8; i++ {
		if b[i] != 0xFF {
			if (b[i] & byte(0x80)) == 0 {
				if i > 0 {
					return b[i-1:]
				}
				return b
			}
			return b[i:]
		}
	}
	return b
}

func (n numeric) UintToBytes(i uint64) []byte {
	if i <= 0 {
		return nil
	}

	r := make([]byte, 8)
	l, _ := n.writeUint(r, i)
	return r[0:l]
}

func (n numeric) IntToBytes(i int64) (isNegative bool, buf []byte) {
	isNegative = i < 0
	if isNegative {
		i = -i // not changed if i==MinInt64
	}
	buf = n.UintToBytes(uint64(i))
	return
}

func (n numeric) Float32ToBytes(f float32) (isNegative bool, buf []byte) {
	isNegative = f < 0
	if isNegative {
		f = -f
	}
	buf = n.UintToBytes(uint64(math.Float32bits(f)))
	return
}

func (n numeric) Float64ToBytes(d float64) (isNegative bool, buf []byte) {
	isNegative = d < 0
	if isNegative {
		d = -d
	}
	buf = n.UintToBytes(math.Float64bits(d))
	return
}

func (n numeric) BigIntToBytes(bi *big.Int) (isNegative bool, buf []byte) {
	cmp := bi.Sign()
	isNegative = cmp < 0
	buf = bi.Bytes()
	return
}

// bytesToUint use last n bytes in b to create an unsigned integer
func (numeric) bytesToUint(b []byte, n int) uint64 {
	var r uint64 = 0
	l := len(b)
	if l > 0 {
		s := l - n
		if s < 0 {
			s = 0
		}
		for i := s; i < l; i++ {
			r <<= 8
			r += uint64(b[i])
		}
	}
	return r
}

func (n numeric) BytesToUint64(b []byte) uint64 {
	return n.bytesToUint(b, 8)
}

func (n numeric) BytesToUint32(b []byte) uint32 {
	return uint32(n.bytesToUint(b, 4))
}

func (n numeric) BytesToUint16(b []byte) uint16 {
	return uint16(n.bytesToUint(b, 2))
}

func (numeric) BytesToUint8(b []byte) uint8 {
	if b == nil || len(b) == 0 {
		return 0
	}
	return uint8(b[len(b)-1])
}

func (n numeric) BytesToIntB(b []byte) (int, bool) {
	if len(b) == 0 {
		return 0, false
	}
	bytesize := (32 << (^uint(0) >> 63)) >> 3
	if len(b) > bytesize {
		return 0, true
	}
	var r uint = 0
	for i := 0; i < len(b); i++ {
		r <<= 8
		r += uint(b[i])
	}
	return int(r), false
}

// Deprecated use BytesToIntB instead
func (n numeric) BytesToInt(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	var r int = 0
	for i := 0; i < len(b); i++ {
		r <<= 8
		r += int(b[i])
	}
	return r
}

// if overflowed return (0, true), or return int64 value and false
func (n numeric) BytesToInt64B(b []byte, isNegative bool) (int64, bool) {
	r := n.bytesToUint(b, 8)
	if !isNegative && r > math.MaxInt64 {
		return 0, true
	}
	i := int64(r)
	if isNegative {
		i = -i
	}
	return i, false
}

func (n numeric) BytesToInt64(b []byte, isNegative bool) int64 {
	r := int64(n.bytesToUint(b, 8))
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) BytesToInt32B(b []byte, isNegative bool) (int32, bool) {
	r := n.bytesToUint(b, 4)
	if !isNegative && r > math.MaxInt32 {
		return 0, true
	}
	i := int32(r)
	if isNegative {
		i = -i
	}
	return i, false
}

// Deprecated use BytesToInt32B instead
func (n numeric) BytesToInt32(b []byte, isNegative bool) int32 {
	r := int32(n.bytesToUint(b, 4))
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) BytesToInt16B(b []byte, isNegative bool) (int16, bool) {
	r := n.bytesToUint(b, 2)
	if !isNegative && r > math.MaxInt16 {
		return 0, true
	}
	i := int16(r)
	if isNegative {
		i = -i
	}
	return i, false
}

// Deprecated use BytesToInt16B instead
func (n numeric) BytesToInt16(b []byte, isNegative bool) int16 {
	r := int16(n.bytesToUint(b, 2))
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) BytesToInt8B(b []byte, isNegative bool) (int8, bool) {
	if len(b) == 0 {
		return 0, false
	}
	r := uint8(b[len(b)-1])
	if !isNegative && r > math.MaxInt8 {
		return 0, true
	}
	i := int8(r)
	if isNegative {
		i = -i
	}
	return i, false
}

// Deprecated use BytesToInt8B instead
func (n numeric) BytesToInt8(b []byte, isNegative bool) int8 {
	if b == nil || len(b) == 0 {
		return 0
	}
	r := int8(b[len(b)-1])
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) ByteToFloat32(b byte, isNegative bool) float32 {
	r := math.Float32frombits(uint32(b))
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) ByteToFloat64(b byte, isNegative bool) float64 {
	r := math.Float64frombits(uint64(b))
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) BytesToFloat32(b []byte, isNegative bool) float32 {
	r := math.Float32frombits(n.BytesToUint32(b))
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) BytesToFloat64(b []byte, isNegative bool) float64 {
	r := math.Float64frombits(n.BytesToUint64(b))
	if isNegative && r > 0 {
		r = -r
	}
	return r
}

func (n numeric) BytesFillBigInt(b []byte, isNegative bool, bi *big.Int) {
	if bi == nil {
		return
	}
	if b == nil || len(b) == 0 {
		bi.SetInt64(0)
		return
	}
	bi.SetBytes(b)
	if isNegative {
		bi.Neg(bi)
	}
}

func (n numeric) BytesToBigInt(b []byte, isNegative bool) *big.Int {
	r := new(big.Int)
	n.BytesFillBigInt(b, isNegative, r)
	return r
}

func ToBinaryBuffer(v interface{}, w io.Writer) error {
	b := ToBinaryBytes(v)
	_, err := w.Write(b)
	return err
}

// ToBinaryBytes
func ToBinaryBytes(v interface{}) (ret []byte) {
	if v == nil {
		return nil
	}
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Uint8:
		ret = make([]byte, 1)
		ret[0] = byte(vv.Uint())
	case reflect.Uint16:
		ret = make([]byte, 2)
		binary.BigEndian.PutUint16(ret, uint16(vv.Uint()))
	case reflect.Uint32:
		ret = make([]byte, 4)
		binary.BigEndian.PutUint32(ret, uint32(vv.Uint()))
	case reflect.Uint64:
		ret = make([]byte, 8)
		binary.BigEndian.PutUint64(ret, uint64(vv.Uint()))
	case reflect.Uint:
		ret = make([]byte, 8)
		binary.BigEndian.PutUint64(ret, vv.Uint())
	case reflect.Int8:
		ret = make([]byte, 1)
		ret[0] = byte(vv.Int())
	case reflect.Int16:
		ret = make([]byte, 2)
		binary.BigEndian.PutUint16(ret, uint16(vv.Int()))
	case reflect.Int32:
		ret = make([]byte, 4)
		binary.BigEndian.PutUint32(ret, uint32(vv.Int()))
	case reflect.Int64, reflect.Int:
		ret = make([]byte, 8)
		binary.BigEndian.PutUint64(ret, uint64(vv.Int()))
	default:
		break
	}
	return ret
}

func BinaryToUint(bs []byte) uint64 {
	switch len(bs) {
	case 1:
		return uint64(bs[0])
	case 2:
		return uint64(binary.BigEndian.Uint16(bs))
	case 4:
		return uint64(binary.BigEndian.Uint32(bs))
	case 8:
		return binary.BigEndian.Uint64(bs)
	default:
		return 0
	}
}

func BinaryToInt(bs []byte) int64 {
	return int64(BinaryToUint(bs))
}
