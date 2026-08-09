// SPDX-License-Identifier: GPL-3.0-only

package player

import (
	"log"
	"os/exec"
	"sync"
	"time"
)

type ProcessInfo struct {
	PID       int       `json:"pid"`
	Name      string    `json:"name"`
	StartedAt time.Time `json:"started_at"`
	Cmd       *exec.Cmd `json:"-"`
}

type ProcessManager struct {
	mu        sync.RWMutex
	processes map[int]*ProcessInfo
}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[int]*ProcessInfo),
	}
}

func (pm *ProcessManager) RegisterProcess(name string, cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	info := &ProcessInfo{
		PID:       cmd.Process.Pid,
		Name:      name,
		StartedAt: time.Now(),
		Cmd:       cmd,
	}

	pm.processes[cmd.Process.Pid] = info
	log.Printf("[ProcessManager] Registered child process '%s' (PID %d)", name, cmd.Process.Pid)

	// Asynchronously monitor process termination
	go func(pid int) {
		_ = cmd.Wait()
		pm.mu.Lock()
		delete(pm.processes, pid)
		pm.mu.Unlock()
		log.Printf("[ProcessManager] Process '%s' (PID %d) exited cleanly.", name, pid)
	}(cmd.Process.Pid)
}

func (pm *ProcessManager) StopAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.processes) == 0 {
		return
	}

	log.Printf("[ProcessManager] Stopping %d active child processes gracefully...", len(pm.processes))

	for _, info := range pm.processes {
		if info.Cmd != nil && info.Cmd.Process != nil {
			_ = stopChildProcess(info.Cmd)
		}
	}

	// Give processes 1.5 seconds to terminate gracefully
	time.Sleep(1500 * time.Millisecond)

	// Force SIGKILL for any remaining processes
	for pid, info := range pm.processes {
		if info.Cmd != nil && info.Cmd.Process != nil {
			log.Printf("[ProcessManager] Force killing un-terminated process '%s' (PID %d)...", info.Name, pid)
			_ = info.Cmd.Process.Kill()
		}
	}

	pm.processes = make(map[int]*ProcessInfo)
	log.Println("[ProcessManager] All child processes terminated. Zero process leak.")
}
