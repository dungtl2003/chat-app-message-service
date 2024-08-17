import type * as grpc from "@grpc/grpc-js";
import type {MessageTypeDefinition} from "@grpc/proto-loader";

import type {
    IdGeneratorClient as _proto_IdGeneratorClient,
    IdGeneratorDefinition as _proto_IdGeneratorDefinition,
} from "./proto/IdGenerator";

type SubtypeConstructor<
    Constructor extends new (...args: any) => any,
    Subtype,
> = {
    new (...args: ConstructorParameters<Constructor>): Subtype;
};

export interface ProtoGrpcType {
    proto: {
        GenerateIdRequest: MessageTypeDefinition;
        GenerateIdResponse: MessageTypeDefinition;
        IdGenerator: SubtypeConstructor<
            typeof grpc.Client,
            _proto_IdGeneratorClient
        > & {service: _proto_IdGeneratorDefinition};
    };
}
