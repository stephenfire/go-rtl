/*
 * Copyright 2022 Stephen Guo (stephen.fire@gmail.com)
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
	"errors"
	"fmt"
	"math/big"
	"reflect"
)

type (
	intHandler struct {
		DefaultHeaderHandler
	}
	uintHandler struct {
		DefaultHeaderHandler
	}
	floatHandler struct {
		DefaultHeaderHandler
	}
	boolHandler struct {
		DefaultHeaderHandler
	}
	stringHandler struct {
		DefaultHeaderHandler
	}
	mapHandler struct {
		DefaultHeaderHandler
	}
	structHandler struct {
		DefaultHeaderHandler
	}
	arrayHandler struct {
		DefaultHeaderHandler
	}
	sliceHandler struct {
		DefaultHeaderHandler
	}
	pointerHandler   struct{}
	interfaceHandler struct {
		DefaultHeaderHandler
	}
)

func init() {
	_systemKindHandler(intHandler{}, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64)
	_systemKindHandler(uintHandler{}, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64)
	_systemKindHandler(floatHandler{}, reflect.Float32, reflect.Float64)
	_systemKindHandler(boolHandler{}, reflect.Bool)
	_systemKindHandler(stringHandler{}, reflect.String)
	_systemKindHandler(mapHandler{}, reflect.Map)
	_systemKindHandler(structHandler{}, reflect.Struct)
	_systemKindHandler(arrayHandler{}, reflect.Array)
	_systemKindHandler(sliceHandler{}, reflect.Slice)
	_systemKindHandler(pointerHandler{}, reflect.Ptr)
	_systemKindHandler(interfaceHandler{}, reflect.Interface)
}

func (intHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	value.SetInt(int64(input))
	return popTodo.Clone(), nil
}

func (intHandler) Zero(value reflect.Value) (*Todo, error) {
	value.SetInt(0)
	return popTodo.Clone(), nil
}

func (intHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	if len(inputs) > 8 {
		return nil, errors.New("too many bytes for int64")
	}
	i := Numeric.BytesToInt64(inputs, !isPositive)
	value.SetInt(i)
	return popTodo.Clone(), nil
}

func (uintHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	value.SetUint(uint64(input))
	return popTodo.Clone(), nil
}

func (uintHandler) Zero(value reflect.Value) (*Todo, error) {
	value.SetUint(0)
	return popTodo.Clone(), nil
}

func (uintHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	if !isPositive {
		return nil, errors.New("negative not supported for uint64")
	}
	if len(inputs) > 8 {
		return nil, errors.New("too many bytes for uint64")
	}
	i := Numeric.BytesToUint64(inputs)
	value.SetUint(i)
	return popTodo.Clone(), nil
}

func (floatHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	value.SetFloat(Numeric.ByteToFloat64(input, false))
	return popTodo.Clone(), nil
}

func (floatHandler) Zero(value reflect.Value) (*Todo, error) {
	value.SetFloat(0)
	return popTodo.Clone(), nil
}

func (floatHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	if len(inputs) > 8 {
		return nil, errors.New("too many bytes for float")
	}
	var f float64
	if len(inputs) == 4 {
		f = float64(Numeric.BytesToFloat32(inputs, !isPositive))
	} else {
		f = Numeric.BytesToFloat64(inputs, !isPositive)
	}
	value.SetFloat(f)
	return popTodo.Clone(), nil
}

func (boolHandler) Zero(value reflect.Value) (*Todo, error) {
	value.SetBool(false)
	return popTodo.Clone(), nil
}

func (boolHandler) True(value reflect.Value) (*Todo, error) {
	value.SetBool(true)
	return popTodo.Clone(), nil
}

func (stringHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	value.SetString(string([]byte{input}))
	return popTodo.Clone(), nil
}

func (stringHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (stringHandler) Bytes(value reflect.Value, inputs []byte) (*Todo, error) {
	value.SetString(string(inputs))
	return popTodo.Clone(), nil
}

func (mapHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (mapHandler) Empty(value reflect.Value) (*Todo, error) {
	nmap := reflect.MakeMapWithSize(value.Type(), 0)
	value.Set(nmap)
	return popTodo.Clone(), nil
}

func (mapHandler) Array(value reflect.Value, length int) (*Todo, error) {
	nested, err := newMapElement(value, length)
	if err != nil {
		return nil, fmt.Errorf("new map nested handler failed: %v", err)
	}
	return &Todo{
		StackTodo: StackNested,
		Nested:    nested,
	}, nil
}

func (structHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (structHandler) Array(value reflect.Value, length int) (*Todo, error) {
	nested, err := newStructElement(value, length)
	if err != nil {
		return nil, fmt.Errorf("new struct nested handler failed: %v", err)
	}
	return &Todo{
		StackTodo: StackNested,
		Nested:    nested,
	}, nil
}

func (a arrayHandler) _bytes(value reflect.Value, inputs ...byte) (*Todo, error) {
	etyp := value.Type().Elem()
	if etyp == typeOfByte {
		reflect.Copy(value, reflect.ValueOf(inputs))
		return popTodo.Clone(), nil
	} else {
		nested, err := newString2ArraySlice(value, inputs)
		if err != nil {
			return nil, fmt.Errorf("new string 2 array nested handler failed: %v", err)
		}
		return &Todo{
			StackTodo: StackNested,
			Nested:    nested,
		}, nil
	}
}

func (a arrayHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	return a._bytes(value, input)
}

func (a arrayHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (a arrayHandler) Bytes(value reflect.Value, inputs []byte) (*Todo, error) {
	return a._bytes(value, inputs...)
}

func (a arrayHandler) Array(value reflect.Value, length int) (*Todo, error) {
	nested, err := newArrayElement(value, length)
	if err != nil {
		return nil, fmt.Errorf("new array nested handler failed: %v", err)
	}
	return &Todo{
		StackTodo: StackNested,
		Nested:    nested,
	}, nil
}

func (s sliceHandler) _bytes(value reflect.Value, inputs ...byte) (*Todo, error) {
	checkSlice0(len(inputs), value)
	etyp := value.Type().Elem()
	if etyp == typeOfByte {
		reflect.Copy(value, reflect.ValueOf(inputs))
		return popTodo.Clone(), nil
	} else {
		nested, err := newString2ArraySlice(value, inputs)
		if err != nil {
			return nil, fmt.Errorf("new string 2 slice nested handler failed: %v", err)
		}
		return &Todo{
			StackTodo: StackNested,
			Nested:    nested,
		}, nil
	}
}

func (s sliceHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	return s._bytes(value, input)
}

func (s sliceHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (s sliceHandler) Empty(value reflect.Value) (*Todo, error) {
	if value.CanSet() {
		nslice := reflect.MakeSlice(value.Type(), 0, 0)
		value.Set(nslice)
	}
	return popTodo.Clone(), nil
}

func (s sliceHandler) Bytes(value reflect.Value, inputs []byte) (*Todo, error) {
	return s._bytes(value, inputs...)
}

func (s sliceHandler) Array(value reflect.Value, length int) (*Todo, error) {
	nested, err := newSliceElement(value, length)
	if err != nil {
		return nil, fmt.Errorf("new slice nested handler failed: %v", err)
	}
	return &Todo{
		StackTodo: StackNested,
		Nested:    nested,
	}, nil
}

func (p pointerHandler) _element(value reflect.Value) (reflect.Value, error) {
	etyp := value.Type().Elem()
	ekind := etyp.Kind()
	if !canBeDecodeTo(ekind) {
		return reflect.Value{}, fmt.Errorf("unsupported pointer to %v (kind: %s) "+
			"for decoding", etyp, ekind.String())
	}
	// create if nil
	evalue := value
	if value.IsNil() {
		if !value.CanSet() {
			return reflect.Value{}, fmt.Errorf("cannot create new value %s", etyp.Name())
		}
		evalue = reflect.New(etyp)
		value.Set(evalue)
	}
	return evalue.Elem(), nil
}

func (p pointerHandler) _handle(value reflect.Value) (*Todo, error) {
	if evalue, err := p._element(value); err != nil {
		return nil, err
	} else {
		return &Todo{
			StackTodo: StackReplaceTop,
			Val:       evalue,
		}, nil
	}
}

func (p pointerHandler) Byte(value reflect.Value, _ byte) (*Todo, error) {
	return p._handle(value)
}

func (p pointerHandler) Zero(value reflect.Value) (*Todo, error) {
	// return p._handle(value)
	if !value.IsNil() {
		value.Set(reflect.Zero(value.Type()))
	}
	return popTodo.Clone(), nil
}

func (p pointerHandler) True(value reflect.Value) (*Todo, error) {
	return p._handle(value)
}

func (p pointerHandler) Empty(value reflect.Value) (*Todo, error) {
	return p._handle(value)
}

func (p pointerHandler) Array(value reflect.Value, _ int) (*Todo, error) {
	return p._handle(value)
}

func (p pointerHandler) Number(value reflect.Value, _ bool, _ []byte) (*Todo, error) {
	return p._handle(value)
}

func (p pointerHandler) Bytes(value reflect.Value, _ []byte) (*Todo, error) {
	return p._handle(value)
}

func (p pointerHandler) Version(value reflect.Value, _ ...byte) (*Todo, error) {
	return p._handle(value)
}

func (interfaceHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	nv := reflect.New(typeOfUint64).Elem()
	nv.SetUint(uint64(input))
	value.Set(nv)
	return popTodo.Clone(), nil
}

func (interfaceHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(typeOfInterface))
	return popTodo.Clone(), nil
}

func (interfaceHandler) Empty(value reflect.Value) (*Todo, error) {
	value.Set(reflect.MakeSlice(typeOfInterfaceSlice, 0, 0))
	return popTodo.Clone(), nil
}

func (interfaceHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	l := len(inputs)
	if l <= 8 {
		if !isPositive {
			i := Numeric.BytesToInt64(inputs, !isPositive)
			nv := reflect.New(typeOfInt64).Elem()
			nv.SetInt(i)
			value.Set(nv)
		} else {
			u := Numeric.BytesToUint64(inputs)
			nv := reflect.New(typeOfUint64).Elem()
			nv.SetUint(u)
			value.Set(nv)
		}
	} else {
		i := new(big.Int).SetBytes(inputs)
		if !isPositive {
			i.Neg(i)
		}
		value.Set(reflect.ValueOf(i))
	}
	return popTodo.Clone(), nil
}

func (interfaceHandler) Bytes(value reflect.Value, inputs []byte) (*Todo, error) {
	nv := reflect.New(typeOfString).Elem()
	nv.SetString(string(inputs))
	value.Set(nv)
	return popTodo.Clone(), nil
}

func (interfaceHandler) Array(value reflect.Value, size int) (*Todo, error) {
	slice := reflect.MakeSlice(typeOfInterfaceSlice, size, size)
	value.Set(slice)
	return &Todo{
		StackTodo: StackReplaceTop,
		Val:       slice,
	}, nil
}
