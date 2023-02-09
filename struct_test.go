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
	"math/big"
	"reflect"
	"testing"
)

func TestStructFields(t *testing.T) {
	type inner struct {
		A int `rtlorder:"0" rtlversion:"0"`
		E int `rtlorder:"5" rtlversion:"2"`
		B int `rtlorder:"1" rtlversion:"1"`
		C int `rtlorder:"3"`
		D int `rtlorder:"4" `
	}

	i := new(inner)

	num, fields := structFields(reflect.TypeOf(*i))
	t.Log(num, fields)
}

func TestVersionedFields(t *testing.T) {
	type inner0 struct {
		A int `rtlorder:"0"`
		E int `rtlorder:"5"`
		B int `rtlorder:"1"`
		C int `rtlorder:"3"`
		D int `rtlorder:"4"`
		// F G
		H *int     `rtlorder:"8"`
		I bool     `rtlorder:"9"`
		J int      `rtlorder:"10"`
		K *big.Int `rtlorder:"11"`
	}
	// no version
	i0 := inner0{}
	n1, f1 := structFields(reflect.TypeOf(i0))
	n2, f2 := versionedFields(reflect.ValueOf(i0), f1)
	if n1 == n2 && reflect.DeepEqual(f1, f2) {
		t.Logf("no versioned: %+v -> n:%d f:%s", i0, n1, f1)
	} else {
		t.Fatalf("no versioned: %+v expecting: num:%d fields:%s but num:%d fields:%s", i0, n1, f1, n2, f2)
	}

	type inner struct {
		A int `rtlorder:"0" rtlversion:"0"`
		E int `rtlorder:"5" rtlversion:"2"`
		B int `rtlorder:"1" rtlversion:"1"`
		C int `rtlorder:"3"`
		D int `rtlorder:"4"`
		// F G
		H *int     `rtlorder:"8"`
		I bool     `rtlorder:"9"`
		J int      `rtlorder:"10" rtlversion:"5"`
		K *big.Int `rtlorder:"11"`
	}
	one := 0
	testDatas := []struct {
		val    inner
		fnum   int
		fields []fieldName
	}{
		{inner{}, 1, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
		}},
		{inner{J: 11}, 12, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
			{index: 2, name: "B", order: 1, version: 1},
			{index: 3, name: "C", order: 3, version: 1},
			{index: 4, name: "D", order: 4, version: 1},
			{index: 1, name: "E", order: 5, version: 2},
			{index: 5, name: "H", order: 8, version: 2},
			{index: 6, name: "I", order: 9, version: 2},
			{index: 7, name: "J", order: 10, version: 5},
			{index: 8, name: "K", order: 11, version: 5},
		}},
		{inner{J: 0, K: big.NewInt(0)}, 12, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
			{index: 2, name: "B", order: 1, version: 1},
			{index: 3, name: "C", order: 3, version: 1},
			{index: 4, name: "D", order: 4, version: 1},
			{index: 1, name: "E", order: 5, version: 2},
			{index: 5, name: "H", order: 8, version: 2},
			{index: 6, name: "I", order: 9, version: 2},
			{index: 7, name: "J", order: 10, version: 5},
			{index: 8, name: "K", order: 11, version: 5},
		}},
		{inner{H: &one}, 10, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
			{index: 2, name: "B", order: 1, version: 1},
			{index: 3, name: "C", order: 3, version: 1},
			{index: 4, name: "D", order: 4, version: 1},
			{index: 1, name: "E", order: 5, version: 2},
			{index: 5, name: "H", order: 8, version: 2},
			{index: 6, name: "I", order: 9, version: 2},
		}},
		{inner{E: 22}, 10, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
			{index: 2, name: "B", order: 1, version: 1},
			{index: 3, name: "C", order: 3, version: 1},
			{index: 4, name: "D", order: 4, version: 1},
			{index: 1, name: "E", order: 5, version: 2},
			{index: 5, name: "H", order: 8, version: 2},
			{index: 6, name: "I", order: 9, version: 2},
		}},
		{inner{C: 22}, 5, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
			{index: 2, name: "B", order: 1, version: 1},
			{index: 3, name: "C", order: 3, version: 1},
			{index: 4, name: "D", order: 4, version: 1},
		}},
		{inner{B: 1}, 5, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
			{index: 2, name: "B", order: 1, version: 1},
			{index: 3, name: "C", order: 3, version: 1},
			{index: 4, name: "D", order: 4, version: 1},
		}},
		{inner{A: 1}, 1, []fieldName{
			{index: 0, name: "A", order: 0, version: 0},
		}},
	}

	for _, data := range testDatas {
		fnum, fields := structFields(reflect.TypeOf(data.val))
		fnum, fields = versionedFields(reflect.ValueOf(data.val), fields)
		if fnum == data.fnum && reflect.DeepEqual(fields, data.fields) {
			t.Logf("%+v -> num:%d fields:%s", data.val, fnum, fields)
		} else {
			t.Fatalf("%+v expecting: num:%d fields:%s but num:%d fields:%s", data.val, data.fnum, data.fields, fnum, fields)
		}
	}
}
