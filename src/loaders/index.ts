import {PrismaClient} from "@prisma/client";
import config from "./config";
import ExpressServer from "./express-server";

export default () => {
    const db = new PrismaClient();
    const expressServer = new ExpressServer(db, {port: config.serverPort});

    expressServer.listen();

    process
        .on("exit", () => {
            expressServer.close();
        })
        .on("SIGINT", () => {
            expressServer.close();
        });
};
