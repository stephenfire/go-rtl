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

import "reflect"

type (
	addressHandler struct{}
	bigintHandler  struct {
		addressHandler
		DefaultHeaderHandler
	}
	bigintPtrhandler struct {
		DefaultHeaderHandler
	}
	bigratHandler struct {
		addressHandler
		DefaultHeaderHandler
	}
	bigratPtrHandler struct {
		DefaultHeaderHandler
	}
	bigfloatHandler struct {
		addressHandler
		DefaultHeaderHandler
	}
	bigfloatPtrHandler struct {
		DefaultHeaderHandler
	}
)

func init() {
	_systemTypeHandler(bigintHandler{}, typeOfBigInt)
	_systemTypeHandler(bigintPtrhandler{}, reflect.PtrTo(typeOfBigInt))
	_systemTypeHandler(bigratHandler{}, typeOfBigRat)
	_systemTypeHandler(bigratPtrHandler{}, reflect.PtrTo(typeOfBigRat))
	_systemTypeHandler(bigfloatHandler{}, typeOfBigFloat)
	_systemTypeHandler(bigfloatPtrHandler{}, reflect.PtrTo(typeOfBigFloat))
}

func (addressHandler) _replace(value reflect.Value) (*Todo, error) {
	return _newTodo().SetReplace(value.Addr()), nil
	// return &Todo{
	// 	StackTodo: StackReplaceTop,
	// 	Val:       value.Addr(),
	// }, nil
}

func (b bigintHandler) Byte(value reflect.Value, _ byte) (*Todo, error) {
	return b._replace(value)
}

func (bigintHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return _popTodo(), nil
}

func (b bigintHandler) Number(value reflect.Value, _ bool, _ []byte) (*Todo, error) {
	return b._replace(value)
}

func (bigintPtrhandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	i := getOrNewBigInt(value)
	i.SetInt64(int64(input))
	return _popTodo(), nil
}

func (bigintPtrhandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return _popTodo(), nil
}

func (bigintPtrhandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	i := getOrNewBigInt(value)
	i.SetBytes(inputs)
	if !isPositive {
		i.Neg(i)
	}
	return _popTodo(), nil
}

func (bigratHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return _popTodo(), nil
}

func (b bigratHandler) Number(value reflect.Value, _ bool, _ []byte) (*Todo, error) {
	return b._replace(value)
}

func (bigratPtrHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return _popTodo(), nil
}

func (bigratPtrHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	if !isPositive {
		return nil, ErrUnsupported
	}
	r := getOrNewBigRat(value)
	if err := r.GobDecode(inputs); err != nil {
		return nil, err
	}
	return _popTodo(), nil
}

func (bigfloatHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return _popTodo(), nil
}

func (b bigfloatHandler) Number(value reflect.Value, _ bool, _ []byte) (*Todo, error) {
	return b._replace(value)
}

func (bigfloatPtrHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return _popTodo(), nil
}

func (bigfloatPtrHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	if !isPositive {
		return nil, ErrUnsupported
	}
	f := getOrNewBigFloat(value)
	if err := f.GobDecode(inputs); err != nil {
		return nil, err
	}
	return _popTodo(), nil
}
