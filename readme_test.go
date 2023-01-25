/*
 * Copyright 2023 Stephen Guo (stephen.fire@gmail.com)
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
	"math/big"
	"reflect"
	"testing"
)

func TestPrimary(t *testing.T) {
	var a, b int
	a = 142857
	if bs, err := Marshal(a); err == nil {
		if err = Unmarshal(bs, &b); err == nil {
			if a == b {
				t.Logf("%d == %d", a, b)
			} else {
				t.Fatalf("%d <> %d", a, b)
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}

	var x, y []int
	x = []int{1, 4, 2, 8, 5, 7}
	y = make([]int, 0)
	if bs, err := Marshal(x); err == nil {
		if err = Unmarshal(bs, &y); err == nil {
			if reflect.DeepEqual(x, y) {
				t.Logf("%v == %v", x, y)
			} else {
				t.Fatalf("%v <> %v", x, y)
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}

func TestBasic(t *testing.T) {
	type (
		embeded struct {
			A uint
			B uint
			C string
			D []byte
		}
		basic struct {
			A uint
			B uint
			C string
			E int
			F *big.Int
			G embeded
		}
	)

	obj := basic{
		A: 22,
		B: 33,
		C: "basic object",
		E: -983,
		F: big.NewInt(9999999),
		G: embeded{A: 44, B: 55, C: "embeded object", D: []byte("byte slice")},
	}

	{
		// Encode - Decode
		buf := new(bytes.Buffer)
		if err := Encode(obj, buf); err != nil {
			t.Fatal(err)
		}
		bs := buf.Bytes()

		decodedObj := new(basic)
		if err := Decode(bytes.NewReader(bs), decodedObj); err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(&obj, decodedObj) {
			t.Logf("%v encode-decode check", decodedObj)
		} else {
			t.Fatalf("%v %v not match", &obj, decodedObj)
		}
	}

	{
		// Marshal - Unmarshal
		bs, err := Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}

		decodedObj := new(basic)
		if err := Unmarshal(bs, decodedObj); err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(&obj, decodedObj) {
			t.Logf("%v marshal-unmarshal check", decodedObj)
		} else {
			t.Fatalf("%v %v not match", &obj, decodedObj)
		}
	}

}

func TestTypes(t *testing.T) {
	type (
		source struct {
			A []byte
			B []byte
		}
		dest struct {
			C string
			D []int
		}
	)

	src := &source{A: []byte("a string"), B: []byte{0x1, 0x2, 0x3, 0x4}}
	if bs, err := Marshal(src); err != nil {
		t.Fatal(err)
	} else {
		dst := new(dest)
		if err := Unmarshal(bs, dst); err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v -> %+v", src, dst)
	}
}

func TestOrder(t *testing.T) {
	type (
		source struct {
			A uint   // `rtlorder:"0"`
			B uint   // `rtlorder:"1"`
			C string // `rtlorder:"2"`
			D []byte // `rtlorder:"3"`
		}
		dest struct {
			E *big.Int `rtlorder:"4"`
			F int      `rtlorder:"5"`
			C string   `rtlorder:"2"`
			B uint     `rtlorder:"1"`
		}
	)

	src := &source{
		A: 1,
		B: 2,
		C: "Charlie",
		D: []byte("not in"),
	}
	if bs, err := Marshal(src); err != nil {
		t.Fatal(err)
	} else {
		dst := new(dest)
		if err := Unmarshal(bs, dst); err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v -> %+v", src, dst)
	}
}
