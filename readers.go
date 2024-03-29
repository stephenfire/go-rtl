/*
 * Copyright 2020 Stephen Guo (stephen.fire@gmail.com)
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
	"encoding/gob"
	"fmt"
	"log"
	"math/big"
	"reflect"
)

type Lenner interface {
	Len() int
}

type typeReaderFunc func(length int, vr ValueReader, value reflect.Value, nesting int) error

func unsupported(_ int, _ ValueReader, _ reflect.Value, _ int) error {
	return ErrUnsupported
}

func unsupportFunc(typ reflect.Type, th TypeHeader) typeReaderFunc {
	return func(length int, vr ValueReader, value reflect.Value, nesting int) error {
		return fmt.Errorf("rtl: unsupported headerType:%s decode to (type:%s kind:%s)", th, typ, typ.Kind())
	}
}

func getFunc(valueType reflect.Type, funcMap map[TypeHeader]typeReaderFunc, th TypeHeader) typeReaderFunc {
	f, _ := funcMap[th]
	if f == nil {
		f = unsupportFunc(valueType, th)
	}
	return f
}

// toSmallBigInt decode single byte header value to big.Int
func toSmallBigInt(length int, vr ValueReader, isNegative bool, value reflect.Value) error {
	buf, err := vr.ReadBytes(length, nil)
	if err != nil {
		return err
	}
	i := getOrNewBigInt(value)
	if isNegative {
		i.SetInt64(Numeric.BytesToInt64(buf, true))
	} else {
		i.SetUint64(Numeric.BytesToUint64(buf))
	}
	return nil
}

// toBigBigInt decode multi bytes header value to big.Int
func toBigBigInt(length int, vr ValueReader, value reflect.Value, _ int) error {
	buf, err := vr.ReadMultiLengthBytes(length, nil)
	if err != nil {
		return err
	}
	i := getOrNewBigInt(value)
	i.SetBytes(buf)
	return nil
}

func toBigNegBigInt(length int, vr ValueReader, value reflect.Value, _ int) error {
	buf, err := vr.ReadMultiLengthBytes(length, nil)
	if err != nil {
		return err
	}
	i := getOrNewBigInt(value)
	i.SetBytes(buf)
	i.Neg(i)
	return nil
}

func toSmallGob(length int, vr ValueReader, value reflect.Value, getGob func(reflect.Value) gob.GobDecoder) error {
	buf, err := vr.ReadBytes(length, nil)
	if err != nil {
		return err
	}
	g := getGob(value)
	return g.GobDecode(buf)
}

func toBigGob(length int, vr ValueReader, value reflect.Value, getGob func(reflect.Value) gob.GobDecoder) error {
	buf, err := vr.ReadMultiLengthBytes(length, nil)
	if err != nil {
		return err
	}
	g := getGob(value)
	return g.GobDecode(buf)
}

// toInt decode single byte header bytes to int value
func toInt(length int, vr ValueReader, isNegative bool, value reflect.Value) error {
	buf, err := vr.ReadBytes(length, nil)
	if err != nil {
		return err
	}
	i := Numeric.BytesToInt64(buf, isNegative)
	value.SetInt(i)
	return nil
}

// toUint decode single byte header bytes to uint value
func toUint(length int, vr ValueReader, value reflect.Value, _ int) error {
	buf, err := vr.ReadBytes(length, nil)
	if err != nil {
		return err
	}
	i := Numeric.BytesToUint64(buf)
	value.SetUint(i)
	return nil
}

// toFloat decode single byte header bytes to float value
func toFloat(length int, vr ValueReader, isNegative bool, value reflect.Value) error {
	buf, err := vr.ReadBytes(length, nil)
	if err != nil {
		return err
	}
	var f float64
	if length == 4 {
		f = float64(Numeric.BytesToFloat32(buf, isNegative))
	} else {
		f = Numeric.BytesToFloat64(buf, isNegative)
	}
	value.SetFloat(f)
	return nil
}

func getOrNewBigInt(v reflect.Value) *big.Int {
	i := v.Interface().(*big.Int)
	if i == nil {
		i = new(big.Int)
		v.Set(reflect.ValueOf(i))
	}
	return i
}

func getOrNewBigRat(v reflect.Value) gob.GobDecoder {
	g := v.Interface().(*big.Rat)
	if g == nil {
		g = new(big.Rat)
		v.Set(reflect.ValueOf(g))
	}
	return g
}

func getOrNewBigFloat(v reflect.Value) gob.GobDecoder {
	g := v.Interface().(*big.Float)
	if g == nil {
		g = new(big.Float)
		v.Set(reflect.ValueOf(g))
	}
	return g
}

// stringToArray decode string(byte slice) to array of type which support single byte value
func stringToArray(buf []byte, vr ValueReader, value reflect.Value, nesting int) error {
	vl := value.Len()
	l := len(buf)
	i := 0

	etyp := value.Type().Elem()
	if etyp == typeOfByte {
		i = reflect.Copy(value, reflect.ValueOf(buf))
	} else {
		for ; i < vl && i < l; i++ {
			evalue := value.Index(i)
			err := valueReader1(THSingleByte, int(buf[i]), vr, evalue, nesting+1)
			if err != nil {
				return err
			}
		}
	}
	if i != vl || i != l {
		log.Printf("rtl: string to array/slice length not match, "+
			"len(string)=%d, len(array)=%d, %d elements writed", l, vl, i)
	}
	return nil
}

func singleByteToArray0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	vl := value.Len()
	if vl >= 1 {
		evalue := value.Index(0)
		return valueReader1(THSingleByte, length, vr, evalue, nesting+1)
	}
	log.Printf("rtl: restore nothing for an 0 length array/slice with %s(byte:%x)", THSingleByte, length)
	return nil
}

// element could be any type which could expressed by a byte
func stringSingleToArray0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	buf, err := vr.ReadBytes(length, nil)
	if err != nil {
		return err
	}
	return stringToArray(buf, vr, value, nesting)
}

func stringMultiToArray0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	buf, err := vr.ReadMultiLengthBytes(length, nil)
	if err != nil {
		return err
	}
	return stringToArray(buf, vr, value, nesting)
}

func arraySingleToArray0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	return toArray0(length, vr, value, nesting)
}

func arrayMultiToArray0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	l, err := vr.ReadMultiLength(length)
	if err != nil {
		return err
	}
	return toArray0(int(l), vr, value, nesting)
}

func toArray0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	vl := value.Len()
	i := 0
	nesting++
	for ; i < length && i < vl; i++ {
		evalue := value.Index(i)
		if err := valueReader0(vr, evalue, nesting); err != nil {
			return err
		}
	}
	if i != vl || i != length {
		log.Printf("rtl: string to array/slice length not match, "+
			"len(string)=%d, len(array)=%d, %d elements writed", length, vl, i)
	}

	return nil
}

func singleByteToSlice0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	checkSlice0(1, value)
	return singleByteToArray0(length, vr, value, nesting)
}

func stringSingleToSlice0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	checkSlice0(length, value)
	return stringSingleToArray0(length, vr, value, nesting)
}

func stringMultiToSlice0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	l, err := vr.ReadMultiLength(length)
	if err != nil {
		return err
	}
	checkSlice0(int(l), value)
	return stringSingleToArray0(int(l), vr, value, nesting)
}

func arraySingleToSlice0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	checkSlice0(length, value)
	return arraySingleToArray0(length, vr, value, nesting)
}

func arrayMultiToSlice0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	l, err := vr.ReadMultiLength(length)
	if err != nil {
		return err
	}
	checkSlice0(int(l), value)
	return arraySingleToArray0(int(l), vr, value, nesting)
}

func checkSlice0(length int, value reflect.Value) {
	if length > value.Cap() {
		newv := reflect.MakeSlice(value.Type(), length, length)
		value.Set(newv)
	}
	if length != value.Len() {
		value.SetLen(length)
	}
}

func arraySingleToMap0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	return toMap0(length, vr, value, nesting)
}

func arrayMultiToMap0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	l, err := vr.ReadMultiLength(length)
	if err != nil {
		return err
	}
	return toMap0(int(l), vr, value, nesting)
}

func toMap0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	if length%2 != 0 {
		return fmt.Errorf("rtl: length of the array must be even when decode to a map, but length=%d", length)
	}
	typ := value.Type()

	if value.IsNil() {
		value.Set(reflect.MakeMapWithSize(typ, length/2))
	}

	ktyp := typ.Key()
	vtyp := typ.Elem()

	nesting++
	for i := 0; i < length; i += 2 {
		kvalue := reflect.New(ktyp).Elem()
		vvalue := reflect.New(vtyp).Elem()
		if err := valueReader0(vr, kvalue, nesting); err != nil {
			return err
		}
		if err := valueReader0(vr, vvalue, nesting); err != nil {
			return err
		}
		value.SetMapIndex(kvalue, vvalue)
	}

	return nil
}

func arraySingleToStruct0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	return toStruct0(length, vr, value, nesting)
}

func arrayMultiToStruct0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	l, err := vr.ReadMultiLength(length)
	if err != nil {
		return err
	}
	return toStruct0(int(l), vr, value, nesting)
}

func toStruct0(length int, vr ValueReader, value reflect.Value, nesting int) error {
	typ := value.Type()
	_, fnames := structFields(typ)
	lth := len(fnames)
	// if lth > length {
	// 	lth = length
	// }

	nesting++
	nextOrder := 0 // 下一个field对应的order
	nextIndex := 0 // 下一个filed的对应下标
	if nextIndex < lth {
		nextOrder = fnames[nextIndex].order
		for i := 0; i < length; i++ {
			if i == nextOrder {
				fvalue := value.Field(fnames[nextIndex].index)
				if err := valueReader0(vr, fvalue, nesting); err != nil {
					return err
				}
				nextIndex++
				if nextIndex >= lth {
					break
				}
				nextOrder = fnames[nextIndex].order
			} else if i < nextOrder {
				if _, err := vr.Skip(); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("illegal status found: dataIndex:%d nextIndex:%d %s",
					i, nextIndex, fnames[nextIndex])
			}
		}
	}
	// 将后面未包含在对象中的字段置空
	for ; nextIndex < lth; nextIndex++ {
		fvalue := value.Field(fnames[nextIndex].index)
		if err := valueReader1(THZeroValue, 0, vr, fvalue, nesting); err != nil {
			return err
		}
	}

	return nil
}

var (
	// fill in value, which must be a *big.Int
	bigIntReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			i := getOrNewBigInt(value)
			i.SetInt64(int64(length))
			return nil
		},
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.Set(reflect.Zero(value.Type()))
			return nil
		},
		THPosNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toSmallBigInt(length, vr, false, value)
		},
		THNegNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toSmallBigInt(length, vr, true, value)
		},
		THPosBigInt: toBigBigInt,
		THNegBigInt: toBigNegBigInt,
	}
	bigRatReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: unsupported,
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.Set(reflect.Zero(value.Type()))
			return nil
		},
		THPosNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toSmallGob(length, vr, value, getOrNewBigRat)
		},
		THNegNumSingle: unsupported,
		THPosBigInt: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toBigGob(length, vr, value, getOrNewBigRat)
		},
		THNegBigInt: unsupported,
	}
	bigFloatReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: unsupported,
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.Set(reflect.Zero(value.Type()))
			return nil
		},
		THPosNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toSmallGob(length, vr, value, getOrNewBigFloat)
		},
		THNegNumSingle: unsupported,
		THPosBigInt: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toBigGob(length, vr, value, getOrNewBigFloat)
		},
		THNegBigInt: unsupported,
	}
	// value SHOULD NOT be a pointer
	intReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetInt(int64(length))
			return nil
		},
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetInt(int64(0))
			return nil
		},
		THPosNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toInt(length, vr, false, value)
		},
		THNegNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toInt(length, vr, true, value)
		},
	}
	// value SHOULD NOT be a pointer
	uintReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetUint(uint64(length))
			return nil
		},
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetUint(uint64(0))
			return nil
		},
		THPosNumSingle: toUint,
	}
	// value SHOULD NOT be a pointer
	floatReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetFloat(Numeric.ByteToFloat64(byte(length), false))
			return nil
		},
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetFloat(0)
			return nil
		},
		THPosNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toFloat(length, vr, false, value)
		},
		THNegNumSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			return toFloat(length, vr, true, value)
		},
	}
	// value SHOULD NOT be a pointer
	boolReaders = map[TypeHeader]typeReaderFunc{
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetBool(false)
			return nil
		},
		THTrue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetBool(true)
			return nil
		},
	}
	// value SHOULD NOT be a pointer
	stringReaders = map[TypeHeader]typeReaderFunc{
		THSingleByte: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.SetString(string([]byte{byte(length)}))
			return nil
		},
		THZeroValue: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			value.Set(reflect.Zero(value.Type()))
			return nil
		},
		THStringSingle: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			buf, err := vr.ReadBytes(length, nil)
			if err != nil {
				return err
			}
			value.SetString(string(buf))
			return nil
		},
		THStringMulti: func(length int, vr ValueReader, value reflect.Value, nesting int) error {
			l, err := vr.ReadMultiLength(length)
			if err != nil {
				return err
			}
			buf, err := vr.ReadBytes(int(l), nil)
			if err != nil {
				return err
			}
			value.SetString(string(buf))
			return nil
		},
	}
)

func valueReader(r ValueReader, value reflect.Value) error {
	return valueReader0(r, value, 0)
}

func valueReader0(vr ValueReader, value reflect.Value, nesting int) error {
	// decode itself if the value implements encoding.Decoder interface
	isDecoder, err := checkTypeOfDecoder(vr, value)
	if isDecoder || err != nil {
		return err
	}

	// if not an encoding.Decoder implementation, use default decoder
	th, length, err := vr.ReadHeader()
	if err != nil {
		// if EndOfFile(err) {
		// 	return nil
		// }
		return err
	}

	return valueReader1(th, length, vr, value, nesting)
}

func valueReader1(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int) error {
	if nesting > MaxNested {
		return ErrNestingOverflow
	}

	typ := value.Type()

	if matched, err := checkPriorStructsReader(th, length, vr, value, nesting); err != nil {
		return err
	} else if matched {
		return err
	}

	kind := value.Kind()
	switch kind {
	case reflect.Array:
		switch th {
		case THSingleByte:
			return singleByteToArray0(length, vr, value, nesting)
		case THZeroValue:
			value.Set(reflect.Zero(typ))
			return nil
		case THStringSingle:
			return stringSingleToArray0(length, vr, value, nesting)
		case THStringMulti:
			return stringMultiToArray0(length, vr, value, nesting)
		case THArraySingle:
			return arraySingleToArray0(length, vr, value, nesting)
		case THArrayMulti:
			return arrayMultiToArray0(length, vr, value, nesting)
		}
	case reflect.Slice:
		switch th {
		case THSingleByte:
			return singleByteToSlice0(length, vr, value, nesting)
		case THZeroValue:
			value.Set(reflect.Zero(typ))
			return nil
		case THEmpty:
			if !value.CanSet() {
				return fmt.Errorf("rtl: slice can not set to empty")
			}
			nslice := reflect.MakeSlice(typ, 0, 0)
			value.Set(nslice)
			return nil
		case THStringSingle:
			return stringSingleToSlice0(length, vr, value, nesting)
		case THStringMulti:
			return stringMultiToSlice0(length, vr, value, nesting)
		case THArraySingle:
			return arraySingleToSlice0(length, vr, value, nesting)
		case THArrayMulti:
			return arrayMultiToSlice0(length, vr, value, nesting)
		}
	case reflect.Map:
		switch th {
		case THZeroValue:
			value.Set(reflect.Zero(typ))
			return nil
		case THEmpty:
			if !value.CanSet() {
				return fmt.Errorf("rtl: map can not set")
			}
			nmap := reflect.MakeMapWithSize(typ, 0)
			value.Set(nmap)
			return nil
		case THArraySingle:
			return arraySingleToMap0(length, vr, value, nesting)
		case THArrayMulti:
			return arrayMultiToMap0(length, vr, value, nesting)
		}
	case reflect.Struct:
		return toStructs(typ, kind, th, length, vr, value, nesting)
	case reflect.Ptr:
		// pointer
		return toPointers(typ, kind, th, length, vr, value, nesting)
	case reflect.Interface:
		// as interface{} type
		return toInterfaces(typ, kind, th, length, vr, value, nesting)
	default:
		funcMap, ok := primKindTypeHeaderMap[kind]
		if ok {
			return typedReader0(th, length, vr, value, nesting, funcMap)
		} else {
			return fmt.Errorf("rtl: unsupported type1 %v (kind: %s, headerType: %s) for decoding", typ, kind, th)
		}
	}
	return fmt.Errorf("rtl: unsupported type2 %v (kind: %s, headerType: %s) for decoding", typ, kind, th)
}

func toStructs(typ reflect.Type, kind reflect.Kind, th TypeHeader, length int, vr ValueReader,
	value reflect.Value, nesting int) error {
	switch th {
	case THZeroValue:
		value.Set(reflect.Zero(typ))
		return nil
	case THArraySingle:
		return arraySingleToStruct0(length, vr, value, nesting)
	case THArrayMulti:
		return arrayMultiToStruct0(length, vr, value, nesting)
	}
	return fmt.Errorf("rtl: unsupported type3 %v (kind: %s, headerType: %s) for decoding", typ, kind, th)
}

func toPointers(typ reflect.Type, _ reflect.Kind, th TypeHeader, length int, vr ValueReader,
	value reflect.Value, nesting int) error {
	etyp := typ.Elem()
	if th == THZeroValue {
		// nil pointer
		if !value.IsNil() {
			value.Set(reflect.Zero(typ))
		}
		return nil
	} else {
		// pointer to something
		// check if the type supported at first
		ekind := etyp.Kind()
		if !canBeDecodeTo(ekind) {
			return fmt.Errorf("rtl: unsupported type4 pointer to %v (kind: %s) "+
				"for decoding", etyp, ekind.String())
		}
		// create if nil
		evalue := value
		if value.IsNil() {
			if !value.CanSet() {
				return fmt.Errorf("rtl: cannot create new value %s", etyp.Name())
			}
			evalue = reflect.New(etyp)
		}
		err := valueReader1(th, length, vr, evalue.Elem(), nesting)
		if err == nil && value.IsNil() {
			value.Set(evalue)
		}
		return err
	}
}

func toInterfaces(typ reflect.Type, kind reflect.Kind, th TypeHeader, length int, vr ValueReader,
	value reflect.Value, nesting int) error {
	if value.Type().NumMethod() != 0 {
		return fmt.Errorf("rtl: unsupported type5 %v (kind: %s, headerType: %s) for decoding", typ, kind, th)
	}
	switch th {
	case THSingleByte:
		nv := reflect.New(typeOfUint64).Elem()
		nv.SetUint(uint64(length))
		value.Set(nv)
		return nil
	case THZeroValue:
		value.Set(reflect.Zero(typeOfInterface))
		return nil
	case THEmpty:
		value.Set(reflect.MakeSlice(typeOfInterfaceSlice, 0, 0))
		return nil
	case THPosNumSingle:
		b, err := vr.ReadBytes(length, nil)
		if err != nil {
			return nil
		}
		nv := reflect.New(typeOfUint64).Elem()
		nv.SetUint(Numeric.BytesToUint64(b))
		value.Set(nv)
		return nil
	case THNegNumSingle:
		b, err := vr.ReadBytes(length, nil)
		if err != nil {
			return nil
		}
		nv := reflect.New(typeOfInt64).Elem()
		nv.SetInt(Numeric.BytesToInt64(b, true))
		value.Set(nv)
		return nil
	case THStringSingle:
		b, err := vr.ReadBytes(length, nil)
		if err != nil {
			return nil
		}
		nv := reflect.New(typeOfString).Elem()
		nv.SetString(string(b))
		value.Set(nv)
		return nil
	case THStringMulti:
		b, err := vr.ReadMultiLengthBytes(length, nil)
		if err != nil {
			return nil
		}
		nv := reflect.New(typeOfString).Elem()
		nv.SetString(string(b))
		value.Set(nv)
		return nil
	case THArraySingle:
		slice := reflect.MakeSlice(typeOfInterfaceSlice, length, length)
		value.Set(slice)
		return arraySingleToArray0(length, vr, slice, nesting)
	case THArrayMulti:
		l, err := vr.ReadMultiLength(length)
		if err != nil {
			return err
		}
		slice := reflect.MakeSlice(typeOfInterfaceSlice, int(l), int(l))
		value.Set(slice)
		return arrayMultiToArray0(int(l), vr, slice, nesting)
	}
	return fmt.Errorf("rtl: unsupported type6 %v (kind: %s, headerType: %s) for decoding", typ, kind, th)
}

func typedReader0(th TypeHeader, length int, vr ValueReader, value reflect.Value, nesting int,
	funcMap map[TypeHeader]typeReaderFunc) error {
	if funcMap == nil {
		return fmt.Errorf("rtl: type mismatch error: expect %s but %s found", value.Type().Name(), th.Name())
	}
	f, ok := funcMap[th]
	if !ok {
		return fmt.Errorf("rtl: type mismatch error: expect %s but %s found", value.Type().Name(), th.Name())
	}
	return f(length, vr, value, nesting)
}

func canBeDecodeTo(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	case reflect.Float32:
	case reflect.Float64:
	case reflect.Bool:
	case reflect.String:
	case reflect.Array:
	case reflect.Slice:
	case reflect.Map:
	case reflect.Struct:
	case reflect.Ptr:
	default:
		return false
	}
	return true
}
