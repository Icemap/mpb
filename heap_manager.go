package mpb

import (
	"container/heap"
)

type heapManager chan heapRequest

type heapCmd int

const (
	h_sync heapCmd = iota
	h_push
	h_iter
	h_pop_all
	h_fix
)

type heapRequest struct {
	cmd  heapCmd
	data interface{}
}

type iterData struct {
	iter chan *Bar
	drop chan struct{}
}

type pushData struct {
	bar  *Bar
	sync bool
}

func (m heapManager) run() {
	var bHeap priorityQueue
	var pMatrix map[int][]chan int
	var aMatrix map[int][]chan int

	var l int
	var sync bool

	for req := range m {
		switch req.cmd {
		case h_push:
			data := req.data.(*pushData)
			heap.Push(&bHeap, data.bar)
			sync = data.sync
		case h_sync:
			if sync || l != bHeap.Len() {
				pMatrix = make(map[int][]chan int)
				aMatrix = make(map[int][]chan int)
				for _, b := range bHeap {
					table := b.wSyncTable()
					for i, ch := range table[0] {
						pMatrix[i] = append(pMatrix[i], ch)
					}
					for i, ch := range table[1] {
						aMatrix[i] = append(aMatrix[i], ch)
					}
				}
			}
			l = bHeap.Len()
			syncWidth(pMatrix)
			syncWidth(aMatrix)
		case h_iter:
			data := req.data.(*iterData)
			for _, b := range bHeap {
				select {
				case data.iter <- b:
				case <-data.drop:
				}
			}
			close(data.iter)
		case h_pop_all:
			data := req.data.(*iterData)
			for bHeap.Len() != 0 {
				select {
				case data.iter <- heap.Pop(&bHeap).(*Bar):
				case <-data.drop:
				}
			}
			close(data.iter)
		case h_fix:
			heap.Fix(&bHeap, req.data.(int))
		}
	}
}

func (m heapManager) sync() {
	m <- heapRequest{cmd: h_sync}
}

func (m heapManager) push(b *Bar, sync bool) {
	data := &pushData{b, sync}
	m <- heapRequest{cmd: h_push, data: data}
}

func (m heapManager) iter(iter chan *Bar, drop chan struct{}) {
	data := &iterData{iter, drop}
	m <- heapRequest{cmd: h_iter, data: data}
}

func (m heapManager) popAll(iter chan *Bar, drop chan struct{}) {
	data := &iterData{iter, drop}
	m <- heapRequest{cmd: h_pop_all, data: data}
}

func (m heapManager) fix(index int) {
	m <- heapRequest{cmd: h_push, data: index}
}
