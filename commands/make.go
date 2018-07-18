package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kevinkenan/cobra"
	"github.com/kevinkenan/subtext/core"
)

func MakeCmd(cmd *cobra.Command, args []string) error {
	cobra.Log("beginning make cmd")
	var err error
	cmd.SilenceUsage = true
	var name, path string
	f := core.NewFolio()
	f.Cmd = cmd

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
	err = f.LoadPackages()
	if err != nil {
		return err
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
