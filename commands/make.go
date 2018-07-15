package commands

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/kevinkenan/cobra"
	"github.com/kevinkenan/subtext/core"
)

func MakeCmd(cmd *cobra.Command, args []string) error {
	cobra.Log("beginning make cmd")
	cmd.SilenceUsage = true
	name := ""
	var input []byte

	if len(args) == 0 || args[0] == "-" {
		cobra.WithField("args", args).Log("reading stdin")
		name = "<stdin>"
		reader := bufio.NewReader(os.Stdin)

		for {
			in, err := reader.ReadByte()

			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			input = append(input, in)
		}
	} else {
		cobra.WithField("files", args).Log("reading file")
		for _, f := range args {
			name = f
			in, err := ioutil.ReadFile(f)
			if err != nil {
				return err
			}
			input = append(input, in...)
		}
	}

	f := core.NewFolio()
	d := core.NewDoc(name, "<stdin>")
	d.Output = cobra.GetString("output")
	d.Packages = cobra.GetStringSlice("packages")
	d.Plain = cobra.GetBool("plain")
	d.Reflow = cobra.GetBool("reflow")
	d.Format = cobra.GetString("format")
	d.Text = string(input)
	f.Append(d)

	output, err := f.Make()
	if err != nil {
		return err
	}
	cobra.Log("folio make complete")

	if d.Output == "-" {
		fmt.Print(output)
	} else {
		f, err := os.Create(d.Output)
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
