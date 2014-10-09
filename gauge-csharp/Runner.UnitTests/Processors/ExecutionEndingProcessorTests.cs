﻿using System.Collections.Generic;
using System.Reflection;
using Gauge.CSharp.Runner.Processors;
using main;
using Moq;
using NUnit.Framework;

namespace Gauge.CSharp.Runner.UnitTests.Processors
{
    [TestFixture]
    public class ExecutionEndingProcessorTests
    {
        private ExecutionEndingProcessor _executionEndingProcessor;
        private Message _request;
        private Mock<IMethodExecutor> _mockMethodExecutor;
        private ProtoExecutionResult _protoExecutionResult;

        public void Foo()
        {
        }

        [SetUp]
        public void Setup()
        {
            var mockHookRegistry = new Mock<IHookRegistry>();
            var hooks = new HashSet<MethodInfo> {GetType().GetMethod("Foo")};
            mockHookRegistry.Setup(x => x.AfterSuiteHooks).Returns(hooks);
            var executionEndingRequest = ExecutionEndingRequest.DefaultInstance;
            _request = Message.CreateBuilder()
                            .SetMessageId(20)
                            .SetMessageType(Message.Types.MessageType.ExecutionEnding)
                            .SetExecutionEndingRequest(executionEndingRequest)
                            .Build();

            _mockMethodExecutor = new Mock<IMethodExecutor>();
            _protoExecutionResult = ProtoExecutionResult.CreateBuilder().SetExecutionTime(0).SetFailed(false).Build();
            _mockMethodExecutor.Setup(x => x.ExecuteHooks(hooks, executionEndingRequest.CurrentExecutionInfo))
                .Returns(_protoExecutionResult);
            _executionEndingProcessor = new ExecutionEndingProcessor(mockHookRegistry.Object, _mockMethodExecutor.Object);
        }
        [Test]
        public void ShouldProcessHooks()
        {
            var message = _executionEndingProcessor.Process(_request);

            _mockMethodExecutor.VerifyAll();
        }

        [Test]
        public void ShouldWrapInMessage()
        {
            var message = _executionEndingProcessor.Process(_request);
            
            Assert.AreEqual(message.MessageId, _request.MessageId);
            Assert.AreEqual(message.MessageType, Message.Types.MessageType.ExecutionStatusResponse);
            Assert.AreSame(message.ExecutionStatusResponse.ExecutionResult, _protoExecutionResult);
        }
    }
}