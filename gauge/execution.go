// This file is part of twist
package main

type execution struct {
	manifest             *manifest
	runner               *testRunner
	specifications       []*specification
	pluginHandler        *pluginHandler
	currentExecutionInfo *ExecutionInfo
	suiteResult          *suiteResult
}

type executionInfo struct {
	currentSpec specification
}

func newExecution(manifest *manifest, specifications []*specification, runner *testRunner, pluginHandler *pluginHandler) *execution {
	e := execution{manifest: manifest, specifications: specifications, runner: runner, pluginHandler: pluginHandler}
	return &e
}

func (e *execution) startExecution() *ProtoExecutionResult {
	message := &Message{MessageType: Message_ExecutionStarting.Enum(),
		ExecutionStartingRequest: &ExecutionStartingRequest{}}
	return e.executeHook(message)
}

func (e *execution) endExecution() *ProtoExecutionResult {
	message := &Message{MessageType: Message_ExecutionEnding.Enum(),
		ExecutionEndingRequest: &ExecutionEndingRequest{CurrentExecutionInfo: e.currentExecutionInfo}}
	return e.executeHook(message)
}

func (e *execution) executeHook(message *Message) *ProtoExecutionResult {
	e.pluginHandler.notifyPlugins(message)
	executionResult := executeAndGetStatus(e.runner.connection, message)
	e.addExecTime(executionResult.GetExecutionTime())
	return executionResult
}

func (e *execution) addExecTime(execTime int64) {
	e.suiteResult.executionTime += execTime
}

func (e *execution) notifyExecutionResult() {
	message := &Message{MessageType: Message_SuiteExecutionResult.Enum(),
		SuiteExecutionResult: &SuiteExecutionResult{SuiteResult: convertToProtoSuiteResult(e.suiteResult)}}
	e.pluginHandler.notifyPlugins(message)
}

func (e *execution) notifyExecutionStop() {
	message := &Message{MessageType: Message_KillProcessRequest.Enum(),
		KillProcessRequest: &KillProcessRequest{}}
	e.pluginHandler.notifyPlugins(message)
	e.pluginHandler.gracefullyKillPlugins()
}

func (e *execution) killProcess() error {
	message := &Message{MessageType: Message_KillProcessRequest.Enum(),
		KillProcessRequest: &KillProcessRequest{}}

	_, err := getResponse(e.runner.connection, message)
	return err
}

func (e *execution) killPlugins() {
	e.pluginHandler.gracefullyKillPlugins()
}

type executionValidationErrors map[*specification][]*stepValidationError

func (exe *execution) validate(conceptDictionary *conceptDictionary) executionValidationErrors {
	validationStatus := make(executionValidationErrors)
	for _, spec := range exe.specifications {
		executor := &specExecutor{specification: spec, runner: exe.runner, conceptDictionary: conceptDictionary}
		validationErrors := executor.validateSpecification()
		if len(validationErrors) != 0 {
			validationStatus[spec] = validationErrors
		}
	}
	if len(validationStatus) > 0 {
		return validationStatus
	} else {
		return nil
	}
}

func (exe *execution) start() *suiteResult {
	exe.suiteResult = newSuiteResult()
	beforeSuiteHookExecResult := exe.startExecution()
	if beforeSuiteHookExecResult.GetFailed() {
		addPreHook(exe.suiteResult, beforeSuiteHookExecResult)
		exe.suiteResult.setFailure()
	}
	for _, specificationToExecute := range exe.specifications {
		executor := newSpecExecutor(specificationToExecute, exe.runner, exe.pluginHandler)
		protoSpecResult := executor.execute()
		exe.suiteResult.addSpecResult(protoSpecResult)
	}
	afterSuiteHookExecResult := exe.endExecution()
	if afterSuiteHookExecResult.GetFailed() {
		addPostHook(exe.suiteResult, afterSuiteHookExecResult)
		exe.suiteResult.setFailure()
	}
	exe.notifyExecutionResult()
	exe.notifyExecutionStop()
	exe.runner.kill()
	return exe.suiteResult
}

func newSpecExecutor(specToExecute *specification, runner *testRunner, pluginHandler *pluginHandler) *specExecutor {
	specExecutor := new(specExecutor)
	specExecutor.initialize(specToExecute, runner, pluginHandler)
	return specExecutor
}
