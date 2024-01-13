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
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type THValueType byte

// type header value
type THValue struct {
	N string      // name of the type header
	C byte        // code
	M byte        // mask
	W byte        // wildcard mask
	T THValueType // type of the header
	B bool        // whether the header is followed by value bytes
	X bool        // whether the type is a nested type, which means there's should be another header follow the current header
}

func (thvalue THValue) Match(b byte) bool {
	return (b & thvalue.M) == thvalue.C
}

func (thvalue THValue) WithNumber(b byte) byte {
	return thvalue.C | (b & thvalue.W)
}

type TypeHeader byte

func (th TypeHeader) Name() string {
	if th == THInvalid {
		return "N/A"
	}
	thv, ok := headerTypeMap[th]
	if ok {
		return thv.N
	}
	return "TypeHeader" + strconv.Itoa(int(th))
}

func (th TypeHeader) String() string {
	return th.Name()
}

func (th TypeHeader) IsValid() bool {
	return th < THInvalid
}

func (th TypeHeader) Nested() bool {
	thv, ok := headerTypeMap[th]
	if ok {
		return thv.X
	}
	return false
}

func (th TypeHeader) ValueType() (THValueType, bool) {
	thv, ok := headerTypeMap[th]
	if ok {
		return thv.T, true
	}
	return THVTInvalid, false
}

func (th TypeHeader) FollowedByHeader() bool {
	thv, ok := headerTypeMap[th]
	if ok {
		return !thv.B
	}
	return false
}

func (th TypeHeader) FollowedByBytes() bool {
	thv, ok := headerTypeMap[th]
	if ok {
		return thv.B
	}
	return false
}

const (
	THSingleByte    TypeHeader = iota // single byte
	THZeroValue                       // zero value (empty string / false of bool)
	THTrue                            // true of bool
	THEmpty                           // empty value
	THArraySingle                     // array with no more than 16 elements
	THArrayMulti                      // array with more than 16 elements
	THPosNumSingle                    // positive number with bytes less and equal to 8
	THNegNumSingle                    // negative number with bytes less and equal to 8
	THPosBigInt                       // positive big.Int
	THNegBigInt                       // negative big.Int
	THStringSingle                    // string with length less and equal to 32
	THStringMulti                     // string with length more than 32
	THVersion                         // 0 <= (version number) <= 15
	THVersionSingle                   // 15 < (version number) < 2^64
	THInvalid
)

const (
	THVTByte         THValueType = iota // one byte value
	THVTSingleHeader                    // single byte header
	THVTMultiHeader                     // multi bytes header
	THVTInvalid
)

const (
	MaxSliceSize = 100 * 1024 * 1024 // max size of creating slice: 100MB
	MaxNested    = 100               // max nested times when encoding. pointer, slice, array, map, struct
)

type (
	// Encoder is the interface which encoding package while invoke the Serialization()
	// when encoding the object.
	// ATTENTION: the receiver of Encoder.Serialization() and Decoder.Deserialization() MUST
	// BE SAME. otherwise, they will not be use in same struct.
	Encoder interface {
		Serialization(w io.Writer) error
	}

	Decoder interface {
		Deserialization(r io.Reader) (shouldBeNil bool, err error)
	}
)

var (
	NilOrFalse   = headerTypeMap[THZeroValue].C
	NotNilOrTrue = headerTypeMap[THTrue].C
	Version0     = headerTypeMap[THVersion].WithNumber(0x0)
	Version1     = headerTypeMap[THVersion].WithNumber(0x1)
	Version2     = headerTypeMap[THVersion].WithNumber(0x2)
	Version3     = headerTypeMap[THVersion].WithNumber(0x3)
	Version4     = headerTypeMap[THVersion].WithNumber(0x4)
	Version5     = headerTypeMap[THVersion].WithNumber(0x5)
	Version6     = headerTypeMap[THVersion].WithNumber(0x6)
	Version7     = headerTypeMap[THVersion].WithNumber(0x7)
	Version8     = headerTypeMap[THVersion].WithNumber(0x8)
	Version9     = headerTypeMap[THVersion].WithNumber(0x9)
	Version10    = headerTypeMap[THVersion].WithNumber(0xa)
	Version11    = headerTypeMap[THVersion].WithNumber(0xb)
	Version12    = headerTypeMap[THVersion].WithNumber(0xc)
	Version13    = headerTypeMap[THVersion].WithNumber(0xd)
	Version14    = headerTypeMap[THVersion].WithNumber(0xe)
	Version15    = headerTypeMap[THVersion].WithNumber(0xf)
)

var (
	// static encoding
	zeroValues  = []byte{headerTypeMap[THZeroValue].C}
	trueBools   = []byte{headerTypeMap[THTrue].C}
	emptyValues = []byte{headerTypeMap[THEmpty].C}

	// header maker of encoding
	HeadMaker headMaker

	// codec for numerics
	Numeric numeric

	// big.Int
	bigint128    = big.NewInt(128)
	typeOfBigInt = reflect.TypeOf(big.Int{})

	// big.Rat && big.Float
	typeOfBigRat   = reflect.TypeOf(big.Rat{})
	typeOfBigFloat = reflect.TypeOf(big.Float{})

	// []interface{} type
	typeOfInterfaceSlice = reflect.TypeOf([]interface{}{})
	typeOfInterface      = reflect.TypeOf((*interface{})(nil)).Elem()

	// uint64
	typeOfUint64 = reflect.TypeOf((*uint64)(nil)).Elem()
	typeOfInt64  = reflect.TypeOf((*int64)(nil)).Elem()
	typeOfString = reflect.TypeOf("")
	typeOfByte   = reflect.TypeOf((*byte)(nil)).Elem()

	// header constants
	headerTypeMap = map[TypeHeader]THValue{
		THSingleByte:    {"Byte", 0x00, 0x80, ^byte(0x80), THVTByte, false, false},
		THZeroValue:     {"Zero", 0x80, 0xFF, 0x00, THVTByte, false, false},
		THTrue:          {"True", 0x81, 0xFF, 0x00, THVTByte, false, false},
		THEmpty:         {"Empty", 0x82, 0xFF, 0x00, THVTByte, false, false},
		THArraySingle:   {"Array", 0x90, 0xF0, ^byte(0xF0), THVTSingleHeader, false, true},
		THArrayMulti:    {"Array+", 0x88, 0xF8, ^byte(0xF8), THVTMultiHeader, false, true},
		THPosNumSingle:  {"PosNum", 0xA0, 0xF8, ^byte(0xF8), THVTSingleHeader, true, false},
		THNegNumSingle:  {"NegNum", 0xA8, 0xF8, ^byte(0xF8), THVTSingleHeader, true, false},
		THPosBigInt:     {"PosNum+", 0xB0, 0xF8, ^byte(0xF8), THVTMultiHeader, true, false},
		THNegBigInt:     {"NegNum+", 0xB8, 0xF8, ^byte(0xF8), THVTMultiHeader, true, false},
		THStringSingle:  {"String", 0xC0, 0xE0, ^byte(0xE0), THVTSingleHeader, true, false},
		THStringMulti:   {"String+", 0xE0, 0xF8, ^byte(0xF8), THVTMultiHeader, true, false},
		THVersion:       {"Ver", 0xF0, 0xF0, ^byte(0xF0), THVTByte, false, false},
		THVersionSingle: {"Ver+", 0xE8, 0xF8, ^byte(0xF8), THVTSingleHeader, false, false},
	}

	// primitive kind to valid TypeHeaders
	primKindTypeHeaderMap = map[reflect.Kind]map[TypeHeader]typeReaderFunc{
		reflect.Int:     intReaders,
		reflect.Int8:    intReaders,
		reflect.Int16:   intReaders,
		reflect.Int32:   intReaders,
		reflect.Int64:   intReaders,
		reflect.Uint:    uintReaders,
		reflect.Uint8:   uintReaders,
		reflect.Uint16:  uintReaders,
		reflect.Uint32:  uintReaders,
		reflect.Uint64:  uintReaders,
		reflect.Float32: floatReaders,
		reflect.Float64: floatReaders,
		reflect.Bool:    boolReaders,
		reflect.String:  stringReaders,
	}

	// cache for structFields
	typeInfoMap = new(sync.Map)

	// serialize/deserialize self
	TypeOfEncoderPtr = reflect.TypeOf((*Encoder)(nil))
	TypeOfEncoder    = TypeOfEncoderPtr.Elem()
	TypeOfDecoderPtr = reflect.TypeOf((*Decoder)(nil))
	TypeOfDecoder    = TypeOfDecoderPtr.Elem()

	// errors
	ErrUnsupported        = errors.New("unsupported")
	ErrNestingOverflow    = fmt.Errorf("nesting overflow: %d times", MaxNested)
	ErrInsufficientLength = errors.New("insufficient length of the slice")
	ErrDecode             = errors.New("decode error")
	ErrLength             = errors.New("length error")
	ErrTooLarge           = errors.New("too large to create")
	ErrDecodeIntoNil      = errors.New("rtl: decode pointer MUST NOT be nil")
	ErrDecodeNoPtr        = errors.New("rtl: value being decode MUST be a pointer")

	ErrEmptyStack   = errors.New("rtl: stack is empty")
	ErrInvalidValue = errors.New("invalid reflect.Value")
)

type headMaker struct{}

// stringBuf put string header into buf, len(buf) must bigger or equal to the number of header bytes
func (headMaker) stringBuf(length int, buf []byte) (int, error) {
	if length <= 0 {
		return 0, nil
	}

	if length <= 32 {
		buf[0] = headerTypeMap[THStringSingle].WithNumber(byte(length))
		return 1, nil
	}

	l, err := Numeric.writeUint(buf[1:], uint64(length))
	if err != nil {
		return 0, err
	}
	buf[0] = headerTypeMap[THStringMulti].WithNumber(byte(l))
	return l + 1, nil
}

func (h headMaker) string(length int) ([]byte, error) {
	if length <= 0 {
		return nil, nil
	}
	r := make([]byte, 9)
	l, err := h.stringBuf(length, r)
	if err != nil {
		return nil, err
	}
	return r[:l], nil
}

// numericBuf put numeric header into buf, len(buf) must bigger or equal to the number of header bytes
func (headMaker) numericBuf(isNegative bool, length int, buf []byte) (int, error) {
	if length <= 0 {
		return 0, nil
	}

	if length <= 8 {
		// single byte header
		if isNegative {
			buf[0] = headerTypeMap[THNegNumSingle].WithNumber(byte(length))
		} else {
			buf[0] = headerTypeMap[THPosNumSingle].WithNumber(byte(length))
		}
		return 1, nil
	}

	// multi bytes header
	l, err := Numeric.writeUint(buf[1:], uint64(length))
	if err != nil {
		return 0, err
	}
	if isNegative {
		buf[0] = headerTypeMap[THNegBigInt].WithNumber(byte(l))
	} else {
		buf[0] = headerTypeMap[THPosBigInt].WithNumber(byte(l))
	}
	return l + 1, nil
}

func (h headMaker) numeric(isNegative bool, length int) ([]byte, error) {
	if length <= 0 {
		return nil, nil
	}

	if length <= 8 {
		r := make([]byte, 1)
		h.numericBuf(isNegative, length, r)
		return r, nil
	}

	r := make([]byte, 9)
	l, err := h.numericBuf(isNegative, length, r)
	if err != nil {
		return nil, err
	}
	return r[:l], nil
}

// arrayBuf put header of an array into buf, len(buf) must bigger or equal to the number of header bytes
func (headMaker) arrayBuf(length int, buf []byte) (int, error) {
	if length <= 0 {
		return 0, nil
	}

	if length <= 16 {
		buf[0] = headerTypeMap[THArraySingle].WithNumber(byte(length))
		return 1, nil
	}

	l, err := Numeric.writeUint(buf[1:], uint64(length))
	if err != nil {
		return 0, err
	}
	buf[0] = headerTypeMap[THArrayMulti].WithNumber(byte(l))
	return l + 1, nil
}

func (h headMaker) array(length int) ([]byte, error) {
	if length <= 0 {
		return nil, nil
	}
	r := make([]byte, 9)
	l, err := h.arrayBuf(length, r)
	if err != nil {
		return nil, err
	}
	return r[:l], nil
}

func (headMaker) versionBuf(version uint64, buf []byte) (int, error) {
	if version <= 15 {
		buf[0] = headerTypeMap[THVersion].WithNumber(byte(version))
		return 1, nil
	}
	l, err := Numeric.writeUint(buf[1:], version)
	if err != nil {
		return 0, err
	}
	buf[0] = headerTypeMap[THVersionSingle].WithNumber(byte(l))
	return l + 1, nil
}

func (h headMaker) version(version uint64) ([]byte, error) {
	r := make([]byte, 9)
	l, err := h.versionBuf(version, r)
	if err != nil {
		return nil, err
	}
	return r[:l], nil
}

type fieldName struct {
	index int
	name  string
	order int
	// version is used to distinguish the fields added in different versions of the struct type upgrade
	// version can only increase sequentially
	version int
}

func (f fieldName) String() string {
	return fmt.Sprintf("field{%d-%s, order:%d, version:%d}", f.index, f.name, f.order, f.version)
}

func structFields(typ reflect.Type) (fieldNum int, fields []fieldName) {
	rv, ok := typeInfoMap.Load(typ)
	if ok {
		fields, _ = rv.([]fieldName)
		fieldNum = 0
		if len(fields) > 0 {
			fieldNum = fields[len(fields)-1].order + 1
		}
		return
	}
	for i := 0; i < typ.NumField(); i++ {
		// exported field
		if f := typ.Field(i); f.PkgPath == "" {
			tagStr := f.Tag.Get("rtl")
			ignored := false
			for _, tag := range strings.Split(tagStr, ",") {
				switch tag = strings.TrimSpace(tag); tag {
				case "-":
					ignored = true
				}
			}
			if ignored {
				continue
			}

			order := -1
			tagStr = f.Tag.Get("rtlorder")
			tagStr = strings.TrimSpace(tagStr)
			if len(tagStr) > 0 {
				if oi, err := strconv.Atoi(tagStr); err != nil {
					panic(fmt.Errorf("illegal rtlorder (%s) for field %s of type %s",
						tagStr, f.Name, typ.Name()))
				} else {
					if oi < 0 {
						panic(fmt.Errorf("illegal rtlorder (%s) for field %s of type %s",
							tagStr, f.Name, typ.Name()))
					}
					order = oi
				}
			}

			version := -1
			tagStr = f.Tag.Get("rtlversion")
			tagStr = strings.TrimSpace(tagStr)
			if len(tagStr) > 0 {
				if oi, err := strconv.Atoi(tagStr); err != nil {
					panic(fmt.Errorf("illegal rtlversion (%s) for filed %s of type %s",
						tagStr, f.Name, typ.Name()))
				} else {
					if oi < 0 {
						panic(fmt.Errorf("illegal rtlversion (%s) for filed %s of type %s",
							tagStr, f.Name, typ.Name()))
					}
					version = oi
				}
			}

			fields = append(fields, fieldName{i, f.Name, order, version})
		}
	}
	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i].order > fields[j].order {
			return false
		}
		if fields[i].order < fields[j].order {
			return true
		}
		return fields[i].index < fields[j].index
	})
	for i := 0; i < len(fields); i++ {
		if fields[i].order < 0 {
			fields[i].order = i
		} else {
			if fields[i].order < i {
				panic(fmt.Errorf("illegal rtlorder (%d) for field %s of type %s, should >= %d",
					fields[i].order, fields[i].name, typ.Name(), i))
			}
			// // fields have been orderred by order, there's no field.order < i
			// break
		}
		if fields[i].version < 0 {
			if i == 0 {
				fields[i].version = 0
			} else {
				fields[i].version = fields[i-1].version
			}
		} else {
			if i > 0 {
				if fields[i].version < fields[i-1].version {
					panic(fmt.Errorf("illegal rtlversion (%d) for field %s of type %s, should >= %d",
						fields[i].version, fields[i].name, typ.Name(), fields[i-1].version))
				}
			}
		}
	}
	// fmt.Printf("%s -> %s\n", typ.Name(), fields)
	typeInfoMap.Store(typ, fields)
	fieldNum = 0
	if len(fields) > 0 {
		fieldNum = fields[len(fields)-1].order + 1
	}
	return
}

// When any field under a certain version is not a zero value, all fields not greater than this version are reserved.
// All fields with version==0 will be reserved
func versionedFields(val reflect.Value, fields []fieldName) (int, []fieldName) {
	if len(fields) == 0 {
		return 0, nil
	}
	maxIndex := len(fields) - 1
	maxVersion := fields[maxIndex].version
	if maxVersion == 0 || maxVersion == fields[0].version {
		return fields[maxIndex].order + 1, fields
	}
	for i := len(fields) - 1; i >= 0; i-- {
		if maxVersion > fields[i].version {
			maxVersion = fields[i].version
			maxIndex = i
		}
		if maxVersion == 0 {
			break
		}
		if v := val.Field(fields[i].index); v.IsZero() {
			continue
		}
		break
	}
	return fields[maxIndex].order + 1, fields[:maxIndex+1]
}

type StructCodec struct {
	structType reflect.Type
	isPtr      bool
}

func NewStructCodec(typ reflect.Type) (*StructCodec, error) {
	if typ == nil {
		return nil, errors.New("NewStructCodec: struct type should not be nil")
	}
	kind := typ.Kind()
	if kind != reflect.Struct {
		if kind == reflect.Ptr {
			if typ = typ.Elem(); typ.Kind() != reflect.Struct {
				panic("type of value must be a struct of ptr to a struct")
			}
		} else {
			panic("type of value must be a struct of ptr to a struct")
		}
	}
	ret := &StructCodec{structType: typ, isPtr: kind == reflect.Ptr}
	return ret, nil
}

func (c *StructCodec) Encode(o interface{}, w io.Writer) error {
	return Encode(o, w)
}

func (c *StructCodec) Decode(r io.Reader) (interface{}, error) {
	val := reflect.New(c.structType)
	if err := Decode(r, val.Interface()); err != nil {
		return nil, err
	}
	if c.isPtr {
		return val.Interface(), nil
	}
	return val.Elem().Interface(), nil
}
