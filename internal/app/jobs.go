package app

import (
	"context"
	"encoding/json/v2"
	"time"

	"github.com/StarsSkyCat/Stars-Admin/internal/core/response"
	"github.com/StarsSkyCat/Stars-Admin/internal/core/scheduler"
)

type demoParams struct {
	Message string `json:"message"`
	DelayMS int    `json:"delayMs"`
}

func jobTargets() (*scheduler.Registry, error) {
	registry := scheduler.NewRegistry()
	parse := func(raw string) (demoParams, error) {
		var p demoParams
		if err := json.Unmarshal([]byte(raw), &p, json.RejectUnknownMembers(true)); err != nil || len([]rune(p.Message)) > 200 || p.DelayMS < 0 || p.DelayMS > 60000 {
			return p, response.ErrInvalidParam
		}
		return p, nil
	}
	err := registry.Register("demoTask", scheduler.Target{Validate: func(raw string) error { _, err := parse(raw); return err }, Redact: func(raw string) string {
		p, _ := parse(raw)
		out, _ := json.Marshal(struct {
			Message string `json:"message"`
			DelayMS int    `json:"delayMs"`
		}{"[REDACTED]", p.DelayMS})
		return string(out)
	}, Run: func(ctx context.Context, raw string) error {
		p, err := parse(raw)
		if err != nil {
			return err
		}
		timer := time.NewTimer(time.Duration(p.DelayMS) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}})
	return registry, err
}
