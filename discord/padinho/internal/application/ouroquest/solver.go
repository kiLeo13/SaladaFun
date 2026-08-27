// Package ouroquest solves Mudae's five-by-five $oq sphere game.
package ouroquest

import (
	"errors"
	"math"
	"math/bits"
)

const (
	// BoardWidth is the number of rows and columns in an $oq board.
	BoardWidth = 5
	// CellCount is the number of buttons in an $oq board.
	CellCount = BoardWidth * BoardWidth
	// TargetCount is the number of purple locations in every legal board.
	TargetCount = 4
	// PaidClickLimit is the number of non-free choices in one game.
	PaidClickLimit      = 7
	defaultSearchDepth  = 2
	deepCandidateLimit  = 96
	exactCandidateLimit = 16
	comparisonEpsilon   = 1e-12
)

// ErrInconsistentBoard reports observations that match no legal target layout.
var ErrInconsistentBoard = errors.New("ouroquest observations are inconsistent")

// Color identifies one revealed $oq sphere color.
type Color uint8

const (
	Unknown Color = iota
	Blue
	Teal
	Green
	Yellow
	Orange
	Purple
	Red
)

// Value returns the base sphere reward for a reveal at the current game phase.
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
	case Purple:
		return 165
	case Red:
		return 150
	default:
		return 0
	}
}

// Board stores row-major observations; Unknown denotes an unrevealed cell.
type Board [CellCount]Color

// Recommendation describes the expected-payout-optimal next click.
type Recommendation struct {
	Position              int
	PurpleProbability     float64
	ImmediateValue        float64
	ExpectedSearchValue   float64
	CompletionProbability float64
	InformationGain       float64
}

// Result contains the exact posterior and one recommended move.
type Result struct {
	CandidateCount      int
	PurpleProbabilities [CellCount]float64
	Recommendation      *Recommendation
}

type world struct {
	mask uint32
}

type actionValue struct {
	position    int
	immediate   float64
	total       float64
	completion  float64
	information float64
	purple      float64
}

var (
	neighborMasks = buildNeighborMasks()
	legalWorlds   = buildWorlds()
)

// Solve filters every legal world and performs bounded finite-horizon expectimax.
func Solve(board Board, unavailable [CellCount]bool) (Result, error) {
	found, spent, valid := boardProgress(board)
	if !valid || spent > PaidClickLimit {
		return Result{}, ErrInconsistentBoard
	}
	candidates := compatibleWorlds(board)
	if len(candidates) == 0 {
		return Result{}, ErrInconsistentBoard
	}
	result := Result{CandidateCount: len(candidates)}
	for _, candidate := range candidates {
		for position := 0; position < CellCount; position++ {
			if candidate.mask&(1<<position) != 0 {
				result.PurpleProbabilities[position]++
			}
		}
	}
	for position := range result.PurpleProbabilities {
		result.PurpleProbabilities[position] /= float64(len(candidates))
	}
	remaining := PaidClickLimit - spent
	if remaining <= 0 {
		return result, nil
	}
	unavailableMask := maskUnavailable(unavailable, board)
	depth := defaultSearchDepth
	if len(candidates) <= deepCandidateLimit || remaining <= 2 {
		depth = 4
	}
	if len(candidates) <= exactCandidateLimit {
		depth = remaining + TargetCount - found
	}
	best, ok := bestAction(candidates, unavailableMask, found, remaining, depth)
	if !ok {
		return result, nil
	}
	result.Recommendation = &Recommendation{
		Position: best.position, PurpleProbability: best.purple,
		ImmediateValue: best.immediate, ExpectedSearchValue: best.total,
		CompletionProbability: best.completion, InformationGain: best.information,
	}
	return result, nil
}

// compatibleWorlds returns all equally likely layouts matching every observation.
func compatibleWorlds(board Board) []world {
	candidates := make([]world, 0, len(legalWorlds))
	for _, candidate := range legalWorlds {
		compatible := true
		for position, color := range board {
			if color == Unknown {
				continue
			}
			isTarget := candidate.mask&(1<<position) != 0
			if color == Purple || color == Red {
				compatible = isTarget
			} else {
				compatible = !isTarget && color == numberedColor(candidate, position)
			}
			if !compatible {
				break
			}
		}
		if compatible {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

// bestAction evaluates every legal action and applies deterministic tie-breaking.
func bestAction(candidates []world, unavailable uint32, found, remaining, depth int) (actionValue, bool) {
	best := actionValue{position: -1}
	for position := 0; position < CellCount; position++ {
		if unavailable&(1<<position) != 0 {
			continue
		}
		candidate := evaluateAction(candidates, unavailable, found, remaining, depth, position)
		if best.position < 0 || betterAction(candidate, best) {
			best = candidate
		}
	}
	return best, best.position >= 0
}

// evaluateAction partitions the posterior by reveal and evaluates future play.
func evaluateAction(candidates []world, unavailable uint32, found, remaining, depth, position int) actionValue {
	groups := make(map[Color][]world, 6)
	for _, candidate := range candidates {
		color := outcome(candidate, position, found)
		groups[color] = append(groups[color], candidate)
	}
	value := actionValue{position: position}
	totalCandidates := float64(len(candidates))
	for color, group := range groups {
		probability := float64(len(group)) / totalCandidates
		if color == Purple || color == Red {
			value.purple += probability
		}
		value.immediate += probability * color.Value()
		value.information -= probability * math.Log2(probability)
		nextFound := found
		if color == Purple || color == Red {
			nextFound++
		}
		cost := 1
		if color == Purple {
			cost = 0
		}
		nextRemaining := remaining - cost
		futureReward := 0.0
		completion := 0.0
		if nextFound >= TargetCount {
			completion = 1
		}
		if nextRemaining > 0 && depth > 1 {
			next, ok := bestAction(group, unavailable|(1<<position), nextFound, nextRemaining, depth-1)
			if ok {
				futureReward = next.total
				completion = next.completion
			}
		}
		value.total += probability * (color.Value() + futureReward)
		value.completion += probability * completion
	}
	return value
}

// betterAction orders utility, completion, information, and stable position.
func betterAction(candidate, current actionValue) bool {
	if math.Abs(candidate.total-current.total) > comparisonEpsilon {
		return candidate.total > current.total
	}
	if math.Abs(candidate.completion-current.completion) > comparisonEpsilon {
		return candidate.completion > current.completion
	}
	if math.Abs(candidate.information-current.information) > comparisonEpsilon {
		return candidate.information > current.information
	}
	return candidate.position < current.position
}

// outcome maps a world and current phase to the Discord-visible reveal color.
func outcome(candidate world, position, found int) Color {
	if candidate.mask&(1<<position) != 0 {
		if found >= TargetCount-1 {
			return Red
		}
		return Purple
	}
	return numberedColor(candidate, position)
}

// numberedColor converts an adjacent-target count to its sphere color.
func numberedColor(candidate world, position int) Color {
	return Color(int(Blue) + bits.OnesCount32(candidate.mask&neighborMasks[position]))
}

// boardProgress validates target order and counts consumed paid clicks.
func boardProgress(board Board) (int, int, bool) {
	found, spent, red := 0, 0, 0
	for _, color := range board {
		switch color {
		case Unknown:
		case Blue, Teal, Green, Yellow, Orange:
			spent++
		case Purple:
			found++
		case Red:
			found++
			spent++
			red++
		default:
			return 0, 0, false
		}
	}
	return found, spent, red <= 1 && (red == 0 || found == TargetCount)
}

// maskUnavailable combines Discord-disabled buttons with all observed cells.
func maskUnavailable(unavailable [CellCount]bool, board Board) uint32 {
	var result uint32
	for position := 0; position < CellCount; position++ {
		if unavailable[position] || board[position] != Unknown {
			result |= 1 << position
		}
	}
	return result
}

// buildWorlds enumerates all C(25,4) equally likely purple layouts.
func buildWorlds() []world {
	result := make([]world, 0, 12650)
	for a := 0; a < CellCount-3; a++ {
		for b := a + 1; b < CellCount-2; b++ {
			for c := b + 1; c < CellCount-1; c++ {
				for d := c + 1; d < CellCount; d++ {
					result = append(result, world{mask: 1<<a | 1<<b | 1<<c | 1<<d})
				}
			}
		}
	}
	return result
}

// buildNeighborMasks precomputes each cell's eight-neighbor geometry.
func buildNeighborMasks() [CellCount]uint32 {
	var result [CellCount]uint32
	for position := 0; position < CellCount; position++ {
		row, column := position/BoardWidth, position%BoardWidth
		for rowDelta := -1; rowDelta <= 1; rowDelta++ {
			for columnDelta := -1; columnDelta <= 1; columnDelta++ {
				neighborRow, neighborColumn := row+rowDelta, column+columnDelta
				if (rowDelta == 0 && columnDelta == 0) || neighborRow < 0 || neighborRow >= BoardWidth || neighborColumn < 0 || neighborColumn >= BoardWidth {
					continue
				}
				result[position] |= 1 << (neighborRow*BoardWidth + neighborColumn)
			}
		}
	}
	return result
}
