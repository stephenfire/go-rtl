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
	"testing"
)

func TestValueReader(t *testing.T) {
	t.Log("start testing ValueReader.Read()")
	bs := []byte("e1384c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e7365637465747572206164697069736963696e6720656c6974")
	buf := bytes.NewBuffer(bs)
	l := len(bs)
	vr := NewValueReader(buf, 0)
	var sum, count int
	b := make([]byte, 10)
	for {
		n, err := vr.Read(b)
		count++
		if n > 0 {
			for i := 0; i < n; i++ {
				if b[i] != bs[i+sum] {
					t.Errorf("not match %x should be %x at pos %d in %d read", b[i], bs[i+sum], i+sum, count)
				}
			}
			sum += n
		}
		if err != nil {
			break
		}
	}
	if sum != l {
		t.Errorf("length not match %d should be %d", sum, l)
	}
}
