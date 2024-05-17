package workerpool

import (
	"context"
	"log"
)

type Result struct {
	Err  error
	Data any
}

type ProcessorFunc[E, V any] func(context.Context, E) V

// Make a worker pool which receives a channel of
// channel
type WorkerPool[E any, V any] struct {
	Pool      chan chan E
	Workers   []*Worker[E, V]
	ResultChs []chan V
}

func NewWorkerPool[E, V any](poolSize int64, processorFunc ProcessorFunc[E, V]) *WorkerPool[E, V] {

	pool := &WorkerPool[E, V]{
		Pool:      make(chan chan E, poolSize),
		ResultChs: []chan V{},
	}

	workers := []*Worker[E, V]{}

	for i := 0; i < int(poolSize); i++ {
		workers = append(workers, &Worker[E, V]{
			ID:        i + 1,
			Bench:     make(chan E, 1),
			Processor: processorFunc,
			Quit:      make(chan bool),
		})
	}

	pool.Workers = workers

	for i := range workers {
		worker := workers[i]
		worker.WorkerPool = pool
	}

	return pool
}

func Dispatch[E, V any](ctx context.Context, pool *WorkerPool[E, V], receiveCh chan E) {
	for {
		select {
		case job := <-receiveCh:
			jobChan := <-pool.Pool
			jobChan <- job
		case <-ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool[E, V]) Stop(ctx context.Context) {
	for _, worker := range wp.Workers {
		worker.Stop()
	}
}

func (wp *WorkerPool[E, V]) Start(ctx context.Context) {
	if len(wp.Workers) == 0 {
		return
	}

	resultChs := []chan V{}

	for _, worker := range wp.Workers {
		resultChs = append(resultChs, worker.Start(ctx))
	}

	wp.ResultChs = resultChs
}

func Merge[V any](ctx context.Context, chans []chan V, out chan<- V) {
	if len(chans) == 0 {
		return
	}

	for _, ch := range chans {
		go func(ch chan V) {
			select {
			case result := <-ch:
				out <- result
			}
		}(ch)
	}
}

type Worker[E, V any] struct {
	ID         int
	WorkerPool *WorkerPool[E, V]
	Bench      chan E
	Processor  ProcessorFunc[E, V]
	Quit       chan bool
}

func (w *Worker[E, V]) Start(ctx context.Context) chan V {
	resultCh := make(chan V, 1)

	go func() {
		defer close(resultCh)

		for {
			// fmt.Printf("pool %v, bench %v\n", w.WorkerPool, w.Bench)
			w.WorkerPool.Pool <- w.Bench

			select {

			case job := <-w.Bench:
				log.Printf("worker:%d", w.ID)
				// utils.Dump(job)
				result := w.Processor(ctx, job)
				_ = result
				// utils.Dump(result)
				// resultCh <- result
			case <-w.Quit:
				log.Println("quiting")
				return
			case <-ctx.Done():
				return

			}
		}

	}()

	return resultCh
}

func (w *Worker[E, V]) Stop() {
	w.Quit <- true
}
