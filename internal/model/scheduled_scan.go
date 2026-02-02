package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const MinScanInterval = 5 * time.Minute

type ScheduledScan struct {
	ID             string     `json:"id"`
	NetworkID      string     `json:"network_id"`
	ProfileID      string     `json:"profile_id"`
	Name           string     `json:"name"`
	CronExpression string     `json:"cron_expression"`
	Enabled        bool       `json:"enabled"`
	Description    string     `json:"description,omitempty"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (s *ScheduledScan) Validate() error {
	if s.NetworkID == "" {
		return fmt.Errorf("network_id is required")
	}

	if s.ProfileID == "" {
		return fmt.Errorf("profile_id is required")
	}

	if s.CronExpression == "" {
		return fmt.Errorf("cron_expression is required")
	}

	parts := strings.Split(s.CronExpression, " ")
	if len(parts) != 5 && len(parts) != 6 {
		return fmt.Errorf("invalid cron expression: must have 5 or 6 parts, got %d", len(parts))
	}

	// Validate cron expression and check minimum interval
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(s.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Check interval between first two runs
	now := time.Now()
	first := schedule.Next(now)
	second := schedule.Next(first)
	interval := second.Sub(first)
	if interval < MinScanInterval {
		return fmt.Errorf("scan interval too short: %v (minimum %v)", interval, MinScanInterval)
	}

	return nil
}
