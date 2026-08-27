// Package ouroharvest calculates optimal expected-value policies for Mudae's $oh game.
package ouroharvest

import (
	"errors"
	"math"
	"sort"
	"sync"
)

const (
	// BoardWidth is the number of rows and columns in an $oh board.
	BoardWidth = 5
	// CellCount is the number of buttons in an $oh board.
	CellCount = BoardWidth * BoardWidth
	// PaidClickLimit is the initial number of paid clicks.
	PaidClickLimit = 5

	comparisonEpsilon              = 1e-9
	earlyHorizon                   = 2
	exactClickWindow               = 3
	revealBonusWeight              = 0.7
	minimumRevealBranchProbability = 1e-5
)

// ErrInvalidState reports counts that cannot describe an $oh board.
var ErrInvalidState = errors.New("invalid Ouroharvest state")

// Color identifies a visible or covered $oh button.
type Color uint8

const (
	Covered Color = iota
	Blue
	Teal
	Green
	Yellow
	Orange
	Red
	Purple
	Dark
	Light
	White
)

var outcomeColors = [...]Color{Blue, Teal, Green, Purple, Light, Yellow, Dark, Orange, Red, White}

var appearanceRates = map[Color]float64{
	Blue: 0.54487797, Teal: 0.23476856, Green: 0.07877798,
	Purple: 0.03934354, Light: 0.02962128, Yellow: 0.02573977,
	Dark: 0.01457667, Orange: 0.00970544, Red: 0.00222858,
	White: 0.00038041,
}

var flatValues = map[Color]float64{
	Green: 35, Yellow: 55, Light: 76.03, Orange: 90, Red: 150, White: 500,
}

// State is the position-independent sufficient statistic for one $oh board.
type State struct {
	ClicksLeft uint8
	Covered    uint8
	Blue       uint8
	Teal       uint8
	Dark       uint8
	Green      uint8
	Yellow     uint8
	Light      uint8
	Orange     uint8
	Red        uint8
	White      uint8
	ChestFound bool
}

// Action identifies one semantically distinct click choice.
type Action Color

// Recommendation describes one action and its complete-policy expectation.
type Recommendation struct {
	Action           Action
	ExpectedSP       float64
	ChestProbability float64
	AdvantageSP      float64
}

// Result contains the ranked available actions.
type Result struct {
	Recommendation *Recommendation
	Actions        []Recommendation
}

type value struct {
	sp    float64
	chest float64
}

type memoKey struct {
	state State
	depth uint8
}

type revealDelta struct {
	Blue, Teal, Dark                         uint8
	Green, Yellow, Light, Orange, Red, White uint8
	Purple                                   uint8
}

type revealBranch struct {
	delta       revealDelta
	probability float64
}

// Solver memoizes optimal continuations and is safe for concurrent games.
type Solver struct {
	mu   sync.Mutex
	memo map[memoKey]value
}

// NewSolver constructs an empty reusable policy cache.
func NewSolver() *Solver { return &Solver{memo: make(map[memoKey]value)} }

// Solve ranks every legal action by expected final sphere points.
func (s *Solver) Solve(state State) (Result, error) {
	if !validState(state) {
		return Result{}, ErrInvalidState
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	actions := reportableActions(state)
	depth := uint8(earlyHorizon)
	if state.ClicksLeft <= exactClickWindow {
		depth = state.ClicksLeft
	}
	recommendations := make([]Recommendation, 0, len(actions))
	for _, action := range actions {
		evaluation := s.evaluate(state, action, depth)
		recommendations = append(recommendations, Recommendation{
			Action: action, ExpectedSP: evaluation.sp, ChestProbability: evaluation.chest,
		})
	}
	sort.SliceStable(recommendations, func(i, j int) bool {
		return better(
			value{sp: recommendations[i].ExpectedSP, chest: recommendations[i].ChestProbability},
			value{sp: recommendations[j].ExpectedSP, chest: recommendations[j].ChestProbability},
		)
	})
	result := Result{Actions: recommendations}
	if len(recommendations) > 0 {
		result.Recommendation = &result.Actions[0]
		if len(recommendations) > 1 {
			result.Recommendation.AdvantageSP = recommendations[0].ExpectedSP - recommendations[1].ExpectedSP
		}
	}
	return result, nil
}

// solve returns the best continuation from state.
func (s *Solver) solve(state State, depth uint8) value {
	if state.ClicksLeft == 0 {
		return value{}
	}
	if depth == 0 {
		return greedyRollout(state)
	}
	key := memoKey{state: state, depth: depth}
	if cached, ok := s.memo[key]; ok {
		return cached
	}
	best := value{sp: -1}
	for _, action := range availableActions(state) {
		candidate := s.evaluate(state, action, depth)
		if best.sp < 0 || better(candidate, best) {
			best = candidate
		}
	}
	if best.sp < 0 {
		best = value{}
	}
	s.memo[key] = best
	return best
}

// evaluate calculates one action's reward plus its optimal continuation.
func (s *Solver) evaluate(state State, action Action, depth uint8) value {
	nextDepth := depth - 1
	color := Color(action)
	switch color {
	case Covered:
		return s.clickCovered(state, nextDepth)
	case Blue:
		state.Blue--
		state.ClicksLeft--
		return s.resolveBlueOrTeal(state, Blue, nextDepth)
	case Teal:
		state.Teal--
		state.ClicksLeft--
		return s.resolveBlueOrTeal(state, Teal, nextDepth)
	case Dark:
		state.Dark--
		state.ClicksLeft--
		return s.resolveDark(state, nextDepth)
	default:
		removeFlat(&state, color)
		state.ClicksLeft--
		continuation := s.solve(state, nextDepth)
		continuation.sp += flatValues[color]
		return continuation
	}
}

// clickCovered resolves the mutually exclusive hidden-$oc and sphere outcomes.
func (s *Solver) clickCovered(state State, depth uint8) value {
	state.Covered--
	state.ClicksLeft--
	colorMass := appearanceMass()
	chestRate := 1 - colorMass
	if state.ChestFound {
		chestRate = 0
	}
	result := value{}
	if chestRate > 0 {
		chestState := state
		chestState.ChestFound = true
		continuation := s.solve(chestState, depth)
		result.sp += chestRate * continuation.sp
		result.chest += chestRate
	}
	colorScale := (1 - chestRate) / colorMass
	for _, color := range outcomeColors {
		branch := s.resolveDirectColor(state, color, depth)
		probability := appearanceRates[color] * colorScale
		result.sp += probability * branch.sp
		result.chest += probability * branch.chest
	}
	return result
}

// resolveDirectColor applies the full behavior of a sphere collected by a paid click.
func (s *Solver) resolveDirectColor(state State, color Color, depth uint8) value {
	switch color {
	case Purple:
		state.ClicksLeft++
		continuation := s.solve(state, depth)
		continuation.sp += 5
		return continuation
	case Blue, Teal:
		return s.resolveBlueOrTeal(state, color, depth)
	case Dark:
		return s.resolveDark(state, depth)
	default:
		continuation := s.solve(state, depth)
		continuation.sp += flatValues[color]
		return continuation
	}
}

// resolveBlueOrTeal adds its reward and averages all cascade reveal combinations.
func (s *Solver) resolveBlueOrTeal(state State, color Color, depth uint8) value {
	reveals := uint8(1)
	reward := 20.0
	if color == Blue {
		reveals = 3
		reward = 10
	}
	if reveals > state.Covered {
		reveals = state.Covered
	}
	result := value{}
	for _, branch := range revealBranches[reveals] {
		next := state
		next.Covered -= reveals
		applyDelta(&next, branch.delta)
		continuation := s.solve(next, depth)
		result.sp += branch.probability * (reward + 5*float64(branch.delta.Purple) + continuation.sp)
		result.chest += branch.probability * continuation.chest
	}
	return result
}

// resolveDark averages its nine equiprobable non-dark transformations.
func (s *Solver) resolveDark(state State, depth uint8) value {
	darkOutcomes := [...]Color{Blue, Teal, Green, Yellow, Orange, Red, Purple, Light, White}
	result := value{}
	for _, color := range darkOutcomes {
		branch := s.resolveDirectColor(state, color, depth)
		result.sp += branch.sp / float64(len(darkOutcomes))
		result.chest += branch.chest / float64(len(darkOutcomes))
	}
	return result
}

// greedyRollout supplies a deterministic tail estimate beyond the exact horizon.
func greedyRollout(state State) value {
	result := value{}
	for state.ClicksLeft > 0 {
		actions := availableActions(state)
		if len(actions) == 0 {
			break
		}
		best := actions[0]
		bestReward := estimatedImmediate(state, Color(best))
		for _, action := range actions[1:] {
			reward := estimatedImmediate(state, Color(action))
			if reward > bestReward {
				best, bestReward = action, reward
			}
		}
		result.sp += bestReward
		applyGreedyAction(&state, Color(best))
		if Color(best) == Covered && !state.ChestFound {
			chestRate := 1 - appearanceMass()
			result.chest = 1 - (1-result.chest)*(1-chestRate)
		}
	}
	return result
}

// estimatedImmediate mirrors the reference solver's conservative reveal bonus.
func estimatedImmediate(state State, color Color) float64 {
	coveredEV := expectedCoveredSphereValue()
	switch color {
	case Covered:
		return coveredEV
	case Blue:
		return 10 + revealBonusWeight*float64(minUint8(3, state.Covered))*coveredEV
	case Teal:
		return 20 + revealBonusWeight*float64(minUint8(1, state.Covered))*coveredEV
	case Dark:
		return 104.3
	default:
		return flatValues[color]
	}
}

// applyGreedyAction removes the resources consumed by one rollout choice.
func applyGreedyAction(state *State, color Color) {
	state.ClicksLeft--
	switch color {
	case Covered:
		state.Covered--
	case Blue:
		state.Blue--
		state.Covered -= minUint8(3, state.Covered)
	case Teal:
		state.Teal--
		state.Covered -= minUint8(1, state.Covered)
	case Dark:
		state.Dark--
	default:
		removeFlat(state, color)
	}
}

// expectedCoveredSphereValue returns the conditional mean of a non-$oc cell.
func expectedCoveredSphereValue() float64 {
	total := 0.0
	for color, probability := range appearanceRates {
		value := flatValues[color]
		switch color {
		case Blue:
			value = 10
		case Teal:
			value = 20
		case Purple:
			value = 5
		case Dark:
			value = 104.3
		}
		total += probability * value
	}
	return total / appearanceMass()
}

// minUint8 returns the smaller unsigned value.
func minUint8(left, right uint8) uint8 {
	if left < right {
		return left
	}
	return right
}

// availableActions returns one action for every present semantic button type.
func availableActions(state State) []Action {
	if state.ClicksLeft == 0 {
		return nil
	}
	actions := make([]Action, 0, 5)
	counts := [...]struct {
		color Color
		count uint8
	}{
		{Covered, state.Covered}, {Blue, state.Blue}, {Teal, state.Teal},
		{Dark, state.Dark},
	}
	for _, item := range counts {
		if item.count > 0 {
			actions = append(actions, Action(item.color))
		}
	}
	// Deterministic flat rewards have no state effects, so a lower-valued flat
	// can never dominate the highest visible one. Evaluating only that sphere is exact.
	for _, item := range [...]struct {
		color Color
		count uint8
	}{
		{White, state.White}, {Red, state.Red}, {Orange, state.Orange},
		{Light, state.Light}, {Yellow, state.Yellow}, {Green, state.Green},
	} {
		if item.count > 0 {
			actions = append(actions, Action(item.color))
			break
		}
	}
	return actions
}

// reportableActions includes dominated flats so the top-level EV margin is truthful.
func reportableActions(state State) []Action {
	actions := availableActions(state)
	if state.ClicksLeft == 0 {
		return actions
	}
	selectedFlat := Color(255)
	for _, action := range actions {
		if _, flat := flatValues[Color(action)]; flat {
			selectedFlat = Color(action)
		}
	}
	for _, item := range [...]struct {
		color Color
		count uint8
	}{
		{White, state.White}, {Red, state.Red}, {Orange, state.Orange},
		{Light, state.Light}, {Yellow, state.Yellow}, {Green, state.Green},
	} {
		if item.count > 0 && item.color != selectedFlat {
			actions = append(actions, Action(item.color))
		}
	}
	return actions
}

// validState checks click and board count bounds.
func validState(state State) bool {
	if state.ClicksLeft > PaidClickLimit {
		return false
	}
	total := int(state.Covered) + int(state.Blue) + int(state.Teal) + int(state.Dark) +
		int(state.Green) + int(state.Yellow) + int(state.Light) + int(state.Orange) +
		int(state.Red) + int(state.White)
	return total <= CellCount
}

// removeFlat removes one deterministic visible reward.
func removeFlat(state *State, color Color) {
	switch color {
	case Green:
		state.Green--
	case Yellow:
		state.Yellow--
	case Light:
		state.Light--
	case Orange:
		state.Orange--
	case Red:
		state.Red--
	case White:
		state.White--
	}
}

// applyDelta adds one precomputed cascade outcome to a state.
func applyDelta(state *State, delta revealDelta) {
	state.Blue += delta.Blue
	state.Teal += delta.Teal
	state.Dark += delta.Dark
	state.Green += delta.Green
	state.Yellow += delta.Yellow
	state.Light += delta.Light
	state.Orange += delta.Orange
	state.Red += delta.Red
	state.White += delta.White
}

// better applies expected SP first and hidden-$oc probability as a strict tie-break.
func better(left, right value) bool {
	if math.Abs(left.sp-right.sp) > comparisonEpsilon {
		return left.sp > right.sp
	}
	return left.chest > right.chest+comparisonEpsilon
}

// appearanceMass returns the published probability assigned to sphere colors.
func appearanceMass() float64 {
	total := 0.0
	for _, probability := range appearanceRates {
		total += probability
	}
	return total
}

var revealBranches = buildRevealBranches()

// buildRevealBranches precomputes conditional color combinations for zero to three reveals.
func buildRevealBranches() [4][]revealBranch {
	var result [4][]revealBranch
	result[0] = []revealBranch{{probability: 1}}
	colorMass := appearanceMass()
	current := map[revealDelta]float64{{}: 1}
	for count := 1; count <= 3; count++ {
		next := make(map[revealDelta]float64)
		for delta, probability := range current {
			for _, color := range outcomeColors {
				updated := delta
				incrementDelta(&updated, color)
				next[updated] += probability * appearanceRates[color] / colorMass
			}
		}
		current = next
		result[count] = make([]revealBranch, 0, len(current))
		retainedMass := 0.0
		for delta, probability := range current {
			if probability >= minimumRevealBranchProbability {
				result[count] = append(result[count], revealBranch{delta: delta, probability: probability})
				retainedMass += probability
			}
		}
		for index := range result[count] {
			result[count][index].probability /= retainedMass
		}
	}
	return result
}

// incrementDelta records one newly revealed sphere.
func incrementDelta(delta *revealDelta, color Color) {
	switch color {
	case Blue:
		delta.Blue++
	case Teal:
		delta.Teal++
	case Dark:
		delta.Dark++
	case Green:
		delta.Green++
	case Yellow:
		delta.Yellow++
	case Light:
		delta.Light++
	case Orange:
		delta.Orange++
	case Red:
		delta.Red++
	case White:
		delta.White++
	case Purple:
		delta.Purple++
	}
}
