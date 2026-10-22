//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestAutomaticStableLockAndStaleShutdown(t *testing.T) {
	dir := t.TempDir()
	first, err := openUpdateLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := openUpdateLock(dir); err == nil {
		closeUpdateLock(second)
		t.Fatal("concurrent lock allowed")
	}
	closeUpdateLock(first)
	again, err := openUpdateLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeUpdateLock(again)
	exe := filepath.Join(dir, "fixture.exe")
	session := filepath.Join(dir, "session-dead")
	os.Mkdir(session, 0700)
	p := updateParticipant{PID: uint32(os.Getpid()), Created: 1, Role: "claude"}
	tx := updateTransaction{Target: exe, Version: "v2.0.0", Token: strings.Repeat("a", 64)}
	writeUpdateJSON(filepath.Join(session, "transaction.json"), tx)
	writeUpdateJSON(filepath.Join(session, "participants.json"), []updateParticipant{p})
	writeUpdateJSON(filepath.Join(dir, "active.json"), updateShutdownRequest{"session-dead", tx.Token, uint32(os.Getpid()), 1})
	if shouldStopForAutomaticUpdate(exe, dir, p, "v1.0.0") {
		t.Fatal("stale marker stopped app while another operation owns lock")
	}
	if shouldStopForAutomaticUpdate(exe, dir, p, "v2.0.0") {
		t.Fatal("replacement stopped for old marker")
	}
}
func TestAutomaticHelperEarlyExitReportsError(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "session-test")
	os.Mkdir(session, 0700)
	automaticUpdateState.Lock()
	prior := automaticUpdateState.status
	automaticUpdateState.status = automaticUpdateStatus{State: "installing", CheckedAt: time.Now(), NextCheckAt: time.Now().Add(automaticUpdateInterval)}
	automaticUpdateState.Unlock()
	defer func() {
		automaticUpdateState.Lock()
		automaticUpdateState.status = prior
		automaticUpdateState.Unlock()
	}()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	monitorAutomaticUpdateHelper(cmd, dir, session, "v2.0.0")
	var persisted automaticUpdateStatus
	if err := readUpdateJSON(filepath.Join(dir, "status.json"), &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.State != "error" || persisted.Error == "" || persisted.Version != "v2.0.0" {
		t.Fatalf("silent helper exit left stale state: %+v", persisted)
	}
}
func TestAutomaticHelperManifestPathAndToken(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "fixture.exe")
	os.WriteFile(target, []byte("old"), 0600)
	work := updateWorkDir(target)
	session := filepath.Join(work, "session-test")
	os.MkdirAll(session, 0700)
	helper := filepath.Join(session, "updater.exe")
	os.WriteFile(helper, []byte("old"), 0600)
	hash, _ := fileSHA256(helper)
	token := strings.Repeat("a", 64)
	tx := updateTransaction{Target: target, OldHash: hash, NewHash: hash, Version: "v2.0.0", Token: token}
	manifest := filepath.Join(session, "transaction.json")
	writeUpdateJSON(manifest, tx)
	if _, err := validateUpdateTransaction(manifest, token, helper); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][3]string{{manifest, "wrong", helper}, {"transaction.json", token, helper}, {manifest, token, target}, {filepath.Join(dir, "transaction.json"), token, helper}} {
		if _, err := validateUpdateTransaction(args[0], args[1], args[2]); err == nil {
			t.Fatal("unsafe helper invocation accepted")
		}
	}
	tx.Target = filepath.Join(dir, "unrelated", "fixture.exe")
	writeUpdateJSON(manifest, tx)
	if _, err := validateUpdateTransaction(manifest, token, helper); err == nil {
		t.Fatal("manifest redirected target")
	}
}
func TestAutomaticAtomicReplaceRetainsOriginalOnLockedTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.exe")
	candidate := filepath.Join(dir, "candidate.exe")
	os.WriteFile(target, []byte("old"), 0600)
	os.WriteFile(candidate, []byte("new"), 0600)
	path, err := windows.UTF16PtrFromString(target)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := replaceUpdateFile(candidate, target); err == nil {
		windows.CloseHandle(h)
		t.Fatal("replaced a file opened without delete sharing")
	}
	windows.CloseHandle(h)
	if time.Since(start) > 7*time.Second {
		t.Fatal("replacement retry exceeded bound")
	}
	if b, _ := os.ReadFile(target); string(b) != "old" {
		t.Fatal("failed atomic replace damaged original")
	}
	if err := replaceUpdateFile(candidate, target); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "new" {
		t.Fatal("atomic replacement failed")
	}
}
func TestAutomaticAtomicReplaceWaitsForLockRelease(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.exe")
	candidate := filepath.Join(dir, "candidate.exe")
	os.WriteFile(target, []byte("old"), 0600)
	os.WriteFile(candidate, []byte("new"), 0600)
	path, _ := windows.UTF16PtrFromString(target)
	h, err := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	go func() { time.Sleep(300 * time.Millisecond); windows.CloseHandle(h) }()
	if err := replaceUpdateFile(candidate, target); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target); string(b) != "new" {
		t.Fatal("replacement failed after transient image lock released")
	}
}

// This fixture uses the production helper and process coordination code, but
// has no tray, network calls, credentials or user data. Every process and file
// stays under t.TempDir. It exercises actual Windows executable file locks.
const automaticFixtureSource = `package main
import("os";"time";"path/filepath";"golang.org/x/sys/windows")
var runtimeAutoDiscover bool
var fixtureRole string
func tag()string{return fixtureRole}
func logf(string,...interface{}){}
func main(){
automaticUpdateHealthTimeout=2*time.Second
if handleAutomaticUpdateMode(){return}
for i:=1;i<len(os.Args);i++{if os.Args[i]=="-only" && i+1<len(os.Args){fixtureRole=os.Args[i+1];i++}}
exe,_:=canonicalUpdateExecutable();dir:=updateWorkDir(exe);os.MkdirAll(dir,0700)
mode,_:=os.ReadFile(filepath.Join(filepath.Dir(exe),"failure-mode"))
if appVersion=="v2.0.0" && string(mode)=="early-exit"{return}
if appVersion=="v2.0.0" && string(mode)=="early-hang"{for{time.Sleep(time.Second)}}
created,_:=processCreated(windows.CurrentProcess());p:=updateParticipant{PID:uint32(os.Getpid()),Created:created,Role:fixtureRole}
writeUpdateJSON(participantPath(dir,p.PID),p)
automaticUpdateHealthy()
if appVersion=="v2.0.0" && string(mode)=="health-crash"{time.Sleep(100*time.Millisecond);return}
for{if shouldStopForAutomaticUpdate(exe,dir,p,appVersion){return};time.Sleep(20*time.Millisecond)}
}
`

func buildAutomaticFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"auto_update.go", "auto_update_windows.go", "update.go"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(automaticFixtureSource), 0600)
	for _, name := range []string{"go.mod", "go.sum"} {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, name), b, 0600)
	}
	old, new := filepath.Join(dir, "old.exe"), filepath.Join(dir, "new.exe")
	for _, item := range []struct{ path, version string }{{old, "v1.0.0"}, {new, "v2.0.0"}} {
		cmd := exec.Command("go", "build", "-ldflags", "-X main.appVersion="+item.version, "-o", item.path, ".")
		cmd.Dir = dir
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build fixture: %v %s", err, b)
		}
	}
	return old, new
}
func waitAutomaticFixture(t *testing.T, target string, count int) []updateProcess {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		ps, err := runningUpdateProcesses(target, updateWorkDir(target))
		if err == nil && len(ps) == count {
			return ps
		}
		closeUpdateProcesses(ps)
		if time.Now().After(deadline) {
			t.Fatalf("expected %d fixture processes, got %d: %v", count, len(ps), err)
		}
		time.Sleep(30 * time.Millisecond)
	}
}
func TestAutomaticHelperRealWindowsLifecycle(t *testing.T) {
	old, new := buildAutomaticFixture(t)
	for _, mode := range []string{"success", "early-exit", "early-hang", "health-crash", "corrupt-candidate"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "fixture.exe")
			if err := copyUpdateFile(old, target); err != nil {
				t.Fatal(err)
			}
			work := updateWorkDir(target)
			os.MkdirAll(work, 0700)
			session, err := os.MkdirTemp(work, "session-")
			if err != nil {
				t.Fatal(err)
			}
			os.WriteFile(filepath.Join(dir, "failure-mode"), []byte(mode), 0600)
			t.Cleanup(func() {
				ps, _ := runningUpdateProcesses(target, work)
				for _, p := range ps {
					windows.TerminateProcess(p.handle, 1)
				}
				waitUpdateProcesses(ps, 5*time.Second)
				closeUpdateProcesses(ps)
			})
			for _, role := range []string{"claude", "codex"} {
				cmd := exec.Command(target, "-only", role)
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				cmd.Process.Release()
			}
			ps := waitAutomaticFixture(t, target, 2)
			closeUpdateProcesses(ps)
			copyUpdateFile(old, filepath.Join(session, "updater.exe"))
			copyUpdateFile(new, filepath.Join(session, "candidate.exe"))
			oh, _ := fileSHA256(old)
			nh, _ := fileSHA256(new)
			tx := updateTransaction{Target: target, OldHash: oh, NewHash: nh, Version: "v2.0.0", Token: strings.Repeat("a", 64)}
			if err := writeUpdateJSON(filepath.Join(session, "transaction.json"), tx); err != nil {
				t.Fatal(err)
			}
			if mode == "corrupt-candidate" {
				os.WriteFile(filepath.Join(session, "candidate.exe"), []byte("incomplete download"), 0600)
			}
			helper := exec.Command(filepath.Join(session, "updater.exe"), updateHelperFlag, filepath.Join(session, "transaction.json"), tx.Token)
			if err := helper.Start(); err != nil {
				t.Fatal(err)
			}
			var contender *exec.Cmd
			if mode == "success" {
				contender = exec.Command(filepath.Join(session, "updater.exe"), updateHelperFlag, filepath.Join(session, "transaction.json"), tx.Token)
				if err := contender.Start(); err != nil {
					t.Fatal(err)
				}
			}
			if err := helper.Wait(); err != nil {
				t.Fatalf("helper: %v", err)
			}
			if contender != nil {
				if err := contender.Wait(); err != nil {
					t.Fatalf("competing helper: %v", err)
				}
			}
			if b, err := os.ReadFile(filepath.Join(session, "error.txt")); err == nil {
				t.Log(string(b))
			}
			got, err := fileSHA256(target)
			if err != nil {
				t.Fatal(err)
			}
			want := oh
			if mode == "success" {
				want = nh
			}
			if got != want {
				t.Fatalf("installed hash %s want %s", got, want)
			}
			ps = waitAutomaticFixture(t, target, 2)
			roles := map[string]bool{}
			for _, p := range ps {
				roles[p.participant.Role] = true
			}
			closeUpdateProcesses(ps)
			if !roles["claude"] || !roles["codex"] {
				t.Fatal("sibling roles not preserved")
			}
			if mode == "success" {
				if h, _ := fileSHA256(filepath.Join(session, "previous.exe")); h != oh {
					t.Fatal("previous binary not preserved")
				}
				retry := exec.Command(filepath.Join(session, "updater.exe"), updateHelperFlag, filepath.Join(session, "transaction.json"), tx.Token)
				if b, err := retry.CombinedOutput(); err != nil {
					t.Fatalf("idempotent retry %v %s", err, b)
				}
			} else {
				b, err := os.ReadFile(filepath.Join(session, "error.txt"))
				if err != nil {
					t.Fatalf("no helper error: %v", err)
				}
				t.Log(string(b))
			}
		})
	}
}
