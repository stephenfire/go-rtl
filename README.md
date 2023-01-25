# go-rtl

这是一款使用简单，灵活方便，兼容性强的序列化工具。使用RTL(Recursive Typed and Length-prefixed)编码方式的golang对象的序列化及反序列化方法库。目前尚不支持其他语言。

由于go基础库中不存在通用序列化工具；且如protobuf的第三方序列化需要预定义文件及相应的生成工具，当序列化类型经常变化或很多时，工作会显得有些繁琐。为了能够简单使用而开发这个通用序列化工具包。

## 特性

- 使用简单，可以将任意对象序列化成字节流，并反序列化为兼容的对象。
- 序列化后，除原值外的附加信息很少。
- struct类型升级可以兼容，即不同版本数据可以反序列化为不同版本类型的对象。
- 支持类型的自定义序列化方式，可以与通用方法混用。

## 使用限制

- 因为使用反射，序列化会比硬编码的序列化方式慢。
- 因为无法确定代码中希望为哪种类型对象，无法将数据反序列化至interface{}中。但序列化是可以的，因为此时的interface{}类型是明确的。
- 在struct类型升级兼容性中，不支持删除其中的自定义序列化方式的字段。

## 使用方法

### 1. 引用

go.mod

```
require github.com/stephenfire/go-rtl v1.0.4
```

go文件

```go
import "github.com/stephenfire/go-rtl"
```

### 2. 序列化对象

由于使用reflect包，所以struct只有public的属性才可以被序列化和反序列化。

```go
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
```

```go
    buf := new(bytes.Buffer)
    if err := Encode(obj, buf); err != nil {
        t.Fatal(err)
    }
    bs := buf.Bytes()
```

或

```go
    bs, err := Marshal(obj)
    if err != nil {
        t.Fatal(err)
    }
```

### 3. 反序列化对象，注意必须传入对象指针

```go
    decodedObj := new(basic)
    if err := Decode(bytes.NewReader(bs), decodedObj); err != nil {
        t.Fatal(err)
    }
```

或

```go
    decodedObj := new(basic)
    if err := Unmarshal(bs, decodedObj); err != nil {
        t.Fatal(err)
    }
```

### 4. 基础类型序列化

基本上所有类型均可

```go
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
```

### 5. 类型兼容转化

结构对象序列化的原数据与目标数据兼容时即可正常反序列化，与属性位置相关

```go
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
```

输出：

```
&{A:[97 32 115 116 114 105 110 103] B:[1 2 3 4]} -> &{C:a string D:[1 2 3 4]}
```



### 6. 结构类型的版本兼容性

通过使用标记 *rtlorder* 对结构属性进行排序。

序列化时，按rtlorder的顺序写入buffer，遇到不连续的情况时，用ZeroValue占位。

反序列化时，按rtlorder的顺序读buffer，遇到不连续的情况时，跳过buffer中对应的位置；如果buffer数据不足，则反序列化对象中的后续属性均为缺省零值。

```go
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
```

输出：

```
&{A:1 B:2 C:Charlie D:[110 111 116 32 105 110]} -> &{E:<nil> F:0 C:Charlie B:2}
```

