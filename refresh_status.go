package main

import (
	"fmt"
	"sync"
	"time"
)

type refreshJobStatus struct {
	ID          uint64     `json:"id"`
	Phase       string     `json:"phase"`
	Message     string     `json:"message"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	NextAt      *time.Time `json:"next_at,omitempty"`
}

var refreshJobs = struct {
	sync.Mutex
	job    refreshJobStatus
	queued bool
	next   map[string]time.Time
}{job: refreshJobStatus{Phase: "idle", Message: "갱신 대기"}, next: make(map[string]time.Time)}

// The tray event loop owns this gate. A job may finish before that loop receives
// its completion signal, so retain any API reservation across that boundary.
type refreshRunGate struct{ running bool }

func (g *refreshRunGate) start(handingOff bool) bool {
	if g.running || handingOff {
		return false
	}
	g.running = true
	return true
}

func (g *refreshRunGate) finish() bool {
	g.running = false
	refreshJobs.Lock()
	defer refreshJobs.Unlock()
	return refreshJobs.queued
}

func currentRefreshJob() refreshJobStatus {
	refreshJobs.Lock()
	defer refreshJobs.Unlock()
	return refreshJobs.job
}

func enqueueRefreshAction(_ time.Time) (refreshJobStatus, bool) {
	refreshJobs.Lock()
	defer refreshJobs.Unlock()
	if refreshJobs.queued || refreshJobs.job.Phase == "running" {
		return refreshJobs.job, true
	}
	select {
	case appActions <- "refresh":
		refreshJobs.job = refreshJobStatus{ID: refreshJobs.job.ID + 1, Phase: "running", Message: "갱신 요청을 처리할 차례를 기다리고 있습니다"}
		refreshJobs.queued = true
		return refreshJobs.job, true
	default:
		return refreshJobs.job, false
	}
}

func beginRefreshJob(now time.Time) {
	refreshJobs.Lock()
	defer refreshJobs.Unlock()
	id := refreshJobs.job.ID
	if !refreshJobs.queued {
		id++
	}
	refreshJobs.queued = false
	refreshJobs.job = refreshJobStatus{ID: id, Phase: "running", Message: "사용량을 조회하고 있습니다", StartedAt: &now}
	collectingRefresh.Store(true)
}

func nextQueryTime(last time.Time, interval time.Duration, backoff time.Time) time.Time {
	var next time.Time
	if !last.IsZero() {
		next = last.Add(interval)
	}
	if backoff.After(next) {
		next = backoff
	}
	return next
}

// Called immediately before each provider fetch, under refreshMu. Earlier
// providers can take time, so their job's start is not this provider's start.
func reserveProviderQuery(now time.Time, last *time.Time, interval time.Duration, backoff time.Time) bool {
	if now.Before(nextQueryTime(*last, interval, backoff)) {
		return false
	}
	*last = now
	return true
}

func finishRefreshJob(now time.Time, enabled enabledAgents, attempts, failures int) {
	next := make(map[string]time.Time)
	if enabled.claude {
		next["Claude"] = nextQueryTime(lastFetch.claude, claudePollMin, claudeBackoffUntil)
	}
	if enabled.codex {
		next["Codex"] = nextQueryTime(lastFetch.codex, codexPollMin, time.Time{})
	}
	if enabled.antigravity {
		next["Antigravity"] = nextQueryTime(lastFetch.antigravity, 2*time.Minute, time.Time{})
	}
	refreshJobs.Lock()
	defer refreshJobs.Unlock()
	refreshJobs.next = next
	j := &refreshJobs.job
	j.CompletedAt = &now
	j.Phase, j.Message = refreshOutcome(attempts, failures, len(next))
	var earliest time.Time
	for _, at := range next {
		if earliest.IsZero() || at.Before(earliest) {
			earliest = at
		}
	}
	if !earliest.IsZero() {
		j.NextAt = &earliest
	}
	collectingRefresh.Store(false)
}

func refreshOutcome(attempts, failures, count int) (string, string) {
	if count == 0 {
		return "waiting", "계정 연결 대기 · 로그인하면 자동으로 연결됩니다"
	}
	if attempts == 0 {
		return "waiting", "최소 조회 간격 또는 요청 제한으로 대기 중입니다"
	}
	if failures == attempts {
		return "failed", "사용량 조회에 실패했습니다. 연결 상태와 로그인을 확인해 주세요"
	}
	if failures > 0 {
		return "completed", fmt.Sprintf("갱신 완료 · %d개 에이전트의 조회에 실패했습니다", failures)
	}
	return "completed", "사용량 갱신이 완료되었습니다"
}

func agentNextQuery(name string) *time.Time {
	refreshJobs.Lock()
	defer refreshJobs.Unlock()
	if at, ok := refreshJobs.next[name]; ok && !at.IsZero() {
		return &at
	}
	return nil
}

func waitingRefreshDue(now time.Time) bool {
	refreshJobs.Lock()
	defer refreshJobs.Unlock()
	j := refreshJobs.job
	return !refreshJobs.queued && j.Phase == "waiting" && j.NextAt != nil && !now.Before(*j.NextAt)
}
