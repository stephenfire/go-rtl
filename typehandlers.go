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
		DefaultEventHandler
	}
	bigintPtrhandler struct {
		DefaultEventHandler
	}
	bigratHandler struct {
		addressHandler
		DefaultEventHandler
	}
	bigratPtrHandler struct {
		DefaultEventHandler
	}
	bigfloatHandler struct {
		addressHandler
		DefaultEventHandler
	}
	bigfloatPtrHandler struct {
		DefaultEventHandler
	}
	binaryUnmarshalerHandler struct {
		addressHandler
		DefaultEventHandler
	}
	binaryUnmarshalerPtrHandler struct {
		DefaultEventHandler
	}
)

func init() {
	_systemTypeHandler(bigintHandler{}, typeOfBigInt)
	_systemTypeHandler(bigintPtrhandler{}, reflect.PtrTo(typeOfBigInt))
	_systemTypeHandler(bigratHandler{}, typeOfBigRat)
	_systemTypeHandler(bigratPtrHandler{}, reflect.PtrTo(typeOfBigRat))
	_systemTypeHandler(bigfloatHandler{}, typeOfBigFloat)
	_systemTypeHandler(bigfloatPtrHandler{}, reflect.PtrTo(typeOfBigFloat))
	_systemTypeHandler(binaryUnmarshalerHandler{}, typeOfTime)
	_systemTypeHandler(binaryUnmarshalerPtrHandler{}, reflect.PtrTo(typeOfTime))
}

func (addressHandler) _replace(ctx *HandleContext, value reflect.Value) error {
	return ctx.ReplaceStack(value.Addr())
}

func (b bigintHandler) Byte(ctx *HandleContext, value reflect.Value, _ byte) error {
	return b._replace(ctx, value)
}

func (bigintHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (b bigintHandler) Number(ctx *HandleContext, value reflect.Value, _ bool, _ []byte) error {
	return b._replace(ctx, value)
}

func (bigintPtrhandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	i := getOrNewBigInt(value)
	i.SetInt64(int64(input))
	return ctx.PopState()
}

func (bigintPtrhandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (bigintPtrhandler) Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error {
	i := getOrNewBigInt(value)
	i.SetBytes(inputs)
	if !isPositive {
		i.Neg(i)
	}
	return ctx.PopState()
}

func (bigratHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (b bigratHandler) Number(ctx *HandleContext, value reflect.Value, _ bool, _ []byte) error {
	return b._replace(ctx, value)
}

func (bigratPtrHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (bigratPtrHandler) Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error {
	if !isPositive {
		return ErrUnsupported
	}
	r := getOrNewBigRat(value)
	if err := r.GobDecode(inputs); err != nil {
		return err
	}
	return ctx.PopState()
}

func (bigfloatHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (b bigfloatHandler) Number(ctx *HandleContext, value reflect.Value, _ bool, _ []byte) error {
	return b._replace(ctx, value)
}

func (bigfloatPtrHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (bigfloatPtrHandler) Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error {
	if !isPositive {
		return ErrUnsupported
	}
	f := getOrNewBigFloat(value)
	if err := f.GobDecode(inputs); err != nil {
		return err
	}
	return ctx.PopState()
}

func (b binaryUnmarshalerHandler) Byte(ctx *HandleContext, value reflect.Value, _ byte) error {
	return b._replace(ctx, value)
}

func (b binaryUnmarshalerHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (b binaryUnmarshalerHandler) Bytes(ctx *HandleContext, value reflect.Value, _ []byte) error {
	return b._replace(ctx, value)
}

func (binaryUnmarshalerPtrHandler) Byte(ctx *HandleContext, value reflect.Value, input byte) error {
	if err := setToBinaryUnmarshaler(value, []byte{input}); err != nil {
		return err
	}
	return ctx.PopState()
}

func (binaryUnmarshalerPtrHandler) Zero(ctx *HandleContext, value reflect.Value) error {
	value.Set(reflect.Zero(value.Type()))
	return ctx.PopState()
}

func (binaryUnmarshalerPtrHandler) Bytes(ctx *HandleContext, value reflect.Value, input []byte) error {
	if err := setToBinaryUnmarshaler(value, input); err != nil {
		return err
	}
	return ctx.PopState()
}
