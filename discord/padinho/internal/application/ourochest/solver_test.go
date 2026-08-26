package ourochest

import (
	"errors"
	"math"
	"testing"
)

func TestSolveInitialBoard(t *testing.T) {
	result, err := Solve(Board{}, [CellCount]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateCount != 24 {
		t.Fatalf("CandidateCount = %d", result.CandidateCount)
	}
	if result.RedProbabilities[centerCell] != 0 {
		t.Fatalf("center red probability = %f", result.RedProbabilities[centerCell])
	}
	assertProbabilitySum(t, result.RedProbabilities)
	if len(result.Recommendations) == 0 || result.Recommendations[0].Kind != Balanced {
		t.Fatalf("recommendations = %#v", result.Recommendations)
	}
	if result.Recommendations[0].Position != 16 {
		t.Fatalf("initial recommendation position = %d", result.Recommendations[0].Position)
	}
}

func TestSolveMatchesReferenceProbabilityAndExpectedValueCases(t *testing.T) {
	tests := []struct {
		name          string
		board         Board
		wantPositions []int
		wantValue     float64
	}{
		{"empty", Board{}, []int{16}, 35.52083333333292},
		{"corner orange", func() Board { var board Board; board[0] = Orange; return board }(), []int{1, 5}, 98.12499999999977},
		{"center blue", func() Board { var board Board; board[centerCell] = Blue; return board }(), []int{1, 3, 5, 9, 15, 19, 21, 23}, 41.562499999999915},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Solve(test.board, [CellCount]bool{})
			if err != nil {
				t.Fatal(err)
			}
			var reward Recommendation
			for _, recommendation := range result.Recommendations {
				if recommendation.ImmediateValue > reward.ImmediateValue {
					reward = recommendation
				}
			}
			positionMatches := false
			for _, position := range test.wantPositions {
				positionMatches = positionMatches || reward.Position == position
			}
			if !positionMatches || math.Abs(reward.ImmediateValue-test.wantValue) > 1e-9 {
				t.Fatalf("reward recommendation = %#v; all = %#v", reward, result.Recommendations)
			}
		})
	}
}

func TestSolveAppliesEveryClueRelation(t *testing.T) {
	tests := map[string]struct {
		color Color
		want  int
	}{
		"blue": {Blue, 8}, "teal": {Teal, 16}, "green": {Green, 8},
		"yellow": {Yellow, 8}, "orange": {Orange, 4},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			board := Board{}
			board[centerCell] = test.color
			result, err := Solve(board, [CellCount]bool{})
			if err != nil {
				t.Fatal(err)
			}
			if result.CandidateCount != test.want {
				t.Fatalf("CandidateCount = %d, want %d", result.CandidateCount, test.want)
			}
		})
	}
}

func TestSolveCornerOrangeLeavesTwoCandidates(t *testing.T) {
	board := Board{}
	board[0] = Orange
	result, err := Solve(board, [CellCount]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateCount != 2 || result.RedProbabilities[1] != 0.5 || result.RedProbabilities[5] != 0.5 {
		t.Fatalf("result = %#v", result)
	}
}

func TestSolveRejectsImpossibleObservations(t *testing.T) {
	tests := []Board{
		func() Board { var board Board; board[centerCell] = Red; return board }(),
		func() Board { var board Board; board[0], board[1] = Red, Red; return board }(),
		func() Board { var board Board; board[0], board[1], board[5] = Orange, Orange, Orange; return board }(),
	}
	for index, board := range tests {
		if _, err := Solve(board, [CellCount]bool{}); !errors.Is(err, ErrInconsistentBoard) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func TestSolveNeverRecommendsObservedOrUnavailableCells(t *testing.T) {
	board := Board{}
	board[centerCell] = Blue
	unavailable := [CellCount]bool{}
	unavailable[3] = true
	result, err := Solve(board, unavailable)
	if err != nil {
		t.Fatal(err)
	}
	for _, recommendation := range result.Recommendations {
		if recommendation.Position == centerCell || recommendation.Position == 3 {
			t.Fatalf("recommendation = %#v", recommendation)
		}
	}
}

func TestSolveCollapsesDuplicateObjectives(t *testing.T) {
	board := Board{}
	board[0] = Orange
	unavailable := [CellCount]bool{}
	unavailable[0] = true
	result, err := Solve(board, unavailable)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[int]struct{})
	for _, recommendation := range result.Recommendations {
		if _, duplicate := seen[recommendation.Position]; duplicate {
			t.Fatalf("duplicate recommendation = %#v", recommendation)
		}
		seen[recommendation.Position] = struct{}{}
	}
}

func TestSolveAfterRedPrioritizesRemainingReward(t *testing.T) {
	board := Board{}
	board[1] = Red
	unavailable := [CellCount]bool{}
	unavailable[1] = true
	result, err := Solve(board, unavailable)
	if err != nil {
		t.Fatal(err)
	}
	if result.CandidateCount != 1 || result.RedProbabilities[1] != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Recommendations) == 0 || result.Recommendations[0].ImmediateValue <= 0 {
		t.Fatalf("recommendations = %#v", result.Recommendations)
	}
}

func TestSolveReturnsNoSuggestionsWhenEveryCellIsUnavailable(t *testing.T) {
	unavailable := [CellCount]bool{}
	for index := range unavailable {
		unavailable[index] = true
	}
	result, err := Solve(Board{}, unavailable)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Recommendations) != 0 {
		t.Fatalf("recommendations = %#v", result.Recommendations)
	}
}

func TestCompletionCountsEnforceRemainingQuotas(t *testing.T) {
	board := Board{}
	board[1], board[5] = Teal, Teal
	if count := completionCount(board, 0); count != 0 {
		t.Fatalf("completionCount() = %f", count)
	}
}

func TestColorValues(t *testing.T) {
	want := map[Color]float64{Unknown: 0, Blue: 10, Teal: 20, Green: 35, Yellow: 55, Orange: 90, Red: 150, Color(99): 0}
	for color, value := range want {
		if got := color.Value(); got != value {
			t.Fatalf("Color(%d).Value() = %f, want %f", color, got, value)
		}
	}
}

func TestCombinations(t *testing.T) {
	tests := []struct {
		total, selected int
		want            float64
	}{{4, 2, 6}, {8, 3, 56}, {3, 0, 1}, {2, 3, 0}, {3, -1, 0}}
	for _, test := range tests {
		if got := combinations(test.total, test.selected); got != test.want {
			t.Fatalf("combinations(%d, %d) = %f, want %f", test.total, test.selected, got, test.want)
		}
	}
}

func assertProbabilitySum(t *testing.T, probabilities [CellCount]float64) {
	t.Helper()
	total := 0.0
	for _, probability := range probabilities {
		total += probability
	}
	if math.Abs(total-1) > 1e-9 {
		t.Fatalf("probability sum = %f", total)
	}
}
