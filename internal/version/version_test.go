package version

import "testing"

func TestCommitShort(t *testing.T) {
	tests := []struct {
		name   string
		commit string
		want   string
	}{
		{"default dev", "dev", "dev"},
		{"short hash", "abc123", "abc123"},
		{"exactly seven", "abcdefg", "abcdefg"},
		{"long hash", "abcdef1234567890", "abcdef1"},
	}

	orig := Commit
	t.Cleanup(func() { Commit = orig })

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			Commit = tc.commit
			if got := CommitShort(); got != tc.want {
				t.Errorf("CommitShort() = %q, want %q", got, tc.want)
			}
		})
	}
}
