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
)

type ValueReader interface {
	HasMore() bool
	ReadHeader() (TypeHeader, uint32, error)
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

func (r *defaultVR) ReadHeader() (TypeHeader, uint32, error) {
	if !r.HasMore() {
		return 0, 0, io.EOF
	}
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, r.filterErr(err)
	}
	return ParseRTLHeader(b)
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

	var stack []headerStack
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
		size := int(length)
		if vt == THVTMultiHeader {
			ml, err := r.ReadMultiLength(int(length))
			skiped += int(length)
			if err != nil {
				return err
			}
			size = int(ml)
		}

		stack = append(stack, headerStack{
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

// type bufValueReader struct {
// 	reader     io.Reader // basic reader
// 	eof        bool      // if the reader EOF
// 	lastError  error     // error of last reading(if exist, except io.EOF)
// 	buffer     []byte    // buffered bytes
// 	available  uint32    // length of available bytes in buffer
// 	offset     uint32    // offset for buffer of reading
// 	readCount  int       // counting the read bytes
// 	readerSize int       // summary size of the reader, if it's a stream, set to MaxSliceSize
// }
//
// func (r *bufValueReader) ResetCount() {
// 	r.readCount = 0
// }
//
// func (r bufValueReader) ReadCount() int {
// 	return r.readCount
// }
//
// func (r *bufValueReader) HasMore() bool {
// 	if r.available > r.offset {
// 		return true
// 	}
// 	return r.next()
// }
//
// func (r *bufValueReader) left() int {
// 	return r.readerSize - r.readCount
// }
//
// // next read more bytes to buffer when buffer is empty,
// // and return if it has more bytes in buffer
// func (r *bufValueReader) next() bool {
// 	if r.available > r.offset {
// 		return true
// 	}
//
// 	if r.eof || r.lastError != nil {
// 		return false
// 	}
//
// 	r.offset = 0
// 	r.available = 0
// 	for {
// 		n, err := r.reader.Read(r.buffer)
// 		if n > 0 {
// 			r.available = uint32(n)
// 		}
// 		if err != nil {
// 			if err == io.EOF {
// 				r.eof = true
// 			} else {
// 				r.lastError = err
// 			}
// 			break
// 		}
// 		if n > 0 {
// 			break
// 		}
// 	}
// 	return r.available > 0
// }
//
// func (r *bufValueReader) forward(count int) {
// 	r.offset += uint32(count)
// 	r.readCount += count
// }
//
// // GetHeader get 1 byte from buffer, parse the byte to (TypeHeader, length)
// // if THSingleByte, length will be the byte value.
// // if THZeroValue/THTrue, length will be 0
// // if TypeHeader is a single byte header, length will be the length of the content
// // if TypeHeader is a multi bytes header, length will be the length of the length of the content
// // if anything goes wrong, error will not be nil, and (TypeHeader, length) are all meaningless
// func (r *bufValueReader) getHeader() (TypeHeader, uint32, error) {
// 	if !r.HasMore() {
// 		if r.lastError != nil {
// 			return 0, 0, r.lastError
// 		}
// 		return 0, 0, io.EOF
// 	}
// 	b := r.buffer[r.offset]
// 	return ParseRTLHeader(b)
// }
//
// // ReadHeader GetHeader and move 1byte forward if success
// func (r *bufValueReader) ReadHeader() (TypeHeader, uint32, error) {
// 	th, l, err := r.getHeader()
// 	if err == nil {
// 		r.offset++
// 		r.readCount++
// 	}
// 	return th, l, err
// }
//
// func (r *bufValueReader) ReadByte() (byte, error) {
// 	if !r.HasMore() {
// 		if r.lastError != nil {
// 			return 0, r.lastError
// 		}
// 		return 0, io.EOF
// 	}
// 	b := r.buffer[r.offset]
// 	r.offset++
// 	r.readCount++
// 	return b, nil
// }
//
// func (r *bufValueReader) Read(p []byte) (int, error) {
// 	if !r.HasMore() {
// 		if r.lastError != nil {
// 			return 0, r.lastError
// 		}
// 		return 0, io.EOF
// 	}
//
// 	// copy buffer to p
// 	n := copy(p, r.buffer[r.offset:r.available])
// 	r.offset += uint32(n)
// 	r.readCount += n
// 	if n >= len(p) {
// 		return n, nil
// 	}
//
// 	// read until fill full p or reader reach EOF
// 	ret := n
// 	for {
// 		if r.next() == false {
// 			// no more data
// 			if r.lastError != nil {
// 				return ret, r.lastError
// 			}
// 			return ret, io.EOF
// 		}
// 		n = copy(p[ret:], r.buffer[r.offset:r.available])
// 		ret += n
// 		r.offset += uint32(n)
// 		r.readCount += n
// 		if ret >= len(p) {
// 			return ret, nil
// 		}
// 	}
// }
//
// // ReadBytes read length bytes and return a slice, if parameter bytes length not
// // sufficient, will create a new slice
// func (r *bufValueReader) ReadBytes(length int, buf []byte) ([]byte, error) {
// 	return ReadBytesFromReader(r, length, buf)
// }
//
// // ReadMultiLength read length of multi bytes' header value's length
// func (r *bufValueReader) ReadMultiLength(length int) (uint64, error) {
// 	ret, err := ReadMultiLengthFromReader(r, length)
// 	if err != nil {
// 		return 0, err
// 	}
// 	left := r.left()
// 	if left <= 0 || ret > uint64(left) {
// 		return 0, fmt.Errorf("%d bytes multi-length(%d) is larger than left(%d)", length, ret, left)
// 	}
// 	return ret, nil
// }
//
// func (r *bufValueReader) ReadMultiLengthBytes(length int, buf []byte) ([]byte, error) {
// 	return ReadMultiLengthBytesFromReader(r, length, buf)
// }

func ParseRTLHeader(b byte) (TypeHeader, uint32, error) {
	for th, thv := range headerTypeMap {
		if thv.Match(b) {
			switch thv.T {
			case THVTByte:
				return th, uint32(b & thv.W), nil
			case THVTSingleHeader, THVTMultiHeader:
				l := uint32(b & thv.W)
				if l == 0 {
					l = uint32(thv.W + 1)
				}
				return th, l, nil
			default:
				// should not be here
				panic("unknown type")
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
	// if bufferSize > 0 {
	// 	return &bufValueReader{
	// 		reader:     r,
	// 		eof:        false,
	// 		buffer:     make([]byte, bufferSize),
	// 		available:  0,
	// 		offset:     0,
	// 		readerSize: l,
	// 	}
	// } else {
	return &defaultVR{
		reader:     r,
		eof:        false,
		readCount:  0,
		readerSize: l,
	}
	// }
}
