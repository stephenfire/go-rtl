# go-rtl
使用RTL(Recursive Typed and Length-prefixed)编码方式的golang对象的序列化及反序列化方法库。目前尚不支持其他语言。

### 特性

- 使用简单，可以将任意对象序列化成字节流，并反序列化为兼容的对象。
- 序列化后，除原值外的附加信息很少。
- struct类型升级可以兼容，即不同版本数据可以反序列化为不同版本类型的对象。
- 支持类型的自定义序列化方式，可以与通用方法混用。

### 使用限制

- 因为使用反射，序列化会比硬编码的序列化方式慢。
- 因为无法确定代码中希望为哪种类型对象，无法将数据反序列化至interface{}中。但序列化是可以的，因为此时的interface{}类型是明确的。
- 在struct类型升级兼容性中，不支持删除其中的自定义序列化方式的字段。
