package routines

type AttemptCounter struct {
	Attempts map[string]int
}

func NewAttemptCounter() AttemptCounter {
	return AttemptCounter{
		Attempts: make(map[string]int),
	}
}

func (attemptCounter *AttemptCounter) Reset(actionID string) {
	attemptCounter.Attempts[actionID] = 0
}

func (attemptCounter *AttemptCounter) Increment(actionID string) {
	val, ok := attemptCounter.Attempts[actionID]
	if !ok {
		attemptCounter.Attempts[actionID] = 1
	} else {
		attemptCounter.Attempts[actionID] = val + 1
	}
}

func (attemptCounter *AttemptCounter) Get(actionID string) int {
	val, ok := attemptCounter.Attempts[actionID]
	if !ok {
		return 0
	}
	return val
}
