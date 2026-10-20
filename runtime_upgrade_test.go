package main

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func preserveRuntimeStatus(t *testing.T) {
	t.Helper()
	refreshJobs.Lock()
	job, queued, next := refreshJobs.job, refreshJobs.queued, refreshJobs.next
	refreshJobs.job, refreshJobs.queued, refreshJobs.next = refreshJobStatus{Phase: "idle"}, false, map[string]time.Time{}
	refreshJobs.Unlock()
	previousFetch, previousBackoff := lastFetch, claudeBackoffUntil
	previousCollecting := collectingRefresh.Load()
	previousActions := appActions
	appActions = make(chan string, 16)
	t.Cleanup(func() {
		refreshJobs.Lock()
		refreshJobs.job, refreshJobs.queued, refreshJobs.next = job, queued, next
		refreshJobs.Unlock()
		lastFetch, claudeBackoffUntil = previousFetch, previousBackoff
		collectingRefresh.Store(previousCollecting)
		appActions = previousActions
	})
}

func TestPausedAlertsPreserveRulesAndResumeAtExpiry(t *testing.T) {
	now := time.Unix(1800000000, 0)
	settings := defaultTraySettings()
	settings.AlertsPausedUntil = now.Add(time.Hour).Unix()
	s := &State{Alerts: map[string]bool{"codex7d": true}, Claude: AgentUsage{Available: true, Short: Window{Left: 2}, Week: Window{Left: 70}}}
	before := maps.Clone(s.Alerts)
	if events := alertTransitionsAt(s, settings, now); len(events) != 0 || !maps.Equal(before, s.Alerts) {
		t.Fatalf("pause changed transitions: %+v %+v", events, s.Alerts)
	}
	if events := alertTransitionsAt(s, settings, time.Unix(settings.AlertsPausedUntil, 0)); len(events) != 1 || !s.Alerts["claude5h"] {
		t.Fatalf("expiry failed to resume: %+v %+v", events, s.Alerts)
	}
	if !settings.AlertsEnabled || !settings.AlertRules["claude5h"].Enabled {
		t.Fatal("pause changed persisted rules")
	}
}

func TestRefreshJobsDistinguishRequestWaitAndCompleted(t *testing.T) {
	preserveRuntimeStatus(t)
	now := time.Unix(1800000000, 0)
	requested, ok := enqueueRefreshAction(now)
	if !ok || requested.ID == 0 || requested.Phase != "running" {
		t.Fatalf("request not reserved: %+v", requested)
	}
	if duplicate, ok := enqueueRefreshAction(now); !ok || duplicate.ID != requested.ID || len(appActions) != 1 {
		t.Fatal("duplicate created another job")
	}
	<-appActions
	beginRefreshJob(now)
	if currentRefreshJob().ID != requested.ID || currentRefreshJob().StartedAt == nil {
		t.Fatal("start lost requested job")
	}
	lastFetch.claude = now
	claudeBackoffUntil = now.Add(10 * time.Minute)
	finishRefreshJob(now.Add(time.Second), enabledAgents{claude: true}, 0, 0)
	waiting := currentRefreshJob()
	if waiting.Phase != "waiting" || waiting.CompletedAt == nil || waiting.NextAt == nil || !waiting.NextAt.Equal(claudeBackoffUntil) || collectingRefresh.Load() {
		t.Fatalf("wait misreported as collection: %+v", waiting)
	}
	if waitingRefreshDue(now.Add(9*time.Minute)) || !waitingRefreshDue(claudeBackoffUntil) {
		t.Fatal("backoff retry deadline ignored")
	}
	if next := agentNextQuery("Claude"); next == nil || !next.Equal(claudeBackoffUntil) {
		t.Fatal("missing provider deadline")
	}
	beginRefreshJob(claudeBackoffUntil)
	finishRefreshJob(claudeBackoffUntil.Add(time.Second), enabledAgents{claude: true}, 1, 0)
	if completed := currentRefreshJob(); completed.Phase != "completed" || completed.ID != requested.ID+1 {
		t.Fatalf("completion: %+v", completed)
	}
	beginRefreshJob(now)
	finishRefreshJob(now, enabledAgents{claude: true}, 1, 1)
	if currentRefreshJob().Phase != "failed" {
		t.Fatal("failed fetch reported success")
	}
}

func TestRefreshWithinMinimumIntervalDoesNotFetch(t *testing.T) {
	preserveRuntimeStatus(t)
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	mu.Lock()
	previousState := state
	state = &State{Alerts: map[string]bool{}, Claude: AgentUsage{Available: true, OK: true, FetchedAt: time.Now(), Short: Window{Left: 90}, Week: Window{Left: 80}}}
	mu.Unlock()
	t.Cleanup(func() { mu.Lock(); state = previousState; mu.Unlock() })
	lastFetch.claude = time.Now()
	claudeBackoffUntil = time.Time{}
	refreshFor(enabledAgents{claude: true})
	job := currentRefreshJob()
	if job.Phase != "waiting" || job.NextAt == nil {
		t.Fatalf("minimum interval performed collection: %+v", job)
	}
	mu.RLock()
	left := state.Claude.Short.Left
	mu.RUnlock()
	if left != 90 {
		t.Fatal("cached quota lost during wait")
	}
}

func TestProviderQueryIntervalStartsAfterEarlierProviderDelay(t *testing.T) {
	now := time.Unix(1800000000, 0)
	var first, second time.Time
	if !reserveProviderQuery(now, &first, claudePollMin, time.Time{}) {
		t.Fatal("first provider was not eligible")
	}
	// The first provider takes ten seconds before the next provider is called.
	secondStarted := now.Add(10 * time.Second)
	if !reserveProviderQuery(secondStarted, &second, codexPollMin, time.Time{}) || !second.Equal(secondStarted) {
		t.Fatal("second provider did not record its own start")
	}
	if reserveProviderQuery(now.Add(codexPollMin), &second, codexPollMin, time.Time{}) || !second.Equal(secondStarted) {
		t.Fatal("job start allowed an early second-provider query")
	}
	deadline := secondStarted.Add(codexPollMin)
	if !reserveProviderQuery(deadline, &second, codexPollMin, time.Time{}) || !second.Equal(deadline) {
		t.Fatal("provider was not eligible at its actual deadline")
	}
	backoff := deadline.Add(10 * time.Minute)
	if reserveProviderQuery(deadline.Add(codexPollMin), &second, codexPollMin, backoff) || !second.Equal(deadline) {
		t.Fatal("minimum interval bypassed backoff or changed the last query")
	}
	if !reserveProviderQuery(backoff, &second, codexPollMin, backoff) {
		t.Fatal("provider did not resume when backoff expired")
	}
}

func TestRefreshReservationSurvivesCompletionBeforeTraySignal(t *testing.T) {
	preserveRuntimeStatus(t)
	now := time.Unix(1800000000, 0)
	var gate refreshRunGate
	if gate.start(true) || !gate.start(false) {
		t.Fatal("handoff guard or initial start failed")
	}
	beginRefreshJob(now)
	finishRefreshJob(now.Add(time.Second), enabledAgents{codex: true}, 1, 0)
	// The collector has finished, but the tray still has its completion signal
	// pending. Consume a newly accepted API action before receiving that signal.
	requested, ok := enqueueRefreshAction(now.Add(2 * time.Second))
	if !ok || <-appActions != "refresh" || gate.start(false) {
		t.Fatal("completion-boundary request did not wait for the tray signal")
	}
	if duplicate, ok := enqueueRefreshAction(now); !ok || duplicate.ID != requested.ID || len(appActions) != 0 {
		t.Fatal("duplicate reservation created another queued action")
	}
	if !gate.finish() || !gate.start(false) {
		t.Fatal("receiving the completion signal lost the pending reservation")
	}
	beginRefreshJob(now.Add(3 * time.Second))
	if job := currentRefreshJob(); job.ID != requested.ID || job.StartedAt == nil {
		t.Fatalf("resumed refresh lost its reserved job ID: %+v", job)
	}
	finishRefreshJob(now.Add(4*time.Second), enabledAgents{codex: true}, 0, 0)
	if gate.finish() {
		t.Fatal("completed reservation scheduled an extra refresh")
	}
}

func TestRefreshReservationStartsBeforeQueuedActionIsConsumed(t *testing.T) {
	preserveRuntimeStatus(t)
	now := time.Unix(1800000000, 0)
	var gate refreshRunGate
	gate.start(false)
	beginRefreshJob(now)
	finishRefreshJob(now.Add(time.Second), enabledAgents{codex: true}, 1, 0)
	requested, ok := enqueueRefreshAction(now.Add(2 * time.Second))
	if !ok || !gate.finish() || !gate.start(false) {
		t.Fatal("completion signal did not restart the reserved job")
	}
	beginRefreshJob(now.Add(3 * time.Second))
	// The completion case ran first. The original API action is still buffered
	// and must not replace the already-started reservation or add another run.
	if <-appActions != "refresh" || gate.start(false) {
		t.Fatal("buffered action started a concurrent refresh")
	}
	if job := currentRefreshJob(); job.ID != requested.ID || job.Phase != "running" {
		t.Fatalf("buffered action lost its reserved job: %+v", job)
	}
	finishRefreshJob(now.Add(4*time.Second), enabledAgents{codex: true}, 0, 0)
	if gate.finish() || len(appActions) != 0 {
		t.Fatal("consumed reservation remained pending")
	}
}

func TestNoAccountDiscoveryConnectsOnlyOwnedAgentAndSkipsOccupiedPort(t *testing.T) {
	previous := agents
	t.Cleanup(func() { agents = previous })
	agents = enabledAgents{}
	var spawned []string
	spawn := func(names []string) { spawned = append(spawned, names...) }
	if reconnectAgents(enabledAgents{}, "claude", true, true, func(string) bool { return false }, spawn) || len(spawned) != 0 {
		t.Fatal("empty discovery started a child")
	}
	found := enabledAgents{claude: true, codex: true}
	if !reconnectAgents(found, "claude", true, true, func(string) bool { return true }, spawn) || !agents.claude || agents.codex || len(spawned) != 0 {
		t.Fatal("occupied sibling was duplicated or owner did not connect")
	}
	reconnectAgents(found, "claude", true, true, func(string) bool { return false }, spawn)
	if !reflect.DeepEqual(spawned, []string{"codex"}) {
		t.Fatalf("missing child not discovered: %v", spawned)
	}
	spawned = nil
	reconnectAgents(found, "claude", false, true, func(string) bool { return false }, spawn)
	if len(spawned) != 0 {
		t.Fatal("explicit only selection spawned another agent")
	}
}

func TestDefaultAccountKeepsURLAndOnlySplitsAfterMultipleAccounts(t *testing.T) {
	previousOnly, previousAgents := only, agents
	t.Cleanup(func() { only, agents = previousOnly, previousAgents })
	owner := ""
	only = &owner
	for _, found := range []enabledAgents{{}, {codex: true}, {claude: true}} {
		agents = enabledAgents{}
		if needsAgentSplit(found, tag(), true) {
			t.Fatal("single or absent account split the default instance")
		}
		spawned := false
		reconnectAgents(found, tag(), true, true, func(string) bool { return false }, func([]string) { spawned = true })
		if spawned || tag() != "" || listenAddr() != "127.0.0.1:47113" || suffix() != "" || !reflect.DeepEqual(agents, found) {
			t.Fatalf("default URL/data ownership changed for %+v", found)
		}
	}
	found := enabledAgents{claude: true, codex: true}
	if !needsAgentSplit(found, "", true) || needsAgentSplit(found, "codex", true) || needsAgentSplit(found, "", false) {
		t.Fatal("split ignored owner or platform")
	}
	var order []string
	err := handoffAgentProcesses(found, func() error { order = append(order, "close"); return nil }, func(names []string) {
		order = append(order, "spawn")
		if !reflect.DeepEqual(names, []string{"claude", "codex"}) {
			t.Errorf("wrong children: %v", names)
		}
	}, func() { order = append(order, "quit") })
	if err != nil || !reflect.DeepEqual(order, []string{"close", "spawn", "quit"}) {
		t.Fatalf("unsafe handoff order: %v %v", order, err)
	}
	order = nil
	failure := errors.New("close failed")
	err = handoffAgentProcesses(found, func() error { return failure }, func([]string) { order = append(order, "spawn") }, func() { order = append(order, "quit") })
	if !errors.Is(err, failure) || len(order) != 0 {
		t.Fatalf("handoff proceeded after failed close: %v %v", order, err)
	}
}

func TestMetadataAPIStableInstanceAndCurrentTheme(t *testing.T) {
	t.Setenv("USAGE_TRAY_DATA_DIR", t.TempDir())
	settings := defaultTraySettings()
	settings.Theme = "dark"
	if err := saveTraySettings(settings); err != nil {
		t.Fatal(err)
	}
	h := appHTTPHandler()
	for range 2 {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, apiRequest("GET", "/api/meta", ""))
		var meta map[string]string
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &meta) != nil || meta["instance_id"] != dashboardInstanceID() || meta["ui_revision"] != dashboardUIRevision() || meta["theme"] != "dark" || meta["version"] != appVersion {
			t.Fatalf("metadata: %d %s", w.Code, w.Body.String())
		}
	}
	if newInstanceID() == dashboardInstanceID() {
		t.Fatal("instance identifier reused")
	}
}
