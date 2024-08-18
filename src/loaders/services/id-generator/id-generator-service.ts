import path from "path";
import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import * as fs from "fs";
import {ProtoGrpcType} from "./proto/id_generator";
import {GenerateIdResponse__Output} from "./proto/proto/GenerateIdResponse";
import {IdGeneratorClient} from "./proto/proto/IdGenerator";
import {Service, ServiceStatus} from "../service";

interface Option {
    protoFile?: string;
    ssl?: {
        rootCert: string;
        clientCert: string;
        clientKey: string;
    };
    debug?: boolean;
    initializeDeadlineInSeconds?: number;
    retries?: number;
}

class IdGeneratorService implements Service {
    private readonly _client: IdGeneratorClient;
    private readonly _debug: boolean;
    private _status: ServiceStatus;

    constructor(idGeneratorServiceEndpoint: string, opt?: Option) {
        if (opt && !this.validate(opt)) {
            console.error("[id generator]: invalid options");
            this._status = ServiceStatus.INITIALIZE_FAILED;
            return;
        }

        this._status = ServiceStatus.INITIALIZING;
        this._debug = opt?.debug ?? false;

        const packageDef = protoLoader.loadSync(
            opt?.protoFile ??
                path.resolve(__dirname, "./proto/id_generator.proto")
        );

        const grpcObj = grpc.loadPackageDefinition(
            packageDef
        ) as unknown as ProtoGrpcType;

        this._client = new grpcObj.proto.IdGenerator(
            idGeneratorServiceEndpoint,
            !opt?.ssl
                ? grpc.credentials.createInsecure()
                : grpc.credentials.createSsl(
                      fs.readFileSync(opt!.ssl!.rootCert),
                      fs.readFileSync(opt!.ssl!.clientKey),
                      fs.readFileSync(opt!.ssl!.clientCert)
                  )
        );

        this.connect(opt?.initializeDeadlineInSeconds ?? 5, opt?.retries ?? 5);
    }

    public getStatus(): ServiceStatus {
        return this._status;
    }

    public disconnect() {
        console.log("[id generator]: Closed");
        if (this._status === ServiceStatus.INITIALIZED) {
            this._client.close();
        }
    }

    public generateId(callback: (error: boolean, id?: bigint) => void) {
        this._debug && console.debug("[id generator]: Requesting for ID...");

        if (this._status !== ServiceStatus.INITIALIZED) {
            this._debug &&
                console.debug("[id generator]: Server isn't initialized");
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
                    this._debug &&
                        console.debug("[id generator]: Error: ", err);
                    callback(true);
                }

                callback(false, BigInt(result!.id!.toString(10)));
            }
        );
    }

    private validate(opt: Option): boolean {
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
        retries: number
    ) {
        const deadline = new Date();
        deadline.setSeconds(
            deadline.getSeconds() + initializeDeadlineInSeconds
        );

        this._client.waitForReady(deadline, (err) => {
            if (err) {
                console.error(
                    "[id generator]: Initialize failed. Error: ",
                    err
                );
                if (retries > 0) {
                    console.log("[id generator]: Retrying...");
                    this.connect(initializeDeadlineInSeconds, retries - 1);
                } else {
                    this._status = ServiceStatus.INITIALIZE_FAILED;
                }
            } else {
                console.log("[id generator]: Ready");
                this._status = ServiceStatus.INITIALIZED;
            }
        });
    }
}

export default IdGeneratorService;
