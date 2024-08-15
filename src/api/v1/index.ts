import express from "express";
import listMessagesByConversationId from "./list-messages-by-conversation-id";
import {PrismaClient} from "@prisma/client";

export default (db: PrismaClient) => {
    const router = express.Router();

    router.get(
        "/conversations/:conversationId/messages",
        listMessagesByConversationId(db)
    );

    return router;
};
