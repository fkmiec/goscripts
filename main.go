package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

/*
Command-line driven program to compile bitfield/script and other go code into unix-like command pipelines.

- compile one-line pipeline of script commands into an executable (Include this src folder on the PATH so the command will be immediately usable)
- default the name of the command to "gocmd"
- optionally specify a unique name for the command
- optionally specify additional imports (e.g. import os if you need to pass arguments to the resulting command)
- optionally specify a file for the code. If so, assume the file is the entire code, including main function and imports.
- provide an option to spit out a skeletal template for a file
- provide an option to save the source in the project under the command name (ie. for name=FindFiles, src file will be <project>/src/FindFiles.go)
- provide an option to list previously compiled commands
- provide an option to output the full path to the source file previously saved to the project (so can edit in your favorite code editor)
- provide an option to output the path to the project folder
- provide an option to execute or run the code after compilation
- support use of shebang to immediately execute the command inline. Shebang invokes the command and passes the filename of the script as the
	first argument. So, "#!/usr/bin/env -S goscripts -x -f" should effectively act the same as combining file and run flags on the command line. For the
	file option, always look for and strip out the shebang line if present.
*/

type Repl struct {
	Imports []string
	Code    string
}

var buf *bytes.Buffer

func readSourceFile(filename string) *bytes.Buffer {
	// Using bufio.Scanner to read line by line
	file, err := os.Open(filename)
	check(err)

	defer file.Close()

	scanner := bufio.NewScanner(file)
	var line string
	buf = bytes.NewBuffer([]byte{})
	for scanner.Scan() {

		line = scanner.Text()
		//strip out the shebang if present
		if strings.HasPrefix(line, "#!") {
			continue
		}
		_, err := buf.WriteString(line + "\n")
		check(err)
	}

	err = scanner.Err() //; err != nil {
	check(err)

	return buf
}

func processTemplate(dir string, repl Repl) *bytes.Buffer {
	var tmplFile = dir + "/script.tmpl"
	tmpl, err := template.New("script.tmpl").ParseFiles(tmplFile)
	check(err)

	buf = bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, repl)
	check(err)

	return buf
}

func writeFile(filename string, buf *bytes.Buffer) bool {
	// Open the file for writing, creates it if it doesn't exist, or truncates if it exists.
	file, err := os.Create(filename)
	check(err)

	// Ensure the file is closed after the function returns.
	defer file.Close()

	// Write the buffer to the file
	_, err = buf.WriteTo(file)
	check(err)

	return true
}

func getProjectPath() string {
	executablePath, err := os.Executable()
	check(err)

	executableDir := filepath.Dir(executablePath)
	return executableDir
}

func getCommandList(dir string) []string {
	cmds := []string{}
	list, err := os.ReadDir(dir)
	check(err)
	for _, entry := range list {
		if !entry.IsDir() {
			cmds = append(cmds, entry.Name())
		}
	}
	sort.Strings(cmds)
	return cmds
}

func checkFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(error, os.ErrNotExist)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	var name string
	var imports string
	var code string
	var inputFile string
	var saveSource bool
	var listCommands bool
	var path string
	var printDir bool
	var printTemplate bool
	var execCode bool

	flag.StringVar(&name, "name", "gocmd", "A name for your command. Defaults to gocmd.")
	flag.StringVar(&name, "n", "gocmd", "A name for your command. Defaults to gocmd.")
	flag.StringVar(&imports, "imports", "", "A comma-separated list of go packages to import. Not used with --file option.")
	flag.StringVar(&imports, "i", "", "A comma-separated list of go packages to import. Not used with --file option.")
	flag.StringVar(&code, "code", "", "The code of your command. Defaults to empty string.")
	flag.StringVar(&code, "c", "", "The code of your command. Defaults to empty string.")

	flag.StringVar(&inputFile, "file", "", "A go src file, complete with main function and imports. Alternative to --code and --imports options.")
	flag.StringVar(&inputFile, "f", "", "A go src file, complete with main function and imports. Alternative to --code and --imports options.")
	flag.BoolVar(&saveSource, "save", false, "Save the source file <name>.go to the project src folder.")
	flag.BoolVar(&saveSource, "s", false, "Save the source file <name>.go to the project src folder.")

	flag.StringVar(&path, "path", "", "Print the path to the source file specified, if exists in the project. Blank if not found.")
	flag.StringVar(&path, "p", "", "Print the path to the source file specified, if exists in the project. Blank if not found.")
	flag.BoolVar(&printDir, "dir", false, "Print the directory path to the project.")
	flag.BoolVar(&printDir, "d", false, "Print the directory path to the project.")
	flag.BoolVar(&printTemplate, "template", false, "Print a template go source file to stdout. After edits, use --file to compile with goscript.")
	flag.BoolVar(&printTemplate, "t", false, "Print a template go source file to stdout. After edits, use --file to compile with goscript.")
	flag.BoolVar(&listCommands, "list", false, "Print the list of previously-compiled commands.")
	flag.BoolVar(&listCommands, "l", false, "Print the list of previously-compiled commands.")

	flag.BoolVar(&execCode, "exec", false, "Execute the resulting binary.")
	flag.BoolVar(&execCode, "x", false, "Execute the resulting binary.")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "goscripts (see https://github.com/fkmiec/goscripts)\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --code|-c string\n\tThe code of your command. Defaults to empty string.")
		fmt.Fprintln(os.Stderr, "  --imports|-i string\n\tA comma-separated list of go packages to import. Not used with --file option.")
		fmt.Fprintln(os.Stderr, "  --file|-f string\n\tA go src file, complete with main function and imports. Alternative to --code and --imports options.")
		fmt.Fprintln(os.Stderr, "  --name|-n string\n\tA name for your command. Defaults to gocmd.")
		fmt.Fprintln(os.Stderr, "  --save|-s\n\tSave the source file <name>.go to the project src folder.")
		fmt.Fprintln(os.Stderr, "  --list|-l\n\tPrint the list of previously-compiled commands.")
		fmt.Fprintln(os.Stderr, "  --dir|-d\n\tPrint the directory path to the project.")
		fmt.Fprintln(os.Stderr, "  --path|-p\n\tPrint the path to the source file specified, if exists in the project. Blank if not found.")
		fmt.Fprintln(os.Stderr, "  --template|-t\n\tPrint a template go source file to stdout. After edits, use --file to compile with goscript.")
		fmt.Fprintln(os.Stderr, "  --exec|-x\n\tExecute the resulting binary.")
		fmt.Fprintln(os.Stderr, "\nExample (Compile with default name gocmd. Execute gocmd.):")
		fmt.Fprintf(os.Stderr, "  %s --code \"script.Echo(\\\" Hello World! \\\").Stdout()\";gocmd\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nExample shebang in 'myscript.go' file:")
		fmt.Fprintf(os.Stderr, "  (1) Add '#!/usr/bin/env -S %s' to the top of your go source file.\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "  (2) Set execute permission and type \"./myscript.go\" as you would with a shell script.\n")
	}

	flag.Parse()

	//Separately evaluate the args to support execution with args and shebang-based execution with no flags
	//Args beyond flags should only be present if executing, so it's (1) shebang, (2) --exec or (3) an error
	//Scenarios (Note any of these could be straight commandline and not shebang):
	// (1) #!/usr/bin/env -S goscript -x -f <filename> <optionally more args> (handled as normal)
	// (2) #!/usr/bin/env -S goscript -x <filename> <optionally more args> (need to determine if first non-flag arg is a filename, and if so, set inputFile=arg[0])
	// (3) #!/usr/bin/env -S goscript <filename> <optionally more args> (need to determine if first non-flag arg is a filename, and if so, set inputFile=arg[0] and execCode=true)
	var subprocessArgs []string
	if len(flag.Args()) > 0 {
		subprocessArgs = flag.Args()
		//check for --file flag. If not present, check if the first non-flag arg is a filename.
		if inputFile == "" {
			isFileExists := checkFileExists(flag.Arg(0))
			//If the first arg is a file that exists
			if isFileExists {
				inputFile = flag.Arg(0) //Account for scenario 2, above
				subprocessArgs = subprocessArgs[1:]
				if !execCode {
					execCode = true //Account for scenario 3, above.
				}
			}
		}
	}

	//Get the path of the executable, which we assume is the project folder.
	//NOTE: While it might typically make sense to install the binary in some other location,
	//  for this project that aims to compile other go code within the same project (in order
	//  to support modules, etc.), we either make this assumption or require a PATH_TO_GOSCRIPTS_PROJECT
	//  environment variable to make this work on the user's system.
	dir := getProjectPath()

	//Print the location of the project folder
	if printDir {
		fmt.Println(dir)
		return //Exit the program after printing the path
	}

	//Print the location of the source file, if it exists, otherwise blank
	if path != "" {
		srcFile := dir + "/src/" + path + ".go"
		isFileExists := checkFileExists(srcFile)
		if isFileExists {
			//print the source file path
			fmt.Println(srcFile)
		}
		return //Exit the program after printing the path
	}

	//List existing commands
	if listCommands {
		cmds := getCommandList(dir + "/bin")
		fmt.Println("Commands:")
		for _, cmd := range cmds {
			fmt.Printf("\t%s\n", cmd)
		}
		return //Exit the program after printing the list of commands
	}

	//Handle a regular go source file (potentially with a shebang (#!) at the top)
	if inputFile != "" {
		buf = readSourceFile(inputFile)

		//Handle typical one-liner code specified on command line
	} else if printTemplate || code != "" {
		//Default use case for this little script builder is the use of bitfield/script.
		//So, we try to include that import if not given explicitly.
		if strings.Contains(code, "script.") {
			if !strings.Contains(imports, "github.com/bitfield/script") {
				if len(imports) > 0 {
					imports += ",github.com/bitfield/script"
				} else {
					imports = "github.com/bitfield/script"
				}
			}
		}

		theImports := strings.Split(imports, ",")

		repl := Repl{
			Imports: theImports,
			Code:    code,
		}

		buf = processTemplate(dir, repl)

		//Helper code prints an empty template to give a starting point when creating an external file manually
		if printTemplate {
			_, err := buf.WriteTo(os.Stdout)
			check(err)
			return //exit the program after printing the template
		}
		//Handle compiling a pre-existing source file located in the project/src folder
	} else if name != "gocmd" {
		srcFilename := dir + "/src/" + name + ".go"
		buf = readSourceFile(srcFilename)
		//Print usage and exit
	} else {
		flag.Usage()
		os.Exit(1)
	}

	//Save source and compile binary
	srcFilename := dir + "/src/" + name + ".go"
	binFilename := dir + "/bin/" + name

	writeFile(srcFilename, buf)
	cmd := exec.Command("go", "build", "-o", binFilename, srcFilename)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}

	// If flag to run the code was given, then execute it
	// Pass stdin and any arguments to the sub-process. This will make shebang and general --exec work as expected.
	if execCode {
		if inputFile != "" {

			cmd := exec.Command(binFilename, subprocessArgs...)
			cmdStdin, err := cmd.StdinPipe()
			check(err)
			stdin := bufio.NewReader(os.Stdin)

			go func() {
				defer cmdStdin.Close()
				for {
					line, err := stdin.ReadString('\n')
					if err != nil || len(strings.TrimSpace(line)) == 0 {
						break
					}
					io.WriteString(cmdStdin, line)
				}
			}()

			out, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("%v: %s\n", err, out)
			} else {
				fmt.Print(string(out))
			}
		}
	}
}
