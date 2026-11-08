package htmlreport

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// scripts/resource-samples.sh turns docker stats, kubectl top and nvidia-smi output into samples
// that ParseSamples reads whole, with the units those tools print.
func TestResourceSamplesScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a POSIX shell script")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	script, err := filepath.Abs("../../../scripts/resource-samples.sh")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	stub := func(name, out string) {
		body := "#!/bin/sh\ncat <<'OUT'\n" + out + "OUT\n"
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub("docker", "masking-cpu,391.20%,2.07GiB / 15.5GiB\nicap,0.50%,120.5MiB / 15.5GiB\nstopped,--,0B / 0B\n")
	stub("kubectl", "masking-cpu-0   3912m   2120Mi\ngateway-7c9f   250m   300Mi\n")
	stub("nvidia-smi", "0, 87, 30500\n1, 3, 512\n")
	for _, c := range []struct {
		mode string
		args []string
		want map[string]float64 // source metric -> value
		bad  int
	}{
		{"docker", nil, map[string]float64{"masking-cpu cpu_cores": 3.912, "masking-cpu mem_bytes": 2.07 * (1 << 30), "icap cpu_cores": 0.005, "icap mem_bytes": 120.5 * (1 << 20), "stopped mem_bytes": 0}, 1},
		{"kubectl", []string{"15", "-n", "my-app"}, map[string]float64{"masking-cpu-0 cpu_cores": 3.912, "masking-cpu-0 mem_bytes": 2120 << 20, "gateway-7c9f cpu_cores": 0.25, "gateway-7c9f mem_bytes": 300 << 20}, 0},
		{"nvidia", nil, map[string]float64{"gpu0 gpu_util_pct": 87, "gpu0 gpu_mem_bytes": 30500 << 20, "gpu1 gpu_util_pct": 3, "gpu1 gpu_mem_bytes": 512 << 20}, 0},
	} {
		cmd := exec.Command("sh", append([]string{script, c.mode}, c.args...)...)
		count := "COUNT=1"
		if c.mode == "nvidia" {
			count = "COUNT=2" // a line per GPU
		}
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), count)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s: %v", c.mode, err)
		}
		got := map[string]float64{}
		st, err := ParseSamples(strings.NewReader(string(out)), func(s Sample) { got[s.Source+" "+s.Metric] = s.Value })
		if err != nil || st.Bad != c.bad {
			t.Errorf("%s: %v, %d bad lines in\n%s", c.mode, err, st.Bad, out)
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v", c.mode, got)
		}
		for k, v := range c.want {
			if g, ok := got[k]; !ok || !near(g, v) {
				t.Errorf("%s: %s = %v, want %v", c.mode, k, g, v)
			}
		}
	}
	if err := exec.Command("sh", script, "nope").Run(); err == nil {
		t.Error("an unknown mode did not fail")
	}
}
