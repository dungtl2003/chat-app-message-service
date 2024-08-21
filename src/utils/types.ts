type ApiError = {
    status: number;
    detail: unknown;
    message: string;
} | null;

export {ApiError};
