package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/cobalto/noppera/internal/config"
	"github.com/cobalto/noppera/internal/models"
	"github.com/cobalto/noppera/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// Archiver manages thread archiving and deletion.
type Archiver struct {
	db    *pgxpool.Pool
	store storage.Storage
	cfg   config.Config
	cron  *cron.Cron
}

// NewArchiver creates a new Archiver instance.
func NewArchiver(db *pgxpool.Pool, store storage.Storage, cfg config.Config) *Archiver {
	return &Archiver{
		db:    db,
		store: store,
		cfg:   cfg,
		cron:  cron.New(),
	}
}

// Start begins the archiving and deletion schedule.
func (a *Archiver) Start() {
	// Run every hour
	_, err := a.cron.AddFunc("@hourly", a.run)
	if err != nil {
		panic(fmt.Errorf("failed to schedule archiver: %w", err))
	}
	a.cron.Start()
}

// Stop stops the cron scheduler.
func (a *Archiver) Stop() {
	a.cron.Stop()
}

// run performs the archiving and deletion tasks.
func (a *Archiver) run() {
	ctx := context.Background()

	// Archive threads inactive for 7 days
	if err := a.archiveThreads(ctx); err != nil {
		fmt.Printf("Archiver: failed to archive threads: %v\n", err)
	}

	// Delete archived threads older than ARCHIVE_DELETE_DAYS
	if err := a.deleteOldThreads(ctx); err != nil {
		fmt.Printf("Archiver: failed to delete old threads: %v\n", err)
	}
}

// archiveThreads marks threads as archived if inactive for 7 days.
func (a *Archiver) archiveThreads(ctx context.Context) error {
	archiveThreshold := time.Now().Add(-7 * 24 * time.Hour)
	_, err := a.db.Exec(ctx,
		"UPDATE posts SET archived_at = $1 WHERE thread_id IS NULL AND archived_at IS NULL AND last_bumped_at < $2",
		time.Now(), archiveThreshold,
	)
	if err != nil {
		return fmt.Errorf("failed to archive threads: %w", err)
	}
	return nil
}

// deleteOldThreads deletes archived threads older than ARCHIVE_DELETE_DAYS.
func (a *Archiver) deleteOldThreads(ctx context.Context) error {
	deleteThreshold := time.Now().Add(-time.Duration(a.cfg.ArchiveDeleteDays) * 24 * time.Hour)
	rows, err := a.db.Query(ctx,
		"SELECT id, image_url FROM posts WHERE archived_at < $1 AND thread_id IS NULL",
		deleteThreshold,
	)
	if err != nil {
		return fmt.Errorf("failed to query old threads: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var imageURL *string
		if err := rows.Scan(&id, &imageURL); err != nil {
			return fmt.Errorf("failed to scan thread: %w", err)
		}

		// Delete associated images
		if imageURL != nil {
			if err := a.store.Delete(ctx, *imageURL); err != nil {
				fmt.Printf("Archiver: failed to delete image for thread %d: %v\n", id, err)
				continue
			}
		}

		// Delete thread and replies
		if err := models.DeletePost(ctx, a.db, id); err != nil {
			fmt.Printf("Archiver: failed to delete thread %d: %v\n", id, err)
			continue
		}
	}

	return nil
}
