// Original file: /home/ilikeblue/Personal/Project/chat-app/service/message/src/loaders/services/id-generator/proto/id_generator.proto

import type * as grpc from "@grpc/grpc-js";
import type {MethodDefinition} from "@grpc/proto-loader";
import type {
    GenerateIdRequest as _proto_GenerateIdRequest,
    GenerateIdRequest__Output as _proto_GenerateIdRequest__Output,
} from "../proto/GenerateIdRequest";
import type {
    GenerateIdResponse as _proto_GenerateIdResponse,
    GenerateIdResponse__Output as _proto_GenerateIdResponse__Output,
} from "../proto/GenerateIdResponse";

export interface IdGeneratorClient extends grpc.Client {
    GenerateId(
        argument: _proto_GenerateIdRequest,
        metadata: grpc.Metadata,
        options: grpc.CallOptions,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
    GenerateId(
        argument: _proto_GenerateIdRequest,
        metadata: grpc.Metadata,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
    GenerateId(
        argument: _proto_GenerateIdRequest,
        options: grpc.CallOptions,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
    GenerateId(
        argument: _proto_GenerateIdRequest,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
    generateId(
        argument: _proto_GenerateIdRequest,
        metadata: grpc.Metadata,
        options: grpc.CallOptions,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
    generateId(
        argument: _proto_GenerateIdRequest,
        metadata: grpc.Metadata,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
    generateId(
        argument: _proto_GenerateIdRequest,
        options: grpc.CallOptions,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
    generateId(
        argument: _proto_GenerateIdRequest,
        callback: grpc.requestCallback<_proto_GenerateIdResponse__Output>
    ): grpc.ClientUnaryCall;
}

export interface IdGeneratorHandlers extends grpc.UntypedServiceImplementation {
    GenerateId: grpc.handleUnaryCall<
        _proto_GenerateIdRequest__Output,
        _proto_GenerateIdResponse
    >;
}

export interface IdGeneratorDefinition extends grpc.ServiceDefinition {
    GenerateId: MethodDefinition<
        _proto_GenerateIdRequest,
        _proto_GenerateIdResponse,
        _proto_GenerateIdRequest__Output,
        _proto_GenerateIdResponse__Output
    >;
}
