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

type mapElement struct {
	val            reflect.Value
	dataSize       int
	dataIdx        int
	kType, vType   reflect.Type
	kValue, vValue reflect.Value
}

var typeOfMapElement = reflect.TypeOf((*mapElement)(nil)).Elem()

func newMapElement(ctx *HandleContext, val reflect.Value, size int) (*mapElement, error) {
	if !val.IsValid() {
		return nil, ErrInvalidValue
	}
	typ := val.Type()
	if typ.Kind() != reflect.Map {
		return nil, errors.New("not a map")
	}
	if size%2 != 0 {
		return nil, fmt.Errorf("length of the array must be even when decode to a map, but length=%d", size)
	}
	if val.IsNil() {
		val.Set(reflect.MakeMapWithSize(typ, size/2))
	}
	ktyp := typ.Key()
	vtyp := typ.Elem()
	ret := ctx.NewNested(typeOfMapElement).(*mapElement)
	ret.val = val
	ret.dataSize = size
	ret.dataIdx = -1
	ret.kType = ktyp
	ret.vType = vtyp
	// ctx._count("mapElement")
	return ret, nil
	// return &mapElement{
	// 	val:      val,
	// 	dataSize: size,
	// 	dataIdx:  -1,
	// 	kType:    ktyp,
	// 	vType:    vtyp,
	// }, nil
}

func (m *mapElement) String() string {
	if m == nil {
		return "mapElem<nil>"
	}
	return fmt.Sprintf("mapElem[%d/%d]", m.dataIdx, m.dataSize)
}

func (m *mapElement) Element(ctx *HandleContext) error {
	if m.dataIdx > 0 && m.dataIdx%2 == 1 {
		// put value
		if !m.kValue.IsValid() || !m.vValue.IsValid() {
			return fmt.Errorf("missing k-v values when %d/%d", m.dataIdx, m.dataSize)
		}
		m.val.SetMapIndex(m.kValue, m.vValue)
		m.kValue = reflect.Value{}
		m.vValue = reflect.Value{}
	}
	m.dataIdx++
	if m.dataIdx >= m.dataSize {
		// map finished
		return ctx.PopState()
	}
	if m.dataIdx%2 == 0 {
		if m.kValue.IsValid() {
			return fmt.Errorf("a valid key value already in cache when %d/%d", m.dataIdx, m.dataSize)
		}
		m.kValue = reflect.New(m.kType).Elem()
		return ctx.PushState(m.kValue, THInvalid, 0, nil, nil)
	} else {
		if m.vValue.IsValid() {
			return fmt.Errorf("a valid v value already in cache when %d/%d", m.dataIdx, m.dataSize)
		}
		if !m.kValue.IsValid() {
			return fmt.Errorf("missing key value when %d/%d", m.dataIdx, m.dataSize)
		}
		m.vValue = reflect.New(m.vType).Elem()
		return ctx.PushState(m.vValue, THInvalid, 0, nil, nil)
	}
}

func (m *mapElement) Index() int {
	return m.dataIdx
}

type sliceElement struct {
	val      reflect.Value
	dataSize int
	dataIdx  int
}

var typeOfsliceElement = reflect.TypeOf((*sliceElement)(nil)).Elem()

func newSliceElement(ctx *HandleContext, val reflect.Value, size int) (*sliceElement, error) {
	if !val.IsValid() {
		return nil, ErrInvalidValue
	}
	typ := val.Type()
	if typ.Kind() != reflect.Slice {
		return nil, errors.New("not a slice")
	}
	checkSlice0(size, val)
	if size > val.Cap() {
		newv := reflect.MakeSlice(typ, size, size)
		val.Set(newv)
	}
	if size != val.Len() {
		val.SetLen(size)
	}
	ret := ctx.NewNested(typeOfsliceElement).(*sliceElement)
	ret.val = val
	ret.dataIdx = -1
	ret.dataSize = size
	// ctx._count("sliceElement")
	return ret, nil
	// return &sliceElement{
	// 	val:      val,
	// 	dataSize: size,
	// 	dataIdx:  -1,
	// }, nil
}

func (s *sliceElement) String() string {
	if s == nil {
		return "sliceElem<nil>"
	}
	return fmt.Sprintf("sliceElem[%d/%d]", s.dataIdx, s.dataSize)
}

func (s *sliceElement) Element(ctx *HandleContext) error {
	s.dataIdx++
	if s.dataIdx >= s.dataSize {
		return ctx.PopState()
	}
	evalue := s.val.Index(s.dataIdx)
	return ctx.PushState(evalue, THInvalid, 0, nil, nil)
}

func (s *sliceElement) Index() int {
	return s.dataIdx
}

type arrayElement struct {
	val       reflect.Value
	dataSize  int
	dataIdx   int
	valueSize int
}

var typeOfArrayElement = reflect.TypeOf((*arrayElement)(nil)).Elem()

func newArrayElement(ctx *HandleContext, val reflect.Value, size int) (*arrayElement, error) {
	if !val.IsValid() {
		return nil, ErrInvalidValue
	}
	typ := val.Type()
	if typ.Kind() != reflect.Array {
		return nil, errors.New("not an array")
	}
	ret := ctx.NewNested(typeOfArrayElement).(*arrayElement)
	ret.val = val
	ret.dataSize = size
	ret.dataIdx = -1
	ret.valueSize = val.Len()
	// ctx._count("arrayElement")
	return ret, nil
	// return &arrayElement{
	// 	val:       val,
	// 	dataSize:  size,
	// 	dataIdx:   -1,
	// 	valueSize: val.Len(),
	// }, nil
}

func (s *arrayElement) String() string {
	if s == nil {
		return "arrayElem<nil>"
	}
	return fmt.Sprintf("arrayElem[%d/%d->%d]", s.dataIdx, s.dataSize, s.valueSize)
}

func (s *arrayElement) Element(ctx *HandleContext) error {
	s.dataIdx++
	if s.dataIdx >= s.dataSize || s.dataIdx >= s.valueSize {
		if err := ctx.PopState(); err != nil {
			return err
		}
		if s.dataSize > s.valueSize {
			if err := ctx.SkipReader(s.dataSize - s.valueSize); err != nil {
				return err
			}
		}
		return nil
	}
	evalue := s.val.Index(s.dataIdx)
	return ctx.PushState(evalue, THInvalid, 0, nil, nil)
}

func (s *arrayElement) Index() int {
	return s.dataIdx
}

type string2ArraySlice struct {
	val       reflect.Value
	buf       []byte
	idx       int
	valueSize int
}

var typeOfString2ArraySlice = reflect.TypeOf((*string2ArraySlice)(nil)).Elem()

func newString2ArraySlice(ctx *HandleContext, val reflect.Value, buf []byte) (*string2ArraySlice, error) {
	if !val.IsValid() {
		return nil, ErrInvalidValue
	}
	if len(buf) == 0 {
		return nil, errors.New("missing data")
	}
	typ := val.Type()
	kind := typ.Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		return nil, errors.New("not an array or a slice")
	}
	if kind == reflect.Slice {
		checkSlice0(len(buf), val)
	}
	ret := ctx.NewNested(typeOfString2ArraySlice).(*string2ArraySlice)
	ret.val = val
	ret.buf = buf
	ret.idx = -1
	ret.valueSize = val.Len()
	// ctx._count("string2ArraySlice")
	return ret, nil
	// return &string2ArraySlice{
	// 	val:       val,
	// 	buf:       buf,
	// 	idx:       -1,
	// 	valueSize: val.Len(),
	// }, nil
}

func (s *string2ArraySlice) String() string {
	if s == nil {
		return "string2Array<nil>"
	}
	return fmt.Sprintf("string2Array[%d/%d->%d]", s.idx, len(s.buf), s.valueSize)
}

func (s *string2ArraySlice) Element(ctx *HandleContext) error {
	s.idx++
	if s.idx >= s.valueSize || s.idx >= len(s.buf) {
		return ctx.PopState()
	}
	evalue := s.val.Index(s.idx)
	return ctx.PushState(evalue, THSingleByte, int(s.buf[s.idx]), nil, nil)
}

func (s *string2ArraySlice) Index() int {
	return s.idx
}

type structElement struct {
	val      reflect.Value
	dataSize int         // data size
	dataIdx  int         // the last processed data index
	fields   []fieldName // structure
	fieldIdx int         // the last processed field index of the structure
}

var typeOfStructElement = reflect.TypeOf((*structElement)(nil)).Elem()

func newStructElement(ctx *HandleContext, val reflect.Value, size int) (*structElement, error) {
	if !val.IsValid() {
		return nil, ErrInvalidValue
	}
	typ := val.Type()
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("not a struct")
	}
	_, fields := structFields(typ)
	ret := ctx.NewNested(typeOfStructElement).(*structElement)
	ret.val = val
	ret.dataSize = size
	ret.dataIdx = -1
	ret.fields = fields
	ret.fieldIdx = -1
	// ctx._count("structElement")
	return ret, nil
	// return &structElement{
	// 	val:      val,
	// 	dataSize: size,
	// 	dataIdx:  -1,
	// 	fields:   fields,
	// 	fieldIdx: -1,
	// }, nil
}

func (s *structElement) String() string {
	if s == nil {
		return "structElem<nil>"
	}
	return fmt.Sprintf("structElem[%d/%d->%d/%d]", s.dataIdx, s.dataSize, s.fieldIdx, len(s.fields))
}

func (s *structElement) Element(ctx *HandleContext) error {
	nextField := s.fieldIdx + 1
	if nextField < len(s.fields) {
		fieldOrder := s.fields[nextField].order
		s.dataIdx++
		if s.dataIdx < s.dataSize {
			if s.dataIdx == fieldOrder {
				fvalue := s.val.Field(s.fields[nextField].index)
				s.fieldIdx = nextField
				return ctx.PushState(fvalue, THInvalid, 0, nil, nil)
			} else if s.dataIdx < fieldOrder {
				return ctx.SkipReader(1)
			} else {
				return fmt.Errorf("illegal status found: dataIdx:%d fieldIdx:%d %s",
					s.dataIdx, nextField, s.fields[nextField])
			}
		}
	}
	// set zero values
	for i := s.fieldIdx + 1; i < len(s.fields); i++ {
		fvalue := s.val.Field(s.fields[i].index)
		if fvalue.CanSet() {
			fvalue.Set(reflect.Zero(fvalue.Type()))
		}
	}

	if s.dataIdx < s.dataSize-1 {
		// skip datas and pop stack
		if err := ctx.PopState(); err != nil {
			return err
		}
		return ctx.SkipReader(s.dataSize - 1 - s.dataIdx)
	} else {
		return ctx.PopState()
	}
}

func (s *structElement) Index() int {
	return s.dataIdx
}
