// Original file: /home/ilikeblue/Personal/Project/chat-app/service/message/src/loaders/services/id-generator/proto/id_generator.proto

import type {Long} from "@grpc/proto-loader";

export interface GenerateIdResponse {
    id?: number | string | Long;
}

export interface GenerateIdResponse__Output {
    id?: Long;
}
