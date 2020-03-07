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
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func Unhex(str string) []byte {
	b, err := hex.DecodeString(strings.Replace(str, " ", "", -1))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}

type encodeTest struct {
	i1    int8
	i2    int16
	i3    int32
	i4    int64
	u1    uint8
	u2    uint16
	u3    uint32
	u4    uint64
	bytes [32]byte
}

func (t *encodeTest) Serialization(w io.Writer) error {
	w.Write(ToBinaryBytes(t.i1))
	w.Write(ToBinaryBytes(t.i2))
	w.Write(ToBinaryBytes(t.i3))
	w.Write(ToBinaryBytes(t.i4))
	w.Write(ToBinaryBytes(t.u1))
	w.Write(ToBinaryBytes(t.u2))
	w.Write(ToBinaryBytes(t.u3))
	w.Write(ToBinaryBytes(t.u4))
	w.Write(t.bytes[:])
	return nil
}

func (t *encodeTest) Deserialization(r io.Reader) (shouldBeNil bool, err error) {
	bs := make([]byte, 32)
	r.Read(bs[:1])
	t.i1 = int8(BinaryToInt(bs[:1]))

	r.Read(bs[:2])
	t.i2 = int16(BinaryToInt(bs[:2]))

	r.Read(bs[:4])
	t.i3 = int32(BinaryToInt(bs[:4]))

	r.Read(bs[:8])
	t.i4 = int64(BinaryToInt(bs[:8]))

	r.Read(bs[:1])
	t.u1 = uint8(BinaryToUint(bs[:1]))

	r.Read(bs[:2])
	t.u2 = uint16(BinaryToUint(bs[:2]))

	r.Read(bs[:4])
	t.u3 = uint32(BinaryToUint(bs[:4]))

	r.Read(bs[:8])
	t.u4 = uint64(BinaryToUint(bs[:8]))

	r.Read(bs[0:32])
	copy(t.bytes[:], bs)

	return false, nil
}

type namedByteType byte

type RawValue []byte

type simplestruct struct {
	A uint
	B string
}

type recstruct struct {
	I     uint
	Child *recstruct
}

type tailRaw struct {
	A    uint
	Tail []RawValue
}

type hasIgnoredField struct {
	A uint
	B uint
	C uint
	D uint
}

type param struct {
	val interface{}
}

type mapstruct struct {
	A map[string]int64
	B map[int64]*string
}

type bigstruct struct {
	I *big.Int
	R *big.Rat
	F *big.Float
}

type stringAndSlice struct {
	A string
	B []byte
}

type arrayAndSlice struct {
	A [32]byte
	B []byte
}

var (
	string1 = "string1"
	string2 = "string2"
)

var encTests = []param{

	{val: float32(111.3)},
	{val: float64(34343434.333)},

	{val: &encodeTest{i1: 0x12, i2: 0x3456, i3: 0x567890ab, i4: -1, u1: 0xF1, u2: 0xFFF2, u3: 0xFFFFFFF3,
		u4: 0xFFFFFFFFFFFFFFF4, bytes: [...]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0',
			'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
			'0', '1'}}},

	{val: mapstruct{map[string]int64{"key1": 1, "key2": 2}, map[int64]*string{1: &string1, 2: &string2}}},

	// booleans
	{val: true},
	{val: false},

	// integers
	{val: uint32(0)},
	{val: uint32(127)},
	{val: uint32(128)},
	{val: uint32(256)},
	{val: uint32(1024)},
	{val: uint32(0xFFFFFF)},
	{val: uint32(0xFFFFFFFF)},
	{val: uint64(0xFFFFFFFF)},
	{val: uint64(0xFFFFFFFFFF)},
	{val: uint64(0xFFFFFFFFFFFF)},
	{val: uint64(0xFFFFFFFFFFFFFF)},
	{val: uint64(0xFFFFFFFFFFFFFFFF)},

	{val: int8(0)},
	{val: int8(127)},
	{val: int8(-127)},
	{val: int16(128)},
	{val: int16(-128)},
	{val: int32(256)},
	{val: int32(-1024)},
	{val: int32(0xFFFFFF)},
	{val: int64(0xFFFFFFFF)},
	{val: int64(-0xFFFFFFFF)},
	{val: int64(0xFFFFFFFFFF)},
	{val: int64(-0xFFFFFFFFFF)},
	{val: int64(0xFFFFFFFFFFFF)},
	{val: int64(-0xFFFFFFFFFFFF)},
	{val: int64(0xFFFFFFFFFFFFFF)},
	{val: int64(-0xFFFFFFFFFFFFFF)},
	{val: int64(0x7FFFFFFFFFFFFFFF)},
	{val: int64(-0x7FFFFFFFFFFFFFFF)},

	// big integers (should match uint for small values)
	{val: big.NewInt(0)},
	{val: big.NewInt(1)},
	{val: big.NewInt(127)},
	{val: big.NewInt(128)},
	{val: big.NewInt(256)},
	{val: big.NewInt(1024)},
	{val: big.NewInt(0xFFFFFF)},
	{val: big.NewInt(0xFFFFFFFF)},
	{val: big.NewInt(0xFFFFFFFFFF)},
	{val: big.NewInt(0xFFFFFFFFFFFF)},
	{val: big.NewInt(0xFFFFFFFFFFFFFF)},
	{
		val: big.NewInt(0).SetBytes(Unhex("102030405060708090A0B0C0D0E0F2")),
	},
	{
		val: big.NewInt(0).SetBytes(Unhex("0100020003000400050006000700080009000A000B000C000D000E01")),
	},
	{
		val: big.NewInt(0).SetBytes(Unhex("010000000000000000000000000000000000000000000000000000000000000000")),
	},
	{
		val: big.NewInt(0).Sub(big.NewInt(0), big.NewInt(0).SetBytes(Unhex("0100020003000400050006000700080009000A000B000C000D000E01"))),
	},

	// non-pointer big.Int
	{val: *big.NewInt(0)},
	{val: *big.NewInt(0xFFFFFF)},

	// negative ints are not supported
	{val: big.NewInt(-1)},

	// byte slices, strings
	{val: []byte{}},
	{val: []byte{0x7E}},
	{val: []byte{0x7F}},
	{val: []byte{0x80}},
	{val: []byte{1, 2, 3}},

	{val: []namedByteType{1, 2, 3}},
	{val: [...]namedByteType{1, 2, 3}},

	{val: ""},
	{val: "\x7E"},
	{val: "\x7F"},
	{val: "\x80"},
	{val: "dog"},
	{
		val: "Lorem ipsum dolor sit amet, consectetur adipisicing eli",
	},
	{
		val: "Lorem ipsum dolor sit amet, consectetur adipisicing elit",
	},
	{
		val: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur mauris magna, suscipit sed vehicula non, iaculis faucibus tortor. Proin suscipit ultricies malesuada. Duis tortor elit, dictum quis tristique eu, ultrices at risus. Morbi a est imperdiet mi ullamcorper aliquet suscipit nec lorem. Aenean quis leo mollis, vulputate elit varius, consequat enim. Nulla ultrices turpis justo, et posuere urna consectetur nec. Proin non convallis metus. Donec tempor ipsum in mauris congue sollicitudin. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Suspendisse convallis sem vel massa faucibus, eget lacinia lacus tempor. Nulla quis ultricies purus. Proin auctor rhoncus nibh condimentum mollis. Aliquam consequat enim at metus luctus, a eleifend purus egestas. Curabitur at nibh metus. Nam bibendum, neque at auctor tristique, lorem libero aliquet arcu, non interdum tellus lectus sit amet eros. Cras rhoncus, metus ac ornare cursus, dolor justo ultrices metus, at ullamcorper volutpat",
	},

	// slices
	{val: []uint{}},
	{val: []uint{1, 2, 3}},
	{
		// [ [], [[]], [ [], [[]] ] ]
		val: []interface{}{[]interface{}{}, []interface{}{[]interface{}{}}, []interface{}{[]interface{}{}, []interface{}{[]interface{}{}}}},
	},
	{
		val: []string{"aaa", "bbb", "ccc", "ddd", "eee", "fff", "ggg", "hhh", "iii", "jjj", "kkk", "lll", "mmm", "nnn", "ooo"},
	},
	{
		val: []interface{}{uint64(1), uint64(0xFFFFFF), []interface{}{[]interface{}{uint64(4), uint64(5), uint64(5)}}, "abc"},
	},
	{
		val: [][]string{
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
			{"asdf", "qwer", "zxcv"},
		},
	},

	// RawValue
	{val: RawValue(Unhex("01"))},
	{val: RawValue(Unhex("82FFFF"))},
	{val: []RawValue{Unhex("01"), Unhex("02")}},

	// structs
	{val: simplestruct{}},
	{val: simplestruct{A: 3, B: "foo"}},
	{val: &recstruct{5, nil}},
	{val: &recstruct{5, &recstruct{4, &recstruct{3, nil}}}},
	{val: &tailRaw{A: 1, Tail: []RawValue{Unhex("02"), Unhex("03")}}},
	{val: &tailRaw{A: 1, Tail: []RawValue{Unhex("02")}}},
	{val: &tailRaw{A: 1, Tail: []RawValue{}}},
	{val: &tailRaw{A: 1, Tail: nil}},
	{val: &hasIgnoredField{A: 1, B: 0, C: 3, D: 0}},

	// nil
	{val: (*uint)(nil)},
	{val: (*string)(nil)},
	{val: (*[]byte)(nil)},
	{val: (*[10]byte)(nil)},
	{val: (*big.Int)(nil)},
	{val: (*[]string)(nil)},
	{val: (*[10]string)(nil)},
	{val: (*[]interface{})(nil)},
	{val: (*[]struct{ uint })(nil)},
	{val: (*interface{})(nil)},

	// bigs
	{val: &bigstruct{I: big.NewInt(3234234543)}},
	{val: &bigstruct{R: big.NewRat(1233423323, 3545)}},
	{val: &bigstruct{F: big.NewFloat(239842.23354345456)}},
	{val: &bigstruct{I: big.NewInt(3234234543), R: big.NewRat(1233423323, 3545)}},
	{val: &bigstruct{I: big.NewInt(3234234543), F: big.NewFloat(239842.23354345456)}},
	{val: &bigstruct{R: big.NewRat(1233423323, 3545), F: big.NewFloat(239842.23354345456)}},
	{val: &bigstruct{I: big.NewInt(3234234543), R: big.NewRat(1233423323, 3545), F: big.NewFloat(239842.23354345456)}},

	{val: &stringAndSlice{A: "name1", B: []byte("slice1slice1")}},
	{val: &stringAndSlice{A: "name1", B: nil}},

	{val: &arrayAndSlice{A: [32]byte{8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8}, B: []byte("sssssssss")}},
	{val: &arrayAndSlice{A: [32]byte{8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8}, B: nil}},
}

func TestEncode(t *testing.T) {
	buf := new(bytes.Buffer)
	for _, test := range encTests {
		val := reflect.ValueOf(test.val)
		buf.Reset()
		// valueWriter(buf, val)
		Encode(test.val, buf)
		bs := buf.Bytes()
		// fmt.Println(test.val, "->", hex.EncodeToString(buf.Bytes()))

		typ := val.Type()
		nv := reflect.New(typ)
		// vr := NewValueReader(buf, 100)
		nvv := nv.Elem()
		// if err := valueReader(vr, nvv); err != nil {
		// vr := NewValueReader(buf, 256)
		vr := buf
		if err := Decode(vr, nv.Interface()); err != nil {
			t.Error(err)
		}

		fmt.Printf("%v: %#v\n\t%X\n%v: %#v\n", typ, test.val, bs, nvv.Type(), nvv)
		if reflect.DeepEqual(test.val, nvv.Interface()) {
			t.Log(test.val, "check")
		} else {
			t.Error(test.val, "error")
		}
	}
}

type version1 struct {
	A uint
	B uint
}

type version2 struct {
	A uint
	B uint
	C string `rtlorder:"5"`
	D []byte
}

type version3 struct {
	A uint
	B uint
	C string   `rtlorder:"5"`
	E int      `rtlorder:"3"`
	F *big.Int `rtlorder:"4"`
	G version1 `rtlorder:"6"`
}

type version4 struct {
	C string   `rtlorder:"5"`
	E int      `rtlorder:"3"`
	F *big.Int `rtlorder:"4"`
	G version2 `rtlorder:"6"`
}

func TestVersion(t *testing.T) {
	{
		v1 := &version1{A: 87690, B: 12345}
		buf := new(bytes.Buffer)
		Encode(v1, buf)
		bs1 := buf.Bytes()
		v2 := new(version2)
		if err := Decode(buf, v2); err != nil {
			t.Error(err)
		}

		if v1.A == v2.A && v1.B == v2.B && v2.C == "" && v2.D == nil {
			t.Log("version1 -> version2 check")
		} else {
			t.Errorf("version1 -> version2 failed, %+v -> %+v", v1, v2)
		}

		v3 := &version3{A: 22, B: 33, C: "ccc", E: 8888, F: big.NewInt(99999), G: version1{A: 12, B: 34}}
		if err := Decode(bytes.NewReader(bs1), v3); err != nil {
			t.Error(err)
		}

		if v3.A == v1.A && v3.B == v1.B && v3.C == "" && v3.E == 0 && v3.F == nil && v3.G.A == 0 && v3.G.B == 0 {
			t.Log("version1 -> version3 check")
		} else {
			t.Errorf("version1 -> version3 failed, %+v -> %+v", v1, v3)
		}
	}

	{
		v2 := &version2{A: 222999, B: 333000, C: "version2", D: []byte("xxxxxxx")}
		buf := new(bytes.Buffer)
		Encode(v2, buf)
		bs2 := buf.Bytes()

		v3 := &version3{A: 22, B: 33, C: "ccc", E: 8888, F: big.NewInt(99999), G: version1{A: 12, B: 34}}
		if err := Decode(bytes.NewReader(bs2), v3); err != nil {
			t.Error(err)
		}
		if v3.A == v2.A && v3.B == v2.B && v3.C == v2.C && v3.E == 0 && v3.F == nil && v3.G.A == 0 && v3.G.B == 0 {
			t.Log("version2 -> version3 check")
		} else {
			t.Errorf("version2 -> version3 failed, %+v -> %x -> %+v", v2, bs2, v3)
		}

		v1 := &version1{A: 87690, B: 12345}
		if err := Decode(bytes.NewReader(bs2), v1); err != nil {
			t.Error(err)
		}
		if v1.A == v2.A && v1.B == v2.B {
			t.Log("version2 -> version1 check")
		} else {
			t.Errorf("version2 -> version1 failed, %+v -> %+v", v2, v1)
		}
	}

	{
		v3 := &version3{A: 22, B: 33, C: "ccc", E: 8888, F: big.NewInt(99999), G: version1{A: 12, B: 34}}
		buf := new(bytes.Buffer)
		Encode(v3, buf)
		bs := buf.Bytes()

		v1 := &version1{A: 87690, B: 12345}
		if err := Decode(bytes.NewReader(bs), v1); err != nil {
			t.Error(err)
		}
		if v1.A == v3.A && v1.B == v3.B {
			t.Log("version3 -> version1 check")
		} else {
			t.Errorf("version3 -> version1 failed, %+v -> %+v", v3, v1)
		}

		v2 := &version2{A: 8802, B: 65623, C: "xxxx", D: []byte("yyyyyy")}
		if err := Decode(bytes.NewReader(bs), v2); err != nil {
			t.Error(err)
		}
		if v3.A == v2.A && v3.B == v2.B && v3.C == v2.C && v2.D == nil {
			t.Log("version3 -> version2 check")
		} else {
			t.Errorf("version3 -> version2 failed, %+v -> %+v", v3, v2)
		}

		v4 := &version4{
			C: "version4",
			E: 99999,
			F: big.NewInt(100000),
			G: version2{A: 222999, B: 333000, C: "version2", D: []byte("xxxxxxx")},
		}
		if err := Decode(bytes.NewReader(bs), v4); err != nil {
			t.Error(err)
		}
		if v4.C == v3.C && v4.E == v3.E && v4.F.Cmp(v3.F) == 0 && v4.G.A == v3.G.A && v4.G.B == v3.G.B && v4.G.C == "" && v4.G.D == nil {
			t.Log("version3 -> version4 check")
		} else {
			t.Errorf("version3 -> version4 failed, %+v -> %+v", v3, v4)
		}
	}

	{
		v4 := &version4{
			C: "version4",
			E: 99999,
			F: big.NewInt(100000),
			G: version2{A: 222999, B: 333000, C: "version2", D: []byte("xxxxxxx")},
		}
		buf := new(bytes.Buffer)
		Encode(v4, buf)
		bs := buf.Bytes()

		v3 := &version3{A: 22, B: 33, C: "ccc", E: 8888, F: big.NewInt(99999), G: version1{A: 12, B: 34}}
		if err := Decode(bytes.NewReader(bs), v3); err != nil {
			t.Error(err)
		}
		if v3.C == v4.C && v3.E == v4.E && v3.F.Cmp(v4.F) == 0 && v3.G.A == v4.G.A && v3.G.B == v4.G.B {
			t.Log("version4 -> version3 check")
		} else {
			t.Errorf("version4 -> version3 failed, %+v -> %x -> %+v", v4, bs, v3)
		}

		v1 := &version1{A: 87690, B: 12345}
		if err := Decode(bytes.NewReader(bs), v1); err != nil {
			t.Error(err)
		}
		if v1.A == 0 && v1.B == 0 {
			t.Log("version4 -> version1 check")
		} else {
			t.Errorf("version4 -> version1 failed, %+v -> %+v", v3, v1)
		}
	}
}
