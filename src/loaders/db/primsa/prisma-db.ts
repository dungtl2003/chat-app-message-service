import {$Enums, Prisma, PrismaClient} from "@prisma/client";
import {DbClient, Message, MessageType, TableName} from "../db";
import {ApiError} from "@/utils/types";

class PrismaDb implements DbClient {
    private readonly _db;

    constructor() {
        this._db = new PrismaClient();
    }

    public async getMessageById(
        messageId: bigint
    ): Promise<[ApiError, Message | null]> {
        try {
            const message = await this._db.message.findFirst({
                where: {
                    id: messageId,
                },
            });

            if (!message) {
                return [
                    {
                        status: 400,
                        message: `message with ID ${messageId} does not exist in database`,
                        detail: `message with ID ${messageId} does not exist in database`,
                    },
                    null,
                ];
            }

            return [
                null,
                {
                    id: message.id,
                    senderId: message.senderId,
                    receiverId: message.receiverId,
                    content: message.content,
                    type: message.type as MessageType,
                    createdAt: message.createdAt.toJSON(),
                    updatedAt: message.updatedAt?.toJSON(),
                    deletedAt: message.deletedAt?.toJSON(),
                } as Message,
            ];
        } catch (error) {
            return [
                {status: 500, message: "server error", detail: error},
                null,
            ];
        }
    }

    public async wipe(table: TableName): Promise<ApiError> {
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
            return {status: 500, message: "server error", detail: error};
        }
    }

    public async insertMessage(message: Message): Promise<ApiError> {
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
            return {status: 500, message: "server error", detail: error};
        }
    }

    public async insertMessages(messages: Message[]): Promise<ApiError> {
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
            return {status: 500, message: "server error", detail: error};
        }
    }

    public async listMessagesByConversationId(
        receiverId: bigint,
        query: {
            after?: bigint;
            limit?: number;
            orderBy?: "id:asc" | "id:desc";
        }
    ): Promise<[ApiError | null, Message[]]> {
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
                        createdAt: msg.createdAt.toJSON(),
                        updatedAt: msg.updatedAt?.toJSON() ?? null,
                        deletedAt: msg.deletedAt?.toJSON() ?? null,
                    } as Message;
                }),
            ];
        } catch (error) {
            return [{status: 500, message: "server error", detail: error}, []];
        }
    }
}

export default PrismaDb;
