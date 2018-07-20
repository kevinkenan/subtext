package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kevinkenan/cobra"
	"github.com/kevinkenan/subtext/core"
)

func Make() (cmd *cobra.Command) {
	cmd = cobra.NewCommand("make")
	cmd.Short = "create a new document"
	cmd.RunE = MakeRunE
	cmd.AddFlags(
		cobra.NewStringFlag("output", cobra.Opts().Abbr("o").Default("-").Desc("path to the output file")),
		cobra.NewBoolFlag("plain", cobra.Opts().Default(false).Desc("process the text in plain mode")),
		cobra.NewBoolFlag("reflow", cobra.Opts().Default(false).Desc("reflow paragraphs")),
		cobra.NewStringFlag("format", cobra.Opts().Desc("the output format")),
		cobra.NewStringSliceFlag("packages", cobra.Opts().Abbr("p").Desc("macro package(s) to apply to input")),
		cobra.NewStringSliceFlag("package-dir", cobra.Opts().Desc("path to a package directory. you may set this multiple times")),
		cobra.NewBoolFlag("default-warnings", cobra.Opts().Default(false).Desc("warn when a default macro is used")))

	return
}

func MakeRunE(cmd *cobra.Command, args []string) error {
	cobra.Log("beginning make cmd")
	var err error
	cmd.SilenceUsage = true
	var name, path string
	f := core.NewFolio()
	f.Cmd = cmd

	for _, pdir := range cobra.GetStringSlice("package-dir") {
		path = filepath.Clean(pdir)
		f.PkgSearchPaths = append(f.PkgSearchPaths, path)
	}

	switch {
	case len(args) > 1:
		return fmt.Errorf("make requires zero or one file")
	case len(args) == 0, args[0] == "-":
		cobra.WithField("args", args).Log("reading stdin")

		// name = "<stdin>"
		d := core.NewDoc("<stdin>", "<stdin>")
		if err := f.AppendDoc(d); err != nil {
			return err
		}
	case len(args) == 1:
		cobra.WithField("files", args).Log("reading file")
		name = args[0]
		path = filepath.Clean(name)

		d := core.NewDoc(name, path)
		if err := f.AppendDoc(d); err != nil {
			return err
		}
		// input = append(input, in...)
	}

	// d := core.NewDoc(name, "<stdin>")
	OutputName := cobra.GetString("output")

	f.Packages = cobra.GetStringSlice("packages")
	if len(f.Packages) > 0 {
		err = f.LoadPackages(f.Packages)
		if err != nil {
			return err
		}
	}

	output, err := f.Make()
	if err != nil {
		return err
	}
	cobra.Log("folio make complete")

	if OutputName == "-" {
		fmt.Print(output)
	} else {
		f, err := os.Create(OutputName)
		if err != nil {
			return err
		}
		defer f.Close()

		f.WriteString(output)
		f.Sync()
	}
	// for _, pkg := range d.Packages {
	// 	fmt.Println(pkg)
	// }
	return nil
}
