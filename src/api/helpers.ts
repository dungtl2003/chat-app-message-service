import {Request} from "express";
import {z, ZodTypeAny} from "zod";

type FieldErrors = {
    [K: string]: string[];
};

type ReturnType<P extends ZodTypeAny, ReqQuery extends ZodTypeAny> =
    | {
          fieldErrors: FieldErrors;
          data?: null;
      }
    | {
          fieldErrors?: null;
          data: {
              params?: z.infer<P>;
              query?: z.infer<ReqQuery>;
          };
      };

function deserializeRequest<P extends ZodTypeAny, ReqQuery extends ZodTypeAny>(
    request: Request<unknown, unknown, unknown, unknown>,
    schema: {
        params?: P;
        query?: ReqQuery;
    }
): ReturnType<P, ReqQuery> {
    const deserializedData: ReturnType<P, ReqQuery> = {data: {}};

    if (schema.params) {
        const validationParams = schema.params.safeParse(request.params);
        if (!validationParams.success) {
            return {
                fieldErrors: validationParams.error.flatten()
                    .fieldErrors as FieldErrors,
            };
        }
        deserializedData.data = {
            ...deserializedData.data,
            params: validationParams.data,
        };
    }

    if (schema.query) {
        const validationQuery = schema.query.safeParse(
            request.query as ReqQuery
        );
        if (!validationQuery.success) {
            return {
                fieldErrors: validationQuery.error.flatten()
                    .fieldErrors as FieldErrors,
            };
        } else {
            deserializedData.data = {
                ...deserializedData.data,
                query: validationQuery.data,
            };
        }
    }

    return deserializedData;
}

export {deserializeRequest};
