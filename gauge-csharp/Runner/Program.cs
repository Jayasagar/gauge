﻿using System;
using System.Reflection;

namespace Gauge.CSharp.Runner
{
    internal static class Program
    {
        private static void Main(string[] args)
        {
            if (args.Length == 0)
            {
                Console.WriteLine("usage: {0} --<start|init>", AppDomain.CurrentDomain.FriendlyName);
                Environment.Exit(1);
            }
            var phase = args[0];
            var phaseExecutor = PhaseExecutorFactory.GetExecutor(phase);
            phaseExecutor.Execute();
        }
    }
}