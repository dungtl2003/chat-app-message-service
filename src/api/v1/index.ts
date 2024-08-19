import express from "express";
import listMessagesByConversationId from "./list-messages-by-conversation-id";
import {DbClient} from "@/loaders/db/db";

export default (db: DbClient) => {
    const router = express.Router();

    router.get(
        "/conversations/:conversationId/messages",
        listMessagesByConversationId(db)
    );

    return router;
};
