import "@/patch";
import ExpressServer from "@/loaders/express-server";
import {getRandomInt} from "@/utils/helpers";
import {assert} from "chai";
import {ResponseMessage} from "@/api/v1/list-messages-by-conversation-id";
import PrismaDb from "@/loaders/db/primsa/prisma-db";
import {DbClient, Message} from "@/loaders/db/db";

async function addTempMessages(
    db: DbClient,
    size: number,
    conversationId: bigint
) {
    const tempMessages: Message[] = Array(size)
        .fill(0)
        .map((_, i) => {
            const message = {
                id: BigInt(i),
                senderId: BigInt(getRandomInt(1, 9999)),
                receiverId: conversationId,
                content: String(i),
                type: "TEXT",
                createdAt: new Date().toJSON(),
                updatedAt: null,
                deletedAt: null,
            } as Message;

            return message;
        });

    await db.insertMessages(tempMessages);

    return tempMessages;
}

describe("list messages by conversation ID endpoint", () => {
    const PORT = 8040;
    const API_VERSION = "v1";

    let db: PrismaDb;
    let expressServer: ExpressServer;

    before("init server", () => {
        db = new PrismaDb();

        expressServer = new ExpressServer(db, {port: PORT});
        expressServer.listen();
    });

    after("shutdown server and clean database", async () => {
        expressServer.close();
    });

    afterEach("clean database", async () => {
        await db.wipe("message");
    });

    it("should return all messages of the conversation in correct order if not specific limit or the limit is bigger than total existed messages", async () => {
        const messages = await addTempMessages(db, 500, 2n);

        let response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:asc",
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        let responsePayload: ResponseMessage = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(500, responsePayload.data!.length);
        responsePayload.data!.forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i]),
                JSON.stringify(msg)
            );
        });

        response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:desc",
                    limit: "600",
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        responsePayload = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(500, responsePayload.data!.length);
        responsePayload.data!.reverse().forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i]),
                JSON.stringify(msg)
            );
        });
    });

    it("should work with a very big conversation ID", async () => {
        const veryBigConversationId = 9223372036854775800n; // close to 2^63
        const messages = await addTempMessages(db, 500, veryBigConversationId);

        let response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/${veryBigConversationId}/messages?${new URLSearchParams(
                {
                    orderBy: "id:asc",
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        let responsePayload: ResponseMessage = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(500, responsePayload.data!.length);
        responsePayload.data!.forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i]),
                JSON.stringify(msg)
            );
        });
    });

    it("should not work with non number conversation ID", async () => {
        await addTempMessages(db, 500, 2n);

        let response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/nonnumber/messages?${new URLSearchParams(
                {
                    orderBy: "id:asc",
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 400);
    });

    it("should return correct amount of messages of the conversation in correct order", async () => {
        const messages = await addTempMessages(db, 500, 2n);

        let response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:asc",
                    limit: "200",
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        let responsePayload: ResponseMessage = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(200, responsePayload.data!.length);
        responsePayload.data!.forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i]),
                JSON.stringify(msg)
            );
        });

        response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:desc",
                    limit: "200", // from 300n to 499n
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        responsePayload = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(200, responsePayload.data!.length);
        responsePayload.data!.reverse().forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i + 300]),
                JSON.stringify(msg)
            );
        });
    });

    it("should return no message if conversation does not exist", async () => {
        let response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:asc",
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        let responsePayload: ResponseMessage = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(0, responsePayload.data!.length);
    });

    it("should return messages after a given message ID in correct order", async () => {
        const messages = await addTempMessages(db, 500, 2n);

        let response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:asc",
                    after: "199", // from 200
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        let responsePayload: ResponseMessage = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(300, responsePayload.data!.length);
        responsePayload.data!.forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i + 200]),
                JSON.stringify(msg)
            );
        });

        response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:desc",
                    after: "201", // from 200
                    limit: "100",
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        responsePayload = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(100, responsePayload.data!.length);
        responsePayload.data!.reverse().forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i + 101]),
                JSON.stringify(msg)
            );
        });

        response = await fetch(
            `http://localhost:${PORT}/api/${API_VERSION}/conversations/2/messages?${new URLSearchParams(
                {
                    orderBy: "id:desc",
                    after: "201", // from 200
                    limit: "300", // should not matter
                }
            )}`,
            {
                method: "GET",
                headers: {
                    Accept: "application/json, text/plain",
                    "Content-Type": "application/json;charset=UTF-8",
                },
            }
        );

        assert.strictEqual(response.status, 200);
        responsePayload = await response.json();
        assert.isNotNull(responsePayload.data);
        assert.strictEqual(201, responsePayload.data!.length);
        responsePayload.data!.reverse().forEach((msg, i) => {
            assert.strictEqual(
                JSON.stringify(messages[i]),
                JSON.stringify(msg)
            );
        });
    });
});
