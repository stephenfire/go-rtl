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
	"fmt"
	"reflect"
)

type headerValueReader func(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error

var (
	_priorStructReaders = map[reflect.Type]headerValueReader{
		typeOfBigInt:   bigIntReader0,
		typeOfBigRat:   bigRatReader0,
		typeOfBigFloat: bigFloatReader0,
	}
)

func checkPriorStructsReader(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) (matched bool, err error) {
	typ := value.Type()
	for _, prior := range _priorStructOrder {
		if typ.AssignableTo(prior) || typ.AssignableTo(reflect.PtrTo(prior)) {
			fn, exist := _priorStructReaders[prior]
			if exist {
				err = fn(th, length, vr, value, nesting)
				return true, err
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

func bigRatReader0(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error {
	typ := value.Type()

	// big.Rat
	if typ.AssignableTo(typeOfBigRat) {
		f := getFunc(typ, bigRatReaders, th)
		return f(length, vr, value.Addr(), nesting)
	}
	// *big.Rat
	if typ.AssignableTo(reflect.PtrTo(typeOfBigRat)) {
		f := getFunc(typ, bigRatReaders, th)
		return f(length, vr, value, nesting)
	}

	return fmt.Errorf("rtl: should be big.Rat or *big.Rat, but %s", typ.Name())
}

func bigFloatReader0(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error {
	typ := value.Type()

	// big.Float
	if typ.AssignableTo(typeOfBigFloat) {
		f := getFunc(typ, bigFloatReaders, th)
		return f(length, vr, value.Addr(), nesting)
	}
	// *big.Float
	if typ.AssignableTo(reflect.PtrTo(typeOfBigFloat)) {
		f := getFunc(typ, bigFloatReaders, th)
		return f(length, vr, value, nesting)
	}

	return fmt.Errorf("rtl: should be big.Float or *big.Float, but %s", typ.Name())
}
