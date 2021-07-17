## Adaptive Radix Tree

[The Adaptive Radix Tree: ARTful Indexing for Main-Memory Databases](https://db.in.tum.de/~leis/papers/ART.pdf)

#### Usage

```go
package main

import (
	"fmt"
	
    "github.com/rayzui/art"
)

func main() {
    tree := art.New()

    tree.Insert([]byte("key"), "value")
    value := tree.Search([]byte("key"))
    fmt.Println(value)

    tree.Each(func(n Node) {
    	fmt.Println(n.Key(), n.Value())
    })
}
```
