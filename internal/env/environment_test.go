package env

import "testing"

func TestLaunchdJobReloadsUserEnvironment(t *testing.T) {
	job := LaunchdJob("/usr/local/bin/macctl")

	want := []string{"/usr/local/bin/macctl", "--user", "daemon-reload"}
	if len(job.ProgramArguments) != len(want) {
		t.Fatalf("ProgramArguments length = %d, want %d: %#v", len(job.ProgramArguments), len(want), job.ProgramArguments)
	}
	for i := range want {
		if job.ProgramArguments[i] != want[i] {
			t.Fatalf("ProgramArguments[%d] = %q, want %q: %#v", i, job.ProgramArguments[i], want[i], job.ProgramArguments)
		}
	}
}
