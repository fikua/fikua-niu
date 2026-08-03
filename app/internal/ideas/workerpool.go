// workerpool.go implements the bounded scrape worker pool (design.md
// ADR-03, amended F-05, tasks.md T-07/T-07a/T-08): Service.Add never
// spawns an unbounded goroutine per POST — it submits a job to this pool,
// which runs at most workerPoolSize scrapes concurrently via a
// buffered-channel semaphore.
package ideas

import (
	"context"
	"sync"
)

// workerPoolSize is the confirmed concurrency cap for the scrape worker
// pool (design.md ADR-03/R-08, tasks.md T-07a — range 4-8 fixed by
// design, exact value confirmed here during /code).
//
// 6 workers x ~2MiB retained per in-flight scrape (fetchsafe's T-03g
// streaming limit) ≈ 12MiB peak, comfortably under the container's
// 128M/0.5CPU limit (app/compose.yaml) even with the rest of the process'
// baseline footprint (SQLite connection pool, goose, the HTTP server
// itself) — chosen at the midpoint of the approved 4-8 range rather than
// the ceiling, since a two-person household app is never expected to
// queue more than a handful of concurrent "add idea" submissions at
// once.
const workerPoolSize = 6

// job is one unit of scrape work submitted to the pool.
type job struct {
	ideaID int64
	url    string
	run    func(ctx context.Context, ideaID int64, url string)
}

// WorkerPool is a bounded-concurrency background worker pool for the
// scrape step of adding an idea (ADR-03). It is fed by Service.Add and
// runs on a background context supplied by main.go — independent from
// any single HTTP request's context, which is already cancelled by the
// time a worker picks up the job (the request already got its immediate
// 201 response).
type WorkerPool struct {
	ctx  context.Context
	jobs chan job
	wg   sync.WaitGroup
}

// NewWorkerPool starts workerPoolSize goroutines consuming from an
// internal job queue, all running under bgCtx (main.go's
// context.Background(), NOT any single request's context — ADR-03).
// Call Close at application shutdown for an orderly drain.
func NewWorkerPool(bgCtx context.Context) *WorkerPool {
	p := &WorkerPool{
		ctx:  bgCtx,
		jobs: make(chan job, workerPoolSize*4), // headroom so Submit never blocks the HTTP handler under a burst
	}
	p.wg.Add(workerPoolSize)
	for i := 0; i < workerPoolSize; i++ {
		go p.worker()
	}
	return p
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case j, ok := <-p.jobs:
			if !ok {
				return
			}
			j.run(p.ctx, j.ideaID, j.url)
		}
	}
}

// Submit enqueues a scrape job. Never blocks the caller (Service.Add,
// itself called from an HTTP handler) beyond the buffered channel's
// capacity — if the pool is fully occupied, the job simply waits in the
// buffer until a worker frees up; the idea row has already been inserted
// as 'pending' before this is ever called, so a slower-than-usual
// resolution is just a longer-lived Estat D card, not an error (ADR-03).
func (p *WorkerPool) Submit(ideaID int64, url string, run func(ctx context.Context, ideaID int64, url string)) {
	select {
	case p.jobs <- job{ideaID: ideaID, url: url, run: run}:
	case <-p.ctx.Done():
		// Shutting down — drop the job rather than leak a blocked send.
	}
}

// Close stops accepting new jobs and waits for in-flight workers to
// finish draining the queue, up to bgCtx's own cancellation (main.go
// cancels bgCtx on shutdown, which also makes worker() return promptly
// without waiting for the queue to fully drain).
func (p *WorkerPool) Close() {
	close(p.jobs)
	p.wg.Wait()
}
