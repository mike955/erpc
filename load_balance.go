package erpc

import (
	"math/rand"
	"sync/atomic"
)

type BalanceType int

const (
	Random BalanceType = iota
	RoundRobin
	LeastConnections
)

type balance struct {
	bl             BalanceType
	loadBalanceMap map[int32][]int
	loadSize       int32
	balanceSize    int32
}

func NewBalance(b BalanceType) (bl *balance) {
	bl = &balance{
		bl:             b,
		loadBalanceMap: map[int32][]int{},
		loadSize:       0,
		balanceSize:    0,
	}
	return
}

func (b *balance) addLoad(load int32) {
	if _, ok := b.loadBalanceMap[load]; !ok {
		b.loadSize++
		b.loadBalanceMap[load] = []int{}
	}
}

func (b *balance) next() (load int32) {
	switch b.bl {
	case Random:
		load = rand.Int31n(b.loadSize - 1)
	case RoundRobin:
		if atomic.LoadInt32(&b.loadSize) == 0 {
			load = 0
		} else {
			load = atomic.LoadInt32(&b.loadSize) % atomic.LoadInt32(&b.loadSize)
		}
	case LeastConnections:
		load = 0
		minBalance := 0
		for index, balance := range b.loadBalanceMap {
			if len(balance) < minBalance {
				load = index
				minBalance = len(balance)
			}
		}
	default:
		load = rand.Int31n(b.loadSize - 1)
	}
	return load
}

func (b *balance) addBalance(load int32, fd int) {
	if _, ok := b.loadBalanceMap[load]; !ok {
		atomic.AddInt32(&b.loadSize, 1)
		b.loadBalanceMap[load] = []int{fd}
	} else {
		b.loadBalanceMap[load] = append(b.loadBalanceMap[load], fd)
	}
	atomic.AddInt32(&b.balanceSize, 1)
}

func (b *balance) removeBalance(load int32, fd int) {
	if _, ok := b.loadBalanceMap[load]; !ok {
		balances := b.loadBalanceMap[load]
		for i := 0; i < len(balances); i++ {
			ba := balances[i]
			if ba == fd {
				b.loadBalanceMap[load] = append(balances[:i], balances[i+1:]...)
				atomic.AddInt32(&b.balanceSize, -1)
				return
			}
		}
	} else {
		b.loadBalanceMap[load] = []int{}
	}
}
