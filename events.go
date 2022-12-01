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
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
)

type handleState struct {
	val reflect.Value
	typ reflect.Type
	// record type header match the value, THInvalid means not initialized
	th     TypeHeader
	length int
	buf    []byte
	// NestedHandler with the current nested kind val object parsing state, created when val is
	// pushed to the stack for the first time, and collected when it is popped from the stack
	handler NestedHandler
}

func (s *handleState) String() string {
	if s == nil {
		return "<nil>"
	}
	if !s.val.IsValid() {
		return "<NA>"
	}
	typ := s.val.Type()
	kind := typ.Kind()
	if s.handler != nil {
		return fmt.Sprintf("{%s(%s) TH:%s, %s}", typ.Name(), kind, s.th, s.handler)
	} else {
		return fmt.Sprintf("{%s(%s) TH:%s}", typ.Name(), kind, s.th)
	}
}

func (s *handleState) info() string {
	if s == nil {
		return "<nil>"
	}
	if !s.val.IsValid() {
		return "<NA>"
	}
	hstr := ""
	if s.handler != nil {
		hstr = fmt.Sprintf(", %s", s.handler.String())
	}
	return fmt.Sprintf("{%s(%s)-%s%s}", s.typ.Name(), s.typ.Kind(), s.th, hstr)
}

func (s *handleState) isValid() bool {
	return s != nil && s.val.IsValid() && s.typ != nil
}

func (s *handleState) updateValue(val reflect.Value) error {
	if s.handler != nil {
		return errors.New("invalid updating state value when state in handling")
	}
	s.val = val
	s.typ = val.Type()
	return nil
}

type HandleContext struct {
	// input reader
	vr ValueReader
	// top of stack (last handleState) is the current processing value
	stack       []*handleState
	stackPool   sync.Pool
	nestedPools map[reflect.Type]*sync.Pool
	// counter     map[string]int
}

// func (ctx *HandleContext) _count(name string) {
// 	c := ctx.counter[name]
// 	ctx.counter[name] = c + 1
// }

func NewHandleContext(r io.Reader) *HandleContext {
	vr, ok := r.(ValueReader)
	if !ok {
		vr = NewValueReader(r)
	}
	ctx := &HandleContext{
		vr:    vr,
		stack: nil,
		stackPool: sync.Pool{New: func() interface{} {
			return &handleState{}
		}},
		// counter: make(map[string]int),
	}
	return ctx
}

func (ctx *HandleContext) String() string {
	if ctx == nil {
		return "Ctx<nil>"
	}
	if len(ctx.stack) == 0 {
		// return fmt.Sprintf("Ctx{%v}", ctx.counter)
		return "Ctx{}"
	}
	return fmt.Sprintf("Ctx{Stack:%d Top:%s}", len(ctx.stack), ctx.stack[len(ctx.stack)-1])
}

func (ctx *HandleContext) StackInfo() string {
	if ctx == nil {
		return "<nil>"
	}
	if len(ctx.stack) == 0 {
		return "[]"
	}
	buf := new(bytes.Buffer)
	buf.WriteByte('[')
	for i, state := range ctx.stack {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(state.info())
	}
	buf.WriteByte(']')
	return buf.String()
}

func (ctx *HandleContext) top() (*handleState, bool) {
	if ctx == nil || len(ctx.stack) == 0 {
		return nil, false
	}
	return ctx.stack[len(ctx.stack)-1], true
}

func (ctx *HandleContext) PopState() error {
	if len(ctx.stack) == 0 {
		return ErrEmptyStack
	}
	last := ctx.stack[len(ctx.stack)-1]
	ctx.stack = ctx.stack[:len(ctx.stack)-1]

	if last.handler != nil {
		typ := reflect.TypeOf(last.handler)
		nestedPool, exist := ctx.nestedPools[typ]
		if exist && nestedPool != nil {
			nestedPool.Put(last.handler)
		}
		last.handler = nil
	}
	ctx.stackPool.Put(last)
	// ctx._count("pop")
	return nil
}

func (ctx *HandleContext) PushState(val reflect.Value, th TypeHeader, length int, buf []byte, handler NestedHandler) error {
	if !val.IsValid() {
		return ErrInvalidValue
	}
	state := ctx.stackPool.Get().(*handleState)
	state.val = val
	state.typ = val.Type()
	state.th = th
	state.length = length
	state.buf = buf
	state.handler = handler
	ctx.stack = append(ctx.stack, state)
	// ctx._count("push")
	return nil
}

func (ctx *HandleContext) ReplaceStack(val reflect.Value) error {
	if len(ctx.stack) == 0 {
		return ErrEmptyStack
	}
	// ctx._count("replace")
	return ctx.stack[len(ctx.stack)-1].updateValue(val)
}

func (ctx *HandleContext) NestedStack(handler NestedHandler) error {
	if len(ctx.stack) == 0 {
		return ErrEmptyStack
	}
	if ctx.stack[len(ctx.stack)-1].handler != nil {
		return errors.New("already has nested handler")
	}
	ctx.stack[len(ctx.stack)-1].handler = handler
	// ctx._count("nested")
	return nil
}

func (ctx *HandleContext) SkipReader(length int) error {
	for i := 0; i < length; i++ {
		if _, err := ctx.vr.Skip(); err != nil {
			return fmt.Errorf("reader skipping %d/%d failed: %v", i, length, err)
		}
	}
	// ctx._count("skip")
	return nil
}

func (ctx *HandleContext) NewNested(typ reflect.Type) interface{} {
	if ctx.nestedPools == nil {
		ctx.nestedPools = make(map[reflect.Type]*sync.Pool)
	}
	pool, exist := ctx.nestedPools[typ]
	if !exist || pool == nil {
		pool = &sync.Pool{New: func() interface{} {
			val := reflect.New(typ)
			// ctx._count(fmt.Sprintf("new(%s)", typ.Name()))
			return val.Interface()
		}}
		ctx.nestedPools[typ] = pool
	}
	return pool.Get()
}

type (
	EventHandler interface {
		Byte(ctx *HandleContext, value reflect.Value, input byte) error
		Zero(ctx *HandleContext, value reflect.Value) error
		True(ctx *HandleContext, value reflect.Value) error
		Empty(ctx *HandleContext, value reflect.Value) error
		Array(ctx *HandleContext, value reflect.Value, length int) error
		Number(ctx *HandleContext, value reflect.Value, isPositive bool, inputs []byte) error
		Bytes(ctx *HandleContext, value reflect.Value, inputs []byte) error
		Version(ctx *HandleContext, value reflect.Value, inputs ...byte) error
	}

	DefaultEventHandler struct{}
)

func (h DefaultEventHandler) Byte(_ *HandleContext, _ reflect.Value, _ byte) error {
	return ErrUnsupported
}
func (h DefaultEventHandler) Zero(_ *HandleContext, _ reflect.Value) error  { return ErrUnsupported }
func (h DefaultEventHandler) True(_ *HandleContext, _ reflect.Value) error  { return ErrUnsupported }
func (h DefaultEventHandler) Empty(_ *HandleContext, _ reflect.Value) error { return ErrUnsupported }
func (h DefaultEventHandler) Array(_ *HandleContext, _ reflect.Value, _ int) error {
	return ErrUnsupported
}
func (h DefaultEventHandler) Number(_ *HandleContext, _ reflect.Value, _ bool, _ []byte) error {
	return ErrUnsupported
}
func (h DefaultEventHandler) Bytes(_ *HandleContext, _ reflect.Value, _ []byte) error {
	return ErrUnsupported
}
func (h DefaultEventHandler) Version(_ *HandleContext, _ reflect.Value, _ ...byte) error {
	return ErrUnsupported
}

type (
	NestedHandler interface {
		String() string
		Element(ctx *HandleContext) error
		Index() int
	}
)

var (
	_systemKindHandlers = make(map[reflect.Kind]EventHandler)
	_systemTypeHandlers = make(map[reflect.Type]EventHandler)
)

func _systemKindHandler(handler EventHandler, kinds ...reflect.Kind) {
	for _, k := range kinds {
		_systemKindHandlers[k] = handler
	}
}

func _systemTypeHandler(handler EventHandler, typs ...reflect.Type) {
	for _, typ := range typs {
		_systemTypeHandlers[typ] = handler
	}
}

type EventDecoder struct {
	kindHandlers map[reflect.Kind]EventHandler
	typeHandlers map[reflect.Type]EventHandler
}

func (e *EventDecoder) runDecoder(r io.Reader, value reflect.Value, typ reflect.Type) (bool, error) {
	if typ.Implements(TypeOfDecoder) {
		newCreate := false
		if typ.Kind() == reflect.Ptr && value.IsNil() {
			nvalue := reflect.New(typ.Elem())
			value.Set(nvalue)
			newCreate = true
		}
		decoder, _ := value.Interface().(Decoder)
		shouldBeNil, err := decoder.Deserialization(r)
		if err != nil {
			return true, err
		}
		if newCreate && shouldBeNil {
			value.Set(reflect.Zero(typ))
		}
		return true, nil
	} else {
		return false, nil
	}
}

func (e *EventDecoder) checkTypeOfDecoder(r io.Reader, value reflect.Value) (bool, error) {
	typ := value.Type()
	isDecoder, err := e.runDecoder(r, value, typ)
	if isDecoder || err != nil {
		return isDecoder, err
	}
	if typ.Kind() == reflect.Ptr {
		etyp := typ.Elem()
		elem := value.Elem()
		return e.runDecoder(r, elem, etyp)
	}
	return false, nil
}

func (e *EventDecoder) _getTypeHandler(typ reflect.Type) (EventHandler, error) {
	if len(e.typeHandlers) > 0 {
		handler, exist := e.typeHandlers[typ]
		if exist && handler != nil {
			return handler, nil
		}
	}
	handler, exist := _systemTypeHandlers[typ]
	if exist && handler != nil {
		return handler, nil
	}
	return nil, nil
}

func (e *EventDecoder) _getKindHandler(kind reflect.Kind) (EventHandler, error) {
	if len(e.kindHandlers) > 0 {
		handler, exist := e.kindHandlers[kind]
		if exist && handler != nil {
			return handler, nil
		}
	}
	handler, exist := _systemKindHandlers[kind]
	if exist && handler != nil {
		return handler, nil
	}
	return nil, nil
}

func (e *EventDecoder) handle(ctx *HandleContext) error {
	var err error
	for {
		state, exist := ctx.top()
		if !exist {
			return nil
		}
		if !state.isValid() {
			if err = ctx.PopState(); err != nil {
				return err
			}
			continue
		}

		// var todo *Todo
		if state.handler != nil {
			err = state.handler.Element(ctx)
			if err != nil {
				return fmt.Errorf("rtl: element(%d) handle failed: %v, at %s",
					state.handler.Index(), err, ctx.StackInfo())
			}
		} else {
			if !state.th.IsValid() {
				isDecoder, err := e.checkTypeOfDecoder(ctx.vr, state.val)
				if err != nil {
					return err
				}
				if isDecoder {
					if err = ctx.PopState(); err != nil {
						return err
					}
					continue
				}

				th, length, err := ctx.vr.ReadFullHeader()
				if err != nil {
					return fmt.Errorf("rtl: read header failed: %v, at %s", err, ctx.StackInfo())
				}
				state.th = th
				state.length = length

				if th.FollowedByBytes() {
					buf, err := ctx.vr.ReadBytes(state.length, nil)
					if err != nil {
						return fmt.Errorf("rtl: read value failed: %v, at %s", err, ctx.StackInfo())
					}
					state.buf = buf
				}
			}

			handler, err := e._getTypeHandler(state.typ)
			if err != nil {
				return fmt.Errorf("rtl: get handler for type %s failed: %v", state.typ.Name(), err)
			}
			if handler == nil {
				handler, err = e._getKindHandler(state.typ.Kind())
				if err != nil || handler == nil {
					return fmt.Errorf("rtl: get handler for type:%s, kind:%s failed: %v",
						state.typ.Name(), state.typ.Kind(), err)
				}
			}

			switch state.th {
			case THSingleByte:
				err = handler.Byte(ctx, state.val, byte(state.length))
			case THZeroValue:
				err = handler.Zero(ctx, state.val)
			case THTrue:
				err = handler.True(ctx, state.val)
			case THEmpty:
				err = handler.Empty(ctx, state.val)
			case THArraySingle, THArrayMulti:
				err = handler.Array(ctx, state.val, state.length)
			case THPosNumSingle, THNegNumSingle, THPosBigInt, THNegBigInt:
				err = handler.Number(ctx, state.val, state.th == THPosNumSingle || state.th == THPosBigInt, state.buf)
			case THStringSingle, THStringMulti:
				err = handler.Bytes(ctx, state.val, state.buf)
			case THVersion:
				err = handler.Version(ctx, state.val, byte(state.length))
			case THVersionSingle:
				err = handler.Version(ctx, state.val, state.buf...)
			}

			if err != nil {
				return fmt.Errorf("rtl: header handle failed: %v, at %s", err, ctx.StackInfo())
			}
		}
	}
}

func (e *EventDecoder) Decode(r io.Reader, obj interface{}) error {
	if obj == nil {
		return ErrDecodeIntoNil
	}

	rv := reflect.ValueOf(obj)
	if rv.Kind() != reflect.Ptr {
		return ErrDecodeNoPtr
	}
	if rv.IsNil() {
		return ErrDecodeIntoNil
	}

	// Check if obj is a Decoder, and if so, return directly. Because the pointer will be detached
	// later, resulting in a change in the nature of its interface.
	isDecoder, err := e.checkTypeOfDecoder(r, rv)
	if isDecoder || err != nil {
		return err
	}

	rev := rv.Elem()
	if !rev.CanSet() {
		return errors.New("obj cannot set")
	}
	rtyp := rev.Type()
	rkind := rtyp.Kind()
	if !canBeDecodeTo(rkind) {
		return fmt.Errorf("unsupported decoding to %v (kind: %s)", rtyp, rkind.String())
	}

	vr, ok := r.(ValueReader)
	if !ok {
		vr = NewValueReader(r)
	}

	ctx := NewHandleContext(vr)
	if err := ctx.PushState(rev, THInvalid, 0, nil, nil); err != nil {
		return err
	}
	err = e.handle(ctx)
	// log.Printf("%s", ctx)
	return err
}
