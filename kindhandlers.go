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
		DefaultEventHandler
	}
	uintHandler struct {
		DefaultEventHandler
	}
	floatHandler struct {
		DefaultEventHandler
	}
	boolHandler struct {
		DefaultEventHandler
	}
	stringHandler struct {
		DefaultEventHandler
	}
	mapHandler struct {
		DefaultEventHandler
	}
	structHandler struct {
		DefaultEventHandler
	}
	arrayHandler struct {
		DefaultEventHandler
	}
	sliceHandler struct {
		DefaultEventHandler
	}
	pointerHandler   struct{}
	interfaceHandler struct {
		DefaultEventHandler
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

func (intHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	value.SetInt(int64(input))
	return ctx.PopState()
}

func (intHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.SetInt(0)
	return ctx.PopState()
}

func (intHandler) Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error {
	if len(inputs) > 8 {
		return errors.New("too many bytes for int64")
	}
	i := Numeric.BytesToInt64(inputs, !isPositive)
	value.SetInt(i)
	return ctx.PopState()
}

func (uintHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	value.SetUint(uint64(input))
	return ctx.PopState()
}

func (uintHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.SetUint(0)
	return ctx.PopState()
}

func (uintHandler) Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error {
	if !isPositive {
		return errors.New("negative not supported for uint64")
	}
	if len(inputs) > 8 {
		return errors.New("too many bytes for uint64")
	}
	i := Numeric.BytesToUint64(inputs)
	value.SetUint(i)
	return ctx.PopState()
}

func (floatHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	value.SetFloat(Numeric.ByteToFloat64(input, false))
	return ctx.PopState()
}

func (floatHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.SetFloat(0)
	return ctx.PopState()
}

func (floatHandler) Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error {
	if len(inputs) > 8 {
		return errors.New("too many bytes for float")
	}
	var f float64
	if len(inputs) == 4 {
		f = float64(Numeric.BytesToFloat32(inputs, !isPositive))
	} else {
		f = Numeric.BytesToFloat64(inputs, !isPositive)
	}
	value.SetFloat(f)
	return ctx.PopState()
}

func (boolHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.SetBool(false)
	return ctx.PopState()
}

func (boolHandler) True(ctx *HandleContext, value reflect.Value) error {
	value.SetBool(true)
	return ctx.PopState()
}

func (stringHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	value.SetString(string([]byte{input}))
	return ctx.PopState()
}

func (stringHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (stringHandler) Bytes(ctx *HandleContext, value reflect.Value, inputs []byte) error {
	value.SetString(string(inputs))
	return ctx.PopState()
}

func (mapHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (mapHandler) Empty(ctx *HandleContext, value reflect.Value) error {
	nmap := reflect.MakeMapWithSize(value.Type(), 0)
	value.Set(nmap)
	return ctx.PopState()
}

func (mapHandler) Array(ctx *HandleContext, value reflect.Value, length int) error {
	nested, err := newMapElement(value, length)
	if err != nil {
		return fmt.Errorf("new map nested handler failed: %v", err)
	}
	return ctx.NestedStack(nested)
}

func (structHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (structHandler) Array(ctx *HandleContext, value reflect.Value, length int) error {
	nested, err := newStructElement(value, length)
	if err != nil {
		return fmt.Errorf("new struct nested handler failed: %v", err)
	}
	return ctx.NestedStack(nested)
}

func (a arrayHandler) _bytes(ctx *HandleContext, value reflect.Value, inputs ...byte) error {
	etyp := value.Type().Elem()
	if etyp == typeOfByte {
		reflect.Copy(value, reflect.ValueOf(inputs))
		return ctx.PopState()
	} else {
		nested, err := newString2ArraySlice(value, inputs)
		if err != nil {
			return fmt.Errorf("new string 2 array nested handler failed: %v", err)
		}
		return ctx.NestedStack(nested)
	}
}

func (a arrayHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	return a._bytes(ctx, value, input)
}

func (a arrayHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (a arrayHandler) Bytes(ctx *HandleContext, value reflect.Value, inputs []byte) error {
	return a._bytes(ctx, value, inputs...)
}

func (a arrayHandler) Array(ctx *HandleContext, value reflect.Value, length int) error {
	nested, err := newArrayElement(value, length)
	if err != nil {
		return fmt.Errorf("new array nested handler failed: %v", err)
	}
	return ctx.NestedStack(nested)
}

func (s sliceHandler) _bytes(ctx *HandleContext, value reflect.Value, inputs ...byte) error {
	checkSlice0(len(inputs), value)
	etyp := value.Type().Elem()
	if etyp == typeOfByte {
		reflect.Copy(value, reflect.ValueOf(inputs))
		return ctx.PopState()
	} else {
		nested, err := newString2ArraySlice(value, inputs)
		if err != nil {
			return fmt.Errorf("new string 2 slice nested handler failed: %v", err)
		}
		return ctx.NestedStack(nested)
	}
}

func (s sliceHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	return s._bytes(ctx, value, input)
}

func (s sliceHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (s sliceHandler) Empty(ctx *HandleContext, value reflect.Value) error {
	if value.CanSet() {
		nslice := reflect.MakeSlice(value.Type(), 0, 0)
		value.Set(nslice)
	}
	return ctx.PopState()
}

func (s sliceHandler) Bytes(ctx *HandleContext, value reflect.Value, inputs []byte) error {
	return s._bytes(ctx, value, inputs...)
}

func (s sliceHandler) Array(ctx *HandleContext, value reflect.Value, length int) error {
	nested, err := newSliceElement(value, length)
	if err != nil {
		return fmt.Errorf("new slice nested handler failed: %v", err)
	}
	return ctx.NestedStack(nested)
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

func (p pointerHandler) _handle(ctx *HandleContext, value reflect.Value) error {
	if evalue, err := p._element(value); err != nil {
		return err
	} else {
		return ctx.ReplaceStack(evalue)
	}
}

func (p pointerHandler) Byte(ctx *HandleContext, value reflect.Value, _ byte) error {
	return p._handle(ctx, value)
}

func (p pointerHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	// return p._handle(value)
	if !value.IsNil() {
		value.Set(reflect.Zero(value.Type()))
	}
	return ctx.PopState()
}

func (p pointerHandler) True(ctx *HandleContext, value reflect.Value) error {
	return p._handle(ctx, value)
}

func (p pointerHandler) Empty(ctx *HandleContext, value reflect.Value) error {
	return p._handle(ctx, value)
}

func (p pointerHandler) Array(ctx *HandleContext, value reflect.Value, _ int) error {
	return p._handle(ctx, value)
}

func (p pointerHandler) Number(ctx *HandleContext, value reflect.Value, _ bool, _ []byte) error {
	return p._handle(ctx, value)
}

func (p pointerHandler) Bytes(ctx *HandleContext, value reflect.Value, _ []byte) error {
	return p._handle(ctx, value)
}

func (p pointerHandler) Version(ctx *HandleContext, value reflect.Value, _ ...byte) error {
	return p._handle(ctx, value)
}

func (interfaceHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	nv := reflect.New(typeOfUint64).Elem()
	nv.SetUint(uint64(input))
	value.Set(nv)
	return ctx.PopState()
}

func (interfaceHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(typeOfInterface))
	return ctx.PopState()
}

func (interfaceHandler) Empty(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.MakeSlice(typeOfInterfaceSlice, 0, 0))
	return ctx.PopState()
}

func (interfaceHandler) Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error {
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
	return ctx.PopState()
}

func (interfaceHandler) Bytes(ctx *HandleContext, value reflect.Value, inputs []byte) error {
	nv := reflect.New(typeOfString).Elem()
	nv.SetString(string(inputs))
	value.Set(nv)
	return ctx.PopState()
}

func (interfaceHandler) Array(ctx *HandleContext, value reflect.Value, size int) error {
	slice := reflect.MakeSlice(typeOfInterfaceSlice, size, size)
	value.Set(slice)
	return ctx.ReplaceStack(slice)
}
