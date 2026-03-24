package browserdsl

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	domainworkspace "github.com/garyellow/axle/internal/domain/workspace"
)

const (
	DefaultExtractTarget = "body"
	MaxWait              = 30 * time.Second
)

type Step struct {
	Action string `json:"action"`
	Arg    string `json:"arg,omitempty"`
}

type Plan struct {
	Steps []Step `json:"steps"`
}

func ParseScript(input string) (Plan, error) {
	lines := strings.Split(input, "\n")
	plan := Plan{Steps: make([]Step, 0, len(lines))}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		action := strings.ToLower(parts[0])
		arg := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))

		switch action {
		case "open":
			if arg == "" {
				return Plan{}, fmt.Errorf("browser script: open 缺少 URL")
			}
			parsed, err := url.Parse(strings.TrimSpace(arg))
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return Plan{}, fmt.Errorf("browser script: 無效 URL %q", arg)
			}
			if err := ValidateTarget(parsed); err != nil {
				return Plan{}, err
			}
		case "wait":
			if arg == "" {
				return Plan{}, fmt.Errorf("browser script: wait 缺少 duration")
			}
			d, err := time.ParseDuration(strings.TrimSpace(arg))
			if err != nil {
				return Plan{}, fmt.Errorf("browser script: wait duration 無效: %w", err)
			}
			if err := ValidateWaitDuration(d); err != nil {
				return Plan{}, err
			}
		case "extract":
			if strings.TrimSpace(arg) == "" {
				arg = DefaultExtractTarget
			}
		case "screenshot":
			if err := ValidateScreenshotPathArg(arg); err != nil {
				return Plan{}, err
			}
		default:
			return Plan{}, fmt.Errorf("browser script: 不支援的指令 %q", action)
		}

		plan.Steps = append(plan.Steps, Step{Action: action, Arg: strings.TrimSpace(arg)})
	}

	if err := ValidatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func ValidatePlan(plan Plan) error {
	if len(plan.Steps) == 0 {
		return fmt.Errorf("browser script 不能為空")
	}
	if plan.Steps[0].Action != "open" {
		return fmt.Errorf("browser script 第一個步驟必須是 open")
	}
	return nil
}

func ValidateWaitDuration(d time.Duration) error {
	if d <= 0 || d > MaxWait {
		return fmt.Errorf("browser script: wait duration 超過上限 %s", MaxWait)
	}
	return nil
}

func ValidateTarget(parsed *url.URL) error {
	ip := net.ParseIP(strings.TrimSpace(parsed.Hostname()))
	if ip == nil {
		return fmt.Errorf("browser script: 僅允許 literal public IP URL")
	}
	if isNonPublicIP(ip) {
		return fmt.Errorf("browser script: 禁止存取非 public IP 目標")
	}
	return nil
}

func ValidateScreenshotPathArg(relPath string) error {
	if strings.TrimSpace(relPath) == "" {
		return nil
	}
	cleaned, err := domainworkspace.ValidateRelativePath(relPath)
	if err != nil {
		return err
	}
	if cleaned == "." || cleaned == "" {
		return fmt.Errorf("browser screenshot 路徑不能為空")
	}
	return nil
}

func NormalizeScreenshotPath(artifactRel, relPath string) (string, error) {
	cleaned, err := domainworkspace.ValidateRelativePath(relPath)
	if err != nil {
		return "", err
	}
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("browser screenshot 路徑不能為空")
	}
	browserRoot := filepath.Join(".axle", "browser")
	if cleaned == browserRoot {
		return "", fmt.Errorf("browser screenshot 路徑不能直接指向 `%s`", browserRoot)
	}
	if strings.HasPrefix(cleaned, browserRoot+string(os.PathSeparator)) {
		cleaned = strings.TrimPrefix(cleaned, browserRoot+string(os.PathSeparator))
	}
	artifactBase := filepath.Base(artifactRel)
	if cleaned == artifactBase {
		return "", fmt.Errorf("browser screenshot 路徑不能直接指向本次 run 目錄")
	}
	if strings.HasPrefix(cleaned, artifactBase+string(os.PathSeparator)) {
		cleaned = strings.TrimPrefix(cleaned, artifactBase+string(os.PathSeparator))
	}
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("browser screenshot 路徑不能為空")
	}
	return filepath.Join(artifactRel, cleaned), nil
}

func isNonPublicIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	switch {
	case v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127:
		return true
	case v4[0] == 169 && v4[1] == 254:
		return true
	case v4[0] == 198 && (v4[1] == 18 || v4[1] == 19):
		return true
	case v4[0] == 192 && v4[1] == 0 && v4[2] == 2:
		return true
	case v4[0] == 198 && v4[1] == 51 && v4[2] == 100:
		return true
	case v4[0] == 203 && v4[1] == 0 && v4[2] == 113:
		return true
	case v4[0] >= 240:
		return true
	case v4[0] == 255:
		return true
	default:
		return false
	}
}
