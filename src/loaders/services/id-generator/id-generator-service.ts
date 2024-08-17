import path from "path";
import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import * as fs from "fs";
import {ProtoGrpcType} from "./proto/id_generator";
import {GenerateIdResponse__Output} from "./proto/proto/GenerateIdResponse";
import {IdGeneratorClient} from "./proto/proto/IdGenerator";

interface Option {
    port?: number;
    protoFile?: string;
    ssl?: {
        rootCert: string;
        clientCert: string;
        clientKey: string;
    };
    debug?: boolean;
}

class IdGeneratorService {
    private readonly PORT = 9000;

    private readonly _client: IdGeneratorClient;
    private readonly _debug: boolean;
    private _connected: boolean;

    constructor(opt?: Option) {
        this._connected = false;
        this._debug = opt?.debug ?? false;

        const packageDef = protoLoader.loadSync(
            opt?.protoFile ??
                path.resolve(__dirname, "./proto/id_generator.proto")
        );

        const grpcObj = grpc.loadPackageDefinition(
            packageDef
        ) as unknown as ProtoGrpcType;

        this._client = new grpcObj.proto.IdGenerator(
            `localhost:${opt?.port ?? this.PORT}`,
            !opt?.ssl
                ? grpc.credentials.createInsecure()
                : grpc.credentials.createSsl(
                      fs.readFileSync(opt!.ssl!.rootCert),
                      fs.readFileSync(opt!.ssl!.clientKey),
                      fs.readFileSync(opt!.ssl!.clientCert)
                  )
        );

        const deadline = new Date();
        deadline.setSeconds(deadline.getSeconds() + 5);
        this._client.waitForReady(deadline, () => {
            console.log("[id generator]: Ready");
            this._connected = true;
        });
    }

    public isConnected(): boolean {
        return this._connected;
    }

    public disconnect() {
        console.log("[id generator]: Closed");
        this._client.close();
    }

    public generateId(callback: (error: boolean, id?: bigint) => void) {
        this._debug && console.debug("[id generator]: Requesting for ID...");

        if (!this._connected) {
            this._debug &&
                console.debug("[id generator]: Server does not connect");
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
}

export default IdGeneratorService;
