package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	"transaction-api-w-go/pkg/domain"
)

type TransactionJob struct {
	TransactionID uint
	FromUserID    uint
	ToUserID      uint
	Amount        float64
	Description   string
}

type TransactionWorker struct {
	id                 int
	jobQueue           <-chan TransactionJob
	transactionService domain.TransactionService
	balanceService     domain.BalanceService
	processedCount     uint64
	failedCount        uint64
	mu                 sync.RWMutex
	ctx                context.Context
}

type TransactionWorkerPool struct {
	workers            []*TransactionWorker
	jobQueue           chan TransactionJob
	wg                 sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
	transactionService domain.TransactionService
	balanceService     domain.BalanceService
}

type TransactionStats struct {
	TotalProcessed     uint64
	TotalFailed        uint64
	TotalAmount        float64
	AverageProcessTime float64
	mu                 sync.RWMutex
}

func NewTransactionWorkerPool(
	workerCount int,
	transactionService domain.TransactionService,
	balanceService domain.BalanceService,
) *TransactionWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &TransactionWorkerPool{
		workers:            make([]*TransactionWorker, workerCount),
		jobQueue:           make(chan TransactionJob, 1000),
		ctx:                ctx,
		cancel:             cancel,
		transactionService: transactionService,
		balanceService:     balanceService,
	}

	for i := 0; i < workerCount; i++ {
		pool.workers[i] = &TransactionWorker{
			id:                 i,
			jobQueue:           pool.jobQueue,
			transactionService: transactionService,
			balanceService:     balanceService,
			ctx:                ctx,
		}
	}

	return pool
}

func (p *TransactionWorkerPool) Start() {
	for _, worker := range p.workers {
		p.wg.Add(1)
		go worker.start(&p.wg)
	}
}

func (p *TransactionWorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
	close(p.jobQueue)
}

func (p *TransactionWorkerPool) SubmitJob(job TransactionJob) {
	select {
	case p.jobQueue <- job:
	case <-p.ctx.Done():
	}
}

func (p *TransactionWorkerPool) GetStats() *domain.TransactionStats {
	stats := p.transactionService.GetStats()
	return stats
}

func (w *TransactionWorker) start(wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range w.jobQueue {
		startTime := time.Now()

		err := w.processTransaction(job)

		if err != nil {
			atomic.AddUint64(&w.failedCount, 1)
			atomic.AddUint64(&w.transactionService.GetStats().TotalFailed, 1)
		} else {
			atomic.AddUint64(&w.processedCount, 1)
			atomic.AddUint64(&w.transactionService.GetStats().TotalProcessed, 1)
		}

		processTime := time.Since(startTime).Seconds()

		stats := w.transactionService.GetStats()
		stats.UpdateStats(job.Amount, processTime)
	}
}

func (w *TransactionWorker) processTransaction(job TransactionJob) error {
	startTime := time.Now()

	err := w.transactionService.ProcessTransaction(w.ctx, job.TransactionID)
	if err != nil {
		atomic.AddUint64(&w.failedCount, 1)
		return err
	}

	atomic.AddUint64(&w.processedCount, 1)

	processTime := time.Since(startTime).Seconds()

	stats := w.transactionService.GetStats()
	stats.UpdateStats(job.Amount, processTime)

	return nil
}
