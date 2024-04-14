package emojiprogressbar

const PASSED = "🟩"
const CURRENT = "🟦"
const PENDING = "⬜"

type ProgressBar struct {
	passed_symbol  string
	current_symbol string
	pending_symbol string
	reversed       bool
}

func NewProgressBar() ProgressBar {
	return ProgressBar{
		passed_symbol:  PASSED,
		current_symbol: CURRENT,
		pending_symbol: PENDING,
		reversed:       false,
	}
}

func (pb *ProgressBar) SetPassedSymbol(symbol string) {
	pb.passed_symbol = symbol
}

func (pb *ProgressBar) SetCurrentSymbol(symbol string) {
	pb.current_symbol = symbol
}

func (pb *ProgressBar) SetPendingSymbol(symbol string) {
	pb.pending_symbol = symbol
}

func (pb *ProgressBar) SetReversed(reversed bool) {
	pb.reversed = reversed
}

// `emoji_progress_bar` returns a string with a progress bar using emojis.
func (pb *ProgressBar) BuildProgressBar(total, current int, current_symbol string) string {

	var current_symbol_selected string
	if current_symbol == "" {
		current_symbol_selected = pb.current_symbol
	} else {
		current_symbol_selected = current_symbol
	}

	var bar string
	for i := 0; i < total; i++ {
		if i < current-1 {
			if pb.reversed {
				bar = pb.passed_symbol + bar
				continue
			}
			bar += pb.passed_symbol
		} else if i == current-1 {
			if pb.reversed {
				bar = current_symbol_selected + bar
				continue
			}
			bar += current_symbol_selected
		} else {
			if pb.reversed {
				bar = pb.pending_symbol + bar
				continue
			}
			bar += pb.pending_symbol
		}
	}
	return bar + " " + current_symbol
}
