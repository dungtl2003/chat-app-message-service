import {NextFunction, Request, Response} from "express";

function errorHandler() {
    return function (err: any, _: Request, res: Response, next: NextFunction) {
        console.error("Unhandled error: ", err);
        res.status(500).send("server error");
    };
}

export default errorHandler;
