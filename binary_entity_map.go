// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package gosmonaut

import (
	"math"
	"sort"
)

// binaryNodeEntityMap uses a binary search table for storing
// entites. It is well suited in this case since we only have to sort it once
// between the reads and writes. Also we avoid the memory overhead of storing
// the IDs twice and instead just read them from the struct. The (fake-)generic
// solution performs much better than it would using the OSMEntity interface.
type binaryNodeEntityMap struct {
	buckets [][]Node
	n       uint64
}

func newBinaryNodeEntityMap(n int) *binaryNodeEntityMap {
	// Calculate the number of buckets, exponent defines max number of lookups
	nb := n / int(math.Pow(2, 20))
	if nb < 1 {
		nb = 1
	}

	// Calculate the bucket sizes. Leave a small array overhead since the
	// distribution is not completely even.
	var bs int
	if nb == 1 {
		bs = n
	} else {
		bs = int(float64(n/nb) * 1.05)
	}

	// Create the buckets
	buckets := make([][]Node, nb)
	for i := 0; i < nb; i++ {
		buckets[i] = make([]Node, 0, bs)
	}
	return &binaryNodeEntityMap{
		buckets: buckets,
		n:       uint64(nb),
	}
}

func (m *binaryNodeEntityMap) hash(id int64) uint64 {
	return uint64(id) % m.n
}

// Must not be called after calling prepare()
func (m *binaryNodeEntityMap) add(e Node) {
	h := m.hash(e.ID)
	m.buckets[h] = append(m.buckets[h], e)
}

// Must be called between the last write and the first read.
func (m *binaryNodeEntityMap) prepare() {
	// Sort buckets
	for _, b := range m.buckets {
		sort.Slice(b, func(i, j int) bool {
			return b[i].ID < b[j].ID
		})
	}
}

// Must not be called before calling prepare()
func (m *binaryNodeEntityMap) get(id int64) (Node, bool) {
	h := m.hash(id)
	bucket := m.buckets[h]

	// Binary search (we can't use sort.Search as we use int64)
	lo := 0
	hi := len(bucket) - 1
	for lo <= hi {
		mid := (lo + hi) / 2
		midID := bucket[mid].ID

		if midID < id {
			lo = mid + 1
		} else if midID > id {
			hi = mid - 1
		} else {
			return bucket[mid], true
		}
	}
	return Node{}, false
}

// binaryWayEntityMap uses a binary search table for storing
// entites. It is well suited in this case since we only have to sort it once
// between the reads and writes. Also we avoid the memory overhead of storing
// the IDs twice and instead just read them from the struct. The (fake-)generic
// solution performs much better than it would using the OSMEntity interface.
type binaryWayEntityMap struct {
	buckets [][]Way
	n       uint64
}

func newBinaryWayEntityMap(n int) *binaryWayEntityMap {
	// Calculate the number of buckets, exponent defines max number of lookups
	nb := n / int(math.Pow(2, 20))
	if nb < 1 {
		nb = 1
	}

	// Calculate the bucket sizes. Leave a small array overhead since the
	// distribution is not completely even.
	var bs int
	if nb == 1 {
		bs = n
	} else {
		bs = int(float64(n/nb) * 1.05)
	}

	// Create the buckets
	buckets := make([][]Way, nb)
	for i := 0; i < nb; i++ {
		buckets[i] = make([]Way, 0, bs)
	}
	return &binaryWayEntityMap{
		buckets: buckets,
		n:       uint64(nb),
	}
}

func (m *binaryWayEntityMap) hash(id int64) uint64 {
	return uint64(id) % m.n
}

// Must not be called after calling prepare()
func (m *binaryWayEntityMap) add(e Way) {
	h := m.hash(e.ID)
	m.buckets[h] = append(m.buckets[h], e)
}

// Must be called between the last write and the first read.
func (m *binaryWayEntityMap) prepare() {
	// Sort buckets
	for _, b := range m.buckets {
		sort.Slice(b, func(i, j int) bool {
			return b[i].ID < b[j].ID
		})
	}
}

// Must not be called before calling prepare()
func (m *binaryWayEntityMap) get(id int64) (Way, bool) {
	h := m.hash(id)
	bucket := m.buckets[h]

	// Binary search (we can't use sort.Search as we use int64)
	lo := 0
	hi := len(bucket) - 1
	for lo <= hi {
		mid := (lo + hi) / 2
		midID := bucket[mid].ID

		if midID < id {
			lo = mid + 1
		} else if midID > id {
			hi = mid - 1
		} else {
			return bucket[mid], true
		}
	}
	return Way{}, false
}
