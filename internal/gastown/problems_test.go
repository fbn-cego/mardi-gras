package gastown

import "testing"

func TestDetectProblemsNil(t *testing.T) {
	problems := DetectProblems(nil)
	if len(problems) != 0 {
		t.Errorf("expected 0 problems from nil status, got %d", len(problems))
	}
}

func TestDetectProblemsStalled(t *testing.T) {
	status := &TownStatus{
		Agents: []AgentRuntime{
			{Name: "Toast", Role: "polecat", HasWork: true, State: "idle"},
		},
	}
	problems := DetectProblems(status)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}
	if problems[0].Type != "stalled" {
		t.Errorf("expected type 'stalled', got %q", problems[0].Type)
	}
	if problems[0].Severity != "warn" {
		t.Errorf("expected severity 'warn', got %q", problems[0].Severity)
	}
	if problems[0].Agent.Name != "Toast" {
		t.Errorf("expected agent 'Toast', got %q", problems[0].Agent.Name)
	}
}

func TestDetectProblemsBackoff(t *testing.T) {
	status := &TownStatus{
		Agents: []AgentRuntime{
			{Name: "Muffin", Role: "polecat", State: "backoff"},
		},
	}
	problems := DetectProblems(status)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}
	if problems[0].Type != "backoff" {
		t.Errorf("expected type 'backoff', got %q", problems[0].Type)
	}
}

func TestDetectProblemsZombie(t *testing.T) {
	// Zombie on a rig WITH polecats — should still emit zombie.
	status := &TownStatus{
		Rigs: []RigStatus{
			{Name: "myrig", PolecatCount: 1},
		},
		Agents: []AgentRuntime{
			{Name: "Stale", Role: "polecat", Rig: "myrig", Running: false, HookBead: "bd-e5f6"},
		},
	}
	problems := DetectProblems(status)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}
	if problems[0].Type != "zombie" {
		t.Errorf("expected type 'zombie', got %q", problems[0].Type)
	}
	if problems[0].Severity != "error" {
		t.Errorf("expected severity 'error', got %q", problems[0].Severity)
	}
}

func TestDetectProblemsStuck(t *testing.T) {
	status := &TownStatus{
		Agents: []AgentRuntime{
			{Name: "Granite", Role: "polecat", State: "stuck"},
		},
	}
	problems := DetectProblems(status)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(problems))
	}
	if problems[0].Type != "stuck" {
		t.Errorf("expected type 'stuck', got %q", problems[0].Type)
	}
	if problems[0].Severity != "error" {
		t.Errorf("expected severity 'error', got %q", problems[0].Severity)
	}
	if problems[0].Agent.Name != "Granite" {
		t.Errorf("expected agent 'Granite', got %q", problems[0].Agent.Name)
	}
}

func TestDetectProblemsHealthy(t *testing.T) {
	status := &TownStatus{
		Agents: []AgentRuntime{
			{Name: "Toast", Role: "polecat", Running: true, HasWork: true, State: "working"},
			{Name: "Muffin", Role: "polecat", Running: true, State: "idle"},
			{Name: "Witness", Role: "witness", Running: true, State: "working"},
		},
	}
	problems := DetectProblems(status)
	if len(problems) != 0 {
		t.Errorf("expected 0 problems for healthy agents, got %d", len(problems))
	}
}

func TestDetectProblemsDeadRig(t *testing.T) {
	status := &TownStatus{
		Rigs: []RigStatus{
			{Name: "mardi_gras", PolecatCount: 0},
		},
		Agents: []AgentRuntime{
			{Name: "obsidian", Role: "polecat", Rig: "mardi_gras",
				Running: false, HookBead: "mg-001", WorkTitle: "Fix auth"},
			{Name: "quartz", Role: "polecat", Rig: "mardi_gras",
				Running: false, HookBead: "mg-002", WorkTitle: "Add tests"},
		},
	}
	problems := DetectProblems(status)

	// Should emit one dead_rig, NOT two zombies.
	var deadRig, zombie int
	for _, p := range problems {
		switch p.Type {
		case "dead_rig":
			deadRig++
		case "zombie":
			zombie++
		}
	}
	if deadRig != 1 {
		t.Errorf("expected 1 dead_rig problem, got %d", deadRig)
	}
	if zombie != 0 {
		t.Errorf("expected 0 zombie problems (suppressed by dead_rig), got %d", zombie)
	}

	// Verify orphans are attached to the dead_rig problem.
	for _, p := range problems {
		if p.Type == "dead_rig" {
			if len(p.Orphans) != 2 {
				t.Errorf("expected 2 orphans on dead_rig, got %d", len(p.Orphans))
			}
			if p.RigName != "mardi_gras" {
				t.Errorf("expected rig name mardi_gras, got %q", p.RigName)
			}
		}
	}
}

func TestDetectProblemsDeadRigDoesNotSuppressOtherRigZombie(t *testing.T) {
	status := &TownStatus{
		Rigs: []RigStatus{
			{Name: "dead_rig", PolecatCount: 0},
			{Name: "live_rig", PolecatCount: 2},
		},
		Agents: []AgentRuntime{
			{Name: "obsidian", Role: "polecat", Rig: "dead_rig",
				Running: false, HookBead: "dr-001"},
			{Name: "quartz", Role: "polecat", Rig: "live_rig",
				Running: false, HookBead: "lr-001"},
		},
	}
	problems := DetectProblems(status)

	types := map[string]int{}
	for _, p := range problems {
		types[p.Type]++
	}
	if types["dead_rig"] != 1 {
		t.Errorf("expected 1 dead_rig, got %d", types["dead_rig"])
	}
	if types["zombie"] != 1 {
		t.Errorf("expected 1 zombie (from live_rig), got %d", types["zombie"])
	}
}

func TestDetectProblemsMultiple(t *testing.T) {
	status := &TownStatus{
		Agents: []AgentRuntime{
			{Name: "Toast", Role: "polecat", HasWork: true, State: "idle"},        // stalled
			{Name: "Muffin", Role: "polecat", State: "backoff"},                   // backoff
			{Name: "Stale", Role: "polecat", Running: false, HookBead: "bd-e5f6"}, // zombie
		},
	}
	problems := DetectProblems(status)
	if len(problems) != 3 {
		t.Fatalf("expected 3 problems, got %d", len(problems))
	}

	types := map[string]bool{}
	for _, p := range problems {
		types[p.Type] = true
	}
	for _, expected := range []string{"stalled", "backoff", "zombie"} {
		if !types[expected] {
			t.Errorf("expected problem type %q not found", expected)
		}
	}
}
