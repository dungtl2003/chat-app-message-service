package model

import "dungtl2003/chat-app-message-service/internal/types"

// model Participant {
//     id BigInt @id @db.BigInt
//
//     name String
//     joinedAt DateTime @default(now()) @map("joined_at")
//     leftAt DateTime? @map("left_at")
//
//     user ChatUser @relation(fields: [userId], references: [id], onDelete: Cascade)
//     userId BigInt @db.BigInt @map("user_id")
//     conversation Conversation @relation(fields: [conversationId], references: [id], onDelete: Cascade)
//     conversationId BigInt @db.BigInt @map("conversation_id")
//
//     @@map("participant")
//     @@schema("conversation")
// }

type Participant struct {
	Id             types.JsonInt64    `json:"id"`
	Name           string             `json:"name"`
	JoinedAt       types.JsonTime     `json:"joined_at"`
	LeftAt         types.JsonNullTime `json:"left_at"`
	UserId         types.JsonInt64    `json:"user_id"`
	ConversationId types.JsonInt64    `json:"conversation_id"`
}
