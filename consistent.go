package tinyconsistent

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/minio/blake2b-simd"
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

const replicationFactor = 10

// ErrNoHosts .
var ErrNoHosts = errors.New("no hosts added")

// Host .
type Host struct {
	Name string
	Load int64
}

// Consistent .
type Consistent struct {
	hosts     map[uint64]string
	sortedSet []uint64
	loadMap   map[string]*Host
	totalLoad int64

	sync.RWMutex
}

// New create a Consistent
func New() *Consistent {
	return &Consistent{
		hosts:     map[uint64]string{},
		sortedSet: []uint64{},
		loadMap:   map[string]*Host{},
	}
}

// Add .
func (c *Consistent) Add(host string) {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.loadMap[host]; ok {
		return
	}

	c.loadMap[host] = &Host{
		Name: host,
		Load: 0,
	}
	for i := 0; i < replicationFactor; i++ {
		h := c.hash(fmt.Sprintf("%s%d", host, i))
		c.hosts[h] = host
		c.sortedSet = append(c.sortedSet, h)
	}

	sort.Slice(c.sortedSet, func(i int, j int) bool {
		if c.sortedSet[i] < c.sortedSet[j] {
			return true
		}
		return false
	})
}

// Get the location
func (c *Consistent) Get(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.hosts) == 0 {
		return "", ErrNoHosts
	}

	h := c.hash(key)
	idx := c.search(h)
	return c.hosts[c.sortedSet[idx]], nil
}

// GetLeast .
func (c *Consistent) GetLeast(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.hosts) == 0 {
		return "", ErrNoHosts
	}

	h := c.hash(key)
	idx := c.search(h)

	i := idx
	for {
		host := c.hosts[c.sortedSet[i]]
		if c.loadOK(host) {
			return host, nil
		}
		i++
		if i >= len(c.hosts) {
			i = 0
		}
	}
}

func (c *Consistent) search(key uint64) int {
	idx := sort.Search(len(c.sortedSet), func(i int) bool {
		return c.sortedSet[i] >= key
	})

	if idx >= len(c.sortedSet) {
		idx = 0
	}
	return idx
}

// UpdateLoad .
func (c *Consistent) UpdateLoad(host string, load int64) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}

	c.totalLoad -= c.loadMap[host].Load
	c.loadMap[host].Load = load
	c.totalLoad += load
}

// Inc .
func (c *Consistent) Inc(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}
	atomic.AddInt64(&c.loadMap[host].Load, 1)
	atomic.AddInt64(&c.totalLoad, 1)
}

// Done .
func (c *Consistent) Done(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}

	atomic.AddInt64(&c.loadMap[host].Load, -1)
	atomic.AddInt64(&c.loadMap[host].Load, -1)
}

// Remove .
func (c *Consistent) Remove(host string) bool {
	c.Lock()
	defer c.Unlock()

	for i := 0; i < replicationFactor; i++ {
		h := c.hash(fmt.Sprintf("%s%d", host, i))
		delete(c.hosts, h)
		c.delSlice(h)
	}
	delete(c.loadMap, host)
	return true
}

// Hosts .
func (c *Consistent) Hosts() (hosts []string) {
	c.RLock()
	defer c.RUnlock()
	for k, _ := range c.loadMap {
		hosts = append(hosts, k)
	}
	return hosts
}

// GetLoads .
func (c *Consistent) GetLoads() map[string]int64 {
	loads := map[string]int64{}

	for k, v := range c.loadMap {
		loads[k] = v.Load
	}

	return loads
}

// MaxLoad returns the maximum load of the single host
// which is :
// (total_load/number_of_hosts)*1.25
// total_load is the total number of active requests served by hosts
func (c *Consistent) MaxLoad() int64 {
	if c.totalLoad == 0 {
		c.totalLoad = 1
	}

	var avgLoadPerNode float64
	avgLoadPerNode = float64(c.totalLoad / int64(len(c.loadMap)))
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * 1.25)
	return int64(avgLoadPerNode)
}

// loadOK .
func (c *Consistent) loadOK(host string) bool {
	// a safety check if someone performed c.Done more than needed
	if c.totalLoad < 0 {
		c.totalLoad = 0
	}

	var avgLoadPerNode float64
	avgLoadPerNode = float64(c.totalLoad / int64(len(c.loadMap)))
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * 1.25)

	bHost, ok := c.loadMap[host]
	if !ok {
		panic(fmt.Sprintf("given host(%s) not in loadMap", bHost.Name))
	}

	if float64(bHost.Load)+1 <= avgLoadPerNode {
		return true
	}
	return false
}

func (c *Consistent) delSlice(val uint64) {
	idx := -1
	l := 0
	r := len(c.sortedSet) - 1
	for l <= r {
		m := (l + r) / 2
		curVal := c.sortedSet[m]
		if curVal == val {
			idx = m
			break
		} else if curVal < val {
			l = m + 1
		} else {
			l = m - 1
		}
	}
	if idx != -1 {
		c.sortedSet = append(c.sortedSet[:idx], c.sortedSet[idx+1:]...)
	}
}

func (c *Consistent) hash(key string) uint64 {
	out := blake2b.Sum512([]byte(key))
	return binary.LittleEndian.Uint64(out[:])
}
