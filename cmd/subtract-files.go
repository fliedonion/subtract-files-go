package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/safeexec"
)

const (
	optionnameTarget        = "target"
	optionnameMinus         = "minus"
	optionnameDryRun        = "dryrun"
	optionnameRclone        = "rclone"
	optionDescriptionTarget = "search and delete target (delete match files from this directory)"
	optionDescriptionMinus  = "search source (read only)"
	optionDescriptionDryRun = "execute but only report match files. no files will be removed with this option."
	optionDescriptionRclone = "path for rclone executable. if this parameter is specified, \nonly this path will be used to search for rclone."
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

// interface for wrap os.Stat to make testable.
type statInterface func(string) (fs.FileInfo, error)

func checkRclonePathOpt(rclonePathOpt string, statfunc statInterface) (string, error) {
	if rclonePathOpt == "" {
		return "", fmt.Errorf("path is empty")
	}
	if fi, err := statfunc(rclonePathOpt); err == nil {
		if fi.IsDir() {
			return "", fmt.Errorf("specified rclone path is directory")
		} else {
			if strings.ToUpper(filepath.Base(rclonePathOpt)[0:5]) != "RCLONE" {
				return "", fmt.Errorf("specified rclone path is not starts with rclone")
			}

			if filepath.IsAbs(rclonePathOpt) {
				return filepath.Clean(rclonePathOpt), nil
			}

			if abs, err := filepath.Abs(rclonePathOpt); err != nil {
				return "", err
			} else {
				return filepath.Clean(abs), nil
			}
		}
	} else {
		return "", fmt.Errorf("specified rclone path is invalid")
	}
}

func findRclone(rclonePathOpt string, statfunc statInterface) string {

	// if rclonePathOption is not specified:

	path := os.Getenv("PATH")
	if path != "" && path[len(path)-1] != os.PathListSeparator {
		path = path + string(os.PathListSeparator)
	}
	os.Setenv("PATH", path+filepath.Dir(os.Args[0]))

	s, err := safeexec.LookPath("rclone")
	if err != nil {
		log.Println("rclone not found.")
		log.Fatal(err)
	}
	return s
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s -%s [dir to delete files] -%s [dir to compare]\n\n", os.Args[0], optionnameTarget, optionnameMinus)
	flag.Usage()
}

func main() {

	target := flag.String(optionnameTarget, "", optionDescriptionTarget)
	minus := flag.String(optionnameMinus, "", optionDescriptionMinus)
	dryrun := flag.Bool(optionnameDryRun, false, optionDescriptionDryRun)
	rclonePathOpt := flag.String(optionnameRclone, "", optionDescriptionRclone)

	flag.Parse()

	// required parameters check
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

	// find rclone

	rclonePath := ""
	if *rclonePathOpt != "" {
		if result, err := checkRclonePathOpt(*rclonePathOpt, os.Stat); err != nil {
			log.Printf("-%s parameter is invalid\n", optionnameRclone)
			log.Fatal(err)
		} else {
			rclonePath = result
		}
	} else {
		rclonePath = findRclone(*rclonePathOpt, os.Stat)
	}

	if rclonePath == "" {
		log.Fatal("rclone not found")
	}

	// execute rclone

	args := []string{`check`, *target, *minus, `--match=-`, `--config=dummy-rclone.conf`}
	// log.Println(args)
	cmd := exec.Command(rclonePath, args...)

	var bss bytes.Buffer
	var bes bytes.Buffer
	cmd.Stdout = &bss
	cmd.Stderr = &bes

	cmdErr := cmd.Run()
	// log.Println(bes.String())

	// read result.
	log.Println("Target: " + filepath.Clean(*target))
	log.Println("Minus : " + filepath.Clean(*minus))

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

	// remove same files.
	log.Println("Start remove samefiles...")
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

			if *dryrun {
				log.Printf(" %d same files found (not removed).\n", len(removeTargets))
			} else {
				log.Printf(" %d same files removed.\n", len(removeTargets))
			}

		} else {
			log.Println("No matching files were found.")
		}

	} else {
		log.Println("Error: While executing rclone")
		log.Println(cmdErr)
	}
}
