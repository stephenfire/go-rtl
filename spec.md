# Recursive Typed and Length-prefixed Object Encoding (RTL)

[TOC]

## summary

| header code                                   | pattern         |
| --------------------------------------------- | --------------- |
| single byte                                   | 0xxxxxxx        |
| zero value/false of bool                      | 10000000        |
| true of bool                                  | 10000001        |
| empty value                                   | 10000010        |
| <u>*reserved*</u>                             | <u>10000011</u> |
| <u>*reserved*</u>                             | <u>100001xx</u> |
| array(single byte header)                     | 1001xxxx        |
| array(multi bytes header)                     | 10001xxx        |
| positive numeric(single byte header)          | 10100xxx        |
| negative numeric(single byte header)          | 10101xxx        |
| positive numeric(multi bytes header) big.Int  | 10110xxx        |
| negative numeric(multi bytes header) big.Int  | 10111xxx        |
| string = (array of bytes)(single byte header) | 110xxxxx        |
| string = (array of bytes)(multi bytes header) | 11100xxx        |
| struct version(single byte value)             | <u>1111xxxx</u> |
| struct version(single byte header)            | <u>11101xxx</u> |

## single byte

one byte for byte value <= 127 (0x7F)

## basic value

### zero value

zero value of all type(nil for pointer/slice/map, false of boolean,"" for string, empty for array, 0 for numeric)

### true of bool

true value of bool type

### emtpy value

empty value of slice, map

## array value

- array: followed by elements in array, **byte array excluded (use string instead).**

- map: even index is the key, odd index is the value

- struct: one property of the struct is an element in the array

### single byte header

- bit[7-4]:'1001'
- bit[3-0]: the number of the length of the array. '0000': 16bytes, '0001': 1byte, '0010': 2bytes,...,'1111': 15bytes
- empty array is a zero value, use *zero value ['10000000']* instead
- followed by bytes in array from lower index to upper index

### multi byte header

- bit[7-3]='10001', bit[2-0] is the length of the number of the elements of the array, followed by

`bit[2-0]=len(trimPrefixZeros(hex(len(array))))`

- max bytes in the length number is 8: '000': 8bytes, '001':1byte, '010':2bytes,...,'111':7bytes
- empty array is a zero value, use *zero value ['10000000']* instead
- the max number of elements in the array is $2^{8*8}$
- followed by big-endian prefix-zero-trimed hexadecimal bytes of the length
- followed by bytes in array from lower index to upper index

## Numeric

Signed and unsigned integer, float, big.Int values are supported. We use one bit to identify the positive and negative of the number.

### single byte header

- bit[7-4]: '1010'
- bit[3]: '0': positive number, '1': negative number
- bit[2-0]: the number of the hexadecimal bytes of the number. '000': 8bytes, '001':1byte, '010':2bytes,...,'111':7bytes.
- zero byte not support, use *zero value ['10000000']* instead
- followed by big-endian prefix-zero-trimed hexadecimal bytes of the absolute value of the number.
- for int/int8/int16/int32/int64/uint/uint8/uint16/uint32/uint64/float32/float64 and all there aliases

### multi bytes header

- bit[7-4]: '1011'
- bit[3]: '0': positive number, '1': negative number
- bit[2-0]: the number of the hexadecimal bytes of length of the number. '000': 8bytes, '001':1byte, '010':2bytes,...,'111':7bytes
- zero byte not support, use *zero value ['10000000']* instead
- the max bytes length of the number is: $2^{8*8}$
- followed by big-endian prefix-zero-trimed hexadecimal bytes of the number of the length.
- followed by hexadecimal bytes of the number
- for big.Int

## string & byte array

### single byte header

- bit[7-5]: '110'
- bit[4-0]: the number of the bytes in string. '00000': 32bytes, '00001': 1byte, '00010': 2bytes,..., '11111': 31bytes
- zero byte string is an empty string, use *zero value ['10000000']* instead

### multi bytes header

- bit[7-3]: '11100'
- bit[2-0]: the number of the bytes of the number of the bytes in string. max number of bytes is 8, '000': 8bytes, '001':1byte, '010':2bytes,...,'111':7bytes.
- zero byte string is an empty string, use *zero value ['10000000']* instead
- the max length of the string is: $2^{8*8}$
- followed by big-endian prefix-zero-trimed hexadecimal bytes of the number of the length
- followed by bytes in UTF-8 of the string

## struct version

struct version is an unsigned number to distinguish between different version of struct stream data.

### single byte value

- bit[7-4]: '1111'
- bit[3-0]: the version number of the struct, '0000'-'1111' means version 0 to 15
- default struct version is 0
- struct version absentness means version equals 0

### single byte header

- bit[7-3]: '11101'
- bit[2-0]: the number of the bytes of the version number. Max number of bytes is 8, '000': 8bytes, '001':1byte, '010':2bytes, ..., '111':7bytes.
- followed by big-endian prefix-zero-trimed hexadecimal bytes of the version number
- the max version number is $2^{8*8}$