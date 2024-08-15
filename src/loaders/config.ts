process.env.NODE_ENV = process.env.NODE_ENV || "development";
const env = process.env.NODE_ENV;
console.log(`Running app in ${env} environment`);

process.env.PORT = process.env.PORT || "8040";

export default {
    serverPort: parseInt(process.env.PORT, 10),
};
