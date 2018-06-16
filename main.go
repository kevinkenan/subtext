package main

import (
	"bufio"
	"fmt"
	"github.com/kevinkenan/subtext/document"
	"github.com/kevinkenan/subtext/macros"
	"io"
	"io/ioutil"
	"os"
	// "strings"
	// "text/template"
	"github.com/kevinkenan/subtext/parse"
	// "github.com/kevinkenan/subtext/macros"
	// "github.com/kevinkenan/subtext/commands"
	// "github.com/kevinkenan/subtext/verbose"
	"github.com/kevinkenan/cobra"
)

// func render(root *parse.Section, d *doc) string {
// 	s := new(strings.Builder)
// 	// c := make(chan parse.Node)
// 	// go root.Walk(c, func(n *parse.NodeList){w.WriteString(fmt.Sprintf("<here%s|", n.GetValue()))})
// 	s.WriteString("<body>")
// 	s.WriteString(renderNode(root, d))
// 	s.WriteString("</body>")
// 	return s.String()
// }

// func renderNode(n parse.Node, d *doc) string {
// 	s := new(strings.Builder)
// 	switch n.(type) {
// 	case *parse.Section:
// 		s.WriteString(renderSection(n.(*parse.Section), d))
// 	case *parse.Text:
// 		s.WriteString(n.(*parse.Text).GetText())
// 	case *parse.ErrorNode:
// 		s.WriteString(n.(*parse.ErrorNode).GetErrorMsg())
// 	case *parse.Cmd:
// 		s.WriteString(processMacro(n.(*parse.Cmd), d))
// 	case *parse.Paragraph:
// 		s.WriteString("</p>\n<p>")
// 	default:
// 		panic(fmt.Sprintf("unexpected node %q\n", n))
// 	}
// 	return s.String()
// }

// func renderSection(n *parse.Section, d *doc) string {
// 	s := new(strings.Builder)
// 	for _, l := range n.NodeList {
// 		s.WriteString(renderNode(l, d))
// 	}
// 	return s.String()
// }

// func renderNodeList(n parse.NodeList, d *doc) string {
// 	s := new(strings.Builder)
// 	for _, l := range n {
// 		s.WriteString(renderNode(l, d))
// 	}
// 	return s.String()
// }

// func processMacro(n *parse.Cmd, d *doc) string {
// 	// s := new(strings.Builder)
// 	name := n.GetCmdName()
// 	m := d.macros[name]
// 	args, err := m.ValidateArgs(n)
// 	renArgs := map[string]string{}
// 	for k, v := range args {
// 		renArgs[k] = renderNodeList(v, d)
// 	}
// 	s, err := m.Execute(renArgs)
// 	if err != nil {
// 		panic(fmt.Sprintf("%s: %s", name, err))
// 	}
// 	return s
// }

// const test_temp = "Name: •sc{ {{- .Name -}} }\nAge: {{.Age}}\nHometown: {{.Hometown}}\n"

// type test_t struct {
// 	Name, Age, Hometown string
// }

// func runTempTest() {
// 	// vals := test_t{Name: "Kevin Kenan", Age: "22", Hometown: "Eugene",}
// 	val2 := map[string]string{"Name": "Kevin Kenan", "Age": "22", "Hometown": "Eugene",}
// 	t := template.Must(template.New("test_temp").Parse(test_temp))

// 	err := t.Execute(os.Stdout, val2)
// 	if err != nil {
// 		fmt.Println("executing template:", err)
// 	}
// }

// type doc struct {
// 	macros map[string]*macros.Macro
// }

// func NewDoc() *doc {
// 	var m *macros.Macro
// 	// var opt macros.Optional
// 	// macs := make(map[string]*macros.Macro)
// 	opt := macros.Optional{Name: "second", Default: "def"}
// 	d := doc{make(map[string]*macros.Macro)}
// 	m = macros.NewMacro("a", "<it>{{- .first -}}</it>{{.second}}", []string{"first"}, []*macros.Optional{&opt})
// 	d.macros[m.Name] = m
// 	m = macros.NewMacro("b", "<b>{{.first}}</b>", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macros[m.Name] = m
// 	m = macros.NewMacro("c", "<sc>{{.first}}</sc>", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macros[m.Name] = m
// 	return &d
// }

func AppMain(c *cobra.Command, s []string) error {
	cobra.Out("subtext says hello")
	return nil
}

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
			input = in
		}
	}
	d := document.NewDoc()
	d.Name = name
	// d.ParagraphMode = viper.GetBool("paragraph_mode")
	// if viper.GetBool("paragraph_mode") {
	// 	d.AddParagraphMacros()
	// }
	d.Output = cobra.GetString("output")
	d.Packages = cobra.GetStringSlice("packages")
	d.Plain = cobra.GetBool("plain")
	d.Text = string(input)
	d.AddMacro(macros.NewMacro("paragraph.begin", "<p>", []string{"orig"}, nil))
	d.AddMacro(macros.NewMacro("paragraph.end", "</p>\n\n", []string{"orig"}, nil))
	d.AddMacro(macros.NewMacro("title", "<h1>{{.text}}</h1>\n\n", []string{"text"}, nil))
	d.AddMacro(macros.NewMacro("section", "<h2>{{.text}}</h2>\n\n", []string{"text"}, nil))
	d.AddMacro(macros.NewMacro("emph", "<i>{{.text}}</i>", []string{"text"}, nil))
	d.AddMacro(macros.NewMacro("chapter", "<chapter>¶+{{.text}}</chapter>", []string{"text"}, nil))
	output, err := d.Make()
	if err != nil {
		return err
	}
	cobra.Log("make complete")
	fmt.Print(output)
	// for _, pkg := range d.Packages {
	// 	fmt.Println(pkg)
	// }
	return nil
}

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
	d := document.NewDoc()
	d.Name = name
	// d.ParagraphMode = viper.GetBool("paragraph_mode")
	// if viper.GetBool("paragraph_mode") {
	// 	d.AddParagraphMacros()
	// }
	d.Output = cobra.GetString("output")
	d.Packages = cobra.GetStringSlice("packages")
	d.Text = string(input)
	d.AddMacro(macros.NewMacro("paragraph.begin", "<p>", []string{"orig"}, nil))
	d.AddMacro(macros.NewMacro("paragraph.end", "</p>\n\n", []string{"orig"}, nil))
	d.AddMacro(macros.NewMacro("title", "<h1>{{.text}}</h1>", []string{"text"}, nil))
	d.AddMacro(macros.NewMacro("section", "<h2>{{.text}}</h2>", []string{"text"}, nil))
	root, _ := parse.Parse(name, string(input))

	c := make(chan parse.Node)
	go root.Walk(c)
	// fmt.Printf(":: %s\n", root.String())
	fmt.Printf("> Root Section Node: contains %d nodes\n", len(root.NodeList))
	for n := range c {
		switch n.(type) {
		case *parse.Text:
			fmt.Printf("> Text Node: %q\n", n.(*parse.Text).NodeValue)
		case *parse.Section:
			fmt.Printf("> Section Node: contains %d nodes\n", len(n.(*parse.Section).NodeList))
		case *parse.ErrorNode:
			fmt.Printf("> Error: %q\n", n.(*parse.ErrorNode).NodeValue)
		case *parse.Cmd:
			fmt.Printf("> Cmd Node: %q\n", n.(*parse.Cmd).NodeValue)
			fmt.Printf("     Count: %d nodes\n", n.(*parse.Cmd).Count())
			fmt.Print("     Flags: <")
			for _, f := range n.(*parse.Cmd).Flags {
				fmt.Printf("%s", f)
			}
			fmt.Println(">")
			fmt.Printf("     Anonymous: %t\n", n.(*parse.Cmd).Anonymous)
			if n.(*parse.Cmd).Anonymous {
				for i, nl := range n.(*parse.Cmd).ArgList {
					fmt.Printf("     Text Block %d:\n", i)
					for _, nn := range nl {
						fmt.Printf("       %q\n", nn)
					}
				}
			} else {
				if len(n.(*parse.Cmd).ArgMap) > 0 {
					for k, v := range n.(*parse.Cmd).ArgMap {
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

func init() {
	defer func() {
		if err := recover(); err != nil {
			if cerr, ok := err.(cobra.SerpentError); ok {
				fmt.Println(cerr)
				os.Exit(1)
			}
			panic(err)
		}
	}()

	cfg := cobra.NewConfig("subtext")
	cfg.UseEnvVariables = true
	cfg.EnvVarPrefix = "subtext"
	cfg.SetDefault("dflt", "set_default")

	app := cobra.NewApp("subtext")
	app.Short = "a text processor"
	app.Long = "A text processor which utilizes macros and Go templates."
	app.Version = "0.0.1"
	app.RunE = AppMain

	make := cobra.NewCommand("make")
	make.Short = "create a new document"
	make.RunE = MakeCmd
	make.AddFlags(
		cobra.NewStringFlag("output", cobra.Opts().Abbr("o").Default("-").Desc("filesystem path of the output file")),
		cobra.NewBoolFlag("plain", cobra.Opts().Default(false).Desc("process the text in plain mode")),
		cobra.NewStringSliceFlag("packages", cobra.Opts().Abbr("p").Desc("macro package(s) to apply to input")))

	walk := cobra.NewCommand("walk")
	walk.Short = "walk the parse tree and print info about each node"
	walk.Run = WalkCmd

	// command structure
	root := cobra.Init(app, cfg)
	root.SubCmds(make, walk)

	cobra.OnInitialize(subtextInit)
}

func subtextInit() {
	cobra.Log("hello, subtext is starting")
}

func main() {
	cobra.Execute()
	cobra.Logf("subtext is done, goodbye\n")
	cobra.ShutDown()

	// commands.Execute()
	// verbose.LogAll("subtext is done, goodbye")
	// verbose.CloseVerboseLog()

	// err := commands.Execute()
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// // runTempTest()
	// d := NewDoc()
	// // input := "text •a{one} more text"
	// input := "text •a{one •b{two\n\n•c{three four}}} more text"
	// // input := "text •macro{ one •macro2{two} three} four"
	// root, err := parse.Parse(input)
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Println(render(root, d))
	// }
}
