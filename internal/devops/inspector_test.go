package devops

import (
	"testing"
)

func TestValidateUnitName(t *testing.T) {
	valid := []string{"nginx", "nginx.service", "php8.2-fpm", "my_container-1", "user@1000.service"}
	for _, name := range valid {
		if err := ValidateUnitName(name); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", name, err)
		}
	}

	invalid := []string{
		"",
		"-rf",
		"--help",
		"nginx; rm -rf /",
		"container$(whoami)",
		"../etc/passwd",
		"my container",
	}
	for _, name := range invalid {
		if err := ValidateUnitName(name); err == nil {
			t.Errorf("expected %q to be rejected by SEC-05 validator", name)
		}
	}
}

func TestValidateAction(t *testing.T) {
	for _, act := range []string{"start", "stop", "restart"} {
		if err := ValidateAction(act); err != nil {
			t.Errorf("expected action %q to be valid: %v", act, err)
		}
	}
	for _, act := range []string{"", "kill", "rm", "restart;id"} {
		if err := ValidateAction(act); err == nil {
			t.Errorf("expected action %q to be rejected", act)
		}
	}
}

func TestSanitizeText(t *testing.T) {
	raw := "\x1b[31mError:\x1b[0m failed\x00\x07 to start"
	got := SanitizeText(raw)
	want := "Error: failed to start"
	if got != want {
		t.Fatalf("SanitizeText() = %q, want %q", got, want)
	}
}

func TestParseDevOpsSnapshot(t *testing.T) {
	sample := `===SECTION:DOCKER_INFO===
Docker version 26.1.4, build 5650f9b
===SECTION:DOCKER_PS===
a1b2c3d4e5f67890	web-proxy	nginx:alpine	running	Up 3 days	0.0.0.0:80->80/tcp, 0.0.0.0:443->443/tcp
f9e8d7c6b5a43210	 worker-queue 	redis:7	exited	Exited (0) 2 hours ago	
===SECTION:DOCKER_STATS===
a1b2c3d4e5f67890	1.45%	42.5MiB / 2GiB	2.08%	1.2MB / 890kB	0B / 4.1kB
===SECTION:SERVICES===
nginx.service loaded active running A high performance web server
mysql.service loaded failed failed MySQL Community Server
ssh.service loaded active running OpenBSD Secure Shell server
random-oneshot.service loaded inactive dead Oneshot helper
===SECTION:PORTS===
tcp LISTEN 0 511 0.0.0.0:80 0.0.0.0:* users:(("nginx",pid=1024,fd=6))
tcp LISTEN 0 128 127.0.0.1:6379 0.0.0.0:* users:(("redis-server",pid=2048,fd=7))
udp UNCONN 0 0 127.0.0.53%lo:53 0.0.0.0:* users:(("systemd-resolve",pid=512,fd=13))
===SECTION:FIREWALL===
TOOL:ufw
Status: active
To Action From
22/tcp ALLOW Anywhere
===SECTION:UPDATES===
PKG:apt
Inst libssl3 [3.0.2-0ubuntu1.12] (3.0.2-0ubuntu1.15 Ubuntu:22.04/jammy-security [amd64])
Inst curl [7.81.0-1ubuntu1.14] (7.81.0-1ubuntu1.15 Ubuntu:22.04/jammy-updates [amd64])
===SECTION:SSH_AUDIT===
---FAILED_LOGS---
Nov 03 21:10:01 srv sshd[9111]: Failed password for invalid user admin from 203.0.113.45 port 51234 ssh2
Nov 03 21:10:05 srv sshd[9112]: Failed password for root from 203.0.113.45 port 51238 ssh2
Nov 03 21:11:12 srv sshd[9115]: Invalid user deploy from 198.51.100.77 port 39100 ssh2
---WHO---
deploy   pts/0        2026-11-03 20:00 (10.0.0.5)
---SSHD_CFG---
PermitRootLogin no
PasswordAuthentication no
---FAIL2BAN---
active
`

	snap := ParseDevOpsSnapshot("srv-1", sample)

	if !snap.DockerAvailable {
		t.Errorf("expected DockerAvailable = true")
	}
	if len(snap.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(snap.Containers))
	}
	if snap.Containers[0].Name != "web-proxy" || snap.Containers[0].CPUPercent != 1.45 {
		t.Errorf("unexpected container[0]: %+v", snap.Containers[0])
	}

	if len(snap.Services) != 3 {
		t.Fatalf("expected 3 filtered services, got %d", len(snap.Services))
	}
	// Failed service should be sorted first
	if snap.Services[0].Name != "mysql" || snap.Services[0].ActiveState != "failed" {
		t.Errorf("expected failed service mysql first, got %+v", snap.Services[0])
	}

	if len(snap.ListeningPorts) != 3 {
		t.Fatalf("expected 3 listening ports, got %d", len(snap.ListeningPorts))
	}
	if snap.ListeningPorts[0].Port != 80 || snap.ListeningPorts[0].Exposure != "public" {
		t.Errorf("expected port 80 public first, got %+v", snap.ListeningPorts[0])
	}

	if !snap.Firewall.Installed || !snap.Firewall.Active || snap.Firewall.ToolName != "ufw" {
		t.Errorf("unexpected firewall status: %+v", snap.Firewall)
	}

	if snap.Security.PendingUpdates != 2 || snap.Security.SecurityUpdates != 1 {
		t.Errorf("unexpected updates count: %+v", snap.Security)
	}
	if snap.Security.FailedSSHCount != 3 || len(snap.Security.TopAttackers) != 2 {
		t.Fatalf("unexpected ssh brute-force stats: %+v", snap.Security)
	}
	if snap.Security.TopAttackers[0].IP != "203.0.113.45" || snap.Security.TopAttackers[0].Attempts != 2 {
		t.Errorf("unexpected top attacker: %+v", snap.Security.TopAttackers[0])
	}
	if !snap.Security.Fail2banActive || snap.Security.PermitRootLogin != "no" {
		t.Errorf("unexpected sshd hardening state: %+v", snap.Security)
	}
}
