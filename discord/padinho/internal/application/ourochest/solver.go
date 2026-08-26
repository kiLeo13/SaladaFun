// Package ourochest solves Mudae's five-by-five $oc sphere game.
package ourochest

import (
	"errors"
	"math"
)

const (
	// BoardWidth is the number of rows and columns in an $oc board.
	BoardWidth = 5
	// CellCount is the number of buttons in an $oc board.
	CellCount = BoardWidth * BoardWidth
	// MaxClicks is the number of paid choices in one $oc game.
	MaxClicks = 5

	orangeQuota       = 2
	yellowQuota       = 3
	greenQuota        = 4
	centerCell        = CellCount / 2
	comparisonEpsilon = 1e-12
)

// ErrInconsistentBoard reports observations that cannot occur on a legal board.
var ErrInconsistentBoard = errors.New("ourochest observations are inconsistent")

// Color identifies one revealed $oc sphere color.
type Color uint8

const (
	Unknown Color = iota
	Blue
	Teal
	Green
	Yellow
	Orange
	Red
)

var revealedColors = [...]Color{Blue, Teal, Green, Yellow, Orange, Red}

// Value returns the base sphere reward for a color.
func (c Color) Value() float64 {
	switch c {
	case Blue:
		return 10
	case Teal:
		return 20
	case Green:
		return 35
	case Yellow:
		return 55
	case Orange:
		return 90
	case Red:
		return 150
	default:
		return 0
	}
}

// Board stores row-major observations; Unknown denotes an unrevealed cell.
type Board [CellCount]Color

// RecommendationKind explains the objective that selected a position.
type RecommendationKind uint8

const (
	Balanced RecommendationKind = iota
	Information
	Reward
	DirectRed
)

// Recommendation contains one distinct suggested click and its evidence.
type Recommendation struct {
	Kind               RecommendationKind
	Position           int
	RedProbability     float64
	ImmediateValue     float64
	InformationGain    float64
	ExpectedCandidates float64
}

// Result contains the posterior red distribution and non-dominated suggestions.
type Result struct {
	CandidateCount   int
	RedProbabilities [CellCount]float64
	Recommendations  []Recommendation
}

type posterior struct {
	weights    [CellCount]float64
	total      float64
	candidates int
}

type cellMetrics struct {
	position           int
	redProbability     float64
	immediateValue     float64
	informationGain    float64
	expectedCandidates float64
	balancedScore      float64
	balancedTotal      float64
}

// Solve calculates exact legal-board probabilities and dynamic objective suggestions.
func Solve(board Board, unavailable [CellCount]bool) (Result, error) {
	current := analyze(board)
	if current.total <= 0 {
		return Result{}, ErrInconsistentBoard
	}

	result := Result{CandidateCount: current.candidates}
	for position, weight := range current.weights {
		result.RedProbabilities[position] = weight / current.total
	}

	metrics := make([]cellMetrics, 0, CellCount)
	currentEntropy := entropy(current)
	for position, color := range board {
		if color != Unknown || unavailable[position] {
			continue
		}
		metrics = append(metrics, evaluateCell(board, position, current, currentEntropy))
	}
	if len(metrics) == 0 {
		return result, nil
	}

	applyBalancedScores(metrics)
	selected := make(map[int]struct{}, 4)
	appendRecommendation := func(kind RecommendationKind, metric cellMetrics) {
		if _, exists := selected[metric.position]; exists {
			return
		}
		selected[metric.position] = struct{}{}
		result.Recommendations = append(result.Recommendations, Recommendation{
			Kind: kind, Position: metric.position, RedProbability: metric.redProbability,
			ImmediateValue: metric.immediateValue, InformationGain: metric.informationGain,
			ExpectedCandidates: metric.expectedCandidates,
		})
	}

	appendRecommendation(Balanced, bestMetric(metrics, betterBalanced))
	informationBest := bestMetric(metrics, betterInformation)
	if informationBest.informationGain > comparisonEpsilon {
		appendRecommendation(Information, informationBest)
	}
	appendRecommendation(Reward, bestMetric(metrics, betterReward))
	redBest := bestMetric(metrics, betterRed)
	if redBest.redProbability > comparisonEpsilon {
		appendRecommendation(DirectRed, redBest)
	}
	return result, nil
}

// evaluateCell calculates the outcome distribution and one-click decision metrics.
func evaluateCell(board Board, position int, current posterior, currentEntropy float64) cellMetrics {
	metric := cellMetrics{position: position}
	for _, color := range revealedColors {
		observed := board
		observed[position] = color
		after := analyze(observed)
		probability := after.total / current.total
		if probability <= 0 {
			continue
		}
		if color == Red {
			metric.redProbability = probability
		}
		metric.immediateValue += probability * color.Value()
		metric.expectedCandidates += probability * float64(after.candidates)
		metric.informationGain += probability * entropy(after)
	}
	metric.informationGain = math.Max(0, currentEntropy-metric.informationGain)
	return metric
}

// analyze weighs every red position by its compatible completion fraction.
func analyze(board Board) posterior {
	var result posterior
	for redPosition := 0; redPosition < CellCount; redPosition++ {
		base := completionCount(Board{}, redPosition)
		if base == 0 {
			continue
		}
		compatible := completionCount(board, redPosition)
		if compatible == 0 {
			continue
		}
		weight := compatible / base
		result.weights[redPosition] = weight
		result.total += weight
		result.candidates++
	}
	return result
}

// completionCount counts legal color placements for one fixed red position.
func completionCount(board Board, redPosition int) float64 {
	if redPosition == centerCell || board[redPosition] > Unknown && board[redPosition] != Red {
		return 0
	}

	knownRed := 0
	knownOrange := 0
	knownYellow := 0
	knownGreen := 0
	unknownOrthogonal := 0
	unknownDiagonal := 0
	unknownRowColumn := 0

	for position, color := range board {
		if color == Red {
			knownRed++
			if position != redPosition {
				return 0
			}
		}
		if position == redPosition {
			continue
		}

		relation := relationBetween(position, redPosition)
		switch color {
		case Unknown:
			if relation.orthogonal {
				unknownOrthogonal++
			}
			if relation.diagonal {
				unknownDiagonal++
			}
			if relation.rowColumn {
				unknownRowColumn++
			}
		case Orange:
			knownOrange++
			if !relation.orthogonal {
				return 0
			}
		case Yellow:
			knownYellow++
			if !relation.diagonal {
				return 0
			}
		case Green:
			knownGreen++
			if !relation.rowColumn {
				return 0
			}
		case Teal:
			if !relation.rowColumn && !relation.diagonal {
				return 0
			}
		case Blue:
			if relation.rowColumn || relation.diagonal {
				return 0
			}
		case Red:
			return 0
		default:
			return 0
		}
	}
	if knownRed > 1 || knownOrange > orangeQuota || knownYellow > yellowQuota || knownGreen > greenQuota {
		return 0
	}

	remainingOrange := orangeQuota - knownOrange
	remainingYellow := yellowQuota - knownYellow
	remainingGreen := greenQuota - knownGreen
	if unknownOrthogonal < remainingOrange || unknownDiagonal < remainingYellow || unknownRowColumn-remainingOrange < remainingGreen {
		return 0
	}
	return combinations(unknownOrthogonal, remainingOrange) *
		combinations(unknownDiagonal, remainingYellow) *
		combinations(unknownRowColumn-remainingOrange, remainingGreen)
}

type relation struct {
	orthogonal bool
	diagonal   bool
	rowColumn  bool
}

// relationBetween classifies the geometric relationship between two cells.
func relationBetween(first, second int) relation {
	firstRow, firstColumn := first/BoardWidth, first%BoardWidth
	secondRow, secondColumn := second/BoardWidth, second%BoardWidth
	rowDelta := abs(firstRow - secondRow)
	columnDelta := abs(firstColumn - secondColumn)
	return relation{
		orthogonal: rowDelta+columnDelta == 1,
		diagonal:   rowDelta == columnDelta && rowDelta > 0,
		rowColumn:  (rowDelta == 0) != (columnDelta == 0),
	}
}

// combinations returns the binomial coefficient total choose selected.
func combinations(total, selected int) float64 {
	if selected < 0 || total < selected {
		return 0
	}
	if selected == 0 || selected == total {
		return 1
	}
	if selected > total-selected {
		selected = total - selected
	}
	result := 1
	for divisor := 1; divisor <= selected; divisor++ {
		result = result * (total - selected + divisor) / divisor
	}
	return float64(result)
}

// entropy returns the Shannon entropy of the posterior red distribution.
func entropy(value posterior) float64 {
	if value.total <= 0 {
		return 0
	}
	result := 0.0
	for _, weight := range value.weights {
		if weight <= 0 {
			continue
		}
		probability := weight / value.total
		result -= probability * math.Log2(probability)
	}
	return result
}

// applyBalancedScores calculates an equal-regret compromise across three objectives.
func applyBalancedScores(metrics []cellMetrics) {
	redMin, redMax := metricRange(metrics, func(metric cellMetrics) float64 { return metric.redProbability })
	infoMin, infoMax := metricRange(metrics, func(metric cellMetrics) float64 { return metric.informationGain })
	valueMin, valueMax := metricRange(metrics, func(metric cellMetrics) float64 { return metric.immediateValue })
	for index := range metrics {
		red := normalize(metrics[index].redProbability, redMin, redMax)
		information := normalize(metrics[index].informationGain, infoMin, infoMax)
		value := normalize(metrics[index].immediateValue, valueMin, valueMax)
		metrics[index].balancedScore = math.Min(red, math.Min(information, value))
		metrics[index].balancedTotal = red + information + value
	}
}

// metricRange returns the minimum and maximum projection across the metrics.
func metricRange(metrics []cellMetrics, value func(cellMetrics) float64) (float64, float64) {
	minimum, maximum := value(metrics[0]), value(metrics[0])
	for _, metric := range metrics[1:] {
		minimum = math.Min(minimum, value(metric))
		maximum = math.Max(maximum, value(metric))
	}
	return minimum, maximum
}

// normalize scales a metric to zero through one and treats a shared value as ideal.
func normalize(value, minimum, maximum float64) float64 {
	if maximum-minimum <= comparisonEpsilon {
		return 1
	}
	return (value - minimum) / (maximum - minimum)
}

// bestMetric selects one metric using an objective-specific comparator.
func bestMetric(metrics []cellMetrics, better func(cellMetrics, cellMetrics) bool) cellMetrics {
	best := metrics[0]
	for _, candidate := range metrics[1:] {
		if better(candidate, best) {
			best = candidate
		}
	}
	return best
}

// betterBalanced compares the worst and total normalized objective regret.
func betterBalanced(candidate, current cellMetrics) bool {
	return compare(candidate.balancedScore, current.balancedScore,
		candidate.balancedTotal, current.balancedTotal, candidate.position, current.position)
}

// betterInformation prefers entropy reduction and then fewer expected candidates.
func betterInformation(candidate, current cellMetrics) bool {
	return compare(candidate.informationGain, current.informationGain,
		-candidate.expectedCandidates, -current.expectedCandidates, candidate.position, current.position)
}

// betterReward prefers immediate expected sphere value and then red probability.
func betterReward(candidate, current cellMetrics) bool {
	return compare(candidate.immediateValue, current.immediateValue,
		candidate.redProbability, current.redProbability, candidate.position, current.position)
}

// betterRed prefers immediate red probability and then information gain.
func betterRed(candidate, current cellMetrics) bool {
	return compare(candidate.redProbability, current.redProbability,
		candidate.informationGain, current.informationGain, candidate.position, current.position)
}

// compare applies numeric objectives followed by the compatibility position order.
func compare(primaryCandidate, primaryCurrent, secondaryCandidate, secondaryCurrent float64, candidatePosition, currentPosition int) bool {
	if math.Abs(primaryCandidate-primaryCurrent) > comparisonEpsilon {
		return primaryCandidate > primaryCurrent
	}
	if math.Abs(secondaryCandidate-secondaryCurrent) > comparisonEpsilon {
		return secondaryCandidate > secondaryCurrent
	}
	return positionRank(candidatePosition) < positionRank(currentPosition)
}

// positionRank preserves the helper-compatible initial symmetry tie preference.
func positionRank(position int) int {
	if position == 16 {
		return -1
	}
	return position
}

// abs returns the non-negative magnitude of an integer.
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
