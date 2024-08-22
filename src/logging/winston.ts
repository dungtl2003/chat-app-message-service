import winston, {format} from "winston";
import DailyRotateFile from "winston-daily-rotate-file";
import BaseLogger, {LogContext, LogLevel, LogMessage} from "./logger";
import {error} from "node:console";

class WistonLogger implements BaseLogger {
    private _logger: winston.Logger;
    private static _appName = "message-service";

    constructor(level?: LogLevel) {
        this._logger = this._initializeWinston(level ?? LogLevel.INFO);
    }

    public logInfo(msg: LogMessage, context?: LogContext) {
        this._log(msg, LogLevel.INFO, context);
    }
    public logTrace(msg: LogMessage, context?: LogContext) {
        this._log(msg, LogLevel.TRACE, context);
    }
    public logWarn(msg: LogMessage, context?: LogContext) {
        this._log(msg, LogLevel.WARN, context);
    }
    public logError(msg: LogMessage, context?: LogContext) {
        this._log(msg, LogLevel.ERROR, context);
    }
    public logFatal(msg: LogMessage, context?: LogContext) {
        this._log(msg, LogLevel.FATAL, context);
    }
    public logDebug(msg: LogMessage, context?: LogContext) {
        if (process.env.NODE_ENV !== "production") {
            this._log(msg, LogLevel.DEBUG, context);
        }
    }

    private _log(msg: LogMessage, level: LogLevel, context?: LogContext) {
        this._logger.log(level, msg, {context: context});
    }

    private _initializeWinston(level: LogLevel) {
        winston.addColors({
            fatal: "redBG white bold",
            error: "red bold",
            warn: "yellow italic",
            info: "green underline",
            debug: "blue dim",
            trace: "magenta",
        });
        const logger = winston.createLogger({
            levels: {
                fatal: 0,
                error: 1,
                warn: 2,
                info: 3,
                debug: 4,
                trace: 5,
            },
            level: level,
            transports: WistonLogger._getTransports(),
        });

        return logger;
    }

    private static _getTransports() {
        const transports: Array<any> = [
            new winston.transports.Console({
                format: this._getMyFormatForConsole(),
            }),
        ];

        if (process.env.NODE_ENV === "production") {
            transports.push(this._getFileTransport()); // Also log file in production
        }

        return transports;
    }

    private static _getMyFormatForConsole() {
        return format.combine(
            format.timestamp(), // Add a timestamp to the log
            format.printf((info) => {
                const logObject = {
                    host: process.platform,
                    application: "message service",
                    timestamp: info.timestamp,
                    level: info.level,
                    pid: process.pid,
                    message: info.message,
                    context: info.context ? info.context : {},
                };
                return JSON.stringify(logObject, null, 2); // Pretty-print JSON with indentation
            }),
            format.colorize({all: true})
        );
    }

    private static _getFileTransport() {
        const transport = new DailyRotateFile({
            filename: `${WistonLogger._appName}-%DATE%.log`,
            zippedArchive: true, // Compress gzip
            maxSize: "10m", // Rotate after 10MB
            maxFiles: "14d", // Only keep last 14 days
            format: format.combine(
                format.timestamp(),
                format((info) => {
                    info.app = this._appName;
                    return info;
                })(),
                format.json()
            ),
        });

        transport.on("error", (error) => {
            console.error("Error writing to log file: ", error);
        });
        transport.on("rotate", (oldFileName, newFileName) => {
            console.info(`Rotated from ${oldFileName} to ${newFileName}`);
        });

        return transport;
    }
}

export default WistonLogger;
