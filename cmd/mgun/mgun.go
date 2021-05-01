package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/imarsman/mgun/cmd/mgun/internal/build"
	"github.com/imarsman/mgun/cmd/mgun/internal/lib"
	"github.com/imarsman/mgun/cmd/mgun/internal/opt"
	yaml "gopkg.in/yaml.v2"
)

// Embed example config file in buiild for use in help output
// This requires Go 1.16 or above
//go:embed internal/assets/example.config.yaml
var readme string

func printHelp() {
	fmt.Println(readme)
}

func main() {
	var output string
	flag.StringVar(&output, "o", "", "output file name - optional")

	var file string
	flag.StringVar(&file, "f", "", "path to configuration yaml file - required")

	var help bool
	flag.BoolVar(&help, "h", false, "print usage")

	var sample bool
	flag.BoolVar(&sample, "s", false, "print sample")

	flag.Parse()

	// A simple function to print the build information
	var buildinfo = func() {
		fmt.Printf("Version........%-18s\n", build.Version)
		fmt.Printf("Platform.......%-18s\n", build.Platform)
		fmt.Printf("Architecture...%-18s\n", build.Architecture)
		fmt.Printf("Build..........%-18s\n", build.Build)
		fmt.Println("")
	}

	// A simple function to print command option information
	var usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

		flag.PrintDefaults()
	}

	// If help output was asked for print it and then exit
	if help == true {
		buildinfo()
		usage()
		os.Exit(0)
	}

	// If sample configuration was asked for, print it and exit
	if sample == true {
		printHelp()
		os.Exit(0)
	}

	// If no config file was specified, exit with error.
	if file == "" {
		buildinfo()
		fmt.Println("No config file name specified")
		usage()
		os.Exit(1)
	}

	// Check for existence of config file
	if _, err := os.Stat(file); os.IsNotExist(err) {
		fmt.Printf("Could not find config file %s\n", file)
		usage()
		os.Exit(1)
	}

	// If output was specified by -o parameter set the opt package Output value
	if output != "" {
		lib.SetOutput(output)
		opt.Output = output
	}

	bytes, err := ioutil.ReadFile(file)
	if err == nil {

		attack := lib.GetAttack()
		target := lib.NewTarget()
		callCollection := lib.GetCallCollection()
		reporter := lib.GetReporter()
		// Try to read in settings for the overall test to be run
		err := yaml.Unmarshal(bytes, attack)
		if err == nil {
			err = yaml.Unmarshal(bytes, target)
			if err == nil {
				// Try to read in settings for reporting
				err = yaml.Unmarshal(bytes, reporter)
				if err == nil {
					// If nothing was specified in command line output parameter
					// try for a value from config.
					if opt.Output == "" {
						opt.Output = reporter.Output
					}
					// Try to read in settings for headers, params, and requests
					err = yaml.Unmarshal(bytes, callCollection)
					if err == nil {
						attack.SetTarget(target)
						attack.SetGun(callCollection)
						attack.Prepare()
						attack.Start()
					} else {
						fmt.Println(err)
					}
				} else {
					fmt.Println(err)
				}
			} else {
				fmt.Println(err)
			}
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Println(err)
	}
}
