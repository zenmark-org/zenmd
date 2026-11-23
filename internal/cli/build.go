package cli

// BuildKind says which zenmd this is: "" for the app's zenmd (the Zenmark
// binary with the command line inside), "command line" for cmd/zenmd, which has
// no app. cmd/zenmd sets it before Run; VersionLine shows it as
// "zenmd 0.8.0 (command line)".
//
// The spec's Interfaces call this `cli.Build`, but that name is the type of
// `zenmd build`'s options, so the variable is BuildKind.
var BuildKind = ""

// buildSuffix is what VersionLine appends for a build kind: " (command line)",
// nothing for the app.
func buildSuffix() string {
	if BuildKind == "" {
		return ""
	}
	return " (" + BuildKind + ")"
}
