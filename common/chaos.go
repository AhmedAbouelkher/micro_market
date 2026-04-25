package common

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type chaosState struct {
	mu        sync.Mutex
	requests  int
	failAt    map[int]struct{}
	windowEnd int
}

var chaosStates sync.Map

func MaybeError(key string) error {
	stateAny, _ := chaosStates.LoadOrStore(key, &chaosState{})
	state := stateAny.(*chaosState)

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.windowEnd == 0 || state.requests >= state.windowEnd {
		state.requests = 0
		state.windowEnd = 20
		state.failAt = buildFailSet()
	}

	state.requests++
	if _, ok := state.failAt[state.requests]; !ok {
		return nil
	}
	_400Status := []int{http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusNotFound}
	_500Status := []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout}
	if rand.Float64() < 0.5 {
		return NewAppError(_400Status[rand.Intn(len(_400Status)-1)], "chaos test: injected client error")
	}
	return NewAppError(_500Status[rand.Intn(len(_500Status)-1)], "chaos test: injected server error")
}

func buildFailSet() map[int]struct{} {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	failCount := 2 + r.Intn(4)
	positions := r.Perm(20)[:failCount]
	failAt := make(map[int]struct{}, failCount)
	for _, pos := range positions {
		failAt[pos+1] = struct{}{}
	}
	return failAt
}
