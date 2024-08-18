import express, {Express} from "express";
import {createServer, Server} from "node:http";
import cors from "cors";
import errorHandler from "@/utils/error-handler";
import v1 from "@/api/v1";
import {PrismaClient} from "@prisma/client";
import healthcheck from "@/api/healthcheck";
import {Service} from "./services/service";

interface Option {
    port?: number;
    services?: Service[];
}

class ExpressServer {
    private static readonly PORT = 8000;

    private _app: Express;
    private _server: Server;
    private _port: number;

    public constructor(db: PrismaClient, opts?: Option) {
        this._app = express();
        this._port = opts?.port ?? ExpressServer.PORT;

        this._app.use(cors());
        this._app.use(errorHandler());
        this._app.use("/api/v1", v1(db));
        this._app.use("/api/health", healthcheck(opts?.services));

        this._server = createServer(this._app);
    }

    public listen(): void {
        this._server.listen(this._port, () => {
            console.log(
                `[express server]: Server is running at port ${this._port}`
            );
        });
    }

    public close(): void {
        this._server.close((error) => {
            if (error) throw error;

            console.log("[express server]: Stopped");
        });
    }
}

export default ExpressServer;
