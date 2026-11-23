package cliset

import (
	"errors"
	"io"

	"github.com/zenmark-org/zenmd/internal/cli"
)

// NoApp is the Launcher of the command-line build: opening a folder or a page
// needs the app, so it says so in one line (cli.Run prefixes "zenmd: ") and
// fails; it never starts a window.
func NoApp(cli.Launch) error {
	return errors.New("the Zenmark app isn't part of this build; zenmd export site <folder> makes a website")
}

// NoPDF is the PDFExporter of the command-line build: the PDF is made by the app on macOS.
func NoPDF(cli.ExportPDF, io.Writer, io.Writer) error {
	return errors.New("export pdf needs the Zenmark app on macOS")
}
