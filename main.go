package main

import (
	"fmt"
	"os"

	"github.com/kevinkenan/cobra"
	"github.com/kevinkenan/subtext/commands"
)

func AppMain(c *cobra.Command, s []string) error {
	cobra.Out("subtext says hello")
	return nil
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

	makedoc := cobra.NewCommand("make")
	makedoc.Short = "create a new document"
	makedoc.RunE = commands.MakeCmd
	makedoc.AddFlags(
		cobra.NewStringFlag("output", cobra.Opts().Abbr("o").Default("-").Desc("path to the output file")),
		cobra.NewBoolFlag("plain", cobra.Opts().Default(false).Desc("process the text in plain mode")),
		cobra.NewBoolFlag("reflow", cobra.Opts().Default(false).Desc("reflow paragraphs")),
		cobra.NewStringFlag("format", cobra.Opts().Desc("the output format")),
		cobra.NewStringSliceFlag("packages", cobra.Opts().Abbr("p").Desc("macro package(s) to apply to input")),
		cobra.NewBoolFlag("default-warnings", cobra.Opts().Default(false).Desc("warn when a default macro is used")))

	build := cobra.NewCommand("build")
	build.Short = "create a site"
	build.Long = `Copies the contents from the specified to directory to the output directory,
processing subtext files as it goes.
`
	build.RunE = commands.Build
	build.AddFlags(
		cobra.NewStringFlag("output", cobra.Opts().Abbr("o").Req(true).Desc("path to the output directory")),
		cobra.NewBoolFlag("recurse", cobra.Opts().Default(false).Desc("includes contents of subdirectories")),
		cobra.NewBoolFlag("reflow", cobra.Opts().Default(false).Desc("reflow paragraphs")),
		cobra.NewStringFlag("format", cobra.Opts().Desc("the output format")),
		cobra.NewStringSliceFlag("packages", cobra.Opts().Abbr("p").Desc("macro package(s) to apply to input")))

	walk := cobra.NewCommand("walk")
	walk.Short = "walk the parse tree and print info about each node"
	walk.Run = commands.WalkCmd

	// command structure
	root := cobra.Init(app, cfg)
	root.SubCmds(makedoc, walk, build)

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
	// root, err := core.Parse(input)
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Println(render(root, d))
	// }
}
