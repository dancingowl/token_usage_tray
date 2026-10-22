//go:build windows

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const updateHelperFlag = "--usage-tray-update-helper"
const updateHealthEnvironment = "USAGE_TRAY_UPDATE_HEALTH"
const updateTokenEnvironment = "USAGE_TRAY_UPDATE_TOKEN"

type updateParticipant struct {
	PID      uint32 `json:"pid"`
	Created  uint64 `json:"created"`
	Role     string `json:"role"`
	Discover bool   `json:"discover"`
}
type updateTransaction struct {
	Target  string `json:"target"`
	OldHash string `json:"old_hash"`
	NewHash string `json:"new_hash"`
	Version string `json:"version"`
	Token   string `json:"token"`
}
type updateProcess struct {
	handle      windows.Handle
	participant updateParticipant
}
type updateShutdownRequest struct {
	Session       string `json:"session"`
	Token         string `json:"token"`
	HelperPID     uint32 `json:"helper_pid"`
	HelperCreated uint64 `json:"helper_created"`
}

func automaticUpdateRestart() bool {
	path, token := os.Getenv(updateHealthEnvironment), os.Getenv(updateTokenEnvironment)
	if path == "" || token == "" {
		return false
	}
	exe, err := canonicalUpdateExecutable()
	if err != nil {
		return false
	}
	session := filepath.Dir(path)
	if !filepath.IsAbs(path) || !sameUpdatePath(filepath.Dir(session), updateWorkDir(exe)) || !strings.HasPrefix(filepath.Base(session), "session-") || !strings.HasPrefix(filepath.Base(path), "healthy-") {
		return false
	}
	var tx updateTransaction
	if readUpdateJSON(filepath.Join(session, "transaction.json"), &tx) != nil || tx.Token != token || !sameUpdatePath(tx.Target, exe) || tx.Version != appVersion {
		return false
	}
	hash, err := fileSHA256(exe)
	if err != nil || hash != tx.NewHash {
		return false
	}
	if _, err := os.Stat(filepath.Join(session, "complete")); err == nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(session, "error.txt")); err == nil {
		return false
	}
	var helper updateShutdownRequest
	if readUpdateJSON(filepath.Join(session, "helper.json"), &helper) != nil || helper.Token != token {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, helper.HelperPID)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	created, err := processCreated(h)
	if err != nil || created != helper.HelperCreated {
		return false
	}
	status, err := windows.WaitForSingleObject(h, 0)
	if err != nil || status != uint32(windows.WAIT_TIMEOUT) {
		return false
	}
	buf := make([]uint16, 32768)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) != nil {
		return false
	}
	return sameUpdatePath(windows.UTF16ToString(buf[:n]), filepath.Join(session, "updater.exe"))
}

var automaticUpdateStart sync.Once
var automaticUpdateHealthTimeout = 90 * time.Second

func automaticUpdateSupported() bool { return runtime.GOARCH == "amd64" }
func canonicalUpdateExecutable() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return canonicalUpdatePath(p)
}
func writeUpdateJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary, path)
	}
	return err
}
func readUpdateJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 16385))
	if err != nil {
		return err
	}
	if len(b) > 16384 {
		return errors.New("update control file too large")
	}
	return json.Unmarshal(b, value)
}
func openUpdateLock(dir string) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(dir, "update.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
func closeUpdateLock(f *os.File) {
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{})
	_ = f.Close()
}
func processCreated(h windows.Handle) (uint64, error) {
	var c, e, k, u windows.Filetime
	if err := windows.GetProcessTimes(h, &c, &e, &k, &u); err != nil {
		return 0, err
	}
	return uint64(c.HighDateTime)<<32 | uint64(c.LowDateTime), nil
}
func participantPath(dir string, pid uint32) string {
	return filepath.Join(dir, "participant-"+strconv.FormatUint(uint64(pid), 10)+".json")
}
func validUpdateRole(p updateParticipant) bool {
	return p.Role == "" || p.Role == "claude" || p.Role == "codex" || p.Role == "antigravity"
}
func startAutomaticUpdates(quit func()) {
	if !automaticUpdateSupported() || !stableUpdateVersion(appVersion) {
		return
	}
	automaticUpdateStart.Do(func() {
		exe, err := canonicalUpdateExecutable()
		if err != nil {
			setAutomaticUpdateState("error", "", err)
			return
		}
		dir := updateWorkDir(exe)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			setAutomaticUpdateState("error", "", err)
			return
		}
		created, err := processCreated(windows.CurrentProcess())
		if err != nil {
			setAutomaticUpdateState("error", "", err)
			return
		}
		p := updateParticipant{PID: uint32(os.Getpid()), Created: created, Role: tag(), Discover: runtimeAutoDiscover}
		if err := writeUpdateJSON(participantPath(dir, p.PID), p); err != nil {
			setAutomaticUpdateState("error", "", err)
			return
		}
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				var persisted automaticUpdateStatus
				if readUpdateJSON(filepath.Join(dir, "status.json"), &persisted) == nil {
					automaticUpdateState.Lock()
					if !persisted.CheckedAt.Before(automaticUpdateState.status.CheckedAt) {
						automaticUpdateState.status = persisted
					}
					automaticUpdateState.Unlock()
				}
				if !shouldStopForAutomaticUpdate(exe, dir, p, appVersion) {
					continue
				}
				quit()
				return
			}
		}()
		go func() {
			for {
				automaticUpdateAttempt(exe, dir)
				wait := time.Until(currentAutomaticUpdateStatus().NextCheckAt)
				if wait < time.Minute {
					wait = time.Minute
				}
				if wait > automaticUpdateInterval {
					wait = automaticUpdateInterval
				}
				time.Sleep(wait)
			}
		}()
	})
}
func shouldStopForAutomaticUpdate(exe, dir string, p updateParticipant, version string) bool {
	var active updateShutdownRequest
	if readUpdateJSON(filepath.Join(dir, "active.json"), &active) != nil || filepath.Base(active.Session) != active.Session || !strings.HasPrefix(active.Session, "session-") {
		return false
	}
	session := filepath.Join(dir, active.Session)
	var tx updateTransaction
	if readUpdateJSON(filepath.Join(session, "transaction.json"), &tx) != nil || tx.Token != active.Token || !sameUpdatePath(tx.Target, exe) || !newerRelease(version, tx.Version) {
		return false
	}
	var participants []updateParticipant
	if readUpdateJSON(filepath.Join(session, "participants.json"), &participants) != nil {
		return false
	}
	included := false
	for _, q := range participants {
		if q.PID == p.PID && q.Created == p.Created {
			included = true
		}
	}
	if !included {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, active.HelperPID)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	created, err := processCreated(h)
	if err != nil || created != active.HelperCreated {
		return false
	}
	status, err := windows.WaitForSingleObject(h, 0)
	if err != nil || status != uint32(windows.WAIT_TIMEOUT) {
		return false
	}
	buf := make([]uint16, 32768)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(h, 0, &buf[0], &n) != nil || !sameUpdatePath(windows.UTF16ToString(buf[:n]), filepath.Join(session, "updater.exe")) {
		return false
	}
	if lock, err := openUpdateLock(dir); err == nil {
		closeUpdateLock(lock)
		return false
	}
	return true
}
func automaticUpdateAttempt(exe, dir string) {
	lock, err := openUpdateLock(dir)
	if err != nil {
		return
	}
	defer closeUpdateLock(lock)
	// Persisted cooldown also covers helper failure and rollback across restarts.
	var persisted automaticUpdateStatus
	if readUpdateJSON(filepath.Join(dir, "status.json"), &persisted) == nil && time.Now().Before(persisted.NextCheckAt) {
		automaticUpdateState.Lock()
		automaticUpdateState.status = persisted
		automaticUpdateState.Unlock()
		return
	}
	defer func() {
		releaseChecker.mu.Lock()
		retryAt := releaseChecker.retryAt
		releaseChecker.mu.Unlock()
		automaticUpdateState.Lock()
		if retryAt.After(automaticUpdateState.status.NextCheckAt) {
			automaticUpdateState.status.NextCheckAt = retryAt
		}
		automaticUpdateState.Unlock()
		_ = writeUpdateJSON(filepath.Join(dir, "status.json"), currentAutomaticUpdateStatus())
	}()
	automaticUpdateState.Lock()
	automaticUpdateState.status.CheckedAt = time.Now()
	automaticUpdateState.status.NextCheckAt = time.Now().Add(automaticUpdateInterval)
	automaticUpdateState.Unlock()
	setAutomaticUpdateState("checking", "", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	r, err := fetchAutomaticRelease(ctx)
	if err != nil {
		setAutomaticUpdateState("error", "", err)
		return
	}
	if _, _, err = r.assets(); err != nil {
		setAutomaticUpdateState("error", r.Tag, err)
		return
	}
	if !newerRelease(appVersion, r.Tag) {
		setAutomaticUpdateState("idle", r.Tag, nil)
		return
	}
	cleanupAutomaticUpdateSessions(dir)
	setAutomaticUpdateState("downloading", r.Tag, nil)
	b, hash, err := verifiedUpdateBinary(ctx, r)
	if err != nil {
		setAutomaticUpdateState("error", r.Tag, err)
		return
	}
	image, err := pe.NewFile(bytes.NewReader(b))
	if err != nil {
		setAutomaticUpdateState("error", r.Tag, errors.New("download is not a Windows executable"))
		return
	}
	machine := image.Machine
	image.Close()
	if machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		setAutomaticUpdateState("error", r.Tag, errors.New("download is not Windows amd64"))
		return
	}
	oldHash, err := fileSHA256(exe)
	if err != nil {
		setAutomaticUpdateState("error", r.Tag, err)
		return
	}
	session, err := os.MkdirTemp(dir, "session-")
	if err != nil {
		setAutomaticUpdateState("error", r.Tag, err)
		return
	}
	var nonce [32]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		setAutomaticUpdateState("error", r.Tag, err)
		return
	}
	tx := updateTransaction{Target: exe, OldHash: oldHash, NewHash: hash, Version: r.Tag, Token: hex.EncodeToString(nonce[:])}
	err = os.WriteFile(filepath.Join(session, "candidate.exe"), b, 0o600)
	if err == nil {
		err = copyUpdateFile(exe, filepath.Join(session, "updater.exe"))
	}
	if err == nil {
		err = writeUpdateJSON(filepath.Join(session, "transaction.json"), tx)
	}
	if err != nil {
		setAutomaticUpdateState("error", r.Tag, err)
		return
	}
	cmd := exec.Command(filepath.Join(session, "updater.exe"), updateHelperFlag, filepath.Join(session, "transaction.json"), tx.Token)
	cmd.Dir = filepath.Dir(exe)
	setAutomaticUpdateState("installing", r.Tag, nil)
	if err = cmd.Start(); err != nil {
		setAutomaticUpdateState("error", r.Tag, err)
		return
	}
	go monitorAutomaticUpdateHelper(cmd, dir, session, r.Tag)
}
func monitorAutomaticUpdateHelper(cmd *exec.Cmd, dir, session, version string) {
	waitErr := cmd.Wait()
	if _, err := os.Stat(filepath.Join(session, "complete")); err == nil {
		return
	}
	message := "업데이트 도우미가 설치를 완료하지 못했습니다. 다음 예약 시도 때 다시 확인합니다."
	if b, err := os.ReadFile(filepath.Join(session, "error.txt")); err == nil && len(b) > 0 {
		message = string(b)
	}
	if waitErr != nil {
		logf("automatic update helper exited: %v", waitErr)
	}
	setAutomaticUpdateState("error", version, errors.New(message))
	deadline := time.Now().Add(30 * time.Second)
	for {
		lock, err := openUpdateLock(dir)
		if err == nil {
			status := currentAutomaticUpdateStatus()
			status.State = "error"
			status.Error = message
			status.Version = version
			if status.NextCheckAt.Before(time.Now()) {
				status.NextCheckAt = time.Now().Add(automaticUpdateInterval)
			}
			_ = writeUpdateJSON(filepath.Join(dir, "status.json"), status)
			automaticUpdateState.Lock()
			automaticUpdateState.status = status
			automaticUpdateState.Unlock()
			closeUpdateLock(lock)
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
func cleanupAutomaticUpdateSessions(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var newestComplete string
	var newestTime time.Time
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "session-") {
			continue
		}
		session := filepath.Join(dir, entry.Name())
		info, err := os.Stat(filepath.Join(session, "complete"))
		if err == nil && info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestComplete = session
		}
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "session-") {
			continue
		}
		session := filepath.Join(dir, entry.Name())
		resolved, err := filepath.EvalSymlinks(session)
		if err != nil || !strings.EqualFold(resolved, session) || !strings.EqualFold(filepath.Dir(resolved), dir) {
			continue
		}
		_, backupErr := os.Stat(filepath.Join(session, "previous.exe"))
		_, completeErr := os.Stat(filepath.Join(session, "complete"))
		// Keep the most recent successful rollback binary and every unfinished
		// transaction that still owns an original binary needing recovery.
		if session == newestComplete || backupErr == nil && completeErr != nil {
			continue
		}
		_ = os.RemoveAll(session)
	}
}
func copyUpdateFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, io.LimitReader(in, automaticUpdateMaxBinary+1))
	if err == nil {
		err = out.Sync()
	}
	ce := out.Close()
	if err == nil {
		err = ce
	}
	return err
}
func handleAutomaticUpdateMode() bool {
	if len(os.Args) < 2 || os.Args[1] != updateHelperFlag {
		return false
	}
	if len(os.Args) != 4 {
		return true
	}
	err := runAutomaticUpdateHelper(os.Args[2], os.Args[3])
	if err != nil {
		logf("automatic update helper: %v", err)
	}
	return true
}
func validateUpdateTransaction(manifest, token, helper string) (updateTransaction, error) {
	var tx updateTransaction
	if !filepath.IsAbs(manifest) {
		return tx, errors.New("invalid helper control path")
	}
	var err error
	manifest, err = longUpdatePath(manifest)
	if err != nil {
		return tx, err
	}
	helper, err = longUpdatePath(helper)
	if err != nil {
		return tx, err
	}
	if !filepath.IsAbs(manifest) || filepath.Base(manifest) != "transaction.json" || filepath.Base(helper) != "updater.exe" || !sameUpdatePath(filepath.Dir(helper), filepath.Dir(manifest)) {
		return tx, errors.New("invalid helper control path")
	}
	if err := readUpdateJSON(manifest, &tx); err != nil {
		return tx, err
	}
	if len(token) != 64 || tx.Token != token || !filepath.IsAbs(tx.Target) || !stableUpdateVersion(tx.Version) {
		return tx, errors.New("invalid helper transaction")
	}
	if _, err := hex.DecodeString(token); err != nil {
		return tx, err
	}
	if _, err := parseSHA256(tx.OldHash); err != nil {
		return tx, err
	}
	if _, err := parseSHA256(tx.NewHash); err != nil {
		return tx, err
	}
	session := filepath.Dir(manifest)
	if !strings.HasPrefix(filepath.Base(session), "session-") || !sameUpdatePath(filepath.Dir(session), updateWorkDir(tx.Target)) {
		return tx, errors.New("transaction is outside executable update directory")
	}
	resolved, err := canonicalUpdatePath(session)
	expanded, expandErr := longUpdatePath(session)
	if err != nil || expandErr != nil || !strings.EqualFold(resolved, expanded) {
		return tx, errors.New("redirected update directory")
	}
	actual, err := fileSHA256(helper)
	if err != nil || actual != tx.OldHash {
		return tx, errors.New("helper binary mismatch")
	}
	tx.Target, err = canonicalUpdatePath(tx.Target)
	if err != nil {
		return tx, err
	}
	return tx, nil
}

// GetLongPathName expands 8.3 aliases without resolving reparse points. Comparing
// it with EvalSymlinks permits aliases while retaining the redirected-directory
// check on the helper control directory.
func longUpdatePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	p, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		return "", err
	}
	buf := make([]uint16, 32768)
	n, err := windows.GetLongPathName(p, &buf[0], uint32(len(buf)))
	if err != nil {
		return "", err
	}
	if n == 0 || n >= uint32(len(buf)) {
		return "", errors.New("invalid expanded update path")
	}
	return windows.UTF16ToString(buf[:n]), nil
}
func runningUpdateProcesses(target, dir string) ([]updateProcess, error) {
	var err error
	target, err = canonicalUpdatePath(target)
	if err != nil {
		return nil, err
	}
	shortBase := filepath.Base(target)
	if p, e := windows.UTF16PtrFromString(target); e == nil {
		buf := make([]uint16, 32768)
		if n, e := windows.GetShortPathName(p, &buf[0], uint32(len(buf))); e == nil && n > 0 && n < uint32(len(buf)) {
			shortBase = filepath.Base(windows.UTF16ToString(buf[:n]))
		}
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	err = windows.Process32First(snapshot, &entry)
	var found []updateProcess
	fail := func(err error) ([]updateProcess, error) { closeUpdateProcesses(found); return nil, err }
	for err == nil {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, filepath.Base(target)) || strings.EqualFold(name, shortBase) {
			h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, entry.ProcessID)
			if e != nil {
				// A snapshot can contain a process that has already disappeared.
				if errors.Is(e, windows.ERROR_INVALID_PARAMETER) {
					err = windows.Process32Next(snapshot, &entry)
					continue
				}
				return fail(fmt.Errorf("cannot inspect executable process: %w", e))
			}
			if status, e := windows.WaitForSingleObject(h, 0); e == nil && status == windows.WAIT_OBJECT_0 {
				windows.CloseHandle(h)
				err = windows.Process32Next(snapshot, &entry)
				continue
			}
			buf := make([]uint16, 32768)
			size := uint32(len(buf))
			e = windows.QueryFullProcessImageName(h, 0, &buf[0], &size)
			if e != nil {
				status, waitErr := windows.WaitForSingleObject(h, 0)
				windows.CloseHandle(h)
				if waitErr == nil && status == windows.WAIT_OBJECT_0 {
					err = windows.Process32Next(snapshot, &entry)
					continue
				}
				return fail(e)
			}
			if sameUpdatePath(windows.UTF16ToString(buf[:size]), target) {
				created, e := processCreated(h)
				if e != nil {
					windows.CloseHandle(h)
					return fail(e)
				}
				var p updateParticipant
				if readUpdateJSON(participantPath(dir, entry.ProcessID), &p) != nil || p.PID != entry.ProcessID || p.Created != created || !validUpdateRole(p) {
					windows.CloseHandle(h)
					return fail(errors.New("running executable has no matching tray registration"))
				}
				found = append(found, updateProcess{h, p})
			} else {
				windows.CloseHandle(h)
			}
		}
		err = windows.Process32Next(snapshot, &entry)
	}
	if err != windows.ERROR_NO_MORE_FILES {
		return fail(err)
	}
	return found, nil
}
func closeUpdateProcesses(processes []updateProcess) {
	for _, p := range processes {
		windows.CloseHandle(p.handle)
	}
}
func waitUpdateProcesses(processes []updateProcess, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		done := true
		for _, p := range processes {
			status, err := windows.WaitForSingleObject(p.handle, 0)
			if err != nil || status != windows.WAIT_OBJECT_0 {
				done = false
				break
			}
		}
		if done {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}
func automaticUpdateHealthy() {
	path, token := os.Getenv(updateHealthEnvironment), os.Getenv(updateTokenEnvironment)
	if path == "" || token == "" {
		return
	}
	exe, err := canonicalUpdateExecutable()
	if err != nil {
		return
	}
	var tx updateTransaction
	if !filepath.IsAbs(path) || !sameUpdatePath(filepath.Dir(filepath.Dir(path)), updateWorkDir(exe)) || !strings.HasPrefix(filepath.Base(filepath.Dir(path)), "session-") || !strings.HasPrefix(filepath.Base(path), "healthy-") {
		return
	}
	if readUpdateJSON(filepath.Join(filepath.Dir(path), "transaction.json"), &tx) != nil || tx.Token != token || !sameUpdatePath(tx.Target, exe) || tx.Version != appVersion {
		return
	}
	hash, err := fileSHA256(exe)
	if err != nil || hash != tx.NewHash {
		return
	}
	_ = os.WriteFile(path, []byte(token), 0o600)
}

func runAutomaticUpdateHelper(manifest, token string) error {
	helper, err := canonicalUpdateExecutable()
	if err != nil {
		return err
	}
	tx, err := validateUpdateTransaction(manifest, token, helper)
	if err != nil {
		return err
	}
	manifest, err = canonicalUpdatePath(manifest)
	if err != nil {
		return err
	}
	session, dir := filepath.Dir(manifest), filepath.Dir(filepath.Dir(manifest))
	var lock *os.File
	deadline := time.Now().Add(30 * time.Second)
	for {
		lock, err = openUpdateLock(dir)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			return errors.New("another update is active")
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer closeUpdateLock(lock)
	result := func() error {
		// Persist the canonical target before replacement: a replacement file
		// can receive a different 8.3 alias from the original file.
		if err := writeUpdateJSON(manifest, tx); err != nil {
			return err
		}
		created, err := processCreated(windows.CurrentProcess())
		if err != nil {
			return err
		}
		if err = writeUpdateJSON(filepath.Join(session, "helper.json"), updateShutdownRequest{filepath.Base(session), tx.Token, uint32(os.Getpid()), created}); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(session, "complete")); err == nil {
			return nil
		}
		if b, err := os.ReadFile(filepath.Join(session, "error.txt")); err == nil {
			return fmt.Errorf("transaction already failed: %s", b)
		}
		err = installAutomaticUpdate(tx, session, dir)
		if err != nil {
			_ = os.WriteFile(filepath.Join(session, "error.txt"), []byte(err.Error()), 0o600)
		} else {
			_ = os.WriteFile(filepath.Join(session, "complete"), []byte(tx.Version), 0o600)
		}
		var status automaticUpdateStatus
		_ = readUpdateJSON(filepath.Join(dir, "status.json"), &status)
		status.State = "idle"
		status.Version = tx.Version
		status.Error = ""
		if err != nil {
			status.State = "error"
			status.Error = err.Error()
		}
		if status.CheckedAt.IsZero() {
			status.CheckedAt = time.Now()
		}
		status.NextCheckAt = time.Now().Add(automaticUpdateInterval)
		_ = writeUpdateJSON(filepath.Join(dir, "status.json"), status)
		return err
	}()
	return result
}
func installAutomaticUpdate(tx updateTransaction, session, dir string) error {
	candidate, backup := filepath.Join(session, "candidate.exe"), filepath.Join(session, "previous.exe")
	newHash, err := fileSHA256(candidate)
	if err != nil || newHash != tx.NewHash {
		return errors.New("staged executable SHA256 mismatch")
	}
	oldHash, err := fileSHA256(tx.Target)
	if err != nil || oldHash != tx.OldHash {
		return errors.New("installed executable changed; refusing update")
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		return errors.New("transaction already started; backup preserved for recovery")
	}
	processes, err := runningUpdateProcesses(tx.Target, dir)
	if err != nil {
		return err
	}
	defer closeUpdateProcesses(processes)
	if len(processes) == 0 {
		return errors.New("no registered tray processes to restart")
	}
	roles := make([]updateParticipant, len(processes))
	for i, p := range processes {
		roles[i] = p.participant
	}
	if err = writeUpdateJSON(filepath.Join(session, "participants.json"), roles); err != nil {
		return err
	}
	active := filepath.Join(dir, "active.json")
	// A crashed transaction's marker is inert while no helper holds the lock.
	_ = os.Remove(active)
	helperCreated, err := processCreated(windows.CurrentProcess())
	if err != nil {
		return err
	}
	if err = writeUpdateJSON(active, updateShutdownRequest{filepath.Base(session), tx.Token, uint32(os.Getpid()), helperCreated}); err != nil {
		return err
	}
	defer os.Remove(active)
	if !waitUpdateProcesses(processes, 20*time.Second) {
		_ = os.Remove(active)
		// Relaunch only roles whose original handles actually exited. A busy
		// or unresponsive tray is kept running and never forcibly terminated.
		for _, p := range processes {
			status, e := windows.WaitForSingleObject(p.handle, 0)
			if e == nil && status == windows.WAIT_OBJECT_0 {
				_, _ = launchUpdatedRole(tx.Target, p.participant, "", "")
			}
		}
		return errors.New("tray shutdown timed out; original executable retained")
	}
	_ = os.Remove(active)
	if err = copyUpdateFile(tx.Target, backup); err != nil {
		restartUpdateRoles(tx.Target, roles)
		return fmt.Errorf("preserve old executable: %w", err)
	}
	if err = replaceUpdateFile(candidate, tx.Target); err != nil {
		restartUpdateRoles(tx.Target, roles)
		return fmt.Errorf("install failed, original retained: %w", err)
	}
	if err = launchAndCheckUpdate(tx, session, roles); err == nil {
		return nil
	}
	// Stop only a process whose open handle still identifies this exact new
	// executable. Retained handles prevent PID reuse from targeting another app.
	newProcesses, inspectErr := runningUpdateProcesses(tx.Target, dir)
	if inspectErr != nil {
		return fmt.Errorf("health failed (%v); cannot safely stop new process (%v); original preserved at %s", err, inspectErr, backup)
	}
	for _, p := range newProcesses {
		_ = windows.TerminateProcess(p.handle, 1)
	}
	stopped := waitUpdateProcesses(newProcesses, 10*time.Second)
	closeUpdateProcesses(newProcesses)
	if !stopped {
		return fmt.Errorf("health failed; new process did not exit; original preserved at %s", backup)
	}
	if restoreErr := replaceUpdateFile(backup, tx.Target); restoreErr != nil {
		return fmt.Errorf("health failed; cannot restore original (%v); original preserved at %s", restoreErr, backup)
	}
	restartUpdateRoles(tx.Target, roles)
	return fmt.Errorf("health failed; original restored: %w", err)
}
func replaceUpdateFile(source, target string) error {
	src, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		err = windows.MoveFileEx(src, dst, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
		if err == nil {
			return nil
		}
		if !errors.Is(err, windows.ERROR_SHARING_VIOLATION) && !errors.Is(err, windows.ERROR_ACCESS_DENIED) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}
func launchUpdatedRole(exe string, p updateParticipant, health, token string) (*exec.Cmd, error) {
	if !validUpdateRole(p) {
		return nil, errors.New("invalid restart role")
	}
	var args []string
	if p.Role != "" {
		args = append(args, "-only", p.Role)
	}
	if p.Discover {
		args = append(args, "-discover")
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(env), updateHealthEnvironment+"=") && !strings.HasPrefix(strings.ToUpper(env), updateTokenEnvironment+"=") {
			cmd.Env = append(cmd.Env, env)
		}
	}
	if health != "" {
		cmd.Env = append(cmd.Env, updateHealthEnvironment+"="+health, updateTokenEnvironment+"="+token)
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
func restartUpdateRoles(exe string, roles []updateParticipant) {
	for _, p := range roles {
		cmd, err := launchUpdatedRole(exe, p, "", "")
		if err == nil {
			_ = cmd.Process.Release()
		}
	}
}
func launchAndCheckUpdate(tx updateTransaction, session string, roles []updateParticipant) (result error) {
	var launched []*exec.Cmd
	var handles []windows.Handle
	defer func() {
		for _, cmd := range launched {
			if result != nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
			_ = cmd.Process.Release()
		}
		for _, h := range handles {
			windows.CloseHandle(h)
		}
	}()
	paths := make([]string, len(roles))
	for i, p := range roles {
		paths[i] = filepath.Join(session, "healthy-"+strconv.Itoa(i))
		_ = os.Remove(paths[i])
		cmd, err := launchUpdatedRole(tx.Target, p, paths[i], tx.Token)
		if err != nil {
			for _, started := range launched {
				_ = started.Process.Kill()
				_ = started.Wait()
			}
			return err
		}
		launched = append(launched, cmd)
		h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(cmd.Process.Pid))
		if err != nil {
			return err
		}
		handles = append(handles, h)
	}
	deadline := time.Now().Add(automaticUpdateHealthTimeout)
	for {
		for _, h := range handles {
			status, err := windows.WaitForSingleObject(h, 0)
			if err != nil || status != uint32(windows.WAIT_TIMEOUT) {
				return errors.New("new tray process exited before health stabilization")
			}
		}
		healthy := true
		for _, path := range paths {
			b, e := os.ReadFile(path)
			if e != nil || string(b) != tx.Token {
				healthy = false
				break
			}
		}
		if healthy {
			// The health acknowledgement must remain valid through a short
			// stabilization period; this catches immediate post-start crashes.
			time.Sleep(2 * time.Second)
			for _, h := range handles {
				status, err := windows.WaitForSingleObject(h, 0)
				if err != nil || status != uint32(windows.WAIT_TIMEOUT) {
					return errors.New("new tray process exited after health acknowledgement")
				}
			}
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("new version did not acknowledge tray readiness within %s", automaticUpdateHealthTimeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
