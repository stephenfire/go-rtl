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

func Marshal(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	var err error
	err = Encode(v, buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Encode(v interface{}, w io.Writer) error {
	vv := reflect.ValueOf(v)
	_, err := valueWriter(w, vv)
	return err
}

func EncodeBigInt(v interface{}, w io.Writer) error {
	value := reflect.ValueOf(v)
	typ := value.Type()
	for typ.Kind() == reflect.Ptr {
		value = value.Elem()
		typ = value.Type()
	}
	if !typ.AssignableTo(typeOfBigInt) {
		return ErrUnsupported
	}
	_, err := bigIntWriter(w, value)
	return err
}
