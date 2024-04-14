package routines

// Routine function that either returns handler for next step or nil if finished.
// If error occurred during execution, it should be returned.
type RoutineFunc[C any] func(*C) (RoutineFunc[C], error)

// Routine is a simple routine executor.
type Routine[C any] struct {
	QueuedFunction RoutineFunc[C]
	Context        C
}

// InitRoutine initializes a new routine with entry function and starting context.
func InitRoutine[C any](entryFunction RoutineFunc[C], starting_context C) Routine[C] {
	routine := Routine[C]{QueuedFunction: entryFunction, Context: starting_context}
	return routine
}

// Finished returns true if routine has no more steps to execute.
func (r *Routine[C]) Finished() bool {
	return r.QueuedFunction == nil
}

// Step executes the next step in the routine.
func (r *Routine[C]) Step() error {
	next_func, err := r.QueuedFunction(&r.Context)
	if err != nil {
		return err
	}

	r.QueuedFunction = next_func
	return nil
}

// Run executes the routine until it's finished or error occurs.
func (r *Routine[C]) Run() error {
	for !r.Finished() {
		err := r.Step()
		if err != nil {
			return err
		}
	}

	return nil
}
