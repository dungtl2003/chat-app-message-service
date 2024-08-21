import {ApiError} from "@/utils/types";
import {z} from "zod";

const MessageTypeSchema = z.enum([
    "TEXT",
    "IMAGE",
    "VIDEO",
    "AUDIO",
    "FILE",
    "GIF",
    "STICKER",
    "LOCATION",
    "POLL",
]);

const MessageSchema = z.object({
    id: z
        .bigint({
            invalid_type_error: "message's ID must have type bigint",
            required_error: "message's ID is required",
            description: "message's ID",
        })
        .nonnegative("message's ID must be non-negative"),
    senderId: z
        .bigint({
            invalid_type_error: "sender's ID must have type bigint",
            required_error: "sender's ID is required",
            description: "ID of the user who sent this message",
        })
        .nonnegative("sender's ID must be non-negative"),
    receiverId: z
        .bigint({
            invalid_type_error: "receiver's ID must have type bigint",
            required_error: "receiver's ID is required",
            description: "ID of the conversation received this message",
        })
        .nonnegative("receiver's ID must be non-negative"),
    content: z.string({
        invalid_type_error: "content must have type string",
        required_error: "content is required",
        description: "content of this message",
    }),
    type: MessageTypeSchema,
    createdAt: z
        .string({
            description: "the time this message was created",
        })
        .datetime(),
    updatedAt: z
        .string({
            description: "the time this message was updated",
        })
        .datetime()
        .nullable(),
    deletedAt: z
        .string({
            description: "the time this message was deleted",
        })
        .datetime()
        .nullable(),
});

type MessageType = z.infer<typeof MessageTypeSchema>;
type Message = z.infer<typeof MessageSchema>;
type TableName = "message" | "attachment";

interface DbClient {
    listMessagesByConversationId: (
        receiverId: bigint,
        query: {
            after?: bigint;
            limit?: number;
            orderBy?: "id:asc" | "id:desc";
        }
    ) => Promise<[ApiError, Message[]]>;

    insertMessage: (message: Message) => Promise<ApiError>;

    insertMessages: (messages: Message[]) => Promise<ApiError>;

    getMessageById: (messageId: bigint) => Promise<[ApiError, Message | null]>;

    wipe: (table: TableName) => Promise<ApiError>;
}

export {MessageType, TableName, Message, DbClient};
