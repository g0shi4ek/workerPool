package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type WorkerPool struct {
	workers          map[int]chan string
	jobs             chan string
	addWorkerChan    chan struct{}
	removeWorkerChan chan struct{}
	stopWorkersChan  chan struct{}
	next             int
	wg               sync.WaitGroup
	mu               sync.Mutex
}

func NewWorkerPool(init int) *WorkerPool {
	pool := &WorkerPool{
		workers:          make(map[int]chan string),
		jobs:             make(chan string),
		addWorkerChan:    make(chan struct{}),
		removeWorkerChan: make(chan struct{}),
		stopWorkersChan:  make(chan struct{}),
	}

	for i := 0; i < init; i++ {
		pool.AddNewWorker()
	}

	go pool.ManagePool()
	return pool
}

func (p *WorkerPool) AddNewWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := p.next
	p.next++

	workerChan := make(chan string)
	p.workers[id] = workerChan

	p.wg.Add(1)
	go func() {
		for job := range workerChan {
			fmt.Printf("Worker %d, task: %s in process\n", id, job)
			time.Sleep(100 * time.Millisecond)
		}
		defer p.wg.Done()
	}()

}

func (p *WorkerPool) RemoveWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.workers) == 0 {
		return
	}

	for id, ch := range p.workers {
		close(ch)
		delete(p.workers, id)
		fmt.Printf("Removed worker %d\n", id)
		return
	}
}

func (p *WorkerPool) AddNewJob(job string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.workers) > 0 {
		for _, workerChan := range p.workers {
			workerChan <- job
			break
		}
	} else {
		fmt.Println("No available workers for job: ", job)
	}
}

func (p *WorkerPool) StopAllWorkers() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, ch := range p.workers {
		close(ch)
		delete(p.workers, id)
		fmt.Printf("Removed worker %d\n", id)
	}
	fmt.Println("No workers in pool")
}

func (p *WorkerPool) ManagePool() {
	for {
		select {
		case <-p.addWorkerChan:
			p.AddNewWorker()
		case <-p.removeWorkerChan:
			p.RemoveWorker()
		case job := <-p.jobs:
			p.AddNewJob(job)
		case <-p.stopWorkersChan:
			p.StopAllWorkers()
			return
		}
	}
}

func (p *WorkerPool) SubmitJob(job string) {
	p.jobs <- job
}

func (p *WorkerPool) ScaleUp() {
	p.addWorkerChan <- struct{}{}
}

func (p *WorkerPool) ScaleDown() {
	p.removeWorkerChan <- struct{}{}
}

func (p *WorkerPool) GracefulShutdown() {
	close(p.stopWorkersChan)
	close(p.jobs)
	p.wg.Wait()
	fmt.Println("Pool shutdown complete")
}

func main() {
	pool := NewWorkerPool(5)
	defer pool.GracefulShutdown()

	fmt.Println("start workerPool")

	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println("Sent new job")
			pool.SubmitJob("Job Num" + strconv.Itoa(i))
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		for i := 0; i < 3; i++ {
			fmt.Println("Sent sygnal to scale up")
			pool.ScaleUp()
			time.Sleep(200 * time.Millisecond)
		}
	}()

	go func() {
		for i := 0; i < 3; i++ {
			fmt.Println("Sent sygnal to scale down")
			pool.ScaleDown()
			time.Sleep(350 * time.Millisecond)
		}
	}()

	time.Sleep(10 * time.Second)
}
