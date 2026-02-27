package skill

import (
	"testing"
)

func TestCheckCommandSafety_Blocked(t *testing.T) {
	tests := []struct {
		cmd  string
		want DangerLevel
	}{
		{"rm -rf /", DangerBlocked},
		{"mkfs.ext4 /dev/sda", DangerBlocked},
		{"dd if=/dev/zero of=/dev/sda", DangerBlocked},
	}
	for _, tt := range tests {
		level, reasons := CheckCommandSafety(tt.cmd)
		if level != tt.want {
			t.Errorf("CheckCommandSafety(%q) = %d, want %d (reasons: %v)", tt.cmd, level, tt.want, reasons)
		}
	}
}

func TestCheckCommandSafety_Warning(t *testing.T) {
	tests := []struct {
		cmd string
	}{
		{"rm file.txt"},
		{"sudo apt update"},
		{"chmod 777 somefile"},
		{"git push --force"},
		{"git reset --hard"},
		{"kill -9 1234"},
		{"mv a b"},
		{"drop table users"},
		{"delete from users"},
		{"truncate table logs"},
	}
	for _, tt := range tests {
		level, _ := CheckCommandSafety(tt.cmd)
		if level != DangerWarning {
			t.Errorf("CheckCommandSafety(%q) = %d, want DangerWarning(%d)", tt.cmd, level, DangerWarning)
		}
	}
}

func TestCheckCommandSafety_Safe(t *testing.T) {
	tests := []string{
		"ls -la",
		"cat file.txt",
		"go build ./...",
		"echo hello",
		"pwd",
		"git status",
		"git log --oneline -10",
	}
	for _, cmd := range tests {
		level, _ := CheckCommandSafety(cmd)
		if level != DangerNone {
			t.Errorf("CheckCommandSafety(%q) = %d, want DangerNone", cmd, level)
		}
	}
}
