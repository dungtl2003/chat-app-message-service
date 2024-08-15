import express, {Express} from "express";
import {createServer, Server} from "node:http";
import cors from "cors";

interface Option {
    port?: number;
}

class ExpressServer {
    private static readonly PORT = 8000;

    private _app: Express;
    private _server: Server;
    private _port: number;

    public constructor(opts?: Option) {
        this._app = express();
        this._app.use(cors());

        this._port = opts?.port ?? ExpressServer.PORT;
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

    public instance(): Server {
        return this._server;
    }
}

export default ExpressServer;
