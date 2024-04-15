package routines

import (
	"log"
	"reflect"
	"runtime"
	"strings"
)

type PrintLevel int

const (
	INFO PrintLevel = iota
	ERROR
)

func (pl PrintLevel) String() string {
	return [...]string{"INFO", "ERROR"}[pl]
}

// PrintHandler is a function that can be used to handle messages from routines.
type PrintHandler = func(PrintLevel, string)

// Routine function that either returns handler for next step or nil if finished.
// If error occurred during execution, it should be returned.
type RoutineFunc[C any] func(*C, PrintHandler) (RoutineFunc[C], error)

// Routine is a simple routine executor.
type Routine[C any] struct {
	QueuedFunction RoutineFunc[C]
	Context        C
	PrintHandler   PrintHandler
}

// InitRoutine initializes a new routine with entry function and starting context.
func InitRoutine[C any](entryFunction RoutineFunc[C], startingContext C, printHandler PrintHandler) Routine[C] {
	routine := Routine[C]{
		QueuedFunction: entryFunction,
		Context:        startingContext,
		PrintHandler:   printHandler,
	}
	return routine
}

// Finished returns true if routine has no more steps to execute.
func (r *Routine[C]) Finished() bool {
	return r.QueuedFunction == nil
}

// Step executes the next step in the routine.
func (r *Routine[C]) Step() error {
	nextFunc, err := r.QueuedFunction(&r.Context, r.PrintHandler)
	if err != nil {
		return err
	}

	r.QueuedFunction = nextFunc
	return nil
}

// Run executes the routine until it's finished or error occurs.
func (r *Routine[C]) Run() error {
	for !r.Finished() {
		routineStepName := runtime.FuncForPC(reflect.ValueOf(r.QueuedFunction).Pointer()).Name()
		routineStepNameSlice := strings.Split(routineStepName, "/")
		routineStepName = routineStepNameSlice[len(routineStepNameSlice)-1]

		log.Printf("Running routine step %s", routineStepName)
		err := r.Step()
		if err != nil {
			return err
		}
	}

	return nil
}
