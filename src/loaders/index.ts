import config from "./config";
import ExpressServer from "./express-server";
import IdGeneratorService from "./services/id-generator/id-generator-service";
import PrismaDb from "./db/primsa/prisma-db";
import BaseLogger, {LogLevel} from "@/logging/logger";
import WistonLogger from "@/logging/winston";

export default () => {
    const logger: BaseLogger = new WistonLogger(
        config.env == "development" ? LogLevel.TRACE : LogLevel.INFO
    );
    const dbClient = new PrismaDb();
    const idGeneratorService = new IdGeneratorService(
        "localhost:9000",
        {retries: 4},
        logger
    );
    const expressServer = new ExpressServer(
        dbClient,
        {
            port: config.serverPort,
            services: [idGeneratorService],
        },
        logger
    );

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
