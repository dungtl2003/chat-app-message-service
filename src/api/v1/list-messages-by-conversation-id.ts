import {DbClient, Message} from "@/loaders/db/db";
import {Request, Response} from "express";
import expressAsyncHandler from "express-async-handler";
import {z} from "zod";
import {deserializeRequest} from "../helpers";
import {objectToString} from "@/utils/helpers";
import {castToBigint, castToNumber} from "@/utils/zod";

const ReqParamsSchema = z.object({
    conversationId: z
        .string({
            required_error: "conversation's ID is required",
            description: "conversation's ID that the message is belonged to",
        })
        .transform(castToBigint()),
});

const ReqQuerySchema = z.object({
    after: z
        .string({
            required_error: "after is required",
            description: "messages that have ID bigger than the given one",
        })
        .transform(castToBigint())
        .optional(),
    limit: z
        .string({
            required_error: "limit is required",
            description: "max amount of messages",
        })
        .transform(castToNumber())
        .refine((num) => num >= 0, "limit cannot be negative")
        .refine((num) => Number.isFinite(num), "limit must be a finite number")
        .refine((num) => Number.isInteger(num), "limit must be an integer")
        .optional(),
    orderBy: z.enum(["id:asc", "id:desc"]).optional(),
});

interface ResponseMessage {
    hasMore?: boolean;
    data?: Message[];
    error?: unknown;
}

function listMessagesByConversationId(db: DbClient) {
    return expressAsyncHandler(async function (
        request: Request<unknown, unknown, unknown, unknown>,
        response: Response<ResponseMessage>
    ): Promise<void> {
        const validationData = deserializeRequest(request, {
            params: ReqParamsSchema,
            query: ReqQuerySchema,
        });
        if (validationData.fieldErrors) {
            response.status(400).send({
                error: `invalid value: ${objectToString(validationData.fieldErrors)}`,
            });
            return;
        }
        const conversationId = validationData.data.params!.conversationId;
        const query = validationData.data.query!;

        const [error, messages] = await db.listMessagesByConversationId(
            conversationId,
            {
                after: query.after,
                limit: query.limit ? query.limit + 1 : undefined, // we add 1 more to check if the database has more messages than the amount of messages we requested. Later, we will remove the extra one before returning the result
                orderBy: query.orderBy,
            }
        );

        if (error) {
            console.error("Error: ", error.detail);
            response.status(error.status).send({
                error: error.message,
            });
            return;
        }

        const hasMore: boolean =
            query.limit !== undefined && messages.length > query.limit;
        if (hasMore) {
            messages.pop();
        }

        response.status(200).send({
            data: messages,
            hasMore,
        });
    });
}

export default listMessagesByConversationId;
export {ResponseMessage};
