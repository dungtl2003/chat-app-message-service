import path from "path";
import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import * as fs from "fs";
import {ProtoGrpcType} from "./proto/id_generator";
import {GenerateIdResponse__Output} from "./proto/proto/GenerateIdResponse";
import {IdGeneratorClient} from "./proto/proto/IdGenerator";
import {Service, ServiceStatus} from "../service";
import BaseLogger from "@/logging/logger";
import {serializeError} from "serialize-error";
import {Service as ServiceName} from "@/utils/types";

interface Configuration {
    protoFile?: string;
    ssl?: {
        rootCert: string;
        clientCert: string;
        clientKey: string;
    };
    initializeDeadlineInSeconds?: number;
    retries?: number;
}

class IdGeneratorService implements Service {
    private readonly _client: IdGeneratorClient;
    private readonly _logger: BaseLogger | undefined;
    private _status: ServiceStatus;

    constructor(
        idGeneratorServiceEndpoint: string,
        config?: Configuration,
        logger?: BaseLogger
    ) {
        this._status = ServiceStatus.INITIALIZING;
        this._logger = logger;

        this._logger?.logTrace("Validating configs", {
            service: "ID generator",
        });
        if (config && !this.validate(config)) {
            this._logger?.logFatal("Invalid options", {
                service: "ID generator",
            });
            process.exit(1);
        }

        const packageDef = protoLoader.loadSync(
            config?.protoFile ??
                path.resolve(__dirname, "./proto/id_generator.proto")
        );

        const grpcObj = grpc.loadPackageDefinition(
            packageDef
        ) as unknown as ProtoGrpcType;

        this._client = new grpcObj.proto.IdGenerator(
            idGeneratorServiceEndpoint,
            !config?.ssl
                ? grpc.credentials.createInsecure()
                : grpc.credentials.createSsl(
                      fs.readFileSync(config!.ssl!.rootCert),
                      fs.readFileSync(config!.ssl!.clientKey),
                      fs.readFileSync(config!.ssl!.clientCert)
                  )
        );
        this._logger?.logDebug("Service configs", {
            service: "ID generator",
            config: {
                ...config,
            },
        });

        this.connect(
            config?.initializeDeadlineInSeconds ?? 5,
            config?.retries ?? 5
        );
    }

    public getStatus(): ServiceStatus {
        return this._status;
    }

    public disconnect() {
        this._logger?.logInfo("Closed", {
            service: "ID generator",
        });
        if (this._status === ServiceStatus.INITIALIZED) {
            this._client.close();
        }
    }

    public generateId(callback: (error: boolean, id?: bigint) => void) {
        this._logger?.logTrace("Requesting for ID", {
            service: "ID generator",
        });

        if (this._status !== ServiceStatus.INITIALIZED) {
            this._logger?.logError("Service isn't initialized", {
                service: "ID generator",
            });
            callback(true);
            return;
        }

        this._client.generateId(
            {},
            (
                err: grpc.ServiceError | null,
                result: GenerateIdResponse__Output | undefined
            ) => {
                if (err) {
                    this._logger?.logError("Unexpected error", {
                        service: "ID generator",
                        error: serializeError(err),
                    });
                    callback(true);
                }

                this._logger?.logDebug("Received ID", {
                    service: "ID generator",
                    "ID received": result!.id,
                });
                callback(false, BigInt(result!.id!.toString(10)));
            }
        );
    }

    private validate(opt: Configuration): boolean {
        if (
            opt.initializeDeadlineInSeconds &&
            opt.initializeDeadlineInSeconds < 0
        ) {
            return false;
        }

        if (opt.retries && opt.retries < 0) {
            return false;
        }

        return true;
    }

    private async connect(
        initializeDeadlineInSeconds: number,
        maxRetries: number,
        attempt: number = 0
    ) {
        const deadline = new Date();
        deadline.setSeconds(
            deadline.getSeconds() + initializeDeadlineInSeconds
        );

        this._client.waitForReady(deadline, (err) => {
            if (err) {
                this._logger?.logError("Failed to connect", {
                    service: "ID generator",
                    error: serializeError(err),
                });
                if (attempt < maxRetries) {
                    attempt++;
                    this._logger?.logDebug(`Retrying (${attempt})`, {
                        service: "ID generator",
                    });
                    this.connect(
                        initializeDeadlineInSeconds,
                        maxRetries,
                        attempt
                    );
                } else {
                    this._status = ServiceStatus.INITIALIZE_FAILED;
                    process.exit(1);
                }
            } else {
                this._logger?.logInfo("Service is ready", {
                    service: "ID generator",
                });
                this._status = ServiceStatus.INITIALIZED;
            }
        });
    }

    public toJSON(): ServiceName {
        return "ID generator";
    }
}

export default IdGeneratorService;
