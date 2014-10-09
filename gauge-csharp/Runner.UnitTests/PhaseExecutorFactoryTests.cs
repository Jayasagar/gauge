﻿using NUnit.Framework;

namespace Gauge.CSharp.Runner.UnitTests
{
    [TestFixture]
    public class PhaseExecutorFactoryTests
    {
        [Test]
        public void ShouldGetSetupPhaseExecutorForInit()
        {
            var phaseExecutor = PhaseExecutorFactory.GetExecutor("--init");
            Assert.AreEqual(phaseExecutor.GetType(), typeof(SetupPhaseExecutor));
        }

        [Test]
        public void ShouldGetStartPhaseExecutorByDefault()
        {
            var phaseExecutor = PhaseExecutorFactory.GetExecutor(default(string));
            Assert.AreEqual(phaseExecutor.GetType(), typeof(StartPhaseExecutor));
        }
    }
}