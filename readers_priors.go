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
	"errors"
	"fmt"
	"reflect"
)

type headerValueReader func(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error

var (
	_readerPriorStructOrder = []reflect.Type{
		typeOfBigInt,
		typeOfBigRat,
		typeOfBigFloat,
		typeOfTime,
	}

	_priorStructReaders = map[reflect.Type]map[TypeHeader]typeReaderFunc{
		typeOfBigInt:   bigIntReaders,
		typeOfBigRat:   bigRatReaders,
		typeOfBigFloat: bigFloatReaders,
		typeOfTime:     binaryUnmarshalerReaders,
	}

	binaryUnmarshalerReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return setToBinaryUnmarshaler(value, []byte{byte(length)})
		},
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.Set(reflect.Zero(value.Type()))
			return nil
		},
		THStringSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			buf, err := vr.ReadBytes(length, nil)
			if err != nil {
				return err
			}
			return setToBinaryUnmarshaler(value, buf)
		},
		THStringMulti: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			l, err := vr.ReadMultiLength(length)
			if err != nil {
				return err
			}
			buf, err := vr.ReadBytes(int(l), nil)
			if err != nil {
				return err
			}
			return setToBinaryUnmarshaler(value, buf)
		},
	}
)

func checkPriorStructsReader(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) (matched bool, err error) {
	typ := value.Type()
	for _, prior := range _readerPriorStructOrder {
		if typ.AssignableTo(prior) || typ.AssignableTo(reflect.PtrTo(prior)) {
			readers, exist := _priorStructReaders[prior]
			if exist {
				fn := getFunc(typ, readers, th)
				if typ.AssignableTo(prior) {
					err = fn(length, vr, value.Addr(), nesting)
				} else {
					err = fn(length, vr, value, nesting)
				}
				return true, err
			}
		} else if Convertible(typ, prior) {
			readers, exist := _priorStructReaders[prior]
			if exist {
				fn := getFunc(prior, readers, th)
				vptr := value.Addr()
				priorVal := vptr.Convert(reflect.PtrTo(prior))
				return true, fn(length, vr, priorVal, nesting)
			}
		} else if ConvertiblePtr(typ, reflect.PtrTo(prior)) {
			readers, exist := _priorStructReaders[prior]
			if exist {
				if th == THZeroValue {
					value.Set(reflect.Zero(typ))
					return true, nil
				}
				fn := getFunc(prior, readers, th)
				if value.IsNil() {
					value.Set(reflect.New(typ.Elem()))
				}
				priorVal := value.Convert(reflect.PtrTo(prior))
				return true, fn(length, vr, priorVal, nesting)
			}
		}
	}
	return false, nil
}

func bigIntReader0(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error {
	typ := value.Type()

	// big.Int
	if typ.AssignableTo(typeOfBigInt) {
		f := getFunc(typ, bigIntReaders, th)
		return f(length, vr, value.Addr(), nesting)
	}
	// *big.Int
	if typ.AssignableTo(reflect.PtrTo(typeOfBigInt)) {
		f := getFunc(typ, bigIntReaders, th)
		return f(length, vr, value, nesting)
	}

	return fmt.Errorf("rtl: should be big.Int or *big.Int, but %s", typ.Name())
}

// func bigRatReader0(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error {
// 	typ := value.Type()
//
// 	// big.Rat
// 	if typ.AssignableTo(typeOfBigRat) {
// 		f := getFunc(typ, bigRatReaders, th)
// 		return f(length, vr, value.Addr(), nesting)
// 	}
// 	// *big.Rat
// 	if typ.AssignableTo(reflect.PtrTo(typeOfBigRat)) {
// 		f := getFunc(typ, bigRatReaders, th)
// 		return f(length, vr, value, nesting)
// 	}
//
// 	return fmt.Errorf("rtl: should be big.Rat or *big.Rat, but %s", typ.Name())
// }
//
// func bigFloatReader0(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error {
// 	typ := value.Type()
//
// 	// big.Float
// 	if typ.AssignableTo(typeOfBigFloat) {
// 		f := getFunc(typ, bigFloatReaders, th)
// 		return f(length, vr, value.Addr(), nesting)
// 	}
// 	// *big.Float
// 	if typ.AssignableTo(reflect.PtrTo(typeOfBigFloat)) {
// 		f := getFunc(typ, bigFloatReaders, th)
// 		return f(length, vr, value, nesting)
// 	}
//
// 	return fmt.Errorf("rtl: should be big.Float or *big.Float, but %s", typ.Name())
// }

// value must be a pointer of a type, and implemented encoding.BinaryUnmarshaler
func setToBinaryUnmarshaler(value reflect.Value, bs []byte) error {
	if value.Kind() != reflect.Pointer {
		return errors.New("rtl: BinaryUnmarshaler need a pointer")
	}
	if value.IsNil() {
		value.Set(reflect.New(value.Type().Elem()))
	}
	bu := value.Interface().(encoding.BinaryUnmarshaler)
	return bu.UnmarshalBinary(bs)
}
