type ApiError = {
    status: number;
    detail: unknown;
    message: string;
} | null;

type Environment = "development" | "production";

type Service = "ID generator" | "Express";

export {ApiError, Environment, Service};
