import {z} from "zod";
import {containsOnlyDigits} from "./helpers";

function castToNumber() {
    return function (
        arg: string,
        ctx: z.RefinementCtx
    ): number | Promise<number> {
        if (!containsOnlyDigits(arg)) {
            ctx.addIssue({
                code: z.ZodIssueCode.invalid_type,
                expected: "number",
                received: "nan",
                message: "must be a number",
            });
            return z.NEVER;
        }

        return Number(arg);
    };
}

function castToBigint() {
    return function (
        arg: string,
        ctx: z.RefinementCtx
    ): bigint | Promise<bigint> {
        if (!containsOnlyDigits(arg)) {
            ctx.addIssue({
                code: z.ZodIssueCode.invalid_type,
                expected: "bigint",
                received: "nan",
                message: "must be a bigint",
            });
            return z.NEVER;
        }

        return BigInt(arg);
    };
}

export {castToBigint, castToNumber};
