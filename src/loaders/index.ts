import config from "./config";
import ExpressServer from "./express-server";
import IdGeneratorService from "./services/id-generator/id-generator-service";
import PrismaDb from "./db/primsa/prisma-db";

export default () => {
    const dbClient = new PrismaDb();
    const idGeneratorService = new IdGeneratorService("localhost:9000", {
        debug: true,
    });
    const expressServer = new ExpressServer(dbClient, {
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
