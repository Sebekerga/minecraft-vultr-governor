package routines

import (
	"fmt"
	"testing"
)

type EmptyContext struct{}

func StepA(ctx *EmptyContext) (RoutineFunc[EmptyContext], error) {
	fmt.Println("╰-Step A")
	return nil, nil
}

func TestSimpleRoutine(t *testing.T) {

	routine := InitRoutine(StepA, EmptyContext{})
	fmt.Println("Running simple routine")
	err := routine.Run()
	if err != nil {
		t.Error(err)
	}
}

func AdvancedStepA(ctx *EmptyContext) (RoutineFunc[EmptyContext], error) {
	fmt.Println("╰-Advanced Step A")
	return AdvancedStepB, nil
}

func AdvancedStepB(ctx *EmptyContext) (RoutineFunc[EmptyContext], error) {
	fmt.Println("╰-Advanced Step B")
	return nil, nil
}

func TestAdvancedRoutine(t *testing.T) {
	routine := InitRoutine(AdvancedStepA, EmptyContext{})
	fmt.Println("Running advanced routine")
	err := routine.Run()
	if err != nil {
		t.Error(err)
	}
}

func WithErrorStepA(ctx *EmptyContext) (RoutineFunc[EmptyContext], error) {
	fmt.Println("╰-With Error Step A")
	return WithErrorStepB, nil
}

func WithErrorStepB(ctx *EmptyContext) (RoutineFunc[EmptyContext], error) {
	fmt.Println("╰-With Error Step B (should fail)")
	return nil, fmt.Errorf("Error in step B")
}

func TestRoutineWithError(t *testing.T) {
	routine := InitRoutine(WithErrorStepA, EmptyContext{})
	fmt.Println("Running routine with error")
	err := routine.Run()
	if err == nil {
		t.Error("Expected error, got nil")
	}

}

const COUNTER_LIMIT = 5

type ContextWithValue struct {
	counter int16
}

func WithContextLoopStep(ctx *ContextWithValue) (RoutineFunc[ContextWithValue], error) {
	fmt.Println("╰-With Context Step A for counter: ", ctx.counter)
	ctx.counter++
	fmt.Println("  ╰-Counter is now: ", ctx.counter)
	if ctx.counter >= COUNTER_LIMIT {
		return nil, nil
	}

	return WithContextLoopStep, nil
}

func TestRoutineWithContext(t *testing.T) {
	routine := InitRoutine(WithContextLoopStep, ContextWithValue{counter: 0})
	fmt.Println("Running routine with context")
	err := routine.Run()
	if err != nil {
		t.Error(err)
	}
	if routine.Context.counter != COUNTER_LIMIT {
		t.Errorf("Expected counter to be %d, got %d", COUNTER_LIMIT, routine.Context.counter)
	}
}
