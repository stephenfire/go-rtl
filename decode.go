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
	"bytes"
	"io"
	"reflect"
)

func Unmarshal(buf []byte, v interface{}) error {
	return Decode(NewValueReader(bytes.NewBuffer(buf)), v)
}

// Decode reads bytes from r unmarshal to v, if you want to use same Reader to Decode multi
// values, you should use encoding.ValueReader as io.Reader.
func Decode(r io.Reader, v interface{}) error {
	return new(EventDecoder).Decode(r, v)
	// if v == nil {
	// 	return ErrDecodeIntoNil
	// }
	//
	// rv := reflect.ValueOf(v)
	// if rv.Kind() != reflect.Ptr {
	// 	return ErrDecodeNoPtr
	// }
	// if rv.IsNil() {
	// 	return ErrDecodeIntoNil
	// }
	//
	// isDecoder, err := checkTypeOfDecoder(r, rv)
	// if isDecoder || err != nil {
	// 	return err
	// }
	//
	// rev := rv.Elem()
	// vr, ok := r.(ValueReader)
	// if !ok {
	// 	vr = NewValueReader(r)
	// }
	// if err := valueReader(vr, rev); err != nil {
	// 	return err
	// }
	// return nil
}

func checkTypeOfDecoder(r io.Reader, value reflect.Value) (isDecoder bool, err error) {
	typ := value.Type()

	if typ.Implements(TypeOfDecoder) {
		isDecoder = true
		newCreate := false
		if typ.Kind() == reflect.Ptr && value.IsNil() {
			nvalue := reflect.New(typ.Elem())
			value.Set(nvalue)
			newCreate = true
		}
		decoder, _ := value.Interface().(Decoder)
		shouldBeNil, err := decoder.Deserialization(r)
		if err != nil {
			return isDecoder, err
		}
		if newCreate && shouldBeNil {
			value.Set(reflect.Zero(typ))
		}
		return isDecoder, err
	}

	if typ.Kind() == reflect.Ptr {
		etyp := typ.Elem()
		if etyp.Implements(TypeOfDecoder) {
			isDecoder = true
			elem := value.Elem()
			newCreate := false
			if etyp.Kind() == reflect.Ptr && elem.IsNil() {
				evalue := reflect.New(etyp.Elem())
				elem.Set(evalue)
				newCreate = true
			}
			decoder, _ := elem.Interface().(Decoder)
			shouldBeNil, err := decoder.Deserialization(r)
			if err != nil {
				return isDecoder, err
			}
			if newCreate && shouldBeNil {
				elem.Set(reflect.Zero(etyp))
			}
			return isDecoder, err
		}
	}

	return
}

func DecodeBigInt(r io.Reader, v interface{}) error {
	typ := reflect.TypeOf(v)
	if !typ.AssignableTo(typeOfBigInt) && !typ.AssignableTo(reflect.PtrTo(typeOfBigInt)) {
		return ErrUnsupported
	}
	vr, ok := r.(ValueReader)
	if !ok {
		vr = NewValueReader(r)
	}
	value := reflect.ValueOf(v)
	th, length, err := vr.ReadHeader()
	if err != nil {
		return nil
	}
	return bigIntReader0(th, int(length), vr, value, 0)
}
