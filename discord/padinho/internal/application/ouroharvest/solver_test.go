package ouroharvest

import (
	"errors"
	"math"
	"testing"
)

func TestSolveRejectsInvalidStates(t *testing.T) {
	solver := NewSolver()
	for _, state := range []State{{ClicksLeft: 6}, {ClicksLeft: 5, Covered: 25, Blue: 1}} {
		if _, err := solver.Solve(state); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("Solve(%#v) error = %v", state, err)
		}
	}
}

func TestSolveReturnsNoActionForTerminalOrEmptyStates(t *testing.T) {
	solver := NewSolver()
	for _, state := range []State{{ClicksLeft: 0, White: 1}, {ClicksLeft: 5}} {
		result, err := solver.Solve(state)
		if err != nil || result.Recommendation != nil || len(result.Actions) != 0 {
			t.Fatalf("Solve(%#v) = %#v, %v", state, result, err)
		}
	}
}

func TestSolvePrefersHighestDeterministicReward(t *testing.T) {
	result, err := NewSolver().Solve(State{ClicksLeft: 1, Green: 1, White: 1})
	if err != nil || result.Recommendation == nil {
		t.Fatalf("Solve() = %#v, %v", result, err)
	}
	if result.Recommendation.Action != Action(White) || result.Recommendation.ExpectedSP != 500 || result.Recommendation.AdvantageSP != 465 {
		t.Fatalf("recommendation = %#v", result.Recommendation)
	}
}

func TestSolveModelsDarkPurpleRefund(t *testing.T) {
	result, err := NewSolver().Solve(State{ClicksLeft: 1, Dark: 1, White: 1})
	if err != nil || result.Recommendation == nil {
		t.Fatalf("Solve() = %#v, %v", result, err)
	}
	if result.Recommendation.Action != Action(White) {
		t.Fatalf("recommended action = %d, want white", result.Recommendation.Action)
	}
	dark := findAction(t, result, Action(Dark))
	want := (10 + 20 + 35 + 55 + 90 + 150 + 5 + 500 + 76.03 + 500) / 9
	if math.Abs(dark.ExpectedSP-want) > 1e-9 {
		t.Fatalf("dark EV = %.12f, want %.12f", dark.ExpectedSP, want)
	}
}

func TestSolveRetainsCoveredChestProbability(t *testing.T) {
	solver := NewSolver()
	result, err := solver.Solve(State{ClicksLeft: 1, Covered: 1})
	if err != nil || result.Recommendation == nil {
		t.Fatalf("Solve() = %#v, %v", result, err)
	}
	want := 1 - appearanceMass()
	if math.Abs(result.Recommendation.ChestProbability-want) > 1e-12 {
		t.Fatalf("chest probability = %.12f, want %.12f", result.Recommendation.ChestProbability, want)
	}
	withoutChest, err := solver.Solve(State{ClicksLeft: 1, Covered: 1, ChestFound: true})
	if err != nil || withoutChest.Recommendation.ChestProbability != 0 {
		t.Fatalf("chest-found result = %#v, %v", withoutChest, err)
	}
}

func TestRevealBranchesAreNormalizedAndConserveCounts(t *testing.T) {
	for reveals, branches := range revealBranches {
		total := 0.0
		for _, branch := range branches {
			total += branch.probability
			count := int(branch.delta.Blue + branch.delta.Teal + branch.delta.Dark + branch.delta.Green + branch.delta.Yellow + branch.delta.Light + branch.delta.Orange + branch.delta.Red + branch.delta.White + branch.delta.Purple)
			if count != reveals {
				t.Fatalf("reveal %d branch count = %d", reveals, count)
			}
		}
		if math.Abs(total-1) > 1e-12 {
			t.Fatalf("reveal %d probability = %.12f", reveals, total)
		}
	}
}

func TestSolveIsStableAcrossCachedCalls(t *testing.T) {
	solver := NewSolver()
	state := State{ClicksLeft: 5, Covered: 10, Blue: 3, Teal: 2, Dark: 1, Green: 2, Yellow: 1}
	first, err := solver.Solve(state)
	if err != nil {
		t.Fatal(err)
	}
	second, err := solver.Solve(state)
	if err != nil || len(first.Actions) != len(second.Actions) {
		t.Fatalf("second solve = %#v, %v", second, err)
	}
	for index := range first.Actions {
		if first.Actions[index].Action != second.Actions[index].Action || math.Abs(first.Actions[index].ExpectedSP-second.Actions[index].ExpectedSP) > 1e-9 || math.Abs(first.Actions[index].ChestProbability-second.Actions[index].ChestProbability) > 1e-9 {
			t.Fatalf("action %d changed: %#v != %#v", index, first.Actions[index], second.Actions[index])
		}
	}
}

func findAction(t *testing.T, result Result, action Action) Recommendation {
	t.Helper()
	for _, recommendation := range result.Actions {
		if recommendation.Action == action {
			return recommendation
		}
	}
	t.Fatalf("action %d not found", action)
	return Recommendation{}
}
