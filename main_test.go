package tinyconsistent

import (
	"log"
	"testing"
)

func Test_Consistent_Hashing(t *testing.T) {
	c := New()

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
		log.Fatalln(err)
	}
	log.Printf(host)
}

func Test_Consistent_Hashing_With_Bounded_Loads(t *testing.T) {
	c := New()

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

	// increases the load of `host`, we have to call it
	// before sending the request
	c.Inc(host)
	c.Inc(host)
	c.Inc(host)
	// send request or do whatever
	log.Println("send request to", host)

	host1, err1 := c.GetLeast("/app.html")
	if err1 != nil {
		log.Fatalln(err1)
	}
	log.Printf(host1)
	// call it when the work is done, to update the load of `host`
	c.Done(host)
	c.Done(host)
	c.Done(host)

}
