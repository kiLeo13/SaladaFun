package ouroquest

import (
	"math"
	"testing"
)

func TestSolveInitialBoardUsesExactPosterior(t *testing.T) {
	result, err := Solve(Board{}, [CellCount]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateCount != 12650 {
		t.Fatalf("candidate count = %d", result.CandidateCount)
	}
	probabilitySum := 0.0
	for _, probability := range result.PurpleProbabilities {
		probabilitySum += probability
	}
	if math.Abs(probabilitySum-TargetCount) > comparisonEpsilon {
		t.Fatalf("probability sum = %f", probabilitySum)
	}
	if result.Recommendation == nil || result.Recommendation.Position != 6 {
		t.Fatalf("recommendation = %#v", result.Recommendation)
	}
	if math.Abs(result.Recommendation.PurpleProbability-0.16) > comparisonEpsilon {
		t.Fatalf("purple probability = %f", result.Recommendation.PurpleProbability)
	}
}

func TestSolveFiltersNumberAndTargetObservations(t *testing.T) {
	board := Board{}
	board[6] = Blue
	result, err := Solve(board, [CellCount]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateCount != 1820 {
		t.Fatalf("candidate count = %d", result.CandidateCount)
	}
	for _, position := range []int{0, 1, 2, 5, 6, 7, 10, 11, 12} {
		if result.PurpleProbabilities[position] != 0 {
			t.Fatalf("position %d probability = %f", position, result.PurpleProbabilities[position])
		}
	}

	board = Board{}
	board[0] = Purple
	result, err = Solve(board, [CellCount]bool{})
	if err != nil || result.CandidateCount != 2024 || result.PurpleProbabilities[0] != 1 {
		t.Fatalf("purple observation = %#v, %v", result, err)
	}
}

func TestSolveRejectsImpossibleAndFinishedBoards(t *testing.T) {
	board := Board{}
	board[0], board[1] = Blue, Purple
	if _, err := Solve(board, [CellCount]bool{}); !errorsIs(err, ErrInconsistentBoard) {
		t.Fatalf("inconsistent error = %v", err)
	}

	board = Board{}
	for position := 0; position < PaidClickLimit; position++ {
		board[position] = Blue
	}
	result, err := Solve(board, [CellCount]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendation != nil {
		t.Fatalf("finished recommendation = %#v", result.Recommendation)
	}

	board = Board{}
	board[0] = Red
	if _, err := Solve(board, [CellCount]bool{}); !errorsIs(err, ErrInconsistentBoard) {
		t.Fatalf("early red error = %v", err)
	}
}

func TestColorValuesAndWorldGeometry(t *testing.T) {
	want := map[Color]float64{Unknown: 0, Blue: 10, Teal: 20, Green: 35, Yellow: 55, Orange: 90, Purple: 165, Red: 150}
	for color, value := range want {
		if got := color.Value(); got != value {
			t.Fatalf("color %d value = %f", color, got)
		}
	}
	if len(legalWorlds) != 12650 {
		t.Fatalf("world count = %d", len(legalWorlds))
	}
	candidate := world{mask: 1<<0 | 1<<1 | 1<<5 | 1<<24}
	if got := numberedColor(candidate, 6); got != Yellow {
		t.Fatalf("numbered color = %d", got)
	}
	if got := outcome(candidate, 0, 0); got != Purple {
		t.Fatalf("first target = %d", got)
	}
	if got := outcome(candidate, 0, 3); got != Red {
		t.Fatalf("fourth target = %d", got)
	}
}

func TestSolveNeverRecommendsUnavailableCell(t *testing.T) {
	unavailable := [CellCount]bool{}
	for position := 0; position < CellCount; position++ {
		unavailable[position] = true
	}
	unavailable[24] = false
	result, err := Solve(Board{}, unavailable)
	if err != nil {
		t.Fatal(err)
	}
	if result.Recommendation == nil || result.Recommendation.Position != 24 {
		t.Fatalf("recommendation = %#v", result.Recommendation)
	}
}

// errorsIs keeps the assertions readable without weakening exact sentinel checks.
func errorsIs(err, target error) bool {
	return err == target
}
