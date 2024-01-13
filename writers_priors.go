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
		typeOfBigInt:   bigIntWriter,
		typeOfBigRat:   gobEncoderNumberWriter,
		typeOfBigFloat: gobEncoderNumberWriter,
		typeOfTime:     binaryMarshalerBytesWriter,
	}

	_priorStructOrder = []reflect.Type{
		typeOfBigInt,
		typeOfBigRat,
		typeOfBigFloat,
		typeOfTime,
	}
)

func checkPriorStructsWriter(w io.Writer, v reflect.Value) (matched bool, n int, err error) {
	typ := v.Type()
	for _, prior := range _priorStructOrder {
		if typ.AssignableTo(prior) {
			fn, exist := _priorStructWriters[prior]
			if exist {
				n, err = fn(w, v)
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

func bigIntWriter(w io.Writer, v reflect.Value) (int, error) {
	bi := v.Interface().(big.Int)

	if !(bi.Sign() < 0) && bi.Cmp(bigint128) < 0 {
		// 0 < bi <128, use one single byte value
		return w.Write([]byte{byte(bi.Uint64())})
	}

	// big int
	negative, b := Numeric.BigIntToBytes(&bi)
	return _writeNumberBytes(w, negative, b)
}

func gobEncoderNumberWriter(w io.Writer, v reflect.Value) (int, error) {
	br := v.Addr().Interface().(gob.GobEncoder)
	b, err := br.GobEncode()
	if err != nil {
		return 0, err
	}
	return _writeNumberBytes(w, false, b)
}

func binaryMarshalerBytesWriter(w io.Writer, v reflect.Value) (int, error) {
	bm := v.Interface().(encoding.BinaryMarshaler)
	b, err := bm.MarshalBinary()
	if err != nil {
		return 0, err
	}
	return bytesWriter(w, b)
}
