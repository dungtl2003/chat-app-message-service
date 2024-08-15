import config from "./config";
import ExpressServer from "./express-server";

export default () => {
    const expressServer = new ExpressServer({port: config.serverPort});

    expressServer.listen();

    process
        .on("exit", () => {
            expressServer.close();
        })
        .on("SIGINT", () => {
            expressServer.close();
        });
};
