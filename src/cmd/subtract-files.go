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
	optionDescriptionTarget = "delete target (delete match files from this directory)"
	optionDescriptionMinus  = "minus source (read only)"
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

func usage() {
	fmt.Fprintf(os.Stderr, "%s -%s [dir to delete files] -%s [dir to compare]", os.Args[0], optionnameTarget, optionnameMinus)
}

func main() {

	target := flag.String(optionnameTarget, "", optionDescriptionTarget)
	minus := flag.String(optionnameMinus, "", optionDescriptionMinus)

	flag.Parse()

	if *target == "" {
		usage()
		log.Fatal("no target to delete file.")
	}
	basedir := filepath.ToSlash(*target)
	if !filepath.IsAbs(basedir) {
		absdir, err := filepath.Abs(basedir)
		// basedir, err := filepath.ToSlash(*target) // errのためにこれにすると basedirもこのスコープ用のものが新設されてしまう。
		if err != nil {
			log.Fatal("can't get Abstrcat path from target")
		}
		basedir = absdir
	}

	if *minus == "" {
		usage()
		log.Fatal("no comparison source ( minus source ).")
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
			// fmt.Println(in.Text())
			removeTargets = append(removeTargets, in.Text())
		}
		if err := in.Err(); err != nil {
			log.Fatal(err)
		}

		if len(removeTargets) > 0 {
			log.Printf("Number of same files = %d\n", len(removeTargets))
			log.Println("Removing...")
			for _, c := range removeTargets {
				c = filepath.Join(basedir, c)
				log.Printf(" > %s ", c)
				os.Remove(c)
			}
		} else {
			log.Println("No matching files were found.")
		}

	} else {
		log.Println("Error: While executing rclone")
		log.Println(cmdErr)
	}
}
