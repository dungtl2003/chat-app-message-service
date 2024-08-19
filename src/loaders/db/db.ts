type MessageType =
    | "TEXT"
    | "IMAGE"
    | "VIDEO"
    | "AUDIO"
    | "FILE"
    | "GIF"
    | "STICKER"
    | "LOCATION"
    | "POLL";

type TableName = "message" | "attachment";

interface Message {
    id: bigint;
    senderId: bigint;
    receiverId: bigint;
    content: string;
    type: MessageType;
    createdAt: Date;
    updatedAt: Date | null;
    deletedAt: Date | null;
}

interface DbClient {
    listMessagesByConversationId: (
        receiverId: bigint,
        query: {
            after?: bigint;
            limit?: number;
            orderBy?: "id:asc" | "id:desc";
        }
    ) => Promise<[unknown, Message[]]>;

    insertMessage: (message: Message) => Promise<unknown>;

    insertMessages: (messages: Message[]) => Promise<unknown>;

    wipe: (table: TableName) => Promise<unknown>;
}

export {MessageType, TableName, Message, DbClient};
