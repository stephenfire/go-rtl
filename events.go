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

func newState(val reflect.Value) (*handleState, error) {
	if !val.IsValid() {
		return nil, ErrInvalidValue
	}
	return &handleState{val: val, typ: val.Type(), th: THInvalid}, nil
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
		return fmt.Sprintf("{%s(%s) TH:%s HANDLING}", typ.Name(), kind, s.th)
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
	return fmt.Sprintf("{%s(%s)-%s}", s.typ.Name(), s.typ.Kind(), s.th)
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
	stack []*handleState
}

func (ctx *HandleContext) String() string {
	if ctx == nil {
		return "Ctx<nil>"
	}
	if len(ctx.stack) == 0 {
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

func (ctx *HandleContext) pop() (*handleState, bool) {
	if ctx == nil || len(ctx.stack) == 0 {
		return nil, false
	}
	last := ctx.stack[len(ctx.stack)-1]
	ctx.stack = ctx.stack[:len(ctx.stack)-1]
	return last, true
}

func (ctx *HandleContext) apply(todo *Todo) error {
	if err := todo.Validate(); err != nil {
		return err
	}
	switch todo.StackTodo {
	case StackNone:
		// do nothing
	case StackReplaceTop:
		if len(ctx.stack) == 0 {
			return ErrEmptyStack
		}
		if err := ctx.stack[len(ctx.stack)-1].updateValue(todo.Val); err != nil {
			return err
		}
	case StackPush:
		state, err := newState(todo.Val)
		if err != nil {
			return err
		}
		if todo.Th.IsValid() {
			state.th = todo.Th
			state.length = todo.Length
			var inputs []byte
			if len(todo.Inputs) > 0 {
				inputs = make([]byte, len(todo.Inputs))
				copy(inputs, todo.Inputs)
			}
			state.buf = inputs
		}
		if todo.Nested != nil {
			state.handler = todo.Nested
		}
		ctx.stack = append(ctx.stack, state)
	case StackNested:
		if len(ctx.stack) == 0 {
			return ErrEmptyStack
		}
		if ctx.stack[len(ctx.stack)-1].handler != nil {
			return errors.New("already has nested handler")
		}
		ctx.stack[len(ctx.stack)-1].handler = todo.Nested
	default:
		// default to StackPop
		if len(ctx.stack) == 0 {
			return ErrEmptyStack
		}
		ctx.stack = ctx.stack[:len(ctx.stack)-1]
	}

	switch todo.DataTodo {
	case DataSkip:
		for i := 0; i < todo.Length; i++ {
			if _, err := ctx.vr.Skip(); err != nil {
				return fmt.Errorf("reader skipping %d/%d failed: %v", i, todo.Length, err)
			}
		}
	}

	return nil
}

type (
	StackOp byte
	DataOp  byte

	Todo struct {
		StackTodo StackOp
		DataTodo  DataOp
		Val       reflect.Value
		Th        TypeHeader // used for
		Length    int
		Inputs    []byte
		Nested    NestedHandler
	}

	// the lowest level event handler which process top of the stack
	HeaderHandler interface {
		Byte(value reflect.Value, input byte) (*Todo, error)
		Zero(value reflect.Value) (*Todo, error)
		True(value reflect.Value) (*Todo, error)
		Empty(value reflect.Value) (*Todo, error)
		Array(value reflect.Value, length int) (*Todo, error)
		Number(value reflect.Value, isPositive bool, inputs []byte) (*Todo, error)
		Bytes(value reflect.Value, inputs []byte) (*Todo, error)
		Version(value reflect.Value, inputs ...byte) (*Todo, error)
	}

	NestedHandler interface {
		Element() (*Todo, error)
		Index() int
	}

	DefaultHeaderHandler struct{}
)

var (
	_systemKindHandlers = make(map[reflect.Kind]HeaderHandler)
	_systemTypeHandlers = make(map[reflect.Type]HeaderHandler)

	popTodo = Todo{
		StackTodo: StackPop,
		Val:       reflect.Value{},
	}
)

func _systemKindHandler(handler HeaderHandler, kinds ...reflect.Kind) {
	for _, k := range kinds {
		_systemKindHandlers[k] = handler
	}
}

func _systemTypeHandler(handler HeaderHandler, typs ...reflect.Type) {
	for _, typ := range typs {
		_systemTypeHandlers[typ] = handler
	}
}

const (
	StackNone       StackOp = 0
	StackPop        StackOp = 1 // normal
	StackReplaceTop StackOp = 2 // for reflect.Ptr
	StackPush       StackOp = 3 // nested element
	StackNested     StackOp = 4 // nested handler

	DataNone DataOp = 0 // no op
	DataSkip DataOp = 1 // skip reader data
)

func (o StackOp) String() string {
	switch o {
	case StackNone:
		return "-"
	case StackPop:
		return "POP"
	case StackReplaceTop:
		return "REPL"
	case StackPush:
		return "PUSH"
	case StackNested:
		return "NESTED"
	default:
		return "N/A"
	}
}

func (o DataOp) String() string {
	switch o {
	case DataNone:
		return "-"
	case DataSkip:
		return "SKIP"
	default:
		return "N/A"
	}
}

func (f *Todo) String() string {
	if f == nil {
		return "Todo<nil>"
	}
	if f.Nested != nil {
		return fmt.Sprintf("Todo{Stack:%s Data:%s Val:%t NESTED}", f.StackTodo, f.DataTodo, f.Val.IsValid())
	} else {
		return fmt.Sprintf("Todo{Stack:%s Data:%s Val:%t}", f.StackTodo, f.DataTodo, f.Val.IsValid())
	}
}

func (f *Todo) Validate() error {
	if f == nil {
		return errors.New("nil todo")
	}
	switch f.StackTodo {
	case StackReplaceTop, StackPush:
		if !f.Val.IsValid() {
			return fmt.Errorf("missing reflect.Value of StackOp:%s", f.StackTodo)
		}
	case StackNested:
		if f.Nested == nil {
			return fmt.Errorf("missing NestedHandler of StateOp:%s", f.StackTodo)
		}
	}
	if f.DataTodo == DataSkip && f.Length <= 0 {
		return fmt.Errorf("invalid length of DataOp:%s", f.DataTodo)
	}
	return nil
}

func (f *Todo) Clone() *Todo {
	if f == nil {
		return nil
	}
	return &Todo{
		StackTodo: f.StackTodo,
		Val:       f.Val,
		Nested:    f.Nested,
	}
}

func (h DefaultHeaderHandler) Byte(_ reflect.Value, _ byte) (*Todo, error) {
	return nil, ErrUnsupported
}
func (h DefaultHeaderHandler) Zero(_ reflect.Value) (*Todo, error)  { return nil, ErrUnsupported }
func (h DefaultHeaderHandler) True(_ reflect.Value) (*Todo, error)  { return nil, ErrUnsupported }
func (h DefaultHeaderHandler) Empty(_ reflect.Value) (*Todo, error) { return nil, ErrUnsupported }
func (h DefaultHeaderHandler) Array(_ reflect.Value, _ int) (*Todo, error) {
	return nil, ErrUnsupported
}
func (h DefaultHeaderHandler) Number(_ reflect.Value, _ bool, _ []byte) (*Todo, error) {
	return nil, ErrUnsupported
}
func (h DefaultHeaderHandler) Bytes(_ reflect.Value, _ []byte) (*Todo, error) {
	return nil, ErrUnsupported
}
func (h DefaultHeaderHandler) Version(_ reflect.Value, _ ...byte) (*Todo, error) {
	return nil, ErrUnsupported
}

type EventDecoder struct {
	kindHandlers map[reflect.Kind]HeaderHandler
	typeHandlers map[reflect.Type]HeaderHandler
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

func (e *EventDecoder) _getTypeHandler(typ reflect.Type) (HeaderHandler, error) {
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

func (e *EventDecoder) _getKindHandler(kind reflect.Kind) (HeaderHandler, error) {
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
			ctx.pop()
			continue
		}

		var todo *Todo
		if state.handler != nil {
			todo, err = state.handler.Element()
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
					ctx.pop()
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
				todo, err = handler.Byte(state.val, byte(state.length))
			case THZeroValue:
				todo, err = handler.Zero(state.val)
			case THTrue:
				todo, err = handler.True(state.val)
			case THEmpty:
				todo, err = handler.Empty(state.val)
			case THArraySingle, THArrayMulti:
				todo, err = handler.Array(state.val, state.length)
			case THPosNumSingle, THNegNumSingle, THPosBigInt, THNegBigInt:
				todo, err = handler.Number(state.val, state.th == THPosNumSingle || state.th == THPosBigInt, state.buf)
			case THStringSingle, THStringMulti:
				todo, err = handler.Bytes(state.val, state.buf)
			case THVersion:
				todo, err = handler.Version(state.val, byte(state.length))
			case THVersionSingle:
				todo, err = handler.Version(state.val, state.buf...)
			}

			if err != nil {
				return fmt.Errorf("rtl: header handle failed: %v, at %s", err, ctx.StackInfo())
			}
		}

		if err = ctx.apply(todo); err != nil {
			return fmt.Errorf("rtl: apply %s failed: %v", todo, err)
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

	ctx := &HandleContext{vr: vr}
	state, err := newState(rev)
	if err != nil {
		return err
	}
	ctx.stack = append(ctx.stack, state)
	return e.handle(ctx)
}
