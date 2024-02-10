/*
 * Copyright 2024 Stephen Guo (stephen.fire@gmail.com)
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
	"encoding"
	"encoding/gob"
	"io"
	"math/big"
	"reflect"
)

var (
	_priorStructWriters = map[reflect.Type]writerFunc{
		typeOfBigIntPtr:   bigIntPtrWriter,
		typeOfBigInt:      bigIntWriter,
		typeOfBigRatPtr:   bigRatPtrWriter,
		typeOfBigFloatPtr: bigFloatPtrWriter,
		typeOfTime:        timeWriter,
	}

	_writerPriorStructOrder = []reflect.Type{
		typeOfBigIntPtr,
		typeOfBigInt,
		typeOfBigRatPtr,
		typeOfBigFloatPtr,
		typeOfTime,
	}
)

func ConvertibleTo(src, dest reflect.Type) bool {
	if src.Kind() == dest.Kind() && src.ConvertibleTo(dest) {
		if dest.Kind() == reflect.Ptr {
			return src.Elem().Kind() == dest.Elem().Kind()
		}
		return true
	}
	return false
}

func Convertible(src, dest reflect.Type) bool {
	return src.Kind() == dest.Kind() && src.ConvertibleTo(dest)
}

func ConvertiblePtr(src, dest reflect.Type) bool {
	return src.Kind() == reflect.Ptr && dest.Kind() == reflect.Ptr &&
		src.Elem().Kind() == dest.Elem().Kind() && src.ConvertibleTo(dest)
}

func checkPriorStructsWriter(w io.Writer, v reflect.Value) (matched bool, n int, err error) {
	typ := v.Type()
	for _, prior := range _writerPriorStructOrder {
		if typ.AssignableTo(prior) {
			fn, exist := _priorStructWriters[prior]
			if exist {
				n, err = fn(w, v)
				return true, n, err
			}
		} else if ConvertibleTo(typ, prior) {
			fn, exist := _priorStructWriters[prior]
			if exist {
				priorVal := v.Convert(prior)
				n, err = fn(w, priorVal)
				return true, n, err
			}
		}
	}
	return false, 0, nil
}

func _writeNumberBytes(w io.Writer, isNegative bool, bs []byte) (int, error) {
	h, err := HeadMaker.numeric(isNegative, len(bs))
	if err != nil {
		return 0, err
	}
	n, err := w.Write(h)
	if err != nil {
		return n, err
	}
	nn, err := w.Write(bs)
	return n + nn, err
}

func writeBigInt(w io.Writer, bi *big.Int) (int, error) {
	if bi == nil {
		return w.Write(zeroValues)
	}
	if !(bi.Sign() < 0) && bi.Cmp(bigint128) < 0 {
		// 0 < bi <128, use one single byte value
		return w.Write([]byte{byte(bi.Uint64())})
	}

	// big int
	negative, b := Numeric.BigIntToBytes(bi)
	return _writeNumberBytes(w, negative, b)
}

func bigIntPtrWriter(w io.Writer, v reflect.Value) (int, error) {
	if v.IsNil() {
		return w.Write(zeroValues)
	}
	var bi *big.Int
	if v.Type().AssignableTo(typeOfBigIntPtr) {
		bi = v.Interface().(*big.Int)
	} else if v.Type().ConvertibleTo(typeOfBigIntPtr) {
		bi = v.Convert(typeOfBigIntPtr).Interface().(*big.Int)
	}
	return writeBigInt(w, bi)
}

func bigIntWriter(w io.Writer, v reflect.Value) (int, error) {
	var bi *big.Int
	if v.Type().AssignableTo(typeOfBigInt) {
		bbi := v.Interface().(big.Int)
		bi = &bbi
	} else if v.Type().ConvertibleTo(typeOfBigInt) {
		bbi := v.Convert(typeOfBigInt).Interface().(big.Int)
		bi = &bbi
	}
	return writeBigInt(w, bi)
}

func bigRatPtrWriter(w io.Writer, v reflect.Value) (int, error) {
	if v.IsNil() {
		return w.Write(zeroValues)
	}
	return gobEncoderNumberWriter(w, v)
}

func bigFloatPtrWriter(w io.Writer, v reflect.Value) (int, error) {
	if v.IsNil() {
		return w.Write(zeroValues)
	}
	return gobEncoderNumberWriter(w, v)
}

func gobEncoderNumberWriter(w io.Writer, v reflect.Value) (int, error) {
	br := v.Interface().(gob.GobEncoder)
	b, err := br.GobEncode()
	if err != nil {
		return 0, err
	}
	return _writeNumberBytes(w, false, b)
}

func timeWriter(w io.Writer, v reflect.Value) (int, error) {
	return binaryMarshalerBytesWriter(w, v)
}

func binaryMarshalerBytesWriter(w io.Writer, v reflect.Value) (int, error) {
	bm := v.Interface().(encoding.BinaryMarshaler)
	b, err := bm.MarshalBinary()
	if err != nil {
		return 0, err
	}
	return bytesWriter(w, b)
}
