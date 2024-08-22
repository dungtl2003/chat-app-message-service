import express, {Express} from "express";
import {createServer, Server} from "node:http";
import cors from "cors";
import errorHandler from "@/utils/error-handler";
import v1 from "@/api/v1";
import healthcheck from "@/api/healthcheck";
import {Service} from "./services/service";
import {DbClient} from "./db/db";
import BaseLogger from "@/logging/logger";
import {serializeError} from "serialize-error";

interface Configuration {
    port?: number;
    services?: Service[];
}

class ExpressServer {
    private static readonly PORT = 8000;

    private readonly _app: Express;
    private readonly _server: Server;
    private readonly _port: number;
    private readonly _logger: BaseLogger | undefined;

    public constructor(
        db: DbClient,
        config?: Configuration,
        logger?: BaseLogger
    ) {
        this._logger?.logTrace("Validating configs", {
            service: "Express",
        });
        if (config && !this.validate(config)) {
            this._logger?.logFatal("Invalid options", {
                service: "Express",
            });
            process.exit(1);
        }

        this._logger = logger;
        this._app = express();
        this._port = config?.port ?? ExpressServer.PORT;

        this._app.use(cors());
        this._app.use(errorHandler());
        this._app.use("/api/v1", v1(db));
        this._app.use("/api/health", healthcheck(config?.services));

        this._server = createServer(this._app);
        this._logger?.logDebug("Server configs", {
            service: "Express",
            config: {
                ...config,
            },
        });
    }

    public listen(): void {
        this._server.listen(this._port, () => {
            this._logger?.logInfo(`Server is running at port ${this._port}`, {
                service: "Express",
            });
        });
    }

    public close(): void {
        this._server.close((error) => {
            if (error) {
                this._logger?.logError("Unexpected error", {
                    service: "Express",
                    error: serializeError(error),
                });
            } else {
                this._logger?.logInfo("Stopped", {
                    service: "Express",
                });
            }
        });
    }

    private validate(config: Configuration): boolean {
        if (config.port && config.port < 0) {
            return false;
        }

        return true;
    }
}

export default ExpressServer;
