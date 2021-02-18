package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/imarsman/mgun/lib"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	var output string
	flag.StringVar(&output, "o", "will not save if no value supplied", "output file name - optional")

	var file string
	flag.StringVar(&file, "f", "/path/to/config/file.yaml", "configuration yaml file - required")

	var help bool
	flag.BoolVar(&help, "h", false, "print usage")

	flag.Parse()

	var usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

		flag.PrintDefaults()
	}

	if help == true {
		usage()
		os.Exit(0)
	}

	if len(file) == 0 {
		fmt.Println("No config file name specified")
		usage()
		os.Exit(1)
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		fmt.Printf("Could not find config file %s\n", file)
		usage()
		os.Exit(1)
	}

	if output != "" {
		lib.SetOutput(output)
	}

	if len(file) > 0 {
		bytes, err := ioutil.ReadFile(file)
		if err == nil {

			kill := lib.GetKill()
			victim := lib.NewVictim()
			gun := lib.GetGun()
			reporter := lib.GetReporter()
			err := yaml.Unmarshal(bytes, kill)
			if err == nil {
				err = yaml.Unmarshal(bytes, victim)
				if err == nil {
					err = yaml.Unmarshal(bytes, reporter)
					if err == nil {
						err = yaml.Unmarshal(bytes, gun)
						if err == nil {
							kill.SetVictim(victim)
							kill.SetGun(gun)
							kill.Prepare()
							kill.Start()
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
	} else {
		fmt.Println("config file not found")
	}
}
