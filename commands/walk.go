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

func WalkCmd(cmd *cobra.Command, args []string) {
	cobra.Log("beginning walk cmd")
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
				return
			}
			input = append(input, in)
		}
	} else {
		cobra.WithField("files", args).Log("reading file")
		for _, f := range args {
			name = f
			in, err := ioutil.ReadFile(f)
			if err != nil {
				return
			}
			input = in
		}
	}

	f := core.NewFolio()
	d := core.NewDoc(name, "<stdin>")
	// d.ParagraphMode = viper.GetBool("paragraph_mode")
	// if viper.GetBool("paragraph_mode") {
	// 	d.AddParagraphMacros()
	// }
	d.Output = cobra.GetString("output")
	d.Packages = cobra.GetStringSlice("packages")
	d.Text = string(input)
	f.Append(d)
	// d.AddMacro(core.NewMacro("paragraph.begin", "<p>", []string{"orig"}, nil))
	// d.AddMacro(core.NewMacro("paragraph.end", "</p>\n\n", []string{"orig"}, nil))
	// d.AddMacro(core.NewMacro("title", "<h1>{{.text}}</h1>", []string{"text"}, nil))
	// d.AddMacro(core.NewMacro("section", "<h2>{{.text}}</h2>", []string{"text"}, nil))

	root, _ := core.Parse(d)
	c := make(chan core.Node)
	go root.Walk(c)
	// fmt.Printf(":: %s\n", root.String())
	fmt.Printf("> Root Section Node: contains %d nodes\n", len(root.NodeList))
	for n := range c {
		switch n.(type) {
		case *core.Text:
			fmt.Printf("> Text Node: %q\n", n.(*core.Text).NodeValue)
		case *core.Section:
			fmt.Printf("> Section Node: contains %d nodes\n", len(n.(*core.Section).NodeList))
		case *core.ErrorNode:
			fmt.Printf("> Error: %q\n", n.(*core.ErrorNode).NodeValue)
		case *core.Cmd:
			fmt.Printf("> Cmd Node: %q\n", n.(*core.Cmd).NodeValue)
			fmt.Printf("     Count: %d nodes\n", n.(*core.Cmd).Count())
			fmt.Print("     Flags: <")
			for _, f := range n.(*core.Cmd).Flags {
				fmt.Printf("%s", f)
			}
			fmt.Println(">")
			fmt.Printf("     Anonymous: %t\n", n.(*core.Cmd).Anonymous)
			if n.(*core.Cmd).Anonymous {
				for i, nl := range n.(*core.Cmd).ArgList {
					fmt.Printf("     Text Block %d:\n", i)
					for _, nn := range nl {
						fmt.Printf("       %q\n", nn)
					}
				}
			} else {
				if len(n.(*core.Cmd).ArgMap) > 0 {
					for k, v := range n.(*core.Cmd).ArgMap {
						fmt.Printf("     Argument %q: %s\n", k, v)
					}
				} else {
					fmt.Println("     Arguments: None")
				}
			}
		default:
			fmt.Printf("> UNEXPECTED Node: %q\n", n.String())
			fmt.Printf("     Type Code: %d\n", n.Typeof())
		}
	}
}
