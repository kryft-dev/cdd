package action

// Opener returns the program that opens a path or URL in the desktop's
// default application on goos (a runtime.GOOS value): "open" on macOS,
// "xdg-open" everywhere else. Built-in Actions that hand something to the
// desktop use it, so the platform choice lives in one place.
func Opener(goos string) string {
	if goos == "darwin" {
		return "open"
	}
	return "xdg-open"
}
