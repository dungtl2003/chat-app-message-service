import {Message, Prisma, PrismaClient} from "@prisma/client";
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

function createMessage(db: PrismaClient) {
    return expressAsyncHandler(async function (
        request: Request<Params, unknown, unknown, ReqQuery>,
        response: Response<ResponseMessage>
    ): Promise<void> {
        const conversationId = request.params.conversationId;
        const {query} = request;
        const orderBy = query.orderBy ?? "id:asc";
        const [key, value] = orderBy.split(":");
        const whereQuery: Prisma.MessageWhereInput = {
            receiverId: conversationId,
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

        const messages = await db.message.findMany({
            take: query.limit ? +query.limit + 1 : undefined, // we add 1 more to check if the database has more messages than the amount of messages we requested. Later, we will remove the extra one before returning the result
            orderBy: {
                [key]: value,
            },
            where: {
                ...whereQuery,
            },
        });

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

export {ResponseMessage};
