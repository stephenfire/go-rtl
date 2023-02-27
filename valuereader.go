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
	"errors"
	"fmt"
	"io"
	"math"
)

type ValueReader interface {
	HasMore() bool
	ReadHeader() (TypeHeader, int, error)
	ReadFullHeader() (TypeHeader, int, error)
	io.ByteReader
	io.Reader
	ReadBytes(length int, buf []byte) ([]byte, error)
	ReadMultiLength(length int) (uint64, error)
	ReadMultiLengthBytes(length int, buf []byte) ([]byte, error)
	Skip() (int, error)
}

type defaultVR struct {
	reader     io.Reader
	eof        bool
	readCount  int
	header     [1]byte
	readerSize int
}

func EndOfFile(err error) bool {
	return err == io.EOF || err == io.ErrUnexpectedEOF
}

func (r *defaultVR) filterErr(err error) error {
	if EndOfFile(err) {
		r.eof = true
		return io.EOF
	}
	return err
}

func (r *defaultVR) HasMore() bool {
	if r.eof {
		return false
	}
	return true
}

func (r *defaultVR) left() int {
	return r.readerSize - r.readCount
}

func (r *defaultVR) ReadHeader() (TypeHeader, int, error) {
	if !r.HasMore() {
		return 0, 0, io.EOF
	}
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, r.filterErr(err)
	}
	return ParseRTLHeader(b)
}

func (r *defaultVR) ReadFullHeader() (TypeHeader, int, error) {
	th, length, err := r.ReadHeader()
	if err != nil {
		return 0, 0, err
	}
	if vt, exist := th.ValueType(); exist && vt == THVTMultiHeader {
		l, err := r.ReadMultiLength(length)
		if err != nil {
			return 0, 0, err
		}
		if l > math.MaxInt32 {
			return 0, 0, errors.New("full length overflow")
		}
		return th, int(l), nil
	} else {
		return th, length, nil
	}
}

func (r *defaultVR) ReadByte() (byte, error) {
	if !r.HasMore() {
		return 0, io.EOF
	}
	n, err := io.ReadFull(r.reader, r.header[:])
	r.readCount += n
	if err != nil {
		return 0, r.filterErr(err)
	}
	if n <= 0 {
		r.eof = true
		return 0, io.EOF
	}
	return r.header[0], nil
}

func (r *defaultVR) Read(p []byte) (int, error) {
	if !r.HasMore() {
		return 0, io.EOF
	}

	n, err := io.ReadFull(r.reader, p)
	r.readCount += n
	return n, r.filterErr(err)
}

func (r *defaultVR) ReadBytes(length int, buf []byte) ([]byte, error) {
	return ReadBytesFromReader(r, length, buf)
}

func (r *defaultVR) ReadMultiLength(length int) (uint64, error) {
	ret, err := ReadMultiLengthFromReader(r, length)
	if err != nil {
		return 0, err
	}
	left := r.left()
	if left <= 0 || ret > uint64(left) {
		return 0, fmt.Errorf("%d bytes multi-length(%d) is larger than left(%d)", length, ret, left)
	}
	return ret, nil
}

func (r *defaultVR) ReadMultiLengthBytes(length int, buf []byte) ([]byte, error) {
	return ReadMultiLengthBytesFromReader(r, length, buf)
}

func (r *defaultVR) _skip(length int) (int, error) {
	if length > 512 {
		buf := make([]byte, 512)
		for i := 0; i < length/512; i++ {
			n, err := io.ReadFull(r.reader, buf)
			r.readCount += n
			if err != nil {
				return i*512 + n, r.filterErr(err)
			}
		}
		if m := length % 512; m > 0 {
			buf = buf[:m]
			n, err := io.ReadFull(r.reader, buf)
			r.readCount += n
			return length - m + n, r.filterErr(err)
		} else {
			return length, nil
		}
	} else {
		buf := make([]byte, length)
		n, err := io.ReadFull(r.reader, buf)
		r.readCount += n
		return n, r.filterErr(err)
	}
}

type headerStack struct {
	th    TypeHeader
	vt    THValueType
	size  int
	index int
}

func (r *defaultVR) Skip() (int, error) {
	if !r.HasMore() {
		return 0, io.EOF
	}

	var stack []*headerStack
	skiped := 0

	readAndPush := func() error {
		th, length, err := r.ReadHeader()
		skiped++
		if err != nil {
			return err
		}

		vt, exist := th.ValueType()
		if !exist {
			return errors.New("invalid value type of the type header")
		}
		size := length
		if vt == THVTMultiHeader {
			ml, err := r.ReadMultiLength(length)
			skiped += length
			if err != nil {
				return err
			}
			size = int(ml)
		}

		stack = append(stack, &headerStack{
			th:    th,
			vt:    vt,
			size:  size,
			index: -1,
		})
		return nil
	}

	if err := readAndPush(); err != nil {
		return skiped, err
	}

	for len(stack) > 0 {
		last := stack[len(stack)-1]
		if last.th.Nested() {
			last.index++
			if last.index >= last.size {
				break
			}
			if err := readAndPush(); err != nil {
				return skiped, err
			}
		} else {
			if last.vt == THVTByte {
				// no more bytes need to skip
			} else {
				n, err := r._skip(last.size)
				skiped += n
				if err != nil {
					return skiped, err
				}
			}
			stack = stack[:len(stack)-1]
		}
	}
	return skiped, nil
}

func ParseRTLHeader(b byte) (TypeHeader, int, error) {
	for th, thv := range headerTypeMap {
		if thv.Match(b) {
			switch thv.T {
			case THVTByte:
				return th, int(b & thv.W), nil
			case THVTSingleHeader, THVTMultiHeader:
				l := int(b & thv.W)
				if l == 0 {
					l = int(thv.W + 1)
				}
				return th, l, nil
			default:
				// should not be here
				// panic("unknown type")
				return THInvalid, 0, errors.New("unknown type")
			}
		}
	}
	return 0, 0, ErrUnsupported
}

func ReadBytesFromReader(r io.Reader, length int, buf []byte) ([]byte, error) {
	if length <= 0 {
		return buf, ErrLength
	}

	if buf == nil && length > len(buf) {
		buf = make([]byte, length)
	} else {
		buf = buf[:length]
	}
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return buf, err
	}
	if n != length {
		return buf, fmt.Errorf("rtl length error: expect %d but %d readed", length, n)
	}
	return buf, nil
}

func ReadMultiLengthFromReader(vr ValueReader, length int) (uint64, error) {
	if length == 1 {
		b, err := vr.ReadByte()
		if err != nil {
			return 0, err
		}
		return uint64(b), nil
	} else {
		lbuf, err := vr.ReadBytes(length, nil)
		if err != nil {
			return 0, err
		}
		return Numeric.BytesToUint64(lbuf), nil
	}
}

func ReadMultiLengthBytesFromReader(vr ValueReader, length int, buf []byte) ([]byte, error) {
	l, err := vr.ReadMultiLength(length)
	if err != nil {
		return nil, err
	}

	bs, err := vr.ReadBytes(int(l), buf)
	if err != nil {
		return buf, err
	}

	return bs, nil
}

func NewValueReader(r io.Reader, _ ...int) ValueReader {
	l := MaxSliceSize
	lenner, ok := r.(Lenner)
	if ok {
		l = lenner.Len()
	}
	return &defaultVR{
		reader:     r,
		eof:        false,
		readCount:  0,
		readerSize: l,
	}
}
