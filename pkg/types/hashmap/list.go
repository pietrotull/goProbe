package hashmap

import (
	"bytes"
	"sort"

	"github.com/els0r/goProbe/pkg/types"
)

// Item denotes a flat key / value pair
type Item struct {
	types.Key
	Val
}

// List denotes a list of key / value pairs
type List []Item

// Flatten converts a flow map to a flat table / list
func (a AggFlowMap) Flatten() (v4List List, v6List List) {
	v4List, v6List = make(List, 0, a.V4Map.count), make(List, 0, a.V6Map.count)

	for i := a.V4Map.Iter(); i.Next(); {
		v4List = append(v4List, Item{i.Key(), i.Val()})
	}
	for i := a.V6Map.Iter(); i.Next(); {
		v6List = append(v6List, Item{i.Key(), i.Val()})
	}

	return
}

// Sort orders relevant flow columns so that they become more compressible
func (l List) Sort() List {
	sort.Slice(l, func(i, j int) bool {

		iv, jv := l[i], l[j]

		if comp := bytes.Compare(iv.GetSip(), jv.GetSip()); comp != 0 {
			return comp < 0
		}
		if comp := bytes.Compare(iv.GetDip(), jv.GetDip()); comp != 0 {
			return comp < 0
		}
		if comp := bytes.Compare(iv.GetDport(), jv.GetDport()); comp != 0 {
			return comp < 0
		}
		if iv.GetProto() != jv.GetProto() {
			return iv.GetProto() < jv.GetProto()
		}

		return false
	})

	return l
}
