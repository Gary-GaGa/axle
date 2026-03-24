package execution

import "testing"

func TestClassifyCommandBlocked(t *testing.T) {
	tests := []string{"rm -rf /", "mkfs.ext4 /dev/sda", "dd if=/dev/zero of=/dev/sda"}
	for _, cmd := range tests {
		report := ClassifyCommand(cmd)
		if report.Level != DangerBlocked {
			t.Fatalf("ClassifyCommand(%q) = %d, want %d", cmd, report.Level, DangerBlocked)
		}
	}
}

func TestClassifyCommandWarning(t *testing.T) {
	tests := []string{"rm file.txt", "sudo apt update", "chmod 777 somefile", "git push --force"}
	for _, cmd := range tests {
		report := ClassifyCommand(cmd)
		if report.Level != DangerWarning {
			t.Fatalf("ClassifyCommand(%q) = %d, want %d", cmd, report.Level, DangerWarning)
		}
	}
}

func TestClassifyCommandSafe(t *testing.T) {
	tests := []string{"ls -la", "cat file.txt", "go build ./...", "echo hello"}
	for _, cmd := range tests {
		report := ClassifyCommand(cmd)
		if report.Level != DangerNone {
			t.Fatalf("ClassifyCommand(%q) = %d, want %d", cmd, report.Level, DangerNone)
		}
	}
}
