Compact Float
=============

A go implementation of [compact float](https://github.com/kstenerud/compact-float/blob/master/compact-float-specification.md).



Library Usage
-------------

```golang
func demonstrateEncodeDecode() {
	originalValue := 0.1473445219134543
	significantDigits := 6
	buffer := make([]byte, 15)
	bytesEncoded, ok := compact_float.Encode(originalValue, significantDigits, buffer)
	if !ok {
		// TODO: There wasn't enough room to encode
	}
	buffer = buffer[:bytesEncoded]
	fmt.Printf("%v encoded using %d significant digits = %v\n", originalValue, significantDigits, buffer)

	decodedValue, decodedDigits, bytesDecoded, ok := compact_float.Decode(buffer)
	if !ok {
		// TODO: The buffer has been truncated
	}
	fmt.Printf("%v decoded %v bytes = %v with %v significant digits\n", buffer, bytesDecoded, decodedValue, decodedDigits)

	// Prints:
	// 0.1473445219134543 encoded using 6 significant digits = [26 136 255 17]
	// [26 136 255 17] decoded 4 bytes = 0.147345 with 6 significant digits
}
```



License
-------

MIT License:

Copyright 2019 Karl Stenerud

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
