package config

import "testing"

func TestParseFixedSize(t *testing.T) {
	c := defaults()
	c.Size = " 80X24 "
	if err := parseSize(&c); err != nil {
		t.Fatal(err)
	}
	if c.Width != 80 || c.Height != 24 || c.Size != "80x24" {
		t.Fatalf("got size=%q width=%d height=%d", c.Size, c.Width, c.Height)
	}
}

func TestParseAutoSize(t *testing.T) {
	c := defaults()
	c.Size = "auto"
	c.Width, c.Height = 80, 24
	if err := parseSize(&c); err != nil {
		t.Fatal(err)
	}
	if c.Width != 0 || c.Height != 0 {
		t.Fatalf("auto size should use terminal dimensions, got %dx%d", c.Width, c.Height)
	}
}

func TestRejectInvalidFixedSize(t *testing.T) {
	for _, size := range []string{"80", "axb", "37x13", "80x12"} {
		c := defaults()
		c.Size = size
		if err := parseSize(&c); err == nil {
			t.Errorf("expected %q to be rejected", size)
		}
	}
}

func TestRejectInvalidEnvironmentDuration(t *testing.T) {
	t.Setenv("BESZEL_REFRESH", "not-a-duration")
	c := defaults()
	if err := applyEnv(&c); err == nil {
		t.Fatal("expected invalid BESZEL_REFRESH to be rejected")
	}
}

func TestSingleModeEnvironment(t *testing.T) {
	t.Setenv("BESZEL_SINGLE", "true")
	c := defaults()
	if err := applyEnv(&c); err != nil {
		t.Fatal(err)
	}
	if !c.SingleMode {
		t.Fatal("BESZEL_SINGLE=true did not enable single mode")
	}
}

func TestLargeMinimumFixedSize(t *testing.T) {
	_, err := Load([]string{"--hub-url", "https://example.invalid", "--token", "test", "--layout", "large", "--size", "80x22"})
	if err != nil {
		t.Fatalf("80x22 large layout should be valid: %v", err)
	}
	for _, size := range []string{"79x22", "80x21"} {
		if _, err := Load([]string{"--hub-url", "https://example.invalid", "--token", "test", "--layout", "large", "--size", size}); err == nil {
			t.Errorf("large layout size %s should be rejected", size)
		}
	}
}
