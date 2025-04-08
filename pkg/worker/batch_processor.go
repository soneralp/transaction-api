package worker

import (
	"context"
	"sync"
	"time"
	"transaction-api-w-go/pkg/domain"
)

type BatchJob struct {
	UserIDs     []uint
	Amount      float64
	Description string
	Operation   string
}

type BatchProcessor struct {
	balanceService domain.BalanceService
	jobQueue       chan BatchJob
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	stats          *BatchStats
}

type BatchStats struct {
	TotalProcessed     uint64
	TotalFailed        uint64
	TotalAmount        float64
	AverageProcessTime float64
	mu                 sync.RWMutex
}

func NewBatchProcessor(balanceService domain.BalanceService) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &BatchProcessor{
		balanceService: balanceService,
		jobQueue:       make(chan BatchJob, 100),
		ctx:            ctx,
		cancel:         cancel,
		stats:          &BatchStats{},
	}
}

func (p *BatchProcessor) Start() {
	p.wg.Add(1)
	go p.process()
}

func (p *BatchProcessor) Stop() {
	close(p.jobQueue)

	p.wg.Wait()
}

func (p *BatchProcessor) SubmitJob(job BatchJob) {
	select {
	case p.jobQueue <- job:
	case <-p.ctx.Done():
	}
}

func (p *BatchProcessor) GetStats() BatchStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	return *p.stats
}

func (p *BatchProcessor) process() {
	defer p.wg.Done()

	for job := range p.jobQueue {
		startTime := time.Now()

		successCount, failedCount, totalAmount := p.processBatch(job)

		p.stats.mu.Lock()
		p.stats.TotalProcessed += uint64(successCount)
		p.stats.TotalFailed += uint64(failedCount)
		p.stats.TotalAmount += totalAmount

		processTime := time.Since(startTime).Seconds()
		currentTotal := p.stats.TotalProcessed
		currentAvg := p.stats.AverageProcessTime
		p.stats.AverageProcessTime = (currentAvg*float64(currentTotal) + processTime) / float64(currentTotal+1)
		p.stats.mu.Unlock()
	}
}

func (p *BatchProcessor) processBatch(job BatchJob) (successCount, failedCount int, totalAmount float64) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, userID := range job.UserIDs {
		wg.Add(1)
		go func(uid uint) {
			defer wg.Done()

			var err error
			if job.Operation == "add" {
				err = p.balanceService.AddFunds(context.Background(), uid, job.Amount)
			} else if job.Operation == "withdraw" {
				err = p.balanceService.WithdrawFunds(context.Background(), uid, job.Amount)
			}

			mu.Lock()
			if err != nil {
				failedCount++
			} else {
				successCount++
				totalAmount += job.Amount
			}
			mu.Unlock()
		}(userID)
	}

	wg.Wait()
	return
}
