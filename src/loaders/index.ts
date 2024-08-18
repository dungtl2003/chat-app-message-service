import {PrismaClient} from "@prisma/client";
import config from "./config";
import ExpressServer from "./express-server";
import IdGeneratorService from "./services/id-generator/id-generator-service";

export default () => {
    const db = new PrismaClient({
        errorFormat: "pretty",
    });
    const idGeneratorService = new IdGeneratorService("localhost:9000", {
        debug: true,
    });
    const expressServer = new ExpressServer(db, {
        port: config.serverPort,
        services: [idGeneratorService],
    });

    expressServer.listen();

    process
        .on("exit", () => {
            expressServer.close();
            idGeneratorService.disconnect();
        })
        .on("SIGINT", () => {
            expressServer.close();
            idGeneratorService.disconnect();
        });
};
