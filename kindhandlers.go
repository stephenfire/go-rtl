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
	pointerHandler struct{}
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

func (arrayHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	if value.Len() >= 1 {
		evalue := value.Index(0)
		return &Todo{
			StackTodo: StackPush,
			Val:       evalue,
			Th:        THSingleByte,
			Length:    int(input),
		}, nil
	} else {
		return nil, nil
	}
}

func (arrayHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (arrayHandler) Bytes(value reflect.Value, inputs []byte) (*Todo, error) {
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

func (arrayHandler) Array(value reflect.Value, length int) (*Todo, error) {
	nested, err := newArrayElement(value, length)
	if err != nil {
		return nil, fmt.Errorf("new array nested handler failed: %v", err)
	}
	return &Todo{
		StackTodo: StackNested,
		Nested:    nested,
	}, nil
}

func (sliceHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	checkSlice0(1, value)
	evalue := value.Index(0)
	return &Todo{
		StackTodo: StackPush,
		Val:       evalue,
		Th:        THSingleByte,
		Length:    int(input),
	}, nil
}

func (sliceHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (sliceHandler) Empty(value reflect.Value) (*Todo, error) {
	if value.CanSet() {
		nslice := reflect.MakeSlice(value.Type(), 0, 0)
		value.Set(nslice)
	}
	return popTodo.Clone(), nil
}

func (sliceHandler) Bytes(value reflect.Value, inputs []byte) (*Todo, error) {
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

func (sliceHandler) Array(value reflect.Value, length int) (*Todo, error) {
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
