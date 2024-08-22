import {Service} from "@/utils/types";

type LogMessage = string;

type LogContext = {
    [K: string]: any;
} & {
    service?: Service;
};

enum LogLevel {
    DEBUG = "debug",
    INFO = "info",
    TRACE = "trace",
    WARN = "warn",
    ERROR = "error",
    FATAL = "fatal",
}

interface BaseLogger {
    logInfo(msg: LogMessage, context?: LogContext): void;
    logTrace(msg: LogMessage, context?: LogContext): void;
    logWarn(msg: LogMessage, context?: LogContext): void;
    logError(msg: LogMessage, context?: LogContext): void;
    logDebug(msg: LogMessage, context?: LogContext): void;
    logFatal(msg: LogMessage, context?: LogContext): void;
}

export default BaseLogger;
export {LogMessage, LogContext, LogLevel};
