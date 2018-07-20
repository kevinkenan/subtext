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

	makedoc := commands.Make()
	build := commands.Build()
	walk := commands.Walk()

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
