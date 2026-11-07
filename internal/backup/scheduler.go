package backup

import (
	"context"
	"sync"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/storage"
)

type BackupScheduler struct {
	engine   *BackupEngine
	storage  *storage.SQLiteStorage
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewBackupScheduler(engine *BackupEngine, store *storage.SQLiteStorage) *BackupScheduler {
	return &BackupScheduler{
		engine:   engine,
		storage:  store,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic evaluation of backup schedules (every 60 seconds)
func (s *BackupScheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.evaluateSchedules(ctx)
			}
		}
	}()
}

// Stop cleanly terminates scheduler
func (s *BackupScheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *BackupScheduler) evaluateSchedules(ctx context.Context) {
	schedules, err := s.storage.ListAllActiveBackupSchedules()
	if err != nil || len(schedules) == 0 {
		return
	}

	now := time.Now()
	for _, sched := range schedules {
		if !s.isDue(sched, now) {
			continue
		}

		go func(sc model.BackupSchedule) {
			srv, err := s.storage.GetServer(sc.ServerID)
			if err != nil {
				return
			}
			dbCfg, err := s.storage.GetDatabaseConfig(sc.DatabaseConfigID)
			if err != nil {
				return
			}

			_, _ = s.engine.ExecuteBackup(ctx, srv, dbCfg, &sc)

			// Calculate next run
			nextRun := s.calcNextRun(sc, now)
			_ = s.storage.UpdateBackupScheduleRun(sc.ID, now, nextRun)
		}(sched)
	}
}

func (s *BackupScheduler) isDue(sc model.BackupSchedule, now time.Time) bool {
	if sc.NextRun != nil && now.After(*sc.NextRun) {
		return true
	}
	if sc.LastRun == nil {
		return true
	}

	// Format HH:MM check
	currentTimeStr := now.Format("15:04")
	if sc.TimeOfDay != "" && sc.TimeOfDay == currentTimeStr {
		switch sc.Frequency {
		case "daily":
			return now.Sub(*sc.LastRun) > 20*time.Hour
		case "weekly":
			return int(now.Weekday()) == sc.DayOfWeek && now.Sub(*sc.LastRun) > 6*24*time.Hour
		case "hourly":
			return now.Sub(*sc.LastRun) > 50*time.Minute
		}
	}

	return false
}

func (s *BackupScheduler) calcNextRun(sc model.BackupSchedule, from time.Time) time.Time {
	switch sc.Frequency {
	case "hourly":
		return from.Add(1 * time.Hour)
	case "weekly":
		return from.Add(7 * 24 * time.Hour)
	default: // daily
		return from.Add(24 * time.Hour)
	}
}
