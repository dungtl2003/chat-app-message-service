import {PrismaClient} from "@prisma/client";
import config from "./config";
import ExpressServer from "./express-server";
import IdGeneratorService from "./services/id-generator/id-generator-service";

export default () => {
    const db = new PrismaClient();
    const expressServer = new ExpressServer(db, {port: config.serverPort});
    const idGeneratorService = new IdGeneratorService({debug: true});

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
