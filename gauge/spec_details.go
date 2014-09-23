package main

import (
	"github.com/getgauge/common"
	"sync"
	"time"
)

const refreshInterval = time.Duration(2) * time.Second

type specInfoGatherer struct {
	availableSpecs    []*specification
	availableStepsMap map[string]*stepValue
	stepsFromRunner   []string
	specStepMapCache  map[string][]*step
	mutex             sync.Mutex
}

func (specInfoGatherer *specInfoGatherer) makeListOfAvailableSteps(runner *testRunner) {
	specInfoGatherer.availableStepsMap = make(map[string]*stepValue)
	specInfoGatherer.specStepMapCache = make(map[string][]*step)
	specInfoGatherer.stepsFromRunner = specInfoGatherer.getStepsFromRunner(runner)
	specInfoGatherer.addStepValuesToAvailableSteps(specInfoGatherer.stepsFromRunner)
	newSpecStepMap := specInfoGatherer.getStepsFromSpecs()
	specInfoGatherer.addStepsToAvailableSteps(newSpecStepMap)
	go specInfoGatherer.refreshSteps(refreshInterval)
}

func (specInfoGatherer *specInfoGatherer) getStepsFromSpecs() map[string][]*step {
	specFiles := findSpecsFilesIn(common.SpecsDirectoryName)
	dictionary, _ := createConceptsDictionary(true)
	specInfoGatherer.availableSpecs = specInfoGatherer.parseSpecFiles(specFiles, dictionary)
	return specInfoGatherer.findAvailableStepsInSpecs(specInfoGatherer.availableSpecs)
}

func (specInfoGatherer *specInfoGatherer) refreshSteps(seconds time.Duration) {
	for {
		time.Sleep(seconds)
		specInfoGatherer.mutex.Lock()
		specInfoGatherer.availableStepsMap = make(map[string]*stepValue, 0)
		specInfoGatherer.addStepValuesToAvailableSteps(specInfoGatherer.stepsFromRunner)
		newSpecStepMap := specInfoGatherer.getStepsFromSpecs()
		specInfoGatherer.addStepsToAvailableSteps(newSpecStepMap)
		specInfoGatherer.mutex.Unlock()
	}
}

func (specInfoGatherer *specInfoGatherer) getStepsFromRunner(runner *testRunner) []string {
	steps := make([]string, 0)
	if runner == nil {
		runner, connErr := startRunnerAndMakeConnection(getProjectManifest())
		if connErr == nil {
			steps = append(steps, requestForSteps(runner)...)
			runner.kill()
		}
	} else {
		steps = append(steps, requestForSteps(runner)...)
	}
	return steps
}

func (specInfoGatherer *specInfoGatherer) parseSpecFiles(specFiles []string, dictionary *conceptDictionary) []*specification {
	specs := make([]*specification, 0)
	for _, file := range specFiles {
		specContent, err := common.ReadFileContents(file)
		if err != nil {
			continue
		}
		parser := new(specParser)
		specification, result := parser.parse(specContent, dictionary)

		if result.ok {
			specs = append(specs, specification)
		}
	}
	return specs
}

func (specInfoGatherer *specInfoGatherer) findAvailableStepsInSpecs(specs []*specification) map[string][]*step {
	specStepsMap := make(map[string][]*step)
	for _, spec := range specs {
		stepsInSpec := make([]*step, 0)
		stepsInSpec = append(stepsInSpec, spec.contexts...)
		for _, scenario := range spec.scenarios {
			stepsInSpec = append(stepsInSpec, scenario.steps...)
		}
		specStepsMap[spec.fileName] = stepsInSpec
	}
	return specStepsMap
}

func (specInfoGatherer *specInfoGatherer) addStepsToAvailableSteps(newSpecStepsMap map[string][]*step) {
	specInfoGatherer.updateCache(newSpecStepsMap)
	for _, steps := range specInfoGatherer.specStepMapCache {
		for _, step := range steps {
			stepValue, err := extractStepValueAndParams(step.lineText, step.hasInlineTable)
			if err == nil {
				if _, ok := specInfoGatherer.availableStepsMap[stepValue.stepValue]; !ok {
					specInfoGatherer.availableStepsMap[stepValue.stepValue] = stepValue
				}
			}
		}
	}

}

func (specInfoGatherer *specInfoGatherer) updateCache(newSpecStepsMap map[string][]*step) {
	for fileName, specsteps := range newSpecStepsMap {
		specInfoGatherer.specStepMapCache[fileName] = specsteps

	}
}

func (specInfoGatherer *specInfoGatherer) addStepValuesToAvailableSteps(stepValues []string) {
	for _, step := range stepValues {
		specInfoGatherer.addToAvailableSteps(step)
	}
}

func (specInfoGatherer *specInfoGatherer) addToAvailableSteps(stepText string) {
	stepValue, err := extractStepValueAndParams(stepText, false)
	if err == nil {
		if _, ok := specInfoGatherer.availableStepsMap[stepValue.stepValue]; !ok {
			specInfoGatherer.availableStepsMap[stepValue.stepValue] = stepValue
		}
	}
}

func (specInfoGatherer *specInfoGatherer) getAvailableSteps() []*stepValue {
	if specInfoGatherer.availableStepsMap == nil {
		specInfoGatherer.makeListOfAvailableSteps(nil)
	}
	specInfoGatherer.mutex.Lock()
	steps := make([]*stepValue, 0)
	for _, stepValue := range specInfoGatherer.availableStepsMap {
		steps = append(steps, stepValue)
	}
	specInfoGatherer.mutex.Unlock()
	return steps
}
