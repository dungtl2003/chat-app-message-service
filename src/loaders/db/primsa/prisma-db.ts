import {$Enums, Prisma, PrismaClient} from "@prisma/client";
import {DbClient, Message, MessageType, TableName} from "../db";

class PrismaDb implements DbClient {
    private readonly _db;

    constructor() {
        this._db = new PrismaClient();
    }

    public async wipe(table: TableName): Promise<unknown> {
        try {
            switch (table) {
                case "message":
                    await this._db.message.deleteMany();
                    break;
                case "attachment":
                    await this._db.attachment.deleteMany();
                    break;
            }

            return null;
        } catch (error) {
            return error;
        }
    }

    public async insertMessage(message: Message): Promise<unknown> {
        try {
            await this._db.message.create({
                data: {
                    id: message.id,
                    senderId: message.senderId,
                    receiverId: message.receiverId,
                    content: message.content,
                    type: message.type as $Enums.MessageType,
                    createdAt: message.createdAt,
                    updatedAt: message.updatedAt,
                    deletedAt: message.deletedAt,
                },
            });
            return null;
        } catch (error) {
            return error;
        }
    }

    public async insertMessages(messages: Message[]): Promise<unknown> {
        try {
            await this._db.message.createMany({
                data: messages.map((message) => {
                    return {
                        id: message.id,
                        senderId: message.senderId,
                        receiverId: message.receiverId,
                        content: message.content,
                        type: message.type as $Enums.MessageType,
                        createdAt: message.createdAt,
                        updatedAt: message.updatedAt,
                        deletedAt: message.deletedAt,
                    };
                }),
            });

            return null;
        } catch (error) {
            return error;
        }
    }

    public async listMessagesByConversationId(
        receiverId: bigint,
        query: {
            after?: bigint;
            limit?: number;
            orderBy?: "id:asc" | "id:desc";
        }
    ): Promise<[unknown, Message[]]> {
        const orderBy = query.orderBy ?? "id:asc";
        const [key, value] = orderBy.split(":");
        const whereQuery: Prisma.MessageWhereInput = {
            receiverId: receiverId,
        };

        if (query.after) {
            switch (orderBy) {
                case "id:desc":
                    whereQuery.id = {
                        lt: query.after,
                    };
                    break;
                case "id:asc":
                    whereQuery.id = {
                        gt: query.after,
                    };
                    break;
            }
        }

        try {
            const messages = await this._db.message.findMany({
                take: query.limit,
                orderBy: {
                    [key]: value,
                },
                where: {
                    ...whereQuery,
                },
            });

            return [
                null,
                messages.map((msg) => {
                    return {
                        id: msg.id,
                        senderId: msg.senderId,
                        receiverId: msg.receiverId,
                        content: msg.content,
                        type: msg.type as MessageType,
                        createdAt: msg.createdAt,
                        updatedAt: msg.updatedAt,
                        deletedAt: msg.deletedAt,
                    } as Message;
                }),
            ];
        } catch (error) {
            return [error, []];
        }
    }
}

export default PrismaDb;
