Compact Float
=============

A go implementation of [compact float](https://github.com/kstenerud/compact-float/blob/master/compact-float-specification.md).



Library Usage
-------------

```golang

import (
	"fmt"

	"github.com/cockroachdb/apd"
	"github.com/kstenerud/go-compact-float"
)

func demonstrateEncodeDecode() {
	originalValue, _, err := apd.NewFromString("-9.4452837206285466345998345667683453466347345e-5000")
	if err != nil {
		// TODO: Handle error
	}

	buffer := make([]byte, 50)
	bytesEncoded, ok := compact_float.Encode(originalValue, buffer)
	if !ok {
		// TODO: There wasn't enough room to encode
	}
	buffer = buffer[:bytesEncoded]
	fmt.Printf("%v encoded = %v\n", originalValue, buffer)

	decodedValue, bytesDecoded, err := Decode(buffer)
	if err != nil {
		// TODO: Check if is compact_float.ErrorIncomplete or something else
	}
	fmt.Printf("%v decoded (%v bytes) = %v\n", buffer, bytesDecoded, decodedValue)

	// Prints:
	// -9.4452837206285466345998345667683453466347345E-5000 encoded = [207 157 1 209 142 162 230 131 138 191 193 187 225 243 223 252 238 172 229 254 225 143 226 67]
	// [207 157 1 209 142 162 230 131 138 191 193 187 225 243 223 252 238 172 229 254 225 143 226 67] decoded (24 bytes) = -9.4452837206285466345998345667683453466347345E-5000
}

func demonstrateEncodeDecodeFloat64() {
	originalValue := 0.1473445219134543
	significantDigits := 6
	buffer := make([]byte, 15)
	bytesEncoded, ok := compact_float.EncodeFloat64(originalValue, significantDigits, buffer)
	if !ok {
		// TODO: There wasn't enough room to encode
	}
	buffer = buffer[:bytesEncoded]
	fmt.Printf("%v encoded using %d significant digits = %v\n", originalValue, significantDigits, buffer)

	decodedValue, bytesDecoded, err := Decode(buffer)
	if err != nil {
		// TODO: Check if is compact_float.ErrorIncomplete or something else
	}
	fmt.Printf("%v decoded (%v bytes) = %v\n", buffer, bytesDecoded, decodedValue)

	// Prints:
	// 0.1473445219134543 encoded using 6 significant digits = [26 145 255 8]
	// [26 145 255 8] decoded (4 bytes) = 0.147345
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
