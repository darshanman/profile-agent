package internal

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
)

//BreakdownNode ...
type BreakdownNode struct {
	name        string
	measurement float64
	numSamples  int64
	reservoir   []float64
	children    map[string]*BreakdownNode
	updateLock  *sync.RWMutex
}

func newBreakdownNode(name string) *BreakdownNode {
	bn := &BreakdownNode{
		name:        name,
		measurement: 0,
		numSamples:  0,
		reservoir:   nil,
		children:    make(map[string]*BreakdownNode),
		updateLock:  &sync.RWMutex{},
	}

	return bn
}

func (bn *BreakdownNode) findChild(name string) *BreakdownNode {
	bn.updateLock.RLock()
	defer bn.updateLock.RUnlock()

	if child, exists := bn.children[name]; exists {
		return child
	}

	return nil
}

func (bn *BreakdownNode) maxChild() *BreakdownNode {
	bn.updateLock.RLock()
	defer bn.updateLock.RUnlock()

	var maxChild *BreakdownNode
	for _, child := range bn.children {
		if maxChild == nil || child.measurement > maxChild.measurement {
			maxChild = child
		}
	}
	return maxChild
}

func (bn *BreakdownNode) minChild() *BreakdownNode {
	bn.updateLock.RLock()
	defer bn.updateLock.RUnlock()

	var minChild *BreakdownNode
	for _, child := range bn.children {
		if minChild == nil || child.measurement < minChild.measurement {
			minChild = child
		}
	}
	return minChild
}

func (bn *BreakdownNode) addChild(child *BreakdownNode) {
	bn.updateLock.Lock()
	defer bn.updateLock.Unlock()

	bn.children[child.name] = child
}

func (bn *BreakdownNode) removeChild(child *BreakdownNode) {
	bn.updateLock.Lock()
	defer bn.updateLock.Unlock()

	delete(bn.children, child.name)
}

func (bn *BreakdownNode) findOrAddChild(name string) *BreakdownNode {
	child := bn.findChild(name)
	if child == nil {
		child = newBreakdownNode(name)
		bn.addChild(child)
	}

	return child
}

func (bn *BreakdownNode) filter(fromLevel int, min float64, max float64) {
	bn.filterLevel(1, fromLevel, min, max)
}

func (bn *BreakdownNode) filterLevel(currentLevel int, fromLevel int, min float64, max float64) {
	for key, child := range bn.children {
		if currentLevel >= fromLevel && (child.measurement < min || child.measurement > max) {
			delete(bn.children, key)
		} else {
			child.filterLevel(currentLevel+1, fromLevel, min, max)
		}
	}
}

func (bn *BreakdownNode) filterByName(filterFunc filterFuncType) {
	for key, child := range bn.children {
		if filterFunc(child.name) {
			child.filterByName(filterFunc)
		} else {
			delete(bn.children, key)
		}
	}
}

func (bn *BreakdownNode) depth() int {
	max := 0
	for _, child := range bn.children {
		cd := child.depth()
		if cd > max {
			max = cd
		}
	}

	return max + 1
}

func (bn *BreakdownNode) propagate() {
	for _, child := range bn.children {
		child.propagate()
		bn.measurement += child.measurement
		bn.numSamples += child.numSamples
	}
}

func (bn *BreakdownNode) increment(value float64, count int64) {
	AddFloat64(&bn.measurement, value)
	atomic.AddInt64(&bn.numSamples, count)
}

func (bn *BreakdownNode) updateP95(value float64) {
	rLen := 0
	rExists := true

	bn.updateLock.RLock()
	if bn.reservoir == nil {
		rExists = false
	} else {
		rLen = len(bn.reservoir)
	}
	bn.updateLock.RUnlock()

	if !rExists {
		bn.updateLock.Lock()
		bn.reservoir = make([]float64, 0, ReservoirSize)
		bn.updateLock.Unlock()
	}

	if rLen < ReservoirSize {
		bn.updateLock.Lock()
		bn.reservoir = append(bn.reservoir, value)
		bn.updateLock.Unlock()
	} else {
		StoreFloat64(&bn.reservoir[rand.Intn(ReservoirSize)], value)
	}

	atomic.AddInt64(&bn.numSamples, 1)
}

func (bn *BreakdownNode) evaluateP95() {
	if bn.reservoir != nil && len(bn.reservoir) > 0 {
		sort.Float64s(bn.reservoir)
		index := int(math.Floor(float64(len(bn.reservoir)) / 100.0 * 95.0))
		bn.measurement = bn.reservoir[index]

		bn.reservoir = bn.reservoir[:0]
	}

	for _, child := range bn.children {
		child.evaluateP95()
	}
}

func (bn *BreakdownNode) convertToPercentage(total float64) {
	bn.measurement = (bn.measurement / total) * 100.0
	for _, child := range bn.children {
		child.convertToPercentage(total)
	}
}

func (bn *BreakdownNode) normalize(factor float64) {
	bn.measurement = bn.measurement / factor
	bn.numSamples = int64(math.Ceil(float64(bn.numSamples) / factor))
	for _, child := range bn.children {
		child.normalize(factor)
	}
}

func (bn *BreakdownNode) clone() *BreakdownNode {
	cln := newBreakdownNode(bn.name)
	cln.measurement = bn.measurement
	cln.numSamples = bn.numSamples

	for _, child := range bn.children {
		cln.addChild(child.clone())
	}

	return cln
}

func (bn *BreakdownNode) toMap() map[string]interface{} {
	childrenMap := make([]interface{}, len(bn.children))
	i := 0
	for _, child := range bn.children {
		childrenMap[i] = child
		i++
		// childrenMap = append(childrenMap, child.toMap())
	}

	nodeMap := map[string]interface{}{
		"name":        bn.name,
		"measurement": bn.measurement,
		"num_samples": bn.numSamples,
		"children":    childrenMap,
	}

	return nodeMap
}

func (bn *BreakdownNode) printLevel(level int) string {
	str := ""

	for i := 0; i < level; i++ {
		str += "  "
	}

	str += fmt.Sprintf("%v - %v (%v)\n", bn.name, bn.measurement, bn.numSamples)
	for _, child := range bn.children {
		str += child.printLevel(level + 1)
	}

	return str
}
