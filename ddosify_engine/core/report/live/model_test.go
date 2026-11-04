package live

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func runCmd(c tea.Cmd) tea.Msg {
	if c == nil {
		return nil
	}
	return c()
}

func isQuit(c tea.Cmd) bool {
	_, ok := runCmd(c).(tea.QuitMsg)
	return ok
}

func TestStopKeysCallStopOnce(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{
		{Code: 'q', Text: "q"},
		{Code: 'c', Mod: tea.ModCtrl},
	} {
		t.Run(k.String(), func(t *testing.T) {
			calls := 0
			m := NewModel(make(chan Snapshot), ModelOptions{Stop: func() { calls++ }})
			_, cmd := m.Update(k)
			if isQuit(cmd) {
				t.Fatal("a stop key quit the view before the engine finished")
			}
			if !m.Stopping() || calls != 1 {
				t.Fatalf("stopping %v, stop called %d times", m.Stopping(), calls)
			}
			_, cmd = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
			runCmd(cmd)
			if calls != 1 {
				t.Fatalf("stop called %d times", calls)
			}
		})
	}
}

func TestSecondCtrlCLeaves(t *testing.T) {
	m := NewModel(make(chan Snapshot), ModelOptions{})
	m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !isQuit(cmd) || !m.Forced() {
		t.Fatal("ctrl+c while stopping did not leave")
	}
}

func TestSignalsBecomeStop(t *testing.T) {
	m := NewModel(make(chan Snapshot), ModelOptions{})
	if _, ok := filter(m, tea.QuitMsg{}).(stopRequestMsg); !ok {
		t.Fatal("SIGTERM's quit was not turned into a stop")
	}
	if _, ok := filter(m, tea.InterruptMsg{}).(stopRequestMsg); !ok {
		t.Fatal("SIGINT was not turned into a stop")
	}
	m.Update(closedMsg{})
	if _, ok := filter(m, tea.QuitMsg{}).(tea.QuitMsg); !ok {
		t.Fatal("the quit after the channel closed was held back")
	}
}

func TestChannelCloseQuits(t *testing.T) {
	ch := make(chan Snapshot, 1)
	m := NewModel(ch, ModelOptions{})
	ch <- synthetic(0)
	msg := runCmd(m.Init())
	_, cmd := m.Update(msg)
	if s, ok := m.Last(); !ok || s.Iterations != synthetic(0).Iterations {
		t.Fatal("snapshot not kept")
	}
	close(ch)
	_, cmd = m.Update(runCmd(cmd))
	if !isQuit(cmd) || !m.finished {
		t.Fatal("closing the channel did not end the view")
	}
}

func TestHistoryIsBounded(t *testing.T) {
	m := NewModel(make(chan Snapshot), ModelOptions{})
	feed(m, historyLen*5)
	if m.rps.len() != historyLen || m.ttft.len() != historyLen {
		t.Fatalf("history holds %d and %d, cap %d", m.rps.len(), m.ttft.len(), historyLen)
	}
	got := m.rps.last(3)
	want := []float64{synthetic(historyLen*5 - 3).AchievedRPS, synthetic(historyLen*5 - 2).AchievedRPS, synthetic(historyLen*5 - 1).AchievedRPS}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("last values %v, want %v", got, want)
		}
	}
}

func TestResizeChangesLayout(t *testing.T) {
	m := NewModel(make(chan Snapshot), ModelOptions{NoColor: true})
	feed(m, 5)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	wide := m.Render()
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 30})
	narrow := m.Render()
	if wide == narrow {
		t.Fatal("resizing did not change the view")
	}
	checkFits(t, narrow, 40, 30)
}

func TestNoColorFromEnv(t *testing.T) {
	cases := []struct {
		env  []string
		want bool
	}{
		{[]string{"TERM=xterm-256color"}, false},
		{[]string{"NO_COLOR=1"}, true},
		{[]string{"NO_COLOR="}, false},
		{[]string{"TERM=dumb"}, true},
	}
	for _, c := range cases {
		if got := NoColorFromEnv(c.env); got != c.want {
			t.Errorf("%v: %v, want %v", c.env, got, c.want)
		}
	}
}

func TestStoppedStageIsNotDone(t *testing.T) {
	m := NewModel(make(chan Snapshot), ModelOptions{NoColor: true})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	feed(m, 1)
	m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	m.Update(closedMsg{})
	v := m.Render()
	if !strings.Contains(v, "stopped at 52%") || !strings.Contains(v, "test finished") {
		t.Fatalf("stopped stage not shown as stopped:\n%s", v)
	}
}
