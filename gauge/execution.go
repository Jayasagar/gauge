// This file is part of twist
package main

import "net"

type execution struct {
	manifest             *manifest
	connection           net.Conn
	specifications       []*specification
	pluginHandler        *pluginHandler
	currentExecutionInfo *ExecutionInfo
}

type executionInfo struct {
	currentSpec specification
}

func newExecution(manifest *manifest, specifications []*specification, conn net.Conn, pluginHandler *pluginHandler) *execution {
	e := execution{manifest: manifest, specifications: specifications, connection: conn, pluginHandler: pluginHandler}
	return &e
}

func (e *execution) startExecution() *ExecutionStatus {
	message := &Message{MessageType: Message_ExecutionStarting.Enum(),
		ExecutionStartingRequest: &ExecutionStartingRequest{}}

	e.pluginHandler.notifyPlugins(message)
	return executeAndGetStatus(e.connection, message)
}

func (e *execution) endExecution() *ExecutionStatus {
	message := &Message{MessageType: Message_ExecutionEnding.Enum(),
		ExecutionEndingRequest: &ExecutionEndingRequest{CurrentExecutionInfo: e.currentExecutionInfo}}

	e.pluginHandler.notifyPlugins(message)
	return executeAndGetStatus(e.connection, message)
}

func (e *execution) notifyExecutionResult(testExecutionStatus *testExecutionStatus) {
	protoSpecs := make([]*ProtoSpec, 0)
	for _, spec := range testExecutionStatus.specifications {
		protoSpec := convertToProtoSpec(spec)
		protoSpecs = append(protoSpecs, protoSpec)
	}
	message := &Message{MessageType: Message_SuiteExecutionResult.Enum(),
		SuiteExecutionResult: &SuiteExecutionResult{Specs: protoSpecs}}
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

	_, err := getResponse(e.connection, message)
	return err
}

func (e *execution) killPlugins() {
	e.pluginHandler.gracefullyKillPlugins()
}

type testExecutionStatus struct {
	specifications         []*specification
	specExecutionStatuses  []*specExecutionStatus
	hooksExecutionStatuses []*ExecutionStatus
}

func (t *testExecutionStatus) isFailed() bool {
	if t.hooksExecutionStatuses != nil {
		for _, s := range t.hooksExecutionStatuses {
			if !s.GetPassed() {
				return true
			}
		}
	}

	for _, specStatus := range t.specExecutionStatuses {
		if specStatus.isFailed() {
			return true
		}
	}

	return false
}

type executionValidationErrors map[*specification][]*stepValidationError

func (exe *execution) validate(conceptDictionary *conceptDictionary) executionValidationErrors {
	validationStatus := make(executionValidationErrors)
	for _, spec := range exe.specifications {
		executor := &specExecutor{specification: spec, connection: exe.connection, conceptDictionary: conceptDictionary}
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

func (exe *execution) start() *testExecutionStatus {
	testExecutionStatus := &testExecutionStatus{specifications: exe.specifications}
	beforeSuiteHookExecStatus := exe.startExecution()
	if beforeSuiteHookExecStatus.GetPassed() {
		for _, specificationToExecute := range exe.specifications {
			executor := newSpecExecutor(specificationToExecute, exe.connection, exe.pluginHandler)
			specExecutionStatus := executor.execute()
			testExecutionStatus.specExecutionStatuses = append(testExecutionStatus.specExecutionStatuses, specExecutionStatus)
		}
	}

	afterSuiteHookExecStatus := exe.endExecution()
	testExecutionStatus.hooksExecutionStatuses = append(testExecutionStatus.hooksExecutionStatuses, beforeSuiteHookExecStatus, afterSuiteHookExecStatus)
	exe.notifyExecutionResult(testExecutionStatus)
	exe.notifyExecutionStop()
	return testExecutionStatus
}

func newSpecExecutor(specToExecute *specification, connection net.Conn, pluginHandler *pluginHandler) *specExecutor {
	specExecutor := new(specExecutor)
	specExecutor.initialize(specToExecute, connection, pluginHandler)
	return specExecutor
}
