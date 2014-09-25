﻿using System.IO;

namespace Gauge.CSharp.Lib
{
    public interface ITcpClientWrapper
    {
        bool Connected { get;}
        Stream GetStream();
        void Close();
    }
}