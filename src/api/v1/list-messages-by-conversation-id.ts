import {DbClient, Message} from "@/loaders/db/db";
import {Request, Response} from "express";
import expressAsyncHandler from "express-async-handler";

interface ResponseMessage {
    hasMore?: boolean;
    data?: Message[];
    error?: unknown;
}

interface Params {
    conversationId: bigint;
}

interface ReqQuery {
    after?: bigint;
    limit?: number;
    orderBy?: "id:asc" | "id:desc";
}

function listMessagesByConversationId(db: DbClient) {
    return expressAsyncHandler(async function (
        request: Request<Params, unknown, unknown, ReqQuery>,
        response: Response<ResponseMessage>
    ): Promise<void> {
        const conversationId = request.params.conversationId;
        const {query} = request;

        const [error, messages] = await db.listMessagesByConversationId(
            conversationId,
            {
                after: query.after,
                limit: query.limit ? +query.limit + 1 : undefined, // we add 1 more to check if the database has more messages than the amount of messages we requested. Later, we will remove the extra one before returning the result
                orderBy: query.orderBy,
            }
        );

        if (error) {
            console.error("Error: ", error);
            response.status(500).send({
                error: "server error",
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
