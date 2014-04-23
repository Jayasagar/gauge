// This file is part of twist
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/dmotylev/goproperties"
	"github.com/twist2/common"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	specsDirName      = "specs"
	skelFileName      = "hello_world.spec"
	envDefaultDirName = "default"
)

var availableSteps []*step
var acceptedExtensions = make(map[string]bool)

func init() {
	acceptedExtensions[".spec"] = true
	acceptedExtensions[".md"] = true
}

type manifest struct {
	Language string
}

// All the environment variables loaded from the
// current environments JSON files will live here
type environmentVariables struct {
	Variables map[string]string
}

func getProjectManifest() *manifest {
	projectRoot, err := common.GetProjectRoot()
	if err != nil {
		fmt.Printf("Failed to read manifest: %s \n", err.Error())
		os.Exit(1)
	}
	contents := common.ReadFileContents(path.Join(projectRoot, common.ManifestFile))
	dec := json.NewDecoder(strings.NewReader(contents))

	var m manifest
	for {
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("Failed to read manifest. %s\n", err.Error())
			// common.PrintError(fmt.Sprintf("Failed to read: %s. %s\n", manifestFile, err.Error()))
			os.Exit(1)
		}
	}

	return &m
}

func findScenarioFiles(fileChan chan<- string) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	walkFn := func(filePath string, info os.FileInfo, err error) error {
		ext := path.Ext(info.Name())
		if strings.ToLower(ext) == ".scn" {
			fileChan <- filePath
		}
		return nil
	}

	filepath.Walk(pwd, walkFn)
	fileChan <- "done"
}

func parseScenarioFiles(fileChan <-chan string) {
	for {
		scenarioFilePath := <-fileChan
		if scenarioFilePath == "done" {
			break
		}

		parser := new(specParser)
		//todo: parse concepts
		specification, result := parser.parse(common.ReadFileContents(scenarioFilePath), new(conceptDictionary))

		if result.ok {
			availableSteps = append(availableSteps, specification.contexts...)
			for _, scenario := range specification.scenarios {
				availableSteps = append(availableSteps, scenario.steps...)
			}
		} else {
			fmt.Println(result.error.message)
		}

	}
}

func makeListOfAvailableSteps() {
	fileChan := make(chan string)
	go findScenarioFiles(fileChan)
	go parseScenarioFiles(fileChan)
}

func startAPIService() {
	http.HandleFunc("/steps", func(w http.ResponseWriter, r *http.Request) {
		js, err := json.Marshal(availableSteps)
		if err != nil {
			io.WriteString(w, err.Error())
		} else {
			w.Header()["Content-Type"] = []string{"application/json"}
			w.Write(js)
		}
	})
	log.Fatal(http.ListenAndServe(":8889", nil))
}

func showMessage(action, filename string) {
	fmt.Printf(" %s  %s\n", action, filename)
}

func createProjectTemplate(language string) error {
	if !common.IsASupportedLanguage(language) {
		return errors.New(fmt.Sprintf("%s is not a supported language", language))
	}

	// Create the project manifest
	showMessage("create", common.ManifestFile)
	if common.FileExists(common.ManifestFile) {
		showMessage("skip", common.ManifestFile)
	}
	manifest := &manifest{Language: language}
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	ioutil.WriteFile(common.ManifestFile, b, common.NewFilePermissions)

	// creating the spec directory
	showMessage("create", specsDirName)
	if !common.DirExists(specsDirName) {
		err = os.Mkdir(specsDirName, common.NewDirectoryPermissions)
		if err != nil {
			showMessage("error", fmt.Sprintf("Failed to create %s. %s", specsDirName, err.Error()))
		}
	} else {
		showMessage("skip", specsDirName)
	}

	// Copying the skeleton file
	skelFile, err := common.GetSkeletonFilePath(skelFileName)
	if err != nil {
		return err
	}
	specFile := path.Join(specsDirName, skelFileName)
	showMessage("create", specFile)
	if common.FileExists(specFile) {
		showMessage("skip", specFile)
	} else {
		err = common.CopyFile(skelFile, specFile)
		if err != nil {
			showMessage("error", fmt.Sprintf("Failed to create %s. %s", specFile, err.Error()))
		}
	}

	// Creating the env directory
	showMessage("create", common.EnvDirectoryName)
	if !common.DirExists(common.EnvDirectoryName) {
		err = os.Mkdir(common.EnvDirectoryName, common.NewDirectoryPermissions)
		if err != nil {
			showMessage("error", fmt.Sprintf("Failed to create %s. %s", common.EnvDirectoryName, err.Error()))
		}
	}
	defaultEnv := path.Join(common.EnvDirectoryName, envDefaultDirName)
	showMessage("create", defaultEnv)
	if !common.DirExists(defaultEnv) {
		err = os.Mkdir(defaultEnv, common.NewDirectoryPermissions)
		if err != nil {
			showMessage("error", fmt.Sprintf("Failed to create %s. %s", defaultEnv, err.Error()))
		}
	}
	defaultJson, err := common.GetSkeletonFilePath(path.Join(common.EnvDirectoryName, common.DefaultEnvFileName))
	if err != nil {
		return err
	}
	defaultJsonDest := path.Join(defaultEnv, common.DefaultEnvFileName)
	showMessage("create", defaultJsonDest)
	err = common.CopyFile(defaultJson, defaultJsonDest)
	if err != nil {
		showMessage("error", fmt.Sprintf("Failed to create %s. %s", defaultJsonDest, err.Error()))
	}

	return executeInitHookForRunner(language)
}

// Loads all the properties files available in the specified env directory
func loadEnvironment(env string) error {
	envDir, err := common.GetDirInProject(common.EnvDirectoryName)
	if err != nil {
		fmt.Printf("Failed to Load environment: %s\n", err.Error())
		os.Exit(1)
	}

	dirToRead := path.Join(envDir, env)
	if !common.DirExists(dirToRead) {
		return errors.New(fmt.Sprintf("%s is an invalid environment", env))
	}

	isProperties := func(fileName string) bool {
		return filepath.Ext(fileName) == ".properties"
	}

	err = filepath.Walk(dirToRead, func(path string, info os.FileInfo, err error) error {
		if isProperties(path) {
			p, e := properties.Load(path)
			if e != nil {
				return errors.New(fmt.Sprintf("Failed to parse: %s. %s", path, e.Error()))
			}

			for k, v := range p {
				err := common.SetEnvVariable(k, v)
				if err != nil {
					return errors.New(fmt.Sprintf("%s: %s", path, err.Error()))
				}
			}
		}
		return nil
	})

	return err
}

// Command line flags
var daemonize = flag.Bool("daemonize", false, "Run as a daemon")
var initialize = flag.String("init", "", "Initializes project structure in the current directory")
var currentEnv = flag.String("env", "default", "Specifies the environment")

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: twist [options] scenario\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func handleWarnings(result *parseResult) {
	if result.warnings != nil {
		for _, warning := range result.warnings {
			fmt.Println(fmt.Sprintf("[Warning] %s : %s", result.specFile, warning))
		}
	}
}

func main() {
	flag.Parse()

	if *daemonize {
		makeListOfAvailableSteps()
		startAPIService()
	} else if *initialize != "" {
		err := createProjectTemplate(*initialize)
		if err != nil {
			fmt.Printf("Failed to initialize. %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Println("Successfully initialized the project")
	} else {
		if len(flag.Args()) == 0 {
			printUsage()
		}

		// Loading default environment and loading user specified env
		// this way user specified env variable can override default if required
		err := loadEnvironment(envDefaultDirName)
		if err != nil {
			fmt.Printf("Failed to load the default environment. %s\n", err.Error())
			os.Exit(1)
		}

		if *currentEnv != envDefaultDirName {
			err := loadEnvironment(*currentEnv)
			if err != nil {
				fmt.Printf("Failed to load the environment: %s. %s\n", *currentEnv, err.Error())
				os.Exit(1)
			}
		}

		specSource := flag.Arg(0)

		//todo pass concept dictionary to the spec parsing
		concepts, conceptParseError := createConceptsDictionary()
		if conceptParseError != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		specs, specParseError := findSpecs(specSource, concepts)
		if specParseError != nil {
			fmt.Println(specParseError)
			os.Exit(1)
		}

		manifest := getProjectManifest()
		_, runnerError := startRunner(manifest)
		if runnerError != nil {
			fmt.Printf("Failed to start a runner. %s\n", runnerError.Error())
			os.Exit(1)
		}

		conn, connectionError := acceptConnection()
		if connectionError != nil {
			fmt.Printf("Failed to get a runner. %s\n", connectionError.Error())
			os.Exit(1)
		}

		execution := newExecution(manifest, specs, conn)
		validationErrors := execution.validate(concepts)
		if len(validationErrors) > 0 {
			fmt.Println("Validation failed. The following steps are not implemented")
			for _, stepValidationErrors := range validationErrors {
				for _, stepValidationError := range stepValidationErrors {
					s := stepValidationError.step
					fmt.Printf("\x1b[31;1m  %s:%d: %s\n\x1b[0m", stepValidationError.fileName, s.lineNo, s.lineText)
				}
			}
			err := execution.killProcess()
			if err != nil {
				fmt.Printf("Failed to kill the process. %s\n", err.Error())
			}
			os.Exit(1)
		} else {
			status := execution.start()
			exitCode := printExecutionStatus(status)
			os.Exit(exitCode)
		}
	}
}

func printExecutionStatus(status *testExecutionStatus) int {
	// Print out all the errors that happened during the execution
	// helps to view all the errors in one view
	noOfSpecificationsExecuted := len(status.specExecutionStatuses)
	noOfScenariosExecuted := 0
	noOfSpecificationsFailed := 0
	noOfScenariosFailed := 0
	exitCode := 0
	if status.isFailed() {
		fmt.Println("\nThe following failures occured:\n")
		exitCode = 1
	}

	for _, hookStatus := range status.hooksExecutionStatuses {
		if !hookStatus.GetPassed() {
			fmt.Printf("\x1b[31;1m%s\n\x1b[0m", hookStatus.GetErrorMessage())
			fmt.Printf("\x1b[31;1m%s\n\x1b[0m", hookStatus.GetStackTrace())
		}
	}

	for _, specExecStatus := range status.specExecutionStatuses {
		specFailing := false
		for _, hookStatus := range specExecStatus.hooksExecutionStatuses {
			if !hookStatus.GetPassed() {
				specFailing = true
				fmt.Printf("\x1b[31;1m%s\n\x1b[0m", specExecStatus.specification.fileName)
				fmt.Printf("\x1b[31;1m%s\n\x1b[0m", hookStatus.GetErrorMessage())
				fmt.Printf("\x1b[31;1m%s\n\x1b[0m", hookStatus.GetStackTrace())
			}
		}

		noOfScenariosExecuted += len(specExecStatus.scenariosExecutionStatuses[0])
		scenariosFailedInThisSpec := printScenarioExecutionStatus(specExecStatus.scenariosExecutionStatuses[0], specExecStatus.specification)
		if scenariosFailedInThisSpec > 0 {
			specFailing = true
			noOfScenariosFailed += scenariosFailedInThisSpec
		}

		if specFailing {
			noOfSpecificationsFailed += 1
		}
	}

	fmt.Printf("\n\n%d scenarios executed, %d failed\n", noOfScenariosExecuted, noOfScenariosFailed)
	fmt.Printf("%d specifications executed, %d failed\n", noOfSpecificationsExecuted, noOfSpecificationsFailed)
	return exitCode
}

func printScenarioExecutionStatus(scenariosExecStatuses []*scenarioExecutionStatus, specification *specification) int {
	noOfScenariosFailed := 0
	scenarioFailing := false
	for _, scenarioExecStatus := range scenariosExecStatuses {
		for _, hookStatus := range scenarioExecStatus.hooksExecutionStatuses {
			if !hookStatus.GetPassed() {
				scenarioFailing = true
				fmt.Printf("\x1b[31;1m%s:%s:%s\n\x1b[0m", specification.fileName,
					scenarioExecStatus.scenario.heading.value, hookStatus.GetErrorMessage())
			}
		}

		for _, stepExecStatus := range scenarioExecStatus.stepExecutionStatuses {
			for _, executionStatus := range stepExecStatus.executionStatus {
				if !executionStatus.GetPassed() {
					scenarioFailing = true
					fmt.Printf("\x1b[31;1m%s:%s\n\x1b[0m", specification.fileName, executionStatus.GetErrorMessage())
				}
			}
		}
		if scenarioFailing {
			noOfScenariosFailed += 1
		}
	}

	return noOfScenariosFailed
}

func findConceptFiles() []string {
	conceptsDir, err := common.GetDirInProject(common.ConceptsDirectoryName)
	if err != nil {
		fmt.Printf("Failed to find concepts directory. %s\n", err.Error())
		os.Exit(1)
	}

	return common.FindFilesInDir(conceptsDir, func(path string) bool {
		return filepath.Ext(path) == common.ConceptFileExtension
	})

}

func createConceptsDictionary() (*conceptDictionary, *parseError) {
	conceptFiles := findConceptFiles()
	conceptsDictionary := new(conceptDictionary)
	for _, conceptFile := range conceptFiles {
		if err := addConcepts(conceptFile, conceptsDictionary); err != nil {
			return nil, err
		}
	}
	return conceptsDictionary, nil
}

func addConcepts(conceptFile string, conceptDictionary *conceptDictionary) *parseError {
	fileText := common.ReadFileContents(conceptFile)
	concepts, err := new(conceptParser).parse(fileText)
	if err != nil {
		return err
	}
	err = conceptDictionary.add(concepts, conceptFile)
	return err
}

func findSpecs(specSource string, conceptDictionary *conceptDictionary) ([]*specification, *parseError) {
	specFiles := make([]string, 0)
	if common.DirExists(specSource) {
		specFiles = append(specFiles, findSpecsFilesIn(specSource)...)
	} else if common.FileExists(specSource) && isValidSpecExtension(specSource) {
		specFile, _ := filepath.Abs(specSource)
		specFiles = append(specFiles, specFile)
	} else {
		fmt.Printf("Spec file or directory does not exist: %s", specSource)
		os.Exit(1)
	}

	specs := make([]*specification, 0)
	for _, specFile := range specFiles {
		specFileContent := common.ReadFileContents(specFile)
		spec, parseResult := new(specParser).parse(specFileContent, conceptDictionary)
		spec.fileName = specFile
		if !parseResult.ok {
			return nil, parseResult.error
		}
		parseResult.specFile = specFile

		handleWarnings(parseResult)
		specs = append(specs, spec)
	}
	return specs, nil
}

func findSpecsFilesIn(dirRoot string) []string {
	absRoot, _ := filepath.Abs(dirRoot)
	return common.FindFilesInDir(absRoot, isValidSpecExtension)
}

func isValidSpecExtension(path string) bool {
	return acceptedExtensions[filepath.Ext(path)]
}
