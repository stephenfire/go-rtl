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
	bigintHandler struct {
		DefaultHeaderHandler
	}
	bigratHandler struct {
		DefaultHeaderHandler
	}
	bigfloatHandler struct {
		DefaultHeaderHandler
	}
)

func init() {
	_systemTypeHandler(bigintHandler{}, typeOfBigInt, reflect.PtrTo(typeOfBigInt))
	_systemTypeHandler(bigratHandler{}, typeOfBigRat, reflect.PtrTo(typeOfBigRat))
	_systemTypeHandler(bigfloatHandler{}, typeOfBigFloat, reflect.PtrTo(typeOfBigFloat))
}

func (bigintHandler) Byte(value reflect.Value, input byte) (*Todo, error) {
	i := getOrNewBigInt(value)
	i.SetInt64(int64(input))
	return popTodo.Clone(), nil
}

func (bigintHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (bigintHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	i := getOrNewBigInt(value)
	i.SetBytes(inputs)
	if !isPositive {
		i.Neg(i)
	}
	return popTodo.Clone(), nil
}

func (bigratHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (bigratHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	if !isPositive {
		return nil, ErrUnsupported
	}
	r := getOrNewBigRat(value)
	if err := r.GobDecode(inputs); err != nil {
		return nil, err
	}
	return popTodo.Clone(), nil
}

func (bigfloatHandler) Zero(value reflect.Value) (*Todo, error) {
	value.Set(reflect.Zero(value.Type()))
	return popTodo.Clone(), nil
}

func (bigfloatHandler) Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error) {
	if !isPositive {
		return nil, ErrUnsupported
	}
	f := getOrNewBigFloat(value)
	if err := f.GobDecode(inputs); err != nil {
		return nil, err
	}
	return popTodo.Clone(), nil
}
