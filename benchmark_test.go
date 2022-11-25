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
	"fmt"
	"math/big"
	"math/rand"
	"testing"
)

type (
	Address   [20]byte
	Hash      [32]byte
	TypedInt0 big.Int
	Included0 struct {
		Addr1  Address
		H1     Hash
		Bs     []byte
		Int256 *TypedInt0
	}

	ForBenchmark0 struct {
		Includes []*Included0
		Maps     map[Address]uint64
		Count    int
		IsOdd    bool
		F32      float32
	}
)

func _randomBytes(l int) []byte {
	if l < 0 {
		return nil
	}
	if l == 0 {
		return make([]byte, 0, 0)
	}
	bs := make([]byte, l, l)
	rand.Read(bs)
	return bs
}

func _randomN(max int) int {
	n := rand.Intn(max)
	if n == max-1 {
		n = -1
	}
	return n
}

func _randomBytesByN(max int) []byte {
	return _randomBytes(_randomN(max))
}

func (a Address) Random() Address {
	b := _randomBytes(len(a))
	var r Address
	copy(r[:], b)
	return r
}

func (h Hash) Random() Hash {
	b := _randomBytes(len(h))
	var r Hash
	copy(r[:], b)
	return r
}

func (i *TypedInt0) Random() *TypedInt0 {
	bs := _randomBytesByN(6)
	if bs == nil {
		return nil
	}
	if len(bs) == 0 {
		return (*TypedInt0)(big.NewInt(0))
	}
	return (*TypedInt0)(new(big.Int).SetBytes(bs))
}

func (c *Included0) Random() *Included0 {
	n := rand.Intn(10)
	if n == 0 {
		// 1/10 possibility to be nil
		return nil
	}
	ret := new(Included0)
	ret.Addr1 = ret.Addr1.Random()
	ret.H1 = ret.H1.Random()
	ret.Bs = _randomBytesByN(50)
	ret.Int256 = ret.Int256.Random()
	return ret
}

func (b *ForBenchmark0) Random() *ForBenchmark0 {
	ret := new(ForBenchmark0)
	ret.F32 = rand.Float32()

	n := _randomN(101)
	if n < 0 {
		return ret
	}
	if n == 0 {
		ret.Includes = make([]*Included0, 0, 0)
		ret.Maps = make(map[Address]uint64)
		return ret
	}
	ret.Includes = make([]*Included0, n, n)
	ret.Maps = make(map[Address]uint64)
	ret.Count = n
	ret.IsOdd = n%2 == 1

	for i := 0; i < n; i++ {
		ret.Includes[i] = ret.Includes[i].Random()
		if ret.Includes[i] != nil {
			ret.Maps[ret.Includes[i].Addr1] = rand.Uint64()
		}
	}
	return ret
}

var (
	_objects []*ForBenchmark0
	_steams  [][]byte
)

func init() {
	n := 100
	_objects = make([]*ForBenchmark0, n)
	_steams = make([][]byte, n)
	for i := 0; i < n; i++ {
		_objects[i] = _objects[i].Random()
		bs, err := Marshal(_objects[i])
		if err != nil {
			panic(fmt.Errorf("marshal failed at index:%d:%v", i, err))
		}
		_steams[i] = bs
	}
}

func BenchmarkDecodeV1(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	n := len(_steams)
	objs := make([]*ForBenchmark0, n)

	for i := 0; i < b.N; i++ {
		j := i % n
		obj := new(ForBenchmark0)
		buf := bytes.NewBuffer(_steams[j])
		err := DecodeV1(buf, obj)
		if err != nil {
			b.Fatalf("decode v1 failed: %v", err)
		}
		objs[j] = obj
	}
}

func BenchmarkDecodeV2(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	n := len(_steams)
	objs := make([]*ForBenchmark0, n)

	for i := 0; i < b.N; i++ {
		j := i % n
		obj := new(ForBenchmark0)
		buf := bytes.NewBuffer(_steams[j])
		err := DecodeV2(buf, obj)
		if err != nil {
			b.Fatalf("decode v1 failed: %v, index: %d, obj: %v, stream: %x", err, i, _objects[j], _steams[j])
		}
		objs[j] = obj
	}
}
