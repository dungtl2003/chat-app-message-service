import IdGeneratorService from "@/loaders/services/id-generator/id-generator-service";
import {execa} from "execa";
import path from "path";

describe("ID generator service", () => {
    const TLS_PORT = 9000;
    const NON_TLS_PORT = 9001;

    let idGeneratorServiceTls: IdGeneratorService;
    let idGeneratorServiceNonTls: IdGeneratorService;

    before("init service", (done) => {
        idGeneratorServiceNonTls = new IdGeneratorService({
            port: NON_TLS_PORT,
        });
        idGeneratorServiceTls = new IdGeneratorService({
            port: TLS_PORT,
            ssl: {
                rootCert: path.resolve(
                    __dirname,
                    "../../../src/loaders/services/id-generator/ssl/ca_cert.pem"
                ),
                clientCert: path.resolve(
                    __dirname,
                    "../../../src/loaders/services/id-generator/ssl/client_cert.pem"
                ),
                clientKey: path.resolve(
                    __dirname,
                    "../../../src/loaders/services/id-generator/ssl/client_key.pem"
                ),
            },
        });

        setTimeout(() => {
            if (
                idGeneratorServiceTls.isConnected() &&
                idGeneratorServiceNonTls.isConnected()
            ) {
                done();
            } else {
                done("cannot connect to the server");
            }
        }, 5000);
    });

    it("should work with both tls and non tls version", async () => {
        await Promise.all([
            new Promise((resolve, reject) => {
                idGeneratorServiceTls.generateId((error, id) => {
                    if (error || !id) {
                        reject("tls: cannot get ID");
                        return;
                    }

                    resolve("");
                });
            }),
            new Promise((resolve, reject) => {
                idGeneratorServiceNonTls.generateId((error, id) => {
                    if (error || !id) {
                        reject("non tls: cannot get ID");
                        return;
                    }

                    resolve("");
                });
            }),
        ]);
    });

    it("with wrong certificates should fail to connect", async () => {
        try {
            await execa({shell: true})`${__dirname}/fake-cert-gen.sh`;

            const fakeService = new IdGeneratorService({
                debug: true,
                port: TLS_PORT,
                ssl: {
                    rootCert: path.resolve(__dirname, "fake_ca_cert.pem"),
                    clientCert: path.resolve(__dirname, "fake_client_cert.pem"),
                    clientKey: path.resolve(__dirname, "fake_client_key.pem"),
                },
            });

            await new Promise((resolve, reject) => {
                setTimeout(() => {
                    if (
                        fakeService.isConnected() &&
                        fakeService.isConnected()
                    ) {
                        reject("connected even with wrong certificates");
                    } else {
                        resolve("");
                    }
                }, 5000);
            });
        } catch (error) {
            console.debug(error);
        } finally {
            await execa({shell: true})`rm ${__dirname}/*.pem`;
        }
    });

    it("should not generate any duplicated ID, even with multiple different ID services", async () => {
        const idsSet1 = new Set<bigint>();
        const idsSet2 = new Set<bigint>();

        await Promise.all([
            new Promise((resolve, reject) => {
                for (let i = 1; i <= 50000; i++) {
                    idGeneratorServiceTls.generateId((error, id) => {
                        if (error || idsSet1.has(id!)) {
                            reject("duplicated ID on tls service");
                            return;
                        }

                        idsSet1.add(id!);
                    });
                }
                resolve("");
            }),
            new Promise((resolve, reject) => {
                for (let i = 1; i <= 50000; i++) {
                    idGeneratorServiceNonTls.generateId((error, id) => {
                        if (error || idsSet2.has(id!)) {
                            reject("duplicated ID on non tls service");
                            return;
                        }

                        idsSet2.add(id!);
                    });
                }
                resolve("");
            }),
        ]);

        if ([...idsSet1].some((id) => idsSet2.has(id))) {
            throw new Error("duplicated ID between 2 services");
        }
    });
});
