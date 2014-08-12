package com.thoughtworks.gauge;

import com.google.protobuf.CodedInputStream;
import com.google.protobuf.CodedOutputStream;
import main.Messages;
import org.reflections.Configuration;
import org.reflections.Reflections;
import org.reflections.scanners.MethodAnnotationsScanner;
import org.reflections.util.ClasspathHelper;
import org.reflections.util.ConfigurationBuilder;

import java.io.*;
import java.lang.reflect.Method;
import java.net.Socket;
import java.util.HashMap;
import java.util.Set;

public class GaugeRuntime {

    private static MessageLength getMessageLength(InputStream is) throws IOException {
        CodedInputStream codedInputStream = CodedInputStream.newInstance(is);
        long size = codedInputStream.readRawVarint64();
        return new MessageLength(size, codedInputStream);
    }

    private static byte[] toBytes(MessageLength messageLength) throws IOException {
        long messageSize = messageLength.length;
        CodedInputStream stream = messageLength.remainingStream;
        ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
        for (int i = 0; i < messageSize; i++) {
            outputStream.write(stream.readRawByte());
        }

        return outputStream.toByteArray();
    }

    private static void writeMessage(Socket socket, Messages.Message message) throws IOException {
        ByteArrayOutputStream stream = new ByteArrayOutputStream();
        CodedOutputStream cos = CodedOutputStream.newInstance(stream);
        byte[] bytes = message.toByteArray();
        cos.writeRawVarint64(bytes.length);
        cos.flush();
        stream.write(bytes);
        socket.getOutputStream().write(stream.toByteArray());
        socket.getOutputStream().flush();
    }

    private static void dispatchMessages(Socket socket, HashMap<Messages.Message.MessageType, IMessageProcessor> messageProcessors) throws Exception {
        InputStream inputStream = socket.getInputStream();
        while (!socket.isClosed()) {
            try {
                MessageLength messageLength = getMessageLength(inputStream);
                byte[] bytes = toBytes(messageLength);
                Messages.Message message = Messages.Message.parseFrom(bytes);
                if (!messageProcessors.containsKey(message.getMessageType())) {
                    System.out.println("Invalid message");
                } else {
                    IMessageProcessor messageProcessor = messageProcessors.get(message.getMessageType());
                    Messages.Message response = messageProcessor.process(message);
                    writeMessage(socket, response);
                    if (message.getMessageType() == Messages.Message.MessageType.ExecutionEnding
                            || message.getMessageType() == Messages.Message.MessageType.KillProcessRequest) {
                        socket.close();
                        break;
                    }
                }
            } catch (Throwable throwable) {
                throwable.printStackTrace();
                System.out.println(throwable.toString());
            }
        }
    }

    private static Socket connect(String portEnvVariable) {
        String gaugePort = System.getenv(portEnvVariable);

        if (gaugePort == null || gaugePort.equalsIgnoreCase("")) {
            throw new RuntimeException(portEnvVariable + " not set");
        }
        int port = Integer.parseInt(gaugePort);
        Socket clientSocket;
        for (; ; ) {
            try {
                clientSocket = new Socket("localhost", port);
                break;
            } catch (Exception e) {
            }
        }

        return clientSocket;
    }

    public static void main(String[] args) throws Exception {
        Socket gaugeSocket = connect(GaugeConstant.GAUGE_INTERNAL_PORT);
        Socket apiSocket = connect(GaugeConstant.GAUGE_API_PORT);
        final GaugeConnection gaugeApiConnection = new GaugeConnection(apiSocket);

        HashMap<Messages.Message.MessageType, IMessageProcessor> messageProcessors = new HashMap<Messages.Message.MessageType, IMessageProcessor>() {{
            put(Messages.Message.MessageType.ExecutionStarting, new SuiteExecutionStartingProcessor());
            put(Messages.Message.MessageType.ExecutionEnding, new SuiteExecutionEndingProcessor());
            put(Messages.Message.MessageType.SpecExecutionStarting, new SpecExecutionStartingProcessor());
            put(Messages.Message.MessageType.SpecExecutionEnding, new SpecExecutionEndingProcessor());
            put(Messages.Message.MessageType.ScenarioExecutionStarting, new ScenarioExecutionStartingProcessor());
            put(Messages.Message.MessageType.ScenarioExecutionEnding, new ScenarioExecutionEndingProcessor());
            put(Messages.Message.MessageType.StepExecutionStarting, new StepExecutionStartingProcessor());
            put(Messages.Message.MessageType.StepExecutionEnding, new StepExecutionEndingProcessor());
            put(Messages.Message.MessageType.ExecuteStep, new ExecuteStepProcessor());
            put(Messages.Message.MessageType.StepValidateRequest, new ValidateStepProcessor());
            put(Messages.Message.MessageType.StepNamesRequest, new StepNamesRequestProcessor());
            put(Messages.Message.MessageType.SuiteDataStoreInit, new DataStoreInitializer());
            put(Messages.Message.MessageType.SpecDataStoreInit, new DataStoreInitializer());
            put(Messages.Message.MessageType.ScenarioDataStoreInit, new DataStoreInitializer());
            put(Messages.Message.MessageType.KillProcessRequest, new KillProcessProcessor());
        }};

        scanForStepImplementations(gaugeApiConnection);

        dispatchMessages(gaugeSocket, messageProcessors);
    }

    private static void scanForHooks(Reflections reflections) {
        HooksRegistry.setBeforeSuiteHooks(reflections.getMethodsAnnotatedWith(BeforeSuite.class));
        HooksRegistry.setAfterSuiteHooks(reflections.getMethodsAnnotatedWith(AfterSuite.class));
        HooksRegistry.setBeforeSpecHooks(reflections.getMethodsAnnotatedWith(BeforeSpec.class));
        HooksRegistry.setAfterSpecHooks(reflections.getMethodsAnnotatedWith(AfterSpec.class));
        HooksRegistry.setBeforeScenarioHooks(reflections.getMethodsAnnotatedWith(BeforeScenario.class));
        HooksRegistry.setAfterScenarioHooks(reflections.getMethodsAnnotatedWith(AfterScenario.class));
        HooksRegistry.setBeforeStepHooks(reflections.getMethodsAnnotatedWith(BeforeStep.class));
        HooksRegistry.setAfterStepHooks(reflections.getMethodsAnnotatedWith(AfterStep.class));
    }

    private static void scanForStepImplementations(GaugeConnection gaugeApiConnection) {
        Configuration config = new ConfigurationBuilder()
                .setScanners(new MethodAnnotationsScanner())
                .addUrls(ClasspathHelper.forJavaClassPath());
        Reflections reflections = new Reflections(config);
        Set<Method> stepImplementations = reflections.getMethodsAnnotatedWith(Step.class);
        for (Method method : stepImplementations) {
            Step annotation = method.getAnnotation(Step.class);
            if (annotation != null) {
                for (String stepName : annotation.value()) {
                    StepValue stepValue = gaugeApiConnection.getStepValue(stepName);
                    StepRegistry.addStepImplementation(stepValue, method);
                }
            }
        }
        scanForHooks(reflections);
    }

}
