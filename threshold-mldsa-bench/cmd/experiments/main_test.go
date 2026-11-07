package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNearestRankStatistics(t *testing.T) {
	v := make([]float64, 100)
	for i := range v {
		v[i] = float64(100 - i)
	}
	s := statistics(v)
	if s.Mean != 50.5 || s.P50 != 50 || s.P95 != 95 || s.Min != 1 || s.Max != 100 || v[0] != 100 {
		t.Fatalf("unexpected statistics %+v or input mutated", s)
	}
}
func TestWritePublicEvidence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "results")
	r := report{Requested: 10, Succeeded: 10, Checks: map[string]bool{"test": true}}
	if err := writeReport(dir, r, map[string]int{"[0 2 4]": 10}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"REPORT.md", "results.json", "samples.csv"} {
		data, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil || len(data) == 0 {
			t.Fatalf("missing %s: %v", name, e)
		}
		if strings.Contains(string(data), `"coordinator_token"`) {
			t.Fatal("credential field in public evidence")
		}
	}
}
