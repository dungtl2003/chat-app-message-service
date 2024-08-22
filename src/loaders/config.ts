import {Environment} from "@/utils/types";

const env: Environment = (process.env.NODE_ENV as Environment) ?? "development";
console.log(`Running app in ${env} environment`);

process.env.PORT = process.env.PORT ?? "8040";

export default {
    serverPort: parseInt(process.env.PORT, 10),
    env: env,
};
