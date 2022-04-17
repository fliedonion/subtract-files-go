package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	optionnameTarget        = "target"
	optionnameMinus         = "minus"
	optionnameDryRun        = "dryrun"
	optionDescriptionTarget = "delete target (delete match files from this directory)"
	optionDescriptionMinus  = "minus source (read only)"
	optionDescriptionDryRun = "execute but only report match files. no files will be removed with this option."
)

func readSingleLine(r io.Reader) (string, error) {
	in := bufio.NewScanner(r)
	if in.Scan() {
		return in.Text(), nil
	} else {
		err := in.Err()
		if err != nil {
			return "", err
		} else {
			return "", fmt.Errorf("bufio.Scanner scan error")
		}
	}
}

func toAbsDirString(path string) (string, error) {
	pathstr := filepath.ToSlash(path)
	if !filepath.IsAbs(pathstr) {
		// pathstr, err := filepath.Abs(pathstr) // errのために :=してると pathstrも新しい変数になるので外に引き継げない。
		absdir, err := filepath.Abs(pathstr)
		if err != nil {
			return "", err
		}
		pathstr = absdir
	}
	pathstr = filepath.Clean(pathstr)
	return pathstr, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s -%s [dir to delete files] -%s [dir to compare]", os.Args[0], optionnameTarget, optionnameMinus)
}

func main() {

	target := flag.String(optionnameTarget, "", optionDescriptionTarget)
	minus := flag.String(optionnameMinus, "", optionDescriptionMinus)
	dryrun := flag.Bool(optionnameDryRun, false, optionDescriptionDryRun)

	flag.Parse()

	if *target == "" {
		usage()
		log.Fatal("no target to delete file.")
	}
	basedir, err := toAbsDirString(*target)
	if err != nil {
		log.Fatal("can't get Abstrcat path from target")
	}

	if *minus == "" {
		usage()
		log.Fatal("no comparison source ( minus source ).")
	}
	_mindir, err := toAbsDirString(*minus)
	if err != nil {
		log.Fatal("can't get Abstrcat path from minus")
	}
	if basedir == _mindir {
		log.Fatal("same directory specified. target is minus.")
	}

	args := []string{`check`, *target, *minus, `--match=-`, `--config=dummy-rclone.conf`}
	// log.Println(args)
	cmd := exec.Command(`.\rclone.exe`, args...)

	var bss bytes.Buffer
	var bes bytes.Buffer
	cmd.Stdout = &bss
	cmd.Stderr = &bes

	cmdErr := cmd.Run()
	// log.Println(bes.String())

	log.Println("Target: " + *target)
	log.Println("Minus : " + *minus)

	log.Println("If a file with the same relative path and the same contents is found ")
	log.Println("in the Minus directory, it is removed from Target.")
	log.Println("")
	log.Println("[Notice] This command may remove file without any backup.")
	log.Println("Input 'ok' and hit enter if you want to continue.")
	if *dryrun {
		log.Println("DryRun mode. Even if you enter 'ok', NO files will remove.")
	}
	fmt.Print("Please input answer :")

	if ans, err := readSingleLine(os.Stdin); err != nil {
		log.Println("Input error.")
		log.Println(err)
		os.Exit(1)
	} else {
		if ans != "ok" {
			log.Println("Your answer is not ok, exit.")
			os.Exit(0)
		}
		log.Printf("Your answer is %s, continue.\n", ans)
	}
	log.Println("")
	log.Println("Start operation.")
	if _, ok := cmdErr.(*exec.ExitError); ok {
		// たぶんファイル内容が全ファイル一致しないと Exit Code が 1 なので、ExitErrorは無視する。
		// log.Printf("Exit Code, %d\n", exitError.ExitCode())

		r := strings.NewReader(bss.String())
		in := bufio.NewScanner(r)
		removeTargets := []string{}

		for in.Scan() {
			removeTargets = append(removeTargets, in.Text())
		}
		if err := in.Err(); err != nil {
			log.Println("Error while reading rclone result.")
			log.Fatal(err)
		}

		if len(removeTargets) > 0 {
			log.Printf("Number of same files = %d\n", len(removeTargets))
			if *dryrun {
				log.Println("Listing...(dryrun mode)")
			} else {
				log.Println("Removing...")
			}

			for _, c := range removeTargets {
				c = filepath.Join(basedir, c)
				if *dryrun {
					log.Printf(" > %s", c)
				} else {
					log.Printf(" > %s ", c)
					os.Remove(c)
				}
			}

			log.Printf(" %d same files\n", len(removeTargets))

		} else {
			log.Println("No matching files were found.")
		}

	} else {
		log.Println("Error: While executing rclone")
		log.Println(cmdErr)
	}
}
