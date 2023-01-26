> **Reference**
> 
> [知乎-负载均衡器](https://zhuanlan.zhihu.com/p/506415782)
> 
> [知乎-一致性哈希](https://zhuanlan.zhihu.com/p/34969168)
> 
> [GitHub](https://github.com/lafikl/consistent)
> 

# Package consistent
A Golang implementation of Consistent Hashing and Consistent Hashing With Bounded Loads.

https://en.wikipedia.org/wiki/Consistent_hashing

https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html


### Consistent Hashing Example

```go

package main

import (
	"log"
	"github.com/lafikl/consistent"
)

func main() {
	c := consistent.New()

	// adds the hosts to the ring
	c.Add("127.0.0.1:8000")
	c.Add("92.0.0.1:8000")

	// Returns the host that owns `key`.
	//
	// As described in https://en.wikipedia.org/wiki/Consistent_hashing
	//
	// It returns ErrNoHosts if the ring has no hosts in it.
	host, err := c.Get("/app.html")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(host)
}

```


### Consistent Hashing With Bounded Loads Example

```go

package main

import (
	"log"
	"github.com/lafikl/consistent"
)

func main() {
	c := consistent.New()

	// adds the hosts to the ring
	c.Add("127.0.0.1:8000")
	c.Add("92.0.0.1:8000")

	// It uses Consistent Hashing With Bounded loads
	// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
	// to pick the least loaded host that can serve the key
	//
	// It returns ErrNoHosts if the ring has no hosts in it.
	//
	host, err := c.GetLeast("/app.html")
	if err != nil {
		log.Fatal(err)
	}

	// increases the load of `host`, we have to call it before sending the request
	c.Inc(host)

	// send request or do whatever
	log.Println("send request to", host)

	// call it when the work is done, to update the load of `host`.
	c.Done(host)

}

```


## Docs

https://godoc.org/github.com/lafikl/consistent