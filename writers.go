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
	"fmt"
	"io"
	"math"
	"reflect"
)

// writerFunc encode v as bytes write to w, returns the length of bytes it write
type writerFunc func(w io.Writer, v reflect.Value) (int, error)

func valueWriter(w io.Writer, value reflect.Value) (int, error) {
	return valueWriter0(w, value, 0)
}

func valueWriter0(w io.Writer, value reflect.Value, nesting int) (int, error) {
	if nesting > MaxNested {
		return 0, ErrNestingOverflow
	}

	typ := value.Type()

	if typ.Implements(TypeOfEncoder) {
		// if the object can be serialized by itself
		encoder, _ := value.Interface().(Encoder)
		return 0, encoder.Serialization(w)
	}

	if matched, n, err := checkPriorStructsWriter(w, value); err != nil {
		return n, err
	} else if matched {
		return n, err
	}

	kind := value.Kind()
	switch kind {
	case reflect.Invalid:
		// zero Value, do nothing
		return 0, nil
		// return w.Write(zeroValues)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intWriter(w, value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintWriter(w, value)
	case reflect.Float32:
		return float32Writer(w, value)
	case reflect.Float64:
		return float64Writer(w, value)
	case reflect.Bool:
		return boolWriter(w, value)
	case reflect.String:
		return stringWriter(w, value)
	case reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			// byte array
			return byteArrayWriter(w, value)
		} else {
			// other type of array, nesting++
			return arrayWriter0(w, value, nesting)
		}
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			// byte slice(array)
			return byteSliceWriter(w, value)
		} else {
			// other type of slice(array), nesting++
			return sliceWriter0(w, value, nesting)
		}
	case reflect.Map:
		// nesting++
		return mapWriter0(w, value, nesting)
	case reflect.Struct:
		// nesting++
		return structWriter0(w, value, nesting)
	case reflect.Ptr:
		return pointerWriter0(w, value, nesting)
	case reflect.Interface:
		return interfaceWriter0(w, value, nesting)
	default:
		return 0, fmt.Errorf("unsupported type %v for encoding", value.Type())
	}
}

func bytesWriter(w io.Writer, bs []byte) (int, error) {
	if bs == nil {
		return w.Write(zeroValues)
	}
	if len(bs) == 0 {
		return w.Write(emptyValues)
	}
	var ret int = 0
	if len(bs) == 1 && bs[0] <= 127 {
		// single byte
		return w.Write(bs)
	}
	// multi bytes
	h, err := HeadMaker.string(len(bs))
	if err != nil {
		return ret, err
	}
	if h == nil {
		// should not be here
		return w.Write(zeroValues)
	}
	n, err := w.Write(h)
	ret += n
	if err != nil {
		return ret, err
	}
	n, err = w.Write(bs)
	ret += n
	return ret, nil
}

func stringWriter(w io.Writer, v reflect.Value) (int, error) {
	s := v.String()
	if s == "" {
		// empty string
		return w.Write(zeroValues)
	}
	str := []byte(s)
	return bytesWriter(w, str)
}

func byteSliceWriter(w io.Writer, v reflect.Value) (int, error) {
	return bytesWriter(w, v.Bytes())
}

func byteArrayWriter(w io.Writer, v reflect.Value) (int, error) {
	if !v.CanAddr() {
		// reflect.Value.Slice() requires the value must be addressable
		cp := reflect.New(v.Type()).Elem()
		cp.Set(v)
		v = cp
	}

	l := v.Len()
	s := v.Slice(0, l).Bytes()
	return bytesWriter(w, s)
}

func boolWriter(w io.Writer, v reflect.Value) (int, error) {
	if v.Bool() {
		return w.Write(trueBools)
	} else {
		return w.Write(zeroValues)
	}
}

func smallNumberWriter(w io.Writer, isNegative bool, i uint64) (int, error) {
	// zero value
	if i < 0 {
		return w.Write(zeroValues)
	}

	// single byte value
	if isNegative == false && i <= 127 {
		return w.Write([]byte{byte(i)})
	}

	// header(1byte) + value(max 8bytes) = max 9bytes
	buf := make([]byte, 9)

	// value
	l, err := Numeric.writeUint(buf[1:], i)
	if err != nil {
		return 0, err
	}
	// header (1 byte)
	lh, err := HeadMaker.numericBuf(isNegative, l, buf)
	if err != nil {
		return 0, err
	}

	return w.Write(buf[:l+lh])

}

func uintWriter(w io.Writer, v reflect.Value) (int, error) {
	i := v.Uint()
	return smallNumberWriter(w, false, i)
}

func intWriter(w io.Writer, v reflect.Value) (int, error) {
	i := v.Int()
	isNegative := false
	if i < 0 {
		isNegative = true
		i = -i
	}
	return smallNumberWriter(w, isNegative, uint64(i))
}

func float32Writer(w io.Writer, v reflect.Value) (int, error) {
	f := float32(v.Float())
	neg := f < 0
	if neg {
		f = -f
	}
	u32 := math.Float32bits(f)
	return smallNumberWriter(w, neg, uint64(u32))
}

func float64Writer(w io.Writer, v reflect.Value) (int, error) {
	f := v.Float()
	neg := f < 0
	if neg {
		f = -f
	}
	u64 := math.Float64bits(f)
	return smallNumberWriter(w, neg, u64)
}

func arrayWriter0(w io.Writer, v reflect.Value, nesting int) (int, error) {
	length := v.Len()
	if length <= 0 {
		if v.Kind() == reflect.Slice {
			return w.Write(emptyValues)
		}
		return w.Write(zeroValues)
	}

	// array would +1 to nesting, so equals to MaxNested is overflowed
	if nesting >= MaxNested {
		return 0, ErrNestingOverflow
	}

	h, err := HeadMaker.array(length)
	if err != nil {
		return 0, err
	}
	ret, err := w.Write(h)
	if err != nil {
		return ret, nil
	}

	// nesting elements
	nesting++
	for i := 0; i < length; i++ {
		vv := v.Index(i)
		n, err := valueWriter0(w, vv, nesting)
		ret += n
		if err != nil {
			return ret, err
		}
	}

	return ret, nil
}

func arrayWriter(w io.Writer, v reflect.Value) (int, error) {
	return arrayWriter0(w, v, 0)
}

func sliceWriter0(w io.Writer, v reflect.Value, nesting int) (int, error) {
	if v.IsNil() {
		return w.Write(zeroValues)
	}
	return arrayWriter0(w, v, nesting)
}

func sliceWriter(w io.Writer, v reflect.Value) (int, error) {
	return sliceWriter0(w, v, 0)
}

func mapWriter0(w io.Writer, v reflect.Value, nesting int) (int, error) {
	if v.IsNil() {
		return w.Write(zeroValues)
	}

	keys := v.MapKeys()
	if keys == nil || len(keys) == 0 {
		return w.Write(emptyValues)
	}

	// map would +1 to nesting, so equals to MaxNested is overflowed
	if nesting >= MaxNested {
		return 0, ErrNestingOverflow
	}

	length := len(keys)
	length <<= 1
	h, err := HeadMaker.array(length)
	if err != nil {
		return 0, err
	}
	ret, err := w.Write(h)
	if err != nil {
		return ret, err
	}

	// nesting elements
	nesting++
	for _, key := range keys {
		value := v.MapIndex(key)
		n, err := valueWriter0(w, key, nesting)
		ret += n
		if err != nil {
			return ret, err
		}
		n, err = valueWriter0(w, value, nesting)
		ret += n
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func mapWriter(w io.Writer, v reflect.Value) (int, error) {
	return mapWriter0(w, v, 0)
}

// TODO: 当count很大时，可以考虑用特殊字节代表多个zero
func zerosPlacehold(w io.Writer, count int) (int, error) {
	for i := 0; i < count; i++ {
		_, err := w.Write(zeroValues)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

func structWriter0(w io.Writer, v reflect.Value, nesting int) (int, error) {
	typ := v.Type()
	fnum, fnames := structFields(typ)

	if len(fnames) <= 0 {
		// no available fields in the struct
		return w.Write(zeroValues)
	}

	// struct would +1 to nesting, so equals to MaxNested is overflowed
	if nesting >= MaxNested {
		return 0, ErrNestingOverflow
	}

	fnum, fnames = versionedFields(v, fnames)

	h, err := HeadMaker.array(fnum)
	if err != nil {
		return 0, err
	}
	ret, err := w.Write(h)
	if err != nil {
		return ret, err
	}

	// next nesting
	nesting++
	order := -1
	// write all exported fields
	for _, fname := range fnames {
		if fname.order > order+1 {
			// 用ZeroValue补足order跳过的字段
			n, err := zerosPlacehold(w, fname.order-order-1)
			if err != nil {
				return 0, err
			}
			ret += n
		}
		order = fname.order
		vv := v.Field(fname.index)
		n, err := valueWriter0(w, vv, nesting)
		ret += n
		if err != nil {
			return ret, err
		}
	}

	return ret, nil
}

func structWriter(w io.Writer, v reflect.Value) (int, error) {
	return structWriter0(w, v, 0)
}

func pointerWriter0(w io.Writer, v reflect.Value, nesting int) (int, error) {
	if v.IsNil() {
		return w.Write(zeroValues)
	}

	return valueWriter0(w, v.Elem(), nesting)
}

func pointerWriter(w io.Writer, v reflect.Value) (int, error) {
	return pointerWriter0(w, v, 0)
}

func interfaceWriter0(w io.Writer, v reflect.Value, nesting int) (int, error) {
	if v.IsNil() {
		return w.Write(zeroValues)
	}
	return valueWriter0(w, v.Elem(), nesting)
}

func interfaceWriter(w io.Writer, v reflect.Value) (int, error) {
	return interfaceWriter0(w, v, 0)
}
