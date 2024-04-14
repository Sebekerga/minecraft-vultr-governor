package routines

import (
	"fmt"
	"testing"
)

func printHandler(level PrintLevel, message string) {
	fmt.Println("["+level.String()+"]", message)
}

type EmptyContext struct{}

func StepA(ctx *EmptyContext, ph PrintHandler) (RoutineFunc[EmptyContext], error) {
	ph(INFO, "╰-Step A")
	return nil, nil
}

func TestSimpleRoutine(t *testing.T) {

	routine := InitRoutine(StepA, EmptyContext{}, printHandler)
	printHandler(INFO, "Running simple routine")
	err := routine.Run()
	if err != nil {
		t.Error(err)
	}
}

func AdvancedStepA(ctx *EmptyContext, ph PrintHandler) (RoutineFunc[EmptyContext], error) {
	ph(INFO, "╰-Advanced Step A")
	return AdvancedStepB, nil
}

func AdvancedStepB(ctx *EmptyContext, ph PrintHandler) (RoutineFunc[EmptyContext], error) {
	ph(INFO, "╰-Advanced Step B")
	return nil, nil
}

func TestAdvancedRoutine(t *testing.T) {
	routine := InitRoutine(AdvancedStepA, EmptyContext{}, printHandler)
	printHandler(INFO, "Running advanced routine")
	err := routine.Run()
	if err != nil {
		t.Error(err)
	}
}

func WithErrorStepA(ctx *EmptyContext, ph PrintHandler) (RoutineFunc[EmptyContext], error) {
	ph(INFO, "╰-With Error Step A")
	return WithErrorStepB, nil
}

func WithErrorStepB(ctx *EmptyContext, ph PrintHandler) (RoutineFunc[EmptyContext], error) {
	ph(INFO, "╰-With Error Step B (should fail)")
	return nil, fmt.Errorf("Error in step B")
}

func TestRoutineWithError(t *testing.T) {
	routine := InitRoutine(WithErrorStepA, EmptyContext{}, printHandler)
	printHandler(INFO, "Running routine with error")
	err := routine.Run()
	if err == nil {
		t.Error("Expected error, got nil")
	}

}

const COUNTER_LIMIT = 5

type ContextWithValue struct {
	counter int16
}

func WithContextLoopStep(ctx *ContextWithValue, ph PrintHandler) (RoutineFunc[ContextWithValue], error) {
	ph(INFO, fmt.Sprintf("╰-With Context Step A for counter: %d", ctx.counter))
	ctx.counter++
	ph(INFO, fmt.Sprintf(" ╰-Counter is now: %d", ctx.counter))
	if ctx.counter >= COUNTER_LIMIT {
		return nil, nil
	}

	return WithContextLoopStep, nil
}

func TestRoutineWithContext(t *testing.T) {
	routine := InitRoutine(WithContextLoopStep, ContextWithValue{counter: 0}, printHandler)
	printHandler(INFO, "Running routine with context")
	err := routine.Run()
	if err != nil {
		t.Error(err)
	}
	if routine.Context.counter != COUNTER_LIMIT {
		t.Errorf("Expected counter to be %d, got %d", COUNTER_LIMIT, routine.Context.counter)
	}
}
