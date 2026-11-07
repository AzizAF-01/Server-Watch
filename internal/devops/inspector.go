package devops

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"serverwatch/internal/model"
	"serverwatch/internal/sshpool"
)

const maxInspectOutputBytes = 1024 * 1024

var (
	// SEC-05: Strict identifier pattern for service units and container names/IDs.
	// Rejects leading hyphens to prevent CLI flag injection.
	unitNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9@._-]{0,127}$`)

	// SEC-03: Strip ANSI escape sequences from remote command output.
	ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

	// IPv4/IPv6 extraction after "from <ip>" in SSH auth logs.
	sshFromIPRegex = regexp.MustCompile(`from\s+([0-9a-fA-F:.]+)`)

	// Process name and PID extraction from ss/netstat output: users:(("nginx",pid=1234,fd=6))
	ssProcessRegex = regexp.MustCompile(`"([^"]+)",pid=(\d+)`)
	netstatPIDRegex = regexp.MustCompile(`(\d+)/([^\s:]+)`)
)

var keyInfrastructureServices = map[string]bool{
	"nginx":            true,
	"apache2":          true,
	"httpd":            true,
	"caddy":            true,
	"haproxy":          true,
	"docker":           true,
	"containerd":       true,
	"mysql":            true,
	"mariadb":          true,
	"postgresql":       true,
	"redis":            true,
	"redis-server":     true,
	"ssh":              true,
	"sshd":             true,
	"ufw":              true,
	"firewalld":        true,
	"fail2ban":         true,
	"cron":             true,
	"crond":            true,
	"supervisor":       true,
	"php-fpm":          true,
	"php8.1-fpm":       true,
	"php8.2-fpm":       true,
	"php8.3-fpm":       true,
}

const devopsInspectScript = `
export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH
export LC_ALL=C

echo "===SECTION:DOCKER_INFO==="
if command -v docker >/dev/null 2>&1; then
  docker -v 2>/dev/null || true
elif command -v podman >/dev/null 2>&1; then
  podman -v 2>/dev/null || true
fi

echo "===SECTION:DOCKER_PS==="
if command -v docker >/dev/null 2>&1; then
  (docker ps -a --no-trunc --format "{{.ID}}	{{.Names}}	{{.Image}}	{{.State}}	{{.Status}}	{{.Ports}}" 2>/dev/null || sudo -n docker ps -a --no-trunc --format "{{.ID}}	{{.Names}}	{{.Image}}	{{.State}}	{{.Status}}	{{.Ports}}" 2>/dev/null) || true
elif command -v podman >/dev/null 2>&1; then
  podman ps -a --no-trunc --format "{{.ID}}	{{.Names}}	{{.Image}}	{{.State}}	{{.Status}}	{{.Ports}}" 2>/dev/null || true
fi

echo "===SECTION:DOCKER_STATS==="
if command -v docker >/dev/null 2>&1; then
  (timeout 4 docker stats --no-stream --no-trunc --format "{{.ID}}	{{.CPUPerc}}	{{.MemUsage}}	{{.MemPerc}}	{{.NetIO}}	{{.BlockIO}}" 2>/dev/null || timeout 4 sudo -n docker stats --no-stream --no-trunc --format "{{.ID}}	{{.CPUPerc}}	{{.MemUsage}}	{{.MemPerc}}	{{.NetIO}}	{{.BlockIO}}" 2>/dev/null) || true
fi

echo "===SECTION:SERVICES==="
if command -v systemctl >/dev/null 2>&1; then
  systemctl list-units --type=service --all --no-pager --no-legend --plain 2>/dev/null | head -n 180 || true
fi

echo "===SECTION:PORTS==="
(sudo -n ss -tulpnH 2>/dev/null || ss -tulpnH 2>/dev/null || ss -tulnH 2>/dev/null || netstat -tulpn 2>/dev/null || netstat -tuln 2>/dev/null) | head -n 120 || true

echo "===SECTION:FIREWALL==="
if command -v ufw >/dev/null 2>&1; then
  echo "TOOL:ufw"
  (sudo -n ufw status 2>/dev/null || ufw status 2>/dev/null) | head -n 15 || true
elif command -v firewall-cmd >/dev/null 2>&1; then
  echo "TOOL:firewalld"
  (sudo -n firewall-cmd --state 2>/dev/null || firewall-cmd --state 2>/dev/null) || true
elif command -v iptables >/dev/null 2>&1; then
  echo "TOOL:iptables"
  (sudo -n iptables -S 2>/dev/null || iptables -S 2>/dev/null) | head -n 10 || true
else
  echo "TOOL:none"
fi

echo "===SECTION:UPDATES==="
if command -v apt-get >/dev/null 2>&1; then
  echo "PKG:apt"
  apt-get -s -o Debug::NoLocking=true upgrade 2>/dev/null | grep -E "^Inst " || true
elif command -v dnf >/dev/null 2>&1; then
  echo "PKG:dnf"
  dnf check-update -q 2>/dev/null | grep -E "^[a-zA-Z0-9]" || true
elif command -v yum >/dev/null 2>&1; then
  echo "PKG:yum"
  yum check-update -q 2>/dev/null | grep -E "^[a-zA-Z0-9]" || true
elif command -v apk >/dev/null 2>&1; then
  echo "PKG:apk"
  apk version -l '<' 2>/dev/null | grep -v "Installed:" || true
else
  echo "PKG:unknown"
fi

echo "===SECTION:SSH_AUDIT==="
echo "---FAILED_LOGS---"
(sudo -n journalctl -u ssh -u sshd -n 400 --no-pager 2>/dev/null || journalctl -u ssh -u sshd -n 400 --no-pager 2>/dev/null || sudo -n tail -n 400 /var/log/auth.log /var/log/secure 2>/dev/null || tail -n 400 /var/log/auth.log /var/log/secure 2>/dev/null) | grep -E "(Failed password|Invalid user)" || true
echo "---WHO---"
who 2>/dev/null || true
echo "---SSHD_CFG---"
(sudo -n grep -iE "^[[:space:]]*(PermitRootLogin|PasswordAuthentication)" /etc/ssh/sshd_config 2>/dev/null || grep -iE "^[[:space:]]*(PermitRootLogin|PasswordAuthentication)" /etc/ssh/sshd_config 2>/dev/null) || true
echo "---FAIL2BAN---"
systemctl is-active fail2ban 2>/dev/null || true
`

// Inspector collects and manages DevOps ecosystem telemetry over SSH.
type Inspector struct {
	pool *sshpool.SSHPool
}

func NewInspector(pool *sshpool.SSHPool) *Inspector {
	return &Inspector{pool: pool}
}

// ValidateUnitName enforces SEC-05 on service names and container identifiers.
func ValidateUnitName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("nama unit/container tidak boleh kosong")
	}
	if !unitNameRegex.MatchString(trimmed) {
		return fmt.Errorf("nama '%s' mengandung karakter tidak valid (SEC-05)", trimmed)
	}
	return nil
}

// ValidateAction restricts control actions to safe, known operations (SEC-04/SEC-05).
func ValidateAction(action string) error {
	switch action {
	case "start", "stop", "restart":
		return nil
	default:
		return fmt.Errorf("aksi '%s' tidak diizinkan; gunakan start, stop, atau restart", action)
	}
}

// SanitizeText removes ANSI sequences and control characters (SEC-03).
func SanitizeText(s string) string {
	cleaned := ansiEscapeRegex.ReplaceAllString(s, "")
	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		if r == '\n' || r == '\t' || r >= 0x20 {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// InspectServer runs a single consolidated SSH probe to gather Docker, Systemd, Port, Firewall, and Security data.
func (ins *Inspector) InspectServer(ctx context.Context, srv *model.Server) (*model.DevOpsSnapshot, error) {
	if ins.pool == nil {
		return nil, fmt.Errorf("ssh pool belum diinisialisasi")
	}

	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rawOut, err := ins.pool.ExecuteCommand(probeCtx, srv, devopsInspectScript)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data DevOps dari server %s: %w", srv.Name, err)
	}

	if len(rawOut) > maxInspectOutputBytes {
		rawOut = rawOut[:maxInspectOutputBytes]
	}

	return ParseDevOpsSnapshot(srv.ID, rawOut), nil
}

// ManageContainer executes start, stop, or restart on a target Docker/Podman container.
func (ins *Inspector) ManageContainer(ctx context.Context, srv *model.Server, containerIDOrName string, action string) error {
	if err := ValidateUnitName(containerIDOrName); err != nil {
		return err
	}
	if err := ValidateAction(action); err != nil {
		return err
	}

	target := strings.TrimSpace(containerIDOrName)
	cmd := fmt.Sprintf(
		"if command -v docker >/dev/null 2>&1; then (docker %s '%s' 2>&1 || sudo -n docker %s '%s' 2>&1); elif command -v podman >/dev/null 2>&1; then podman %s '%s' 2>&1; else echo 'Docker/Podman tidak tersedia' >&2; exit 1; fi",
		action, target, action, target, action, target,
	)

	execCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	out, err := ins.pool.ExecuteCommand(execCtx, srv, cmd)
	if err != nil {
		return fmt.Errorf("gagal %s container %s: %s", action, target, SanitizeText(out))
	}
	return nil
}

// GetContainerLogs retrieves recent stdout/stderr lines from a container.
func (ins *Inspector) GetContainerLogs(ctx context.Context, srv *model.Server, containerIDOrName string, tailLines int) (string, error) {
	if err := ValidateUnitName(containerIDOrName); err != nil {
		return "", err
	}
	if tailLines <= 0 || tailLines > 500 {
		tailLines = 80
	}

	target := strings.TrimSpace(containerIDOrName)
	cmd := fmt.Sprintf(
		"if command -v docker >/dev/null 2>&1; then (docker logs --tail %d '%s' 2>&1 || sudo -n docker logs --tail %d '%s' 2>&1); elif command -v podman >/dev/null 2>&1; then podman logs --tail %d '%s' 2>&1; fi",
		tailLines, target, tailLines, target, tailLines, target,
	)

	execCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	out, err := ins.pool.ExecuteCommand(execCtx, srv, cmd)
	if err != nil && strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("gagal membaca log container %s: %w", target, err)
	}
	sanitized := SanitizeText(out)
	if sanitized == "" {
		return fmt.Sprintf("Tidak ada output log untuk container %s.", target), nil
	}
	return sanitized, nil
}

// ManageService executes start, stop, or restart on a systemd service unit.
func (ins *Inspector) ManageService(ctx context.Context, srv *model.Server, serviceName string, action string) error {
	if err := ValidateUnitName(serviceName); err != nil {
		return err
	}
	if err := ValidateAction(action); err != nil {
		return err
	}

	unit := strings.TrimSpace(serviceName)
	if !strings.HasSuffix(unit, ".service") {
		unit += ".service"
	}

	cmd := fmt.Sprintf(
		"sudo -n systemctl %s '%s' 2>&1 || systemctl %s '%s' 2>&1",
		action, unit, action, unit,
	)

	execCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	out, err := ins.pool.ExecuteCommand(execCtx, srv, cmd)
	if err != nil {
		msg := SanitizeText(out)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("gagal %s service %s (pastikan user memiliki izin sudo NOPASSWD untuk systemctl): %s", action, serviceName, msg)
	}
	return nil
}

// ParseDevOpsSnapshot parses the multi-section script output into a structured DevOpsSnapshot.
func ParseDevOpsSnapshot(serverID string, raw string) *model.DevOpsSnapshot {
	sections := splitSections(raw)

	dockerAvail, dockerVer, containers := parseDockerSection(
		sections["DOCKER_INFO"],
		sections["DOCKER_PS"],
		sections["DOCKER_STATS"],
	)

	return &model.DevOpsSnapshot{
		ServerID:        serverID,
		Timestamp:       time.Now(),
		DockerAvailable: dockerAvail,
		DockerVersion:   dockerVer,
		Containers:      containers,
		Services:        parseServicesSection(sections["SERVICES"]),
		ListeningPorts:  parsePortsSection(sections["PORTS"]),
		Firewall:        parseFirewallSection(sections["FIREWALL"]),
		Security:        parseSecuritySection(sections["UPDATES"], sections["SSH_AUDIT"]),
	}
}

func splitSections(raw string) map[string]string {
	res := make(map[string]string)
	var current string
	var buf strings.Builder

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "===SECTION:") && strings.HasSuffix(trimmed, "===") {
			if current != "" {
				res[current] = buf.String()
			}
			current = strings.TrimSuffix(strings.TrimPrefix(trimmed, "===SECTION:"), "===")
			buf.Reset()
			continue
		}
		if current != "" {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	if current != "" {
		res[current] = buf.String()
	}
	return res
}

type containerStat struct {
	cpuPct  float64
	memUse  string
	memPct  float64
	netIO   string
	blockIO string
}

func parseDockerSection(infoRaw, psRaw, statsRaw string) (bool, string, []model.DockerContainer) {
	version := SanitizeText(infoRaw)
	available := version != ""

	statsMap := make(map[string]containerStat)
	for _, line := range strings.Split(statsRaw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 6 {
			continue
		}
		id := strings.TrimSpace(parts[0])
		cpuVal := parsePercent(parts[1])
		memPctVal := parsePercent(parts[3])
		st := containerStat{
			cpuPct:  cpuVal,
			memUse:  SanitizeText(parts[2]),
			memPct:  memPctVal,
			netIO:   SanitizeText(parts[4]),
			blockIO: SanitizeText(parts[5]),
		}
		statsMap[id] = st
		if len(id) >= 12 {
			statsMap[id[:12]] = st
		}
	}

	var containers []model.DockerContainer
	for _, line := range strings.Split(psRaw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue
		}
		fullID := SanitizeText(parts[0])
		shortID := fullID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		ports := ""
		if len(parts) >= 6 {
			ports = SanitizeText(parts[5])
		}

		c := model.DockerContainer{
			ID:     shortID,
			Name:   SanitizeText(parts[1]),
			Image:  SanitizeText(parts[2]),
			State:  strings.ToLower(SanitizeText(parts[3])),
			Status: SanitizeText(parts[4]),
			Ports:  ports,
		}
		if st, ok := statsMap[fullID]; ok {
			c.CPUPercent = st.cpuPct
			c.MemUsage = st.memUse
			c.MemPercent = st.memPct
			c.NetIO = st.netIO
			c.BlockIO = st.blockIO
		} else if st, ok := statsMap[shortID]; ok {
			c.CPUPercent = st.cpuPct
			c.MemUsage = st.memUse
			c.MemPercent = st.memPct
			c.NetIO = st.netIO
			c.BlockIO = st.blockIO
		}
		containers = append(containers, c)
	}

	if len(containers) > 0 {
		available = true
	}
	return available, version, containers
}

func parseServicesSection(raw string) []model.SystemdService {
	var services []model.SystemdService
	seen := make(map[string]bool)

	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		unitName := SanitizeText(fields[0])
		if !strings.HasSuffix(unitName, ".service") {
			continue
		}
		shortName := strings.TrimSuffix(unitName, ".service")
		if seen[shortName] {
			continue
		}

		loadState := SanitizeText(fields[1])
		activeState := SanitizeText(fields[2])
		subState := SanitizeText(fields[3])
		desc := ""
		if len(fields) > 4 {
			desc = SanitizeText(strings.Join(fields[4:], " "))
		}

		baseName := shortName
		if idx := strings.Index(baseName, "@"); idx != -1 {
			baseName = baseName[:idx]
		}
		isKey := keyInfrastructureServices[shortName] || keyInfrastructureServices[baseName]

		// Include key infrastructure services, failed services, or running services.
		if !isKey && activeState != "failed" && subState != "running" {
			continue
		}

		seen[shortName] = true
		services = append(services, model.SystemdService{
			Name:         shortName,
			LoadState:    loadState,
			ActiveState:  activeState,
			SubState:     subState,
			Description:  desc,
			IsKeyService: isKey,
		})
	}

	sort.Slice(services, func(i, j int) bool {
		if services[i].ActiveState == "failed" && services[j].ActiveState != "failed" {
			return true
		}
		if services[i].ActiveState != "failed" && services[j].ActiveState == "failed" {
			return false
		}
		if services[i].IsKeyService != services[j].IsKeyService {
			return services[i].IsKeyService
		}
		return services[i].Name < services[j].Name
	})

	if len(services) > 60 {
		services = services[:60]
	}
	return services
}

func parsePortsSection(raw string) []model.ListeningPort {
	var ports []model.ListeningPort
	seen := make(map[string]bool)

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Netid") || strings.HasPrefix(line, "Proto") || strings.HasPrefix(line, "Active") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToLower(fields[0])
		if !strings.HasPrefix(proto, "tcp") && !strings.HasPrefix(proto, "udp") {
			continue
		}
		if strings.HasPrefix(proto, "tcp") {
			proto = "tcp"
		} else {
			proto = "udp"
		}

		// In `ss -tulpnH`, fields[4] is Local Address:Port; in `netstat -tuln`, fields[3] is Local Address:Port
		localEndpoint := ""
		if len(fields) >= 5 && strings.Contains(fields[4], ":") {
			localEndpoint = fields[4]
		} else if strings.Contains(fields[3], ":") {
			localEndpoint = fields[3]
		}
		if localEndpoint == "" {
			continue
		}

		addr, portNum := splitHostPortEndpoint(localEndpoint)
		if portNum <= 0 {
			continue
		}

		procName := "-"
		pid := 0
		if m := ssProcessRegex.FindStringSubmatch(line); len(m) == 3 {
			procName = SanitizeText(m[1])
			pid, _ = strconv.Atoi(m[2])
		} else if m := netstatPIDRegex.FindStringSubmatch(line); len(m) == 3 {
			pid, _ = strconv.Atoi(m[1])
			procName = SanitizeText(m[2])
		}

		exposure := "public"
		if addr == "127.0.0.1" || strings.HasPrefix(addr, "127.0.0.") || addr == "::1" || addr == "localhost" {
			exposure = "local"
		}

		key := fmt.Sprintf("%s:%s:%d", proto, addr, portNum)
		if seen[key] {
			continue
		}
		seen[key] = true

		ports = append(ports, model.ListeningPort{
			Protocol:     proto,
			LocalAddress: addr,
			Port:         portNum,
			Process:      procName,
			PID:          pid,
			Exposure:     exposure,
		})
	}

	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Exposure != ports[j].Exposure {
			return ports[i].Exposure == "public"
		}
		return ports[i].Port < ports[j].Port
	})

	return ports
}

func splitHostPortEndpoint(endpoint string) (string, int) {
	// Strip interface suffix like 127.0.0.53%lo:53
	lastColon := strings.LastIndex(endpoint, ":")
	if lastColon == -1 {
		return "", 0
	}
	hostPart := endpoint[:lastColon]
	portPart := endpoint[lastColon+1:]

	portNum, err := strconv.Atoi(portPart)
	if err != nil || portNum <= 0 || portNum > 65535 {
		return "", 0
	}

	if pctIdx := strings.Index(hostPart, "%"); pctIdx != -1 {
		hostPart = hostPart[:pctIdx]
	}
	hostPart = strings.Trim(hostPart, "[]")
	if hostPart == "*" || hostPart == "" {
		hostPart = "0.0.0.0"
	}
	return hostPart, portNum
}

func parseFirewallSection(raw string) model.FirewallStatus {
	status := model.FirewallStatus{
		Installed: false,
		Active:    false,
		ToolName:  "none",
		Summary:   "Firewall tidak terdeteksi",
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var bodyLines []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "TOOL:") {
			status.ToolName = strings.TrimPrefix(trimmed, "TOOL:")
			continue
		}
		if trimmed != "" {
			bodyLines = append(bodyLines, SanitizeText(trimmed))
		}
	}

	switch status.ToolName {
	case "ufw":
		status.Installed = true
		joined := strings.ToLower(strings.Join(bodyLines, " "))
		if strings.Contains(joined, "status: active") {
			status.Active = true
			status.Summary = "UFW Aktif"
		} else if strings.Contains(joined, "status: inactive") {
			status.Active = false
			status.Summary = "UFW Nonaktif (Inactive)"
		} else {
			status.Summary = "UFW Terpasang (butuh sudo untuk melihat status)"
		}
	case "firewalld":
		status.Installed = true
		joined := strings.ToLower(strings.Join(bodyLines, " "))
		if strings.Contains(joined, "running") {
			status.Active = true
			status.Summary = "Firewalld Aktif (Running)"
		} else {
			status.Active = false
			status.Summary = "Firewalld Nonaktif"
		}
	case "iptables":
		status.Installed = true
		if len(bodyLines) > 0 {
			for _, rule := range bodyLines {
				if strings.HasPrefix(rule, "-A ") || strings.Contains(rule, "DROP") || strings.Contains(rule, "REJECT") {
					status.Active = true
					break
				}
			}
			if status.Active {
				status.Summary = "Aturan iptables aktif terdeteksi"
			} else {
				status.Summary = "iptables default policy (ACCEPT)"
			}
		} else {
			status.Summary = "iptables terpasang"
		}
	}

	return status
}

func parseSecuritySection(updatesRaw, sshAuditRaw string) model.SecurityAudit {
	audit := model.SecurityAudit{
		PackageManager:  "unknown",
		TopAttackers:    []model.SSHAttackerIP{},
		ActiveSessions:  []model.ActiveSSHSession{},
		PermitRootLogin: "default",
		PasswordAuth:    "default",
	}

	for _, line := range strings.Split(updatesRaw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "PKG:") {
			audit.PackageManager = strings.TrimPrefix(trimmed, "PKG:")
			continue
		}
		audit.PendingUpdates++
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "security") {
			audit.SecurityUpdates++
		}
	}

	var subSec string
	ipCounts := make(map[string]int)

	for _, line := range strings.Split(sshAuditRaw, "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "---FAILED_LOGS---":
			subSec = "FAILED"
			continue
		case "---WHO---":
			subSec = "WHO"
			continue
		case "---SSHD_CFG---":
			subSec = "SSHD"
			continue
		case "---FAIL2BAN---":
			subSec = "FAIL2BAN"
			continue
		}
		if trimmed == "" {
			continue
		}

		switch subSec {
		case "FAILED":
			audit.FailedSSHCount++
			if m := sshFromIPRegex.FindStringSubmatch(trimmed); len(m) == 2 {
				ipStr := m[1]
				if net.ParseIP(ipStr) != nil {
					ipCounts[ipStr]++
				}
			}
		case "WHO":
			fields := strings.Fields(trimmed)
			if len(fields) >= 3 {
				fromIP := "local"
				lastField := fields[len(fields)-1]
				if strings.HasPrefix(lastField, "(") && strings.HasSuffix(lastField, ")") {
					fromIP = SanitizeText(strings.Trim(lastField, "()"))
				}
				loginTime := SanitizeText(strings.Join(fields[2:minInt(4, len(fields))], " "))
				audit.ActiveSessions = append(audit.ActiveSessions, model.ActiveSSHSession{
					User:      SanitizeText(fields[0]),
					Terminal:  SanitizeText(fields[1]),
					FromIP:    fromIP,
					LoginTime: loginTime,
				})
			}
		case "SSHD":
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				key := strings.ToLower(fields[0])
				val := strings.ToLower(SanitizeText(fields[1]))
				if key == "permitrootlogin" {
					audit.PermitRootLogin = val
				} else if key == "passwordauthentication" {
					audit.PasswordAuth = val
				}
			}
		case "FAIL2BAN":
			if strings.ToLower(trimmed) == "active" {
				audit.Fail2banActive = true
			}
		}
	}

	for ip, count := range ipCounts {
		audit.TopAttackers = append(audit.TopAttackers, model.SSHAttackerIP{
			IP:       ip,
			Attempts: count,
		})
	}
	sort.Slice(audit.TopAttackers, func(i, j int) bool {
		if audit.TopAttackers[i].Attempts != audit.TopAttackers[j].Attempts {
			return audit.TopAttackers[i].Attempts > audit.TopAttackers[j].Attempts
		}
		return audit.TopAttackers[i].IP < audit.TopAttackers[j].IP
	})
	if len(audit.TopAttackers) > 10 {
		audit.TopAttackers = audit.TopAttackers[:10]
	}

	return audit
}

func parsePercent(s string) float64 {
	cleaned := strings.TrimSuffix(strings.TrimSpace(s), "%")
	val, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || val < 0 {
		return 0
	}
	if val > 1000 {
		return 1000
	}
	return val
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
