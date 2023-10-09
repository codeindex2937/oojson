# oojson

Generate code from JSON

## Example

```go
d := json.NewDecoder(bytes.NewBufferString(`{
  "name": "Tom",
  "age": 50,
  "height": 175.0,
  "trace": [[0.0, 0.0], [0.1, 0.2]],
  "properties": [{"type": "pet", "value": "cat"}]
}`))
d.UseNumber()

var obj interface{}
if err := d.Decode(&obj); err != nil {
	log.Fatal(err)
}

value := &oojson.Value{}
value.Observe(obj)

rawCode, _ := oojson.GetGoType(value, 0, oojson.DefaultGoOption())
goCode, err := format.Source([]byte(rawCode))
if err != nil {
	log.Fatal(err)
}
fmt.Printf("go:\n%v\n", string(goCode))

_, javaCode := oojson.GetJavaType(value, "Test", "  ", oojson.DefaultJavaOption())
fmt.Printf("java:\n%v\n", string(javaCode))
```

## Reference

[go-jsonstruct](https://github.com/twpayne/go-jsonstruct)
