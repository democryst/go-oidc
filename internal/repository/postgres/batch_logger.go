package postgres

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/democryst/go-oidc/internal/model"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

// BatchRepository wraps a Repository and buffers audit log writes.
type BatchRepository struct {
	interfaces.Repository
	pool       *pgxpool.Pool
	eventCh    chan *model.AuditEvent
	batchSize  int
	timeout    time.Duration
	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewBatchRepository creates a new BatchRepository and starts the background worker.
func NewBatchRepository(repo interfaces.Repository, pool *pgxpool.Pool, batchSize int, timeout time.Duration) *BatchRepository {
	ctx, cancel := context.WithCancel(context.Background())
	br := &BatchRepository{
		Repository: repo,
		pool:       pool,
		eventCh:    make(chan *model.AuditEvent, 10000), // Large buffer for the channel
		batchSize:  batchSize,
		timeout:    timeout,
		ctx:        ctx,
		cancelFunc: cancel,
	}

	br.wg.Add(1)
	go br.worker()
	return br
}

// AppendAuditLog satisfies the Repository interface by sending the event to the asynchronous buffer.
func (br *BatchRepository) AppendAuditLog(ctx context.Context, event *model.AuditEvent) error {
	select {
	case br.eventCh <- event:
		return nil
	default:
		// Channel full, fallback to synchronous write to avoid data loss, 
		// but log a warning as this indicates we are falling behind.
		log.Printf("Warning: Audit log buffer full, falling back to sync write")
		return br.Repository.AppendAuditLog(ctx, event)
	}
}

// Close gracefully shuts down the background worker after flushing remaining events.
func (br *BatchRepository) Close() {
	br.cancelFunc()
	br.wg.Wait()
}

func (br *BatchRepository) worker() {
	defer br.wg.Done()
	
	buffer := make([]*model.AuditEvent, 0, br.batchSize)
	ticker := time.NewTicker(br.timeout)
	defer ticker.Stop()

	for {
		select {
		case event := <-br.eventCh:
			buffer = append(buffer, event)
			if len(buffer) >= br.batchSize {
				br.flush(buffer)
				buffer = buffer[:0]
				ticker.Reset(br.timeout)
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				br.flush(buffer)
				buffer = buffer[:0]
			}
		case <-br.ctx.Done():
			// Final flush
			if len(buffer) > 0 {
				br.flush(buffer)
			}
			return
		}
	}
}

func (br *BatchRepository) flush(batch []*model.AuditEvent) {
	if len(batch) == 0 {
		return
	}

	// Use COPY for maximum speed
	// Columns: request_id, event_type, actor_id, client_id, metadata
	rows := make([][]any, len(batch))
	for i, ev := range batch {
		rows[i] = []any{ev.RequestID, ev.EventType, ev.ActorID, ev.ClientID, ev.Metadata}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := br.pool.CopyFrom(
		ctx,
		pgx.Identifier{"audit_log"},
		[]string{"request_id", "event_type", "actor_id", "client_id", "metadata"},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		log.Printf("Error flushing audit log batch: %v", err)
		// At this scale, we should probably retry or write to a dead-letter log file.
	}
}
